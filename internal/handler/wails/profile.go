package wails

import (
	"context"
	"log/slog"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
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

// 接続プロファイル保存
func (h *AppHandler) SaveConnectionProfile(request SaveConnectionProfileRequest) Response[ConnectionProfilesResponse] {
	h.logger.Info(context.Background(), "connection profile save requested", slog.String("operation", "connection_profile_save"))

	draft, err := domain.NewProfileDraft(request.ID, request.Name, domain.DBType(request.DBType), request.Host, request.Port, request.Database, request.Schema, request.User, request.Password)
	if err != nil {
		h.logConnectionProfileSaveFailure(apperr.Wrap(apperr.CodeValidationFailed, err))

		return Fail[ConnectionProfilesResponse](apperr.Wrap(apperr.CodeValidationFailed, err))
	}

	profiles, activeID, err := h.appUseCase.SaveConnectionProfile(context.Background(), draft)
	if err != nil {
		h.logConnectionProfileSaveFailure(err)

		return Fail[ConnectionProfilesResponse](err)
	}

	return OK(ConnectionProfilesResponse{
		Profiles:                  toProfileResponses(profiles),
		ActiveConnectionProfileID: activeID,
	})
}

// アクティブ接続プロファイル切替
func (h *AppHandler) ActivateConnectionProfile(profileID string) Response[ConnectionProfilesResponse] {
	h.logger.Info(context.Background(), "connection profile activation requested", slog.String("operation", "connection_profile_activate"))

	profiles, activeID, err := h.appUseCase.ActivateConnectionProfile(context.Background(), profileID)
	if err != nil {
		h.logConnectionProfileActivationFailure(err)

		return Fail[ConnectionProfilesResponse](err)
	}

	return OK(ConnectionProfilesResponse{
		Profiles:                  toProfileResponses(profiles),
		ActiveConnectionProfileID: activeID,
	})
}

// 接続プロファイル保存失敗ログ出力
func (h *AppHandler) logConnectionProfileSaveFailure(err error) {
	code := apperr.CodeUnexpected
	if appErr := apperr.As(err); appErr != nil {
		code = appErr.Code
	}

	h.logger.ErrorCode(context.Background(), "connection profile save failed", string(code), slog.String("operation", "connection_profile_save"))
}

// アクティブ接続プロファイル切替失敗ログ出力
func (h *AppHandler) logConnectionProfileActivationFailure(err error) {
	code := apperr.CodeUnexpected
	if appErr := apperr.As(err); appErr != nil {
		code = appErr.Code
	}

	h.logger.ErrorCode(context.Background(), "connection profile activation failed", string(code), slog.String("operation", "connection_profile_activate"))
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
