package usecase

import (
	"context"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

// アプリケーションリポジトリ
type AppRepository interface {
	LoadProfiles() ([]domain.Profile, *string, error)
	LoadFlowState(string) (domain.FlowState, error)
	SaveFlowState(string, domain.FlowState) error
	SaveProfiles([]domain.Profile, *string) error
	GetCredential(string) (credential string, found bool, err error)
	SetCredential(string, string) error
	DeleteCredential(string) error
	CheckConnection(context.Context, domain.Profile, string) error
	InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error)
	InspectTableStructure(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStructure, error)
}

// アプリケーションユースケース
type AppUseCase struct {
	repository AppRepository
}

// アプリケーションユースケース生成
func NewAppUseCase(repository AppRepository) *AppUseCase {
	return &AppUseCase{
		repository: repository,
	}
}
