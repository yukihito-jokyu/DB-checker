package wails

import (
	"github.com/yukihito-jokyu/DB-checker/internal/config"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
)

type AppHandler struct {
	logger            applogger.Logger
	configStore       *config.Store
	appUseCase        *usecase.AppUseCase
	inspectionUseCase *usecase.InspectionUseCase
}

// アプリハンドラー生成
func NewAppHandler(logger applogger.Logger, configStore *config.Store, appUseCase *usecase.AppUseCase, inspectionUseCase *usecase.InspectionUseCase) *AppHandler {
	return &AppHandler{
		logger:            logger,
		configStore:       configStore,
		appUseCase:        appUseCase,
		inspectionUseCase: inspectionUseCase,
	}
}
