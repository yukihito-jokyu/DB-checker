package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 既定設定ストア生成検証
func TestNewDefaultStore(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T)
		wantFound bool
	}{
		{
			name:      "既定のユーザー設定ディレクトリ",
			wantFound: true,
		},
		{
			name: "ユーザー設定ディレクトリを取得できない",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("HOME", "")
				t.Setenv("XDG_CONFIG_HOME", "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			store, err := NewDefaultStore()
			if gotFound := store != nil; gotFound != tt.wantFound {
				t.Fatalf("NewDefaultStore() store found = %v, want %v", gotFound, tt.wantFound)
			}
			if gotFound := err == nil; gotFound != tt.wantFound {
				t.Errorf("NewDefaultStore() success = %v, want %v (error = %v)", gotFound, tt.wantFound, err)
			}
			if !tt.wantFound {
				return
			}

			configDir, err := os.UserConfigDir()
			if err != nil {
				t.Fatalf("UserConfigDir() error = %v", err)
			}
			if got, want := store.Path(), filepath.Join(configDir, AppDirName, FileName); got != want {
				t.Errorf("Path() = %q, want %q", got, want)
			}
		})
	}
}

// 未作成設定読込検証
func TestStoreLoadDoesNotCreateMissingConfig(t *testing.T) {
	tests := []struct {
		name          string
		wantErrorCode apperr.Code
	}{
		{
			name:          "未作成の設定",
			wantErrorCode: apperr.CodeConfigNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			_, err := store.Load()
			if gotCode := appErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("Load() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if _, err := os.Stat(store.Path()); !os.IsNotExist(err) {
				t.Errorf("config file exists or stat failed: %v", err)
			}
		})
	}
}

// 設定初期化検証
func TestStoreInitialize(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*testing.T, *Store) string
		wantErrorCode apperr.Code
		wantProfiles  int
		wantPreserved bool
	}{
		{
			name:         "未作成の設定を既定値で初期化する",
			wantProfiles: 0,
		},
		{
			name: "既存の設定を維持する",
			setup: func(t *testing.T, store *Store) string {
				t.Helper()
				configuration := Default()
				configuration.FlowStates = map[string]json.RawMessage{
					"step": json.RawMessage(`{"done":true}`),
				}
				if err := store.Save(configuration); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
				content, err := os.ReadFile(store.Path())
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}

				return string(content)
			},
			wantPreserved: true,
		},
		{
			name: "破損した設定をエラーとして返す",
			setup: func(t *testing.T, store *Store) string {
				t.Helper()
				if err := os.MkdirAll(store.baseDir, 0o750); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
				if err := os.WriteFile(store.Path(), []byte(`{"version":`), 0o600); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}

				return ""
			},
			wantErrorCode: apperr.CodeConfigBroken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			before := ""
			if tt.setup != nil {
				before = tt.setup(t, store)
			}

			err := store.Initialize()
			if gotCode := appErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("Initialize() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode != "" {
				return
			}

			result, err := store.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got := len(result.Config.ConnectionProfiles); got != tt.wantProfiles {
				t.Errorf("profiles = %d, want %d", got, tt.wantProfiles)
			}
			if !tt.wantPreserved {
				return
			}

			after, err := os.ReadFile(store.Path())
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if got := string(after); got != before {
				t.Errorf("config content = %q, want %q", got, before)
			}
		})
	}
}

// 設定読込失敗検証
func TestStoreLoadReturnsReadFailure(t *testing.T) {
	tests := []struct {
		name          string
		wantErrorCode apperr.Code
	}{
		{
			name:          "設定ディレクトリが通常ファイル",
			wantErrorCode: apperr.CodeConfigReadFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := filepath.Join(t.TempDir(), "not-a-directory")
			if err := os.WriteFile(baseDir, []byte("file"), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			_, err := NewStore(baseDir).Load()
			if gotCode := appErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("Load() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
		})
	}
}

// 破損設定非復旧検証
func TestStoreLoadDoesNotRecoverBrokenConfig(t *testing.T) {
	tests := []struct {
		name          string
		content       []byte
		wantErrorCode apperr.Code
	}{
		{
			name:          "破損したJSON",
			content:       []byte(`{"version":`),
			wantErrorCode: apperr.CodeConfigBroken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			if err := os.MkdirAll(store.baseDir, 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(store.Path(), tt.content, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := store.Load()
			if gotCode := appErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("Load() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			got, readErr := os.ReadFile(store.Path())
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(got) != string(tt.content) {
				t.Errorf("file content = %q, want %q", got, tt.content)
			}
		})
	}
}

// 不正プロファイル設定拒否
func TestStoreLoadRejectsInvalidConnectionProfileSchema(t *testing.T) {
	tests := []struct {
		name        string
		profileJSON string
	}{
		{
			name:        "パスワードフィールド",
			profileJSON: `{"id":"profile-1","name":"Local DB","dbType":"postgres","host":"localhost","port":5432,"database":"app","schema":"public","user":"user","password":"secret"}`,
		},
		{
			name:        "重複したID",
			profileJSON: `{"id":"profile-1","name":"Local DB","dbType":"postgres","host":"localhost","port":5432,"database":"app","schema":"public","user":"user"},{"id":"profile-1","name":"Replica","dbType":"postgres","host":"localhost","port":5432,"database":"app","schema":"public","user":"user"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			writeRawConfig(t, store, `{"version":1,"connectionProfiles":[`+tt.profileJSON+`],"activeConnectionProfileId":null,"flowStates":{}}`)

			_, err := store.Load()
			if !apperr.IsCode(err, apperr.CodeConfigBroken) {
				t.Errorf("Load() error = %v, want CONFIG_BROKEN", err)
			}
		})
	}
}

// 不正アクティブID拒否
func TestStoreLoadRejectsMissingOrInvalidActiveConnectionProfileID(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "フィールドなし",
			content: `{"version":1,"connectionProfiles":[],"flowStates":{}}`,
		},
		{
			name:    "不正な型",
			content: `{"version":1,"connectionProfiles":[],"activeConnectionProfileId":1,"flowStates":{}}`,
		},
		{
			name:    "存在しないプロファイル",
			content: `{"version":1,"connectionProfiles":[],"activeConnectionProfileId":"profile-1","flowStates":{}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			writeRawConfig(t, store, tt.content)
			_, err := store.Load()
			if !apperr.IsCode(err, apperr.CodeConfigBroken) {
				t.Errorf("Load() error = %v, want CONFIG_BROKEN", err)
			}
		})
	}
}

// 設定スキーマ検証
func TestStoreLoadRejectsInvalidSchema(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "必須フィールドなし",
			content: `{}`,
		},
		{
			name:    "未対応のバージョン",
			content: `{"version":2,"connectionProfiles":[],"activeConnectionProfileId":null,"flowStates":{}}`,
		},
		{
			name:    "バージョンの型が不正",
			content: `{"version":"1","connectionProfiles":[],"activeConnectionProfileId":null,"flowStates":{}}`,
		},
		{
			name:    "プロファイルがnull",
			content: `{"version":1,"connectionProfiles":null,"activeConnectionProfileId":null,"flowStates":{}}`,
		},
		{
			name:    "プロファイルがオブジェクト",
			content: `{"version":1,"connectionProfiles":{},"activeConnectionProfileId":null,"flowStates":{}}`,
		},
		{
			name:    "フロー状態がnull",
			content: `{"version":1,"connectionProfiles":[],"activeConnectionProfileId":null,"flowStates":null}`,
		},
		{
			name:    "フロー状態が配列",
			content: `{"version":1,"connectionProfiles":[],"activeConnectionProfileId":null,"flowStates":[]}`,
		},
		{
			name:    "未知のプロファイルフィールド",
			content: `{"version":1,"connectionProfiles":[{"id":"profile-1","name":"Local DB","dbType":"postgres","host":"localhost","port":5432,"database":"app","schema":"public","user":"user","extra":true}],"activeConnectionProfileId":null,"flowStates":{}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			writeRawConfig(t, store, tt.content)
			_, err := store.Load()
			if !apperr.IsCode(err, apperr.CodeConfigBroken) {
				t.Errorf("Load() error = %v, want CONFIG_BROKEN", err)
			}
		})
	}
}

// パスワード非保存検証
func TestStoreSaveDoesNotWritePassword(t *testing.T) {
	tests := []struct {
		name           string
		profile        ConnectionProfile
		unwantedString string
	}{
		{
			name: "パスワードを含まないプロファイル",
			profile: ConnectionProfile{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			},
			unwantedString: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			configuration := Default()
			configuration.ConnectionProfiles = []ConnectionProfile{tt.profile}
			if err := store.Save(configuration); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			bytes, err := os.ReadFile(store.Path())
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if strings.Contains(string(bytes), tt.unwantedString) {
				t.Errorf("saved config contains %q: %s", tt.unwantedString, bytes)
			}
		})
	}
}

// 設定保存失敗検証
func TestStoreSaveReturnsWriteFailure(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, store *Store)
	}{
		{
			name: "設定ディレクトリを作成できない",
			setup: func(t *testing.T, store *Store) {
				t.Helper()
				if err := os.WriteFile(store.baseDir, []byte("file"), 0o600); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			},
		},
		{
			name: "一時ファイルへ書き込めない",
			setup: func(t *testing.T, store *Store) {
				t.Helper()
				if err := os.MkdirAll(store.Path()+".tmp", 0o750); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			},
		},
		{
			name: "一時ファイルをリネームできない",
			setup: func(t *testing.T, store *Store) {
				t.Helper()
				if err := os.MkdirAll(store.Path(), 0o750); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(filepath.Join(t.TempDir(), "config"))
			tt.setup(t, store)
			if err := store.Save(Default()); !apperr.IsCode(err, apperr.CodeConfigWriteFailed) {
				t.Errorf("Save() error = %v, want CONFIG_WRITE_FAILED", err)
			}
		})
	}
}

// 接続プロファイルID存在確認検証
func TestContainsConnectionProfileID(t *testing.T) {
	profiles := []ConnectionProfile{
		{
			ID: "profile-1",
		},
	}
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "存在するID",
			id:   "profile-1",
			want: true,
		},
		{
			name: "存在しないID",
			id:   "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsConnectionProfileID(profiles, tt.id); got != tt.want {
				t.Errorf("containsConnectionProfileID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// アプリケーションエラーコード取得
func appErrorCode(err error) apperr.Code {
	if appErr := apperr.As(err); appErr != nil {
		return appErr.Code
	}

	return ""
}

// テスト設定ファイル作成
func writeRawConfig(t *testing.T, store *Store, content string) {
	t.Helper()
	if err := os.MkdirAll(store.baseDir, 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if !json.Valid([]byte(content)) {
		t.Fatalf("test config is not valid JSON: %s", content)
	}
	if err := os.WriteFile(store.Path(), []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
