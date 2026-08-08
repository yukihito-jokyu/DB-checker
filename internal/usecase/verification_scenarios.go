package usecase

import (
	"context"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// シナリオ用プロファイルリポジトリ
type VerificationScenarioProfileRepository interface {
	LoadProfiles() ([]domain.Profile, *string, error)
}

// シナリオリポジトリ
type VerificationScenarioRepository interface {
	ListVerificationScenarios(context.Context, string) ([]domain.VerificationScenarioSummary, error)
}

// 検証シナリオユースケース
type VerificationScenarioUseCase struct {
	profiles   VerificationScenarioProfileRepository
	repository VerificationScenarioRepository
}

// 検証シナリオユースケース生成
func NewVerificationScenarioUseCase(profiles VerificationScenarioProfileRepository, repository VerificationScenarioRepository) *VerificationScenarioUseCase {
	return &VerificationScenarioUseCase{profiles: profiles, repository: repository}
}

// アクティブプロファイルのシナリオ一覧取得
func (u *VerificationScenarioUseCase) ListVerificationScenarios(ctx context.Context) ([]domain.VerificationScenarioSummary, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return nil, err
	}
	if activeID == nil {
		return nil, apperr.New(apperr.CodeProfileNotFound)
	}
	if !containsVerificationScenarioProfile(profiles, *activeID) {
		return nil, apperr.New(apperr.CodeProfileNotFound)
	}

	scenarios, err := u.repository.ListVerificationScenarios(ctx, *activeID)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}

	return scenarios, nil
}

// シナリオ用プロファイル存在判定
func containsVerificationScenarioProfile(profiles []domain.Profile, profileID string) bool {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return true
		}
	}

	return false
}
