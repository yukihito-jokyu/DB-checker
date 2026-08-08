package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// シナリオ用プロファイルリポジトリ
type VerificationScenarioProfileRepository interface {
	LoadProfiles() ([]domain.Profile, *string, error)
}

// シナリオリポジトリ
type VerificationScenarioRepository interface {
	CreateVerificationScenario(context.Context, string, domain.VerificationScenario) error
	ListVerificationScenarios(context.Context, string) ([]domain.VerificationScenarioSummary, error)
	GetVerificationScenario(context.Context, string, string) (domain.VerificationScenario, bool, error)
}

// アクティブプロファイルへのシナリオ作成
func (u *VerificationScenarioUseCase) CreateVerificationScenario(ctx context.Context, draft domain.VerificationScenarioDraft) (domain.VerificationScenario, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return domain.VerificationScenario{}, err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeProfileNotFound)
	}

	scenario, err := draft.NewVerificationScenario(uuid.NewString(), time.Now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrPrimaryKeyRequired) {
			return domain.VerificationScenario{}, apperr.Wrap(apperr.CodePrimaryKeyRequired, err)
		}

		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}
	if err := u.repository.CreateVerificationScenario(ctx, *activeID, scenario); err != nil {
		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}

	return scenario, nil
}

// アクティブプロファイルのシナリオ詳細取得
func (u *VerificationScenarioUseCase) GetVerificationScenario(ctx context.Context, scenarioID string) (domain.VerificationScenario, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return domain.VerificationScenario{}, err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeProfileNotFound)
	}

	scenario, found, err := u.repository.GetVerificationScenario(ctx, *activeID, scenarioID)
	if err != nil {
		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !found {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeScenarioNotFound)
	}

	return scenario, nil
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
