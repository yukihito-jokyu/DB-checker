package config

import "encoding/json"

const (
	AppDirName  = "DB-checker"
	FileName    = "config.json"
	FileVersion = 1
)

type Config struct {
	Version                   int                        `json:"version"`
	ConnectionProfiles        []ConnectionProfile        `json:"connectionProfiles"`
	ActiveConnectionProfileID *string                    `json:"activeConnectionProfileId"`
	FlowStates                map[string]json.RawMessage `json:"flowStates"`
}

type ConnectionProfile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DBType   string `json:"dbType"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
	User     string `json:"user"`
}

// 既定設定生成
func Default() Config {
	return Config{
		Version:                   FileVersion,
		ConnectionProfiles:        []ConnectionProfile{},
		ActiveConnectionProfileID: nil,
		FlowStates:                map[string]json.RawMessage{},
	}
}
