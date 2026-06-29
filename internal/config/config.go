package config

import "encoding/json"

const (
	AppDirName   = "DB-checker"
	FileName     = "config.json"
	FileVersion  = 1
	backupSuffix = "broken"
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
	Password string `json:"password"`
}

// Default は初期設定ファイルへ保存する既定値を返す。
func Default() Config {
	return Config{
		Version:                   FileVersion,
		ConnectionProfiles:        []ConnectionProfile{},
		ActiveConnectionProfileID: nil,
		FlowStates:                map[string]json.RawMessage{},
	}
}
