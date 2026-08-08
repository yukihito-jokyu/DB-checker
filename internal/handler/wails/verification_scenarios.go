package wails

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 検証シナリオ作成
func (h *AppHandler) CreateVerificationScenario(request CreateVerificationScenarioRequest) Response[VerificationScenarioResponse] {
	h.logger.Info(context.Background(), "verification scenario create requested", slog.String("operation", "verification_scenario_create"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification scenario create failed", "verification_scenario_create", err)

		return Fail[VerificationScenarioResponse](err)
	}

	draft, err := domain.NewVerificationScenarioDraft(request.Name, request.PrimaryTable, request.Definition)
	if err != nil {
		if errors.Is(err, domain.ErrPrimaryKeyRequired) {
			err = apperr.Wrap(apperr.CodePrimaryKeyRequired, err)
		} else {
			err = apperr.Wrap(apperr.CodeValidationFailed, err)
		}
		h.logFailureWithCode("verification scenario create failed", "verification_scenario_create", err)

		return Fail[VerificationScenarioResponse](err)
	}

	scenario, err := h.verificationScenarios.CreateVerificationScenario(context.Background(), draft)
	if err != nil {
		h.logFailureWithCode("verification scenario create failed", "verification_scenario_create", err)

		return Fail[VerificationScenarioResponse](err)
	}

	return OK(verificationScenarioResponse(scenario))
}

// 検証シナリオ更新
func (h *AppHandler) UpdateVerificationScenario(request UpdateVerificationScenarioRequest) Response[VerificationScenarioResponse] {
	h.logger.Info(context.Background(), "verification scenario update requested", slog.String("operation", "verification_scenario_update"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification scenario update failed", "verification_scenario_update", err)

		return Fail[VerificationScenarioResponse](err)
	}

	draft, err := domain.NewVerificationScenarioDraft(request.Name, request.PrimaryTable, request.Definition)
	if err != nil {
		if errors.Is(err, domain.ErrPrimaryKeyRequired) {
			err = apperr.Wrap(apperr.CodePrimaryKeyRequired, err)
		} else {
			err = apperr.Wrap(apperr.CodeValidationFailed, err)
		}
		h.logFailureWithCode("verification scenario update failed", "verification_scenario_update", err)

		return Fail[VerificationScenarioResponse](err)
	}

	scenario, err := h.verificationScenarios.UpdateVerificationScenario(context.Background(), request.ScenarioID, draft)
	if err != nil {
		h.logFailureWithCode("verification scenario update failed", "verification_scenario_update", err)

		return Fail[VerificationScenarioResponse](err)
	}

	return OK(verificationScenarioResponse(scenario))
}

// 検証シナリオ削除
func (h *AppHandler) DeleteVerificationScenario(request DeleteVerificationScenarioRequest) Response[DeleteScenarioResponse] {
	h.logger.Info(context.Background(), "verification scenario delete requested", slog.String("operation", "verification_scenario_delete"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification scenario delete failed", "verification_scenario_delete", err)

		return Fail[DeleteScenarioResponse](err)
	}

	workspaceRemoved, err := h.verificationScenarios.DeleteVerificationScenario(context.Background(), request.ScenarioID, request.RemoveWorkspace)
	if err != nil {
		h.logFailureWithCode("verification scenario delete failed", "verification_scenario_delete", err)

		return Fail[DeleteScenarioResponse](err)
	}

	return OK(DeleteScenarioResponse{ScenarioID: request.ScenarioID, WorkspaceRemoved: workspaceRemoved})
}

// 検証実行プレビュー取得
func (h *AppHandler) PreviewVerificationRun(request PreviewVerificationRunRequest) Response[VerificationRunPreviewResponse] {
	h.logger.Info(context.Background(), "verification run preview requested", slog.String("operation", "verification_run_preview"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification run preview failed", "verification_run_preview", err)

		return Fail[VerificationRunPreviewResponse](err)
	}
	if (request.ScenarioID == "") == (request.Draft == nil) {
		err := apperr.New(apperr.CodeValidationFailed)
		h.logFailureWithCode("verification run preview failed", "verification_run_preview", err)

		return Fail[VerificationRunPreviewResponse](err)
	}

	var draft *domain.VerificationScenarioDraft
	if request.Draft != nil {
		value, err := domain.NewVerificationScenarioDraft(request.Draft.Name, request.Draft.PrimaryTable, request.Draft.Definition)
		if err != nil {
			if errors.Is(err, domain.ErrPrimaryKeyRequired) {
				err = apperr.Wrap(apperr.CodePrimaryKeyRequired, err)
			} else {
				err = apperr.Wrap(apperr.CodeValidationFailed, err)
			}
			h.logFailureWithCode("verification run preview failed", "verification_run_preview", err)

			return Fail[VerificationRunPreviewResponse](err)
		}
		draft = &value
	}

	preview, err := h.verificationScenarios.PreviewVerificationRun(context.Background(), request.ScenarioID, draft)
	if err != nil {
		h.logFailureWithCode("verification run preview failed", "verification_run_preview", err)

		return Fail[VerificationRunPreviewResponse](err)
	}

	return OK(verificationRunPreviewResponse(preview))
}

// 検証ワークスペース開始
func (h *AppHandler) EnterVerificationWorkspace(scenarioID string) Response[VerificationWorkspaceResponse] {
	h.logger.Info(context.Background(), "verification workspace enter requested", slog.String("operation", "verification_workspace_enter"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification workspace enter failed", "verification_workspace_enter", err)

		return Fail[VerificationWorkspaceResponse](err)
	}
	name, err := h.verificationScenarios.EnterVerificationWorkspace(context.Background(), scenarioID)
	if err != nil {
		h.logFailureWithCode("verification workspace enter failed", "verification_workspace_enter", err)

		return Fail[VerificationWorkspaceResponse](err)
	}

	return OK(VerificationWorkspaceResponse{ScenarioID: scenarioID, WorkspaceName: name, Mode: "test"})
}

// 検証ワークスペース終了
func (h *AppHandler) ExitVerificationWorkspace(scenarioID string) Response[struct{}] {
	h.logger.Info(context.Background(), "verification workspace exit requested", slog.String("operation", "verification_workspace_exit"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification workspace exit failed", "verification_workspace_exit", err)

		return Fail[struct{}](err)
	}
	if err := h.verificationScenarios.ExitVerificationWorkspace(context.Background(), scenarioID); err != nil {
		h.logFailureWithCode("verification workspace exit failed", "verification_workspace_exit", err)

		return Fail[struct{}](err)
	}

	return OK(struct{}{})
}

// 検証実行状態作成
func (h *AppHandler) PrepareVerificationRun(request PrepareVerificationRunRequest) Response[struct{}] {
	h.logger.Info(context.Background(), "verification run prepare requested", slog.String("operation", "verification_run_prepare"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification run prepare failed", "verification_run_prepare", err)

		return Fail[struct{}](err)
	}
	if err := h.verificationScenarios.PrepareVerificationRun(context.Background(), request.ScenarioID, request.RunID); err != nil {
		h.logFailureWithCode("verification run prepare failed", "verification_run_prepare", err)

		return Fail[struct{}](err)
	}

	return OK(struct{}{})
}

// 検証実行状態更新
func (h *AppHandler) UpdateVerificationRunState(request UpdateVerificationRunStateRequest) Response[struct{}] {
	h.logger.Info(context.Background(), "verification run state update requested", slog.String("operation", "verification_run_state_update"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification run state update failed", "verification_run_state_update", err)

		return Fail[struct{}](err)
	}
	if err := h.verificationScenarios.UpdateVerificationRunState(context.Background(), request.RunID, request.State); err != nil {
		h.logFailureWithCode("verification run state update failed", "verification_run_state_update", err)

		return Fail[struct{}](err)
	}

	return OK(struct{}{})
}

// 検証シナリオ一覧取得
func (h *AppHandler) ListVerificationScenarios() Response[[]VerificationScenarioSummaryResponse] {
	h.logger.Info(context.Background(), "verification scenarios requested", slog.String("operation", "verification_scenarios_list"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification scenarios list failed", "verification_scenarios_list", err)

		return Fail[[]VerificationScenarioSummaryResponse](err)
	}

	scenarios, err := h.verificationScenarios.ListVerificationScenarios(context.Background())
	if err != nil {
		h.logFailureWithCode("verification scenarios list failed", "verification_scenarios_list", err)

		return Fail[[]VerificationScenarioSummaryResponse](err)
	}

	responses := make([]VerificationScenarioSummaryResponse, 0, len(scenarios))
	for _, scenario := range scenarios {
		responses = append(responses, VerificationScenarioSummaryResponse{
			ID:           scenario.ID,
			Name:         scenario.Name,
			PrimaryTable: scenario.PrimaryTable,
			UpdatedAt:    scenario.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	return OK(responses)
}

// 検証シナリオ詳細取得
func (h *AppHandler) GetVerificationScenario(scenarioID string) Response[VerificationScenarioResponse] {
	h.logger.Info(context.Background(), "verification scenario requested", slog.String("operation", "verification_scenario_get"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification scenario get failed", "verification_scenario_get", err)

		return Fail[VerificationScenarioResponse](err)
	}

	scenario, err := h.verificationScenarios.GetVerificationScenario(context.Background(), scenarioID)
	if err != nil {
		h.logFailureWithCode("verification scenario get failed", "verification_scenario_get", err)

		return Fail[VerificationScenarioResponse](err)
	}

	return OK(verificationScenarioResponse(scenario))
}

// 検証シナリオ複製
func (h *AppHandler) DuplicateVerificationScenario(scenarioID string) Response[VerificationScenarioResponse] {
	h.logger.Info(context.Background(), "verification scenario duplicate requested", slog.String("operation", "verification_scenario_duplicate"))
	if h.verificationScenarios == nil {
		err := apperr.New(apperr.CodeScenarioStoreFailed)
		h.logFailureWithCode("verification scenario duplicate failed", "verification_scenario_duplicate", err)

		return Fail[VerificationScenarioResponse](err)
	}

	scenario, err := h.verificationScenarios.DuplicateVerificationScenario(context.Background(), scenarioID)
	if err != nil {
		h.logFailureWithCode("verification scenario duplicate failed", "verification_scenario_duplicate", err)

		return Fail[VerificationScenarioResponse](err)
	}

	return OK(verificationScenarioResponse(scenario))
}

// 検証シナリオ応答変換
func verificationScenarioResponse(scenario domain.VerificationScenario) VerificationScenarioResponse {
	return VerificationScenarioResponse{
		ID:            scenario.ID,
		Name:          scenario.Name,
		PrimaryTable:  scenario.PrimaryTable,
		Definition:    scenario.Definition,
		WorkspaceName: scenario.WorkspaceName,
		CreatedAt:     scenario.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     scenario.UpdatedAt.UTC().Format(time.RFC3339Nano),
		LatestRun:     nil,
	}
}

// 検証実行プレビュー応答変換
func verificationRunPreviewResponse(preview domain.VerificationRunPreview) VerificationRunPreviewResponse {
	return VerificationRunPreviewResponse{
		Ready:       preview.Ready,
		InsertOrder: verificationRunPreviewTablesResponse(preview.InsertOrder),
		DeleteOrder: verificationRunPreviewTablesResponse(preview.DeleteOrder),
		Warnings:    preview.Warnings,
	}
}

// 検証実行プレビューテーブル応答変換
func verificationRunPreviewTablesResponse(tables []domain.VerificationRunPreviewTable) []VerificationRunPreviewTableResponse {
	responses := make([]VerificationRunPreviewTableResponse, 0, len(tables))
	for _, table := range tables {
		responses = append(responses, VerificationRunPreviewTableResponse{
			Name:               table.Name,
			RowCount:           table.RowCount,
			AutomaticallyAdded: table.AutomaticallyAdded,
			GeneratedColumns:   append([]string{}, table.GeneratedColumns...),
		})
	}

	return responses
}
