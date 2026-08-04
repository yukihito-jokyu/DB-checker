package usecase

import (
	"context"
	"errors"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type InspectionRepository interface {
	LoadProfiles() ([]domain.Profile, *string, error)
	GetCredential(string) (credential string, found bool, err error)
	InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error)
}

type InspectionUseCase struct {
	repository InspectionRepository
}

// スキーマ閲覧ユースケース生成
func NewInspectionUseCase(repository InspectionRepository) *InspectionUseCase {
	return &InspectionUseCase{repository: repository}
}

// データベーススキーマ取得
func (u *InspectionUseCase) GetDatabaseSchema(ctx context.Context) (domain.Profile, domain.Schema, error) {
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

// 対象名前空間取得
func schemaNamespace(profile domain.Profile) string {
	if profile.DBType == domain.DBTypeMySQL {
		return profile.Database
	}

	return profile.Schema
}
