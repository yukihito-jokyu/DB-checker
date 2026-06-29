package wails

import (
	"context"
	"log/slog"

	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
)

type AppHandler struct {
	logger applogger.Logger
}

type StatusData struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
}

func NewAppHandler(logger applogger.Logger) *AppHandler {
	return &AppHandler{
		logger: logger,
	}
}

func (h *AppHandler) Status() Response[StatusData] {
	h.logger.Info(context.Background(), "app status requested", slog.String("operation", "status"))

	return OK(StatusData{
		Name:    "DB-checker",
		Ready:   true,
		Version: "dev",
	})
}
