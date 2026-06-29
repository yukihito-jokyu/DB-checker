package wails

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
)

type AppHandler struct {
	logger      applogger.Logger
	configStore *config.Store
}

type StatusData struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
}

type ConfigData struct {
	Version                   int                     `json:"version"`
	ConnectionProfiles        []ConnectionProfileData `json:"connectionProfiles"`
	ActiveConnectionProfileID *string                 `json:"activeConnectionProfileId"`
	FlowStates                map[string]any          `json:"flowStates"`
}

type ConnectionProfileData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DBType   string `json:"dbType"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func NewAppHandler(logger applogger.Logger, configStore *config.Store) *AppHandler {
	return &AppHandler{
		logger:      logger,
		configStore: configStore,
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

func (h *AppHandler) Config() Response[ConfigData] {
	h.logger.Info(context.Background(), "app config requested", slog.String("operation", "config"))

	result, err := h.configStore.Load()
	if err != nil {
		h.logger.Error(context.Background(), "app config request failed", err, slog.String("operation", "config"))
		return Fail[ConfigData](err)
	}
	if result.Recovered {
		h.logger.Warn(
			context.Background(),
			"broken app config recovered",
			slog.String("operation", "config"),
			slog.String("backup_path", result.BackupPath),
		)
	}

	return OK(toConfigData(result.Config))
}

func toConfigData(cfg config.Config) ConfigData {
	profiles := make([]ConnectionProfileData, 0, len(cfg.ConnectionProfiles))
	for _, profile := range cfg.ConnectionProfiles {
		profiles = append(profiles, ConnectionProfileData{
			ID:       profile.ID,
			Name:     profile.Name,
			DBType:   profile.DBType,
			Host:     profile.Host,
			Port:     profile.Port,
			Database: profile.Database,
			Schema:   profile.Schema,
			User:     profile.User,
			Password: profile.Password,
		})
	}

	return ConfigData{
		Version:                   cfg.Version,
		ConnectionProfiles:        profiles,
		ActiveConnectionProfileID: cfg.ActiveConnectionProfileID,
		FlowStates:                decodeFlowStates(cfg.FlowStates),
	}
}

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
