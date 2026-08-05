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

// テーブル行一覧取得
func (u *AppUseCase) ListTableRows(ctx context.Context, query domain.TableQuery) (domain.TableRows, error) {
	profiles, activeID, err := u.repository.LoadProfiles()
	if err != nil {
		return domain.TableRows{}, err
	}

	if activeID == nil {
		return domain.TableRows{}, apperr.New(apperr.CodeProfileNotFound)
	}

	profile, found := findProfile(profiles, *activeID)
	if !found {
		return domain.TableRows{}, apperr.New(apperr.CodeProfileNotFound)
	}

	ref, err := domain.NewTableRef(schemaNamespace(profile), query.Table.Name)
	if err != nil || query.Page < 1 {
		return domain.TableRows{}, apperr.Wrap(apperr.CodeValidationFailed, domain.ErrInvalidTableQuery)
	}

	credential, found, err := u.repository.GetCredential(profile.ID)
	if err != nil {
		return domain.TableRows{}, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}

	if !found {
		return domain.TableRows{}, apperr.Wrap(apperr.CodeCredentialUnavailable, errors.New("credential not found"))
	}

	structure, err := u.repository.InspectTableStructure(ctx, profile, credential, ref)
	if err != nil {
		return domain.TableRows{}, apperr.Wrap(apperr.CodeDataLoadFailed, err)
	}

	query.Table = ref
	query.Columns = structure.Table.Columns
	if query.Sort != nil && (!tableQueryHasColumn(query.Columns, query.Sort.Column) || (query.Sort.Direction != domain.SortDirectionAscending && query.Sort.Direction != domain.SortDirectionDescending)) {
		return domain.TableRows{}, apperr.Wrap(apperr.CodeSortApplyFailed, domain.ErrInvalidTableQuery)
	}

	if err := query.Validate(); err != nil {
		if errors.Is(err, domain.ErrInvalidTableFilter) {
			return domain.TableRows{}, apperr.Wrap(apperr.CodeFilterApplyFailed, err)
		}

		return domain.TableRows{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	rows, err := u.repository.ListRows(ctx, profile, credential, query)
	if err != nil {
		return domain.TableRows{}, apperr.Wrap(apperr.CodeDataLoadFailed, err)
	}

	return rows, nil
}

// テーブル行追加
func (u *AppUseCase) InsertTableRow(ctx context.Context, row domain.InsertRow) (domain.AffectedRows, error) {
	profiles, activeID, err := u.repository.LoadProfiles()
	if err != nil {
		return domain.AffectedRows{}, err
	}
	if activeID == nil {
		return domain.AffectedRows{}, apperr.New(apperr.CodeProfileNotFound)
	}

	profile, found := findProfile(profiles, *activeID)
	if !found {
		return domain.AffectedRows{}, apperr.New(apperr.CodeProfileNotFound)
	}

	ref, err := domain.NewTableRef(schemaNamespace(profile), row.Table.Name)
	if err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}
	row.Table = ref

	credential, found, err := u.repository.GetCredential(profile.ID)
	if err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}
	if !found {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeCredentialUnavailable, errors.New("credential not found"))
	}

	structure, err := u.repository.InspectTableStructure(ctx, profile, credential, ref)
	if err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeRowAddFailed, err)
	}

	if err := row.Validate(structure.Table.Columns); err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	affected, err := u.repository.InsertRow(ctx, profile, credential, ref, row)
	if err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeRowAddFailed, err)
	}

	return affected, nil
}

// テーブルセル更新
func (u *AppUseCase) UpdateTableCell(ctx context.Context, change domain.CellUpdate) (domain.AffectedRows, error) {
	profiles, activeID, err := u.repository.LoadProfiles()
	if err != nil {
		return domain.AffectedRows{}, err
	}
	if activeID == nil {
		return domain.AffectedRows{}, apperr.New(apperr.CodeProfileNotFound)
	}

	profile, found := findProfile(profiles, *activeID)
	if !found {
		return domain.AffectedRows{}, apperr.New(apperr.CodeProfileNotFound)
	}

	ref, err := domain.NewTableRef(schemaNamespace(profile), change.Table.Name)
	if err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}
	change.Table = ref

	credential, found, err := u.repository.GetCredential(profile.ID)
	if err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}
	if !found {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeCredentialUnavailable, errors.New("credential not found"))
	}

	structure, err := u.repository.InspectTableStructure(ctx, profile, credential, ref)
	if err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeCellUpdateFailed, err)
	}
	if err := change.Validate(structure.Table.Columns); err != nil {
		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	affected, err := u.repository.UpdateCell(ctx, profile, credential, ref, change)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRowInput) {
			return domain.AffectedRows{}, apperr.Wrap(apperr.CodeValidationFailed, err)
		}

		return domain.AffectedRows{}, apperr.Wrap(apperr.CodeCellUpdateFailed, err)
	}

	return affected, nil
}

// 問い合わせ列存在判定
func tableQueryHasColumn(columns []domain.Column, name string) bool {
	for _, column := range columns {
		if column.Name == name {
			return true
		}
	}

	return false
}

// 対象名前空間取得
func schemaNamespace(profile domain.Profile) string {
	if profile.DBType == domain.DBTypeMySQL {
		return profile.Database
	}

	return profile.Schema
}
