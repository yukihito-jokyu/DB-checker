package wails

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
)

// アプリ設定返却
func (h *AppHandler) GetConfig() Response[ConfigResponse] {
	h.logger.Info(context.Background(), "app config requested", slog.String("operation", "config"))

	if _, _, err := h.appUseCase.LoadProfiles(); err != nil {
		h.logger.Error(context.Background(), "app config request failed", err, slog.String("operation", "config"))

		return Fail[ConfigResponse](err)
	}

	result, err := h.configStore.Load()
	if err != nil {
		h.logger.Error(context.Background(), "app config request failed", err, slog.String("operation", "config"))

		return Fail[ConfigResponse](err)
	}

	return OK(toConfigResponse(result.Config))
}

// 設定レスポンス変換
func toConfigResponse(cfg config.Config) ConfigResponse {
	profiles := make([]ProfileResponse, 0, len(cfg.ConnectionProfiles))
	for _, profile := range cfg.ConnectionProfiles {
		profiles = append(profiles, ProfileResponse{
			ID:       profile.ID,
			Name:     profile.Name,
			DBType:   profile.DBType,
			Host:     profile.Host,
			Port:     profile.Port,
			Database: profile.Database,
			Schema:   profile.Schema,
			User:     profile.User,
		})
	}

	return ConfigResponse{
		Version:                   cfg.Version,
		ConnectionProfiles:        profiles,
		ActiveConnectionProfileID: cfg.ActiveConnectionProfileID,
		FlowStates:                decodeFlowStates(cfg.FlowStates),
	}
}

// フロー状態復号
func decodeFlowStates(flowStates map[string]json.RawMessage) map[string]any {
	decoded := make(map[string]any, len(flowStates))
	for key, raw := range flowStates {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			decoded[key] = string(raw)

			continue
		}
		decoded[key] = value
	}

	return decoded
}
