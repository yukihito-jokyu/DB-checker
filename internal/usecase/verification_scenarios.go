package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// 検証実行プレビュー用リポジトリ
type VerificationRunPreviewRepository interface {
	GetCredential(string) (credential string, found bool, err error)
	InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error)
}

// シナリオリポジトリ
type VerificationScenarioRepository interface {
	CreateVerificationScenario(context.Context, string, domain.VerificationScenario) error
	ListVerificationScenarios(context.Context, string) ([]domain.VerificationScenarioSummary, error)
	GetVerificationScenario(context.Context, string, string) (domain.VerificationScenario, bool, error)
	UpdateVerificationScenario(context.Context, string, domain.VerificationScenario) (bool, error)
	DeleteVerificationScenario(context.Context, string, string, bool) (bool, bool, bool, error)
}

// 検証ワークスペース状態リポジトリ
type VerificationWorkspaceStateRepository interface {
	GetVerificationWorkspace(context.Context, string, string) (string, string, bool, error)
	SaveVerificationWorkspace(context.Context, string, string, string, string) error
	DeleteVerificationWorkspace(context.Context, string, string) error
	CreateVerificationRun(context.Context, string, string, string) error
	GetVerificationRun(context.Context, string, string) (string, string, bool, error)
	UpdateVerificationRunState(context.Context, string, string, string) (bool, error)
	IsVerificationScenarioBusy(context.Context, string, string) (bool, error)
	IsVerificationRunBusy(context.Context, string, string) (bool, error)
}

// 検証先リポジトリ
type VerificationWorkspaceRepository interface {
	CreateWorkspace(context.Context, domain.Profile, string) error
	DeleteWorkspace(context.Context, domain.Profile, string) error
}

// アクティブプロファイルのシナリオ削除
func (u *VerificationScenarioUseCase) DeleteVerificationScenario(ctx context.Context, scenarioID string, removeWorkspace bool) (bool, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return false, err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return false, apperr.New(apperr.CodeProfileNotFound)
	}

	workspaceRemoved := false
	stateRepository, stateSupported := u.repository.(VerificationWorkspaceStateRepository)
	if stateSupported {
		workspaceState, workspaceName, workspaceFound, stateErr := stateRepository.GetVerificationWorkspace(ctx, *activeID, scenarioID)
		if stateErr != nil {
			return false, apperr.Wrap(apperr.CodeScenarioStoreFailed, stateErr)
		}
		if workspaceFound && workspaceName != verificationWorkspaceName(*activeID, scenarioID) {
			return false, apperr.New(apperr.CodeVerificationNamespaceFailed)
		}
		runBusy, runStateErr := stateRepository.IsVerificationRunBusy(ctx, *activeID, scenarioID)
		if runStateErr != nil {
			return false, apperr.Wrap(apperr.CodeScenarioStoreFailed, runStateErr)
		}
		if runBusy {
			return false, apperr.New(apperr.CodeScenarioBusy)
		}
		if workspaceFound && !removeWorkspace {
			return false, apperr.New(apperr.CodeScenarioBusy)
		}
		if workspaceFound && (workspaceState == "active" || workspaceState == "test") {
			return false, apperr.New(apperr.CodeScenarioBusy)
		}
		if workspaceFound && removeWorkspace {
			if u.workspace == nil {
				return false, apperr.New(apperr.CodeVerificationNamespaceFailed)
			}
			if err := stateRepository.SaveVerificationWorkspace(ctx, *activeID, scenarioID, workspaceName, "deleting"); err != nil {
				return false, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
			}
			if err := u.workspace.DeleteWorkspace(ctx, activeProfile(profiles, *activeID), workspaceName); err != nil {
				return false, apperr.Wrap(apperr.CodeVerificationNamespaceFailed, err)
			}
			if err := stateRepository.DeleteVerificationWorkspace(ctx, *activeID, scenarioID); err != nil {
				return false, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
			}
			workspaceRemoved = true
		}
	}

	found, repositoryWorkspaceRemoved, busy, err := u.repository.DeleteVerificationScenario(ctx, *activeID, scenarioID, false)
	if err != nil {
		return false, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if busy {
		return false, apperr.New(apperr.CodeScenarioBusy)
	}
	if !found {
		return false, apperr.New(apperr.CodeScenarioNotFound)
	}

	return workspaceRemoved || repositoryWorkspaceRemoved, nil
}

// 検証ワークスペース終了
func (u *VerificationScenarioUseCase) ExitVerificationWorkspace(ctx context.Context, scenarioID string) error {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return apperr.New(apperr.CodeProfileNotFound)
	}
	states, ok := u.repository.(VerificationWorkspaceStateRepository)
	if !ok {
		return apperr.New(apperr.CodeScenarioStoreFailed)
	}
	_, found, err := u.repository.GetVerificationScenario(ctx, *activeID, scenarioID)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !found {
		return apperr.New(apperr.CodeScenarioNotFound)
	}
	state, name, exists, err := states.GetVerificationWorkspace(ctx, *activeID, scenarioID)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !exists || state != "test" {
		return apperr.New(apperr.CodeScenarioBusy)
	}
	busy, err := states.IsVerificationRunBusy(ctx, *activeID, scenarioID)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if busy {
		return apperr.New(apperr.CodeScenarioBusy)
	}
	if err := states.SaveVerificationWorkspace(ctx, *activeID, scenarioID, name, "inactive"); err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}

	return nil
}

// 検証ワークスペース開始
func (u *VerificationScenarioUseCase) EnterVerificationWorkspace(ctx context.Context, scenarioID string) (string, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return "", err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return "", apperr.New(apperr.CodeProfileNotFound)
	}
	if u.workspace == nil {
		return "", apperr.New(apperr.CodeVerificationNamespaceFailed)
	}
	states, ok := u.repository.(VerificationWorkspaceStateRepository)
	if !ok {
		return "", apperr.New(apperr.CodeScenarioStoreFailed)
	}
	_, found, err := u.repository.GetVerificationScenario(ctx, *activeID, scenarioID)
	if err != nil {
		return "", apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !found {
		return "", apperr.New(apperr.CodeScenarioNotFound)
	}
	state, name, exists, err := states.GetVerificationWorkspace(ctx, *activeID, scenarioID)
	if err != nil {
		return "", apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if exists && name != verificationWorkspaceName(*activeID, scenarioID) {
		return "", apperr.New(apperr.CodeVerificationNamespaceFailed)
	}
	if exists && (state == "active" || state == "test" || state == "deleting") {
		return "", apperr.New(apperr.CodeScenarioBusy)
	}
	if !exists {
		name = verificationWorkspaceName(*activeID, scenarioID)
		if err := states.SaveVerificationWorkspace(ctx, *activeID, scenarioID, name, "creating"); err != nil {
			return "", apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
		}
	} else if state == "inactive" {
		if err := states.SaveVerificationWorkspace(ctx, *activeID, scenarioID, name, "creating"); err != nil {
			return "", apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
		}
	}
	if err := u.workspace.CreateWorkspace(ctx, activeProfile(profiles, *activeID), name); err != nil {
		return "", apperr.Wrap(apperr.CodeVerificationNamespaceFailed, err)
	}
	if err := states.SaveVerificationWorkspace(ctx, *activeID, scenarioID, name, "test"); err != nil {
		return "", apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}

	return name, nil
}

// 検証実行状態作成
func (u *VerificationScenarioUseCase) PrepareVerificationRun(ctx context.Context, scenarioID, runID string) error {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return apperr.New(apperr.CodeProfileNotFound)
	}
	states, ok := u.repository.(VerificationWorkspaceStateRepository)
	if !ok {
		return apperr.New(apperr.CodeScenarioStoreFailed)
	}
	if scenarioID == "" || runID == "" {
		return apperr.New(apperr.CodeValidationFailed)
	}
	_, found, err := u.repository.GetVerificationScenario(ctx, *activeID, scenarioID)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !found {
		return apperr.New(apperr.CodeScenarioNotFound)
	}
	workspaceState, _, workspaceFound, err := states.GetVerificationWorkspace(ctx, *activeID, scenarioID)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !workspaceFound || workspaceState != "test" {
		return apperr.New(apperr.CodeScenarioBusy)
	}
	_, _, runFound, err := states.GetVerificationRun(ctx, *activeID, runID)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if runFound {
		return apperr.New(apperr.CodeValidationFailed)
	}
	if err := states.CreateVerificationRun(ctx, *activeID, scenarioID, runID); err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}

	return nil
}

// 検証実行状態更新
func (u *VerificationScenarioUseCase) UpdateVerificationRunState(ctx context.Context, runID, state string) error {
	if state != "prepared" && state != "running" && state != "canceling" && state != "succeeded" && state != "failed" && state != "canceled" {
		return apperr.New(apperr.CodeValidationFailed)
	}
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return apperr.New(apperr.CodeProfileNotFound)
	}
	states, ok := u.repository.(VerificationWorkspaceStateRepository)
	if !ok {
		return apperr.New(apperr.CodeScenarioStoreFailed)
	}
	_, currentState, found, err := states.GetVerificationRun(ctx, *activeID, runID)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !found {
		return apperr.New(apperr.CodeScenarioNotFound)
	}
	if !isAllowedVerificationRunTransition(currentState, state) {
		return apperr.New(apperr.CodeValidationFailed)
	}
	updated, err := states.UpdateVerificationRunState(ctx, *activeID, runID, state)
	if err != nil {
		return apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !updated {
		return apperr.New(apperr.CodeScenarioNotFound)
	}

	return nil
}

// 検証実行状態遷移判定
func isAllowedVerificationRunTransition(current, next string) bool {
	return (current == "prepared" && (next == "running" || next == "canceling" || next == "canceled")) ||
		(current == "running" && (next == "canceling" || next == "succeeded" || next == "failed")) ||
		(current == "canceling" && (next == "canceled" || next == "failed"))
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

// アクティブプロファイルのシナリオ複製
func (u *VerificationScenarioUseCase) DuplicateVerificationScenario(ctx context.Context, scenarioID string) (domain.VerificationScenario, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return domain.VerificationScenario{}, err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeProfileNotFound)
	}

	existing, found, err := u.repository.GetVerificationScenario(ctx, *activeID, scenarioID)
	if err != nil {
		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !found {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeScenarioNotFound)
	}

	scenario, err := existing.DuplicateVerificationScenario(uuid.NewString(), time.Now().UTC())
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

// アクティブプロファイルのシナリオ更新
func (u *VerificationScenarioUseCase) UpdateVerificationScenario(ctx context.Context, scenarioID string, draft domain.VerificationScenarioDraft) (domain.VerificationScenario, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return domain.VerificationScenario{}, err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeProfileNotFound)
	}
	if states, ok := u.repository.(VerificationWorkspaceStateRepository); ok {
		busy, stateErr := states.IsVerificationScenarioBusy(ctx, *activeID, scenarioID)
		if stateErr != nil {
			return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeScenarioStoreFailed, stateErr)
		}
		if busy {
			return domain.VerificationScenario{}, apperr.New(apperr.CodeScenarioBusy)
		}
	}
	if err := draft.Validate(); err != nil {
		if errors.Is(err, domain.ErrPrimaryKeyRequired) {
			return domain.VerificationScenario{}, apperr.Wrap(apperr.CodePrimaryKeyRequired, err)
		}

		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	existing, found, err := u.repository.GetVerificationScenario(ctx, *activeID, scenarioID)
	if err != nil {
		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !found {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeScenarioNotFound)
	}

	scenario, err := draft.UpdateVerificationScenario(existing, time.Now().UTC())
	if err != nil {
		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	updated, err := u.repository.UpdateVerificationScenario(ctx, *activeID, scenario)
	if err != nil {
		return domain.VerificationScenario{}, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
	}
	if !updated {
		return domain.VerificationScenario{}, apperr.New(apperr.CodeScenarioNotFound)
	}

	return scenario, nil
}

// 検証シナリオユースケース
type VerificationScenarioUseCase struct {
	profiles   VerificationScenarioProfileRepository
	preview    VerificationRunPreviewRepository
	repository VerificationScenarioRepository
	workspace  VerificationWorkspaceRepository
}

// 検証シナリオユースケース生成
func NewVerificationScenarioUseCase(profiles VerificationScenarioProfileRepository, repository VerificationScenarioRepository, workspaces ...VerificationWorkspaceRepository) *VerificationScenarioUseCase {
	var workspace VerificationWorkspaceRepository
	if len(workspaces) > 0 {
		workspace = workspaces[0]
	}

	return &VerificationScenarioUseCase{profiles: profiles, repository: repository, workspace: workspace}
}

// プレビュー対応検証シナリオユースケース生成
func NewVerificationScenarioUseCaseWithPreview(profiles VerificationScenarioProfileRepository, repository VerificationScenarioRepository, preview VerificationRunPreviewRepository, workspaces ...VerificationWorkspaceRepository) *VerificationScenarioUseCase {
	useCase := NewVerificationScenarioUseCase(profiles, repository, workspaces...)
	useCase.preview = preview

	return useCase
}

// アクティブプロファイル取得
func activeProfile(profiles []domain.Profile, profileID string) domain.Profile {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return profile
		}
	}

	return domain.Profile{}
}

// 検証先内部名生成
func verificationWorkspaceName(profileID, scenarioID string) string {
	digest := sha256.Sum256([]byte(profileID + "\x00" + scenarioID))

	return "db_checker_v_" + hex.EncodeToString(digest[:24])
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

// 検証実行プレビュー取得
func (u *VerificationScenarioUseCase) PreviewVerificationRun(ctx context.Context, scenarioID string, draft *domain.VerificationScenarioDraft) (domain.VerificationRunPreview, error) {
	if (scenarioID == "") == (draft == nil) {
		return domain.VerificationRunPreview{}, apperr.New(apperr.CodeValidationFailed)
	}
	if u.preview == nil {
		return domain.VerificationRunPreview{}, apperr.New(apperr.CodeSchemaLoadFailed)
	}

	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return domain.VerificationRunPreview{}, err
	}
	if activeID == nil || !containsVerificationScenarioProfile(profiles, *activeID) {
		return domain.VerificationRunPreview{}, apperr.New(apperr.CodeProfileNotFound)
	}

	profile := activeProfile(profiles, *activeID)
	var previewDraft domain.VerificationScenarioDraft
	if draft != nil {
		if err := draft.Validate(); err != nil {
			return domain.VerificationRunPreview{}, previewValidationError(err)
		}
		previewDraft = *draft
	} else {
		scenario, found, err := u.repository.GetVerificationScenario(ctx, *activeID, scenarioID)
		if err != nil {
			return domain.VerificationRunPreview{}, apperr.Wrap(apperr.CodeScenarioStoreFailed, err)
		}
		if !found {
			return domain.VerificationRunPreview{}, apperr.New(apperr.CodeScenarioNotFound)
		}
		previewDraft, err = domain.NewVerificationScenarioDraft(scenario.Name, scenario.PrimaryTable, scenario.Definition)
		if err != nil {
			return domain.VerificationRunPreview{}, previewValidationError(err)
		}
	}

	credential, found, err := u.preview.GetCredential(profile.ID)
	if err != nil {
		return domain.VerificationRunPreview{}, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}
	if !found {
		return domain.VerificationRunPreview{}, apperr.New(apperr.CodeCredentialUnavailable)
	}

	schema, err := u.preview.InspectSchema(ctx, profile, credential)
	if err != nil {
		return domain.VerificationRunPreview{}, apperr.Wrap(apperr.CodeSchemaLoadFailed, err)
	}
	if err := schema.Validate(schemaNamespace(profile)); err != nil {
		return domain.VerificationRunPreview{}, apperr.Wrap(apperr.CodeSchemaLoadFailed, err)
	}

	return domain.PreviewVerificationRun(previewDraft, schema), nil
}

// プレビュー入力エラー変換
func previewValidationError(err error) error {
	if errors.Is(err, domain.ErrPrimaryKeyRequired) {
		return apperr.Wrap(apperr.CodePrimaryKeyRequired, err)
	}

	return apperr.Wrap(apperr.CodeValidationFailed, err)
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
