package wails

import (
	"context"
	"log/slog"
)

// 接続プロファイル確認
func (h *AppHandler) CheckProfiles() Response[ProfileCheckResponse] {
	h.logger.Info(context.Background(), "profile check requested", slog.String("operation", "profile_check"))

	profiles, err := h.appUseCase.LoadProfiles()
	if err != nil {
		h.logger.Error(context.Background(), "profile check failed", err, slog.String("operation", "profile_check"))

		return Fail[ProfileCheckResponse](err)
	}

	return OK(ProfileCheckResponse{
		Valid:        true,
		ProfileCount: len(profiles),
	})
}
