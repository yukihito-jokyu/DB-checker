package wails

import (
	"context"
	"github.com/yukihito-jokyu/DB-checker/internal/config"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
	"sync"
)

type AppHandler struct {
	logger                applogger.Logger
	configStore           *config.Store
	appUseCase            *usecase.AppUseCase
	verificationScenarios *usecase.VerificationScenarioUseCase
	statisticsMu          sync.Mutex
	statisticsCancel      context.CancelFunc
	statisticsRequestID   uint64
}

// アプリハンドラー生成
func NewAppHandler(logger applogger.Logger, configStore *config.Store, appUseCase *usecase.AppUseCase, verificationScenarios ...*usecase.VerificationScenarioUseCase) *AppHandler {
	var verificationScenarioUseCase *usecase.VerificationScenarioUseCase
	if len(verificationScenarios) > 0 {
		verificationScenarioUseCase = verificationScenarios[0]
	}

	return &AppHandler{
		logger:                logger,
		configStore:           configStore,
		appUseCase:            appUseCase,
		verificationScenarios: verificationScenarioUseCase,
	}
}
