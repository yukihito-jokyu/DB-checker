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

// フロー状態取得
func (h *AppHandler) LoadFlowState() Response[FlowStateResponse] {
	h.logger.Info(context.Background(), "flow state requested", slog.String("operation", "flow_state_load"))

	if h.appUseCase == nil {
		err := apperr.New(apperr.CodeConfigReadFailed)
		h.logFailureWithCode("flow state load failed", "flow_state_load", err)

		return Fail[FlowStateResponse](err)
	}

	state, err := h.appUseCase.LoadFlowState()
	if err != nil {
		h.logFailureWithCode("flow state load failed", "flow_state_load", err)

		return Fail[FlowStateResponse](err)
	}

	return OK(toFlowStateResponse(state))
}

// フロー状態レスポンス変換
func toFlowStateResponse(state domain.FlowState) FlowStateResponse {
	tableStates := make(map[string]TableFlowStateResponse, len(state.TableStates))
	for tableName, tableState := range state.TableStates {
		tableStates[tableName] = TableFlowStateResponse{X: tableState.X, Y: tableState.Y, Expanded: tableState.Expanded}
	}

	return FlowStateResponse{Version: state.Version, TableStates: tableStates}
}

// 接続プロファイル保存
func (h *AppHandler) SaveConnectionProfile(request SaveConnectionProfileRequest) Response[ConnectionProfilesResponse] {
	h.logger.Info(context.Background(), "connection profile save requested", slog.String("operation", "connection_profile_save"))

	schema := ""
	if request.Schema != nil {
		schema = *request.Schema
	}

	draft, err := domain.NewProfileDraft(request.ID, request.Name, domain.DBType(request.DBType), request.Host, request.Port, request.Database, schema, request.User, request.Password)
	if err != nil {
		appErr := apperr.Wrap(apperr.CodeValidationFailed, err)
		h.logFailureWithCode("connection profile save failed", "connection_profile_save", appErr)

		return Fail[ConnectionProfilesResponse](appErr)
	}

	profiles, activeID, err := h.appUseCase.SaveConnectionProfile(context.Background(), draft)
	if err != nil {
		h.logFailureWithCode("connection profile save failed", "connection_profile_save", err)

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
		h.logFailureWithCode("connection profile activation failed", "connection_profile_activate", err)

		return Fail[ConnectionProfilesResponse](err)
	}

	return OK(ConnectionProfilesResponse{
		Profiles:                  toProfileResponses(profiles),
		ActiveConnectionProfileID: activeID,
	})
}

// 接続プロファイル削除
func (h *AppHandler) DeleteConnectionProfile(profileID string) Response[ConnectionProfilesResponse] {
	h.logger.Info(context.Background(), "connection profile deletion requested", slog.String("operation", "connection_profile_delete"))

	profiles, activeID, err := h.appUseCase.DeleteConnectionProfile(profileID)
	if err != nil {
		h.logFailureWithCode("connection profile deletion failed", "connection_profile_delete", err)

		return Fail[ConnectionProfilesResponse](err)
	}

	return OK(ConnectionProfilesResponse{
		Profiles:                  toProfileResponses(profiles),
		ActiveConnectionProfileID: activeID,
	})
}

// エラーコード付き失敗ログ出力
func (h *AppHandler) logFailureWithCode(message, operation string, err error) {
	code := apperr.CodeUnexpected
	if appErr := apperr.As(err); appErr != nil {
		code = appErr.Code
	}

	h.logger.ErrorCode(context.Background(), message, string(code), slog.String("operation", operation))
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
