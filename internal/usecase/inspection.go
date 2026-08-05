package usecase

import (
	"context"
	"errors"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// データベーススキーマ取得
func (u *AppUseCase) GetDatabaseSchema(ctx context.Context) (domain.Profile, domain.Schema, error) {
	profiles, activeID, err := u.repository.LoadProfiles()
	if err != nil {
		return domain.Profile{}, domain.Schema{}, err
	}
	if activeID == nil {
		return domain.Profile{}, domain.Schema{}, apperr.New(apperr.CodeProfileNotFound)
	}

	profile, found := findProfile(profiles, *activeID)
	if !found {
		return domain.Profile{}, domain.Schema{}, apperr.New(apperr.CodeProfileNotFound)
	}

	credential, found, err := u.repository.GetCredential(profile.ID)
	if err != nil {
		return domain.Profile{}, domain.Schema{}, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}
	if !found {
		return domain.Profile{}, domain.Schema{}, apperr.Wrap(apperr.CodeCredentialUnavailable, errors.New("credential not found"))
	}

	schema, err := u.repository.InspectSchema(ctx, profile, credential)
	if err != nil {
		return domain.Profile{}, domain.Schema{}, apperr.Wrap(apperr.CodeSchemaLoadFailed, err)
	}
	if err := schema.Validate(schemaNamespace(profile)); err != nil {
		return domain.Profile{}, domain.Schema{}, apperr.Wrap(apperr.CodeSchemaLoadFailed, err)
	}

	return profile, schema, nil
}

// テーブル構造取得
func (u *AppUseCase) GetTableStructure(ctx context.Context, table string) (domain.TableStructure, error) {
	profiles, activeID, err := u.repository.LoadProfiles()
	if err != nil {
		return domain.TableStructure{}, err
	}
	if activeID == nil {
		return domain.TableStructure{}, apperr.New(apperr.CodeProfileNotFound)
	}

	profile, found := findProfile(profiles, *activeID)
	if !found {
		return domain.TableStructure{}, apperr.New(apperr.CodeProfileNotFound)
	}
	ref, err := domain.NewTableRef(schemaNamespace(profile), table)
	if err != nil {
		return domain.TableStructure{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	credential, found, err := u.repository.GetCredential(profile.ID)
	if err != nil {
		return domain.TableStructure{}, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}
	if !found {
		return domain.TableStructure{}, apperr.Wrap(apperr.CodeCredentialUnavailable, errors.New("credential not found"))
	}

	structure, err := u.repository.InspectTableStructure(ctx, profile, credential, ref)
	if err != nil {
		return domain.TableStructure{}, apperr.Wrap(apperr.CodeSchemaLoadFailed, err)
	}
	if err := structure.Validate(ref); err != nil {
		return domain.TableStructure{}, apperr.Wrap(apperr.CodeSchemaLoadFailed, err)
	}

	return structure, nil
}

// テーブル統計取得
func (u *AppUseCase) GetTableStatistics(ctx context.Context, table string) (domain.TableStatistics, error) {
	profiles, activeID, err := u.repository.LoadProfiles()
	if err != nil {
		return domain.TableStatistics{}, err
	}
	if activeID == nil {
		return domain.TableStatistics{}, apperr.New(apperr.CodeProfileNotFound)
	}

	profile, found := findProfile(profiles, *activeID)
	if !found {
		return domain.TableStatistics{}, apperr.New(apperr.CodeProfileNotFound)
	}
	ref, err := domain.NewTableRef(schemaNamespace(profile), table)
	if err != nil {
		return domain.TableStatistics{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	credential, found, err := u.repository.GetCredential(profile.ID)
	if err != nil {
		return domain.TableStatistics{}, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}
	if !found {
		return domain.TableStatistics{}, apperr.Wrap(apperr.CodeCredentialUnavailable, errors.New("credential not found"))
	}

	statistics, err := u.repository.InspectTableStatistics(ctx, profile, credential, ref)
	if err != nil {
		if contextError := apperr.FromContextError(err); contextError != nil {
			if contextError.Code == apperr.CodeOperationTimeout {
				statistics.Status = domain.StatisticsStatusTimeout

				return statistics, nil
			}

			return domain.TableStatistics{}, contextError
		}

		return domain.TableStatistics{}, apperr.Wrap(apperr.CodeStatsLoadFailed, err)
	}

	return statistics, nil
}

// 対象名前空間取得
func schemaNamespace(profile domain.Profile) string {
	if profile.DBType == domain.DBTypeMySQL {
		return profile.Database
	}

	return profile.Schema
}
