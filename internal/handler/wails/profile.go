package wails

import (
	"context"
	"log/slog"
)

// 接続プロファイル確認
func (h *AppHandler) CheckProfiles() Response[ProfileCheckResponse] {
	h.logger.Info(context.Background(), "profile check requested", slog.String("operation", "profile_check"))

	profiles, _, err := h.appUseCase.LoadProfiles()
	if err != nil {
		h.logger.Error(context.Background(), "profile check failed", err, slog.String("operation", "profile_check"))

		return Fail[ProfileCheckResponse](err)
	}

	return OK(ProfileCheckResponse{
		Valid:        true,
		ProfileCount: len(profiles),
	})
}

// 接続プロファイル一覧取得
func (h *AppHandler) ListConnectionProfiles() Response[ConnectionProfilesResponse] {
	h.logger.Info(context.Background(), "connection profile list requested", slog.String("operation", "connection_profile_list"))

	profiles, activeID, err := h.appUseCase.LoadProfiles()
	if err != nil {
		h.logger.Error(context.Background(), "connection profile list failed", err, slog.String("operation", "connection_profile_list"))

		return Fail[ConnectionProfilesResponse](err)
	}

	return OK(ConnectionProfilesResponse{
		Profiles:                  toProfileResponses(profiles),
		ActiveConnectionProfileID: activeID,
	})
}
