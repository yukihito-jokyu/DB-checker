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
