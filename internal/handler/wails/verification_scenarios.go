package wails

import (
	"context"
	"log/slog"
	"time"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

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
