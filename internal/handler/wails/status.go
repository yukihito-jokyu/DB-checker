package wails

import (
	"context"
	"log/slog"
)

// アプリ状態返却
func (h *AppHandler) GetStatus() Response[StatusResponse] {
	h.logger.Info(context.Background(), "app status requested", slog.String("operation", "status"))

	return OK(StatusResponse{
		Name:    "DB-checker",
		Ready:   true,
		Version: "dev",
	})
}
