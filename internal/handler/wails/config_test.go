package wails

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// アプリ設定返却検証
func TestAppHandlerGetConfig(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T) *AppHandler
		wantData  *ConfigResponse
		wantError *ErrorResponse
	}{
		{
			name: "保存済みの設定を返す",
			setup: func(t *testing.T) *AppHandler {
				activeID := "profile-1"
				store := config.NewStore(t.TempDir())
				cfg := config.Config{
					Version: config.FileVersion,
					ConnectionProfiles: []config.ConnectionProfile{{
						ID:       activeID,
						Name:     "Local DB",
						DBType:   "postgres",
						Host:     "localhost",
						Port:     5432,
						Database: "app",
						Schema:   "public",
						User:     "user",
					}},
					ActiveConnectionProfileID: &activeID,
					FlowStates: map[string]json.RawMessage{
						"table": json.RawMessage(`{"name":"users"}`),
					},
				}
				if err := store.Save(cfg); err != nil {
					t.Fatalf("Save() error = %v", err)
				}

				return newTestAppHandler(t, store, &connectionProfileRepositoryStub{
					profiles: []domain.Profile{newTestProfile(t, activeID, domain.DBTypePostgres)},
					activeID: &activeID,
				})
			},
			wantData: &ConfigResponse{
				Version: config.FileVersion,
				ConnectionProfiles: []ProfileResponse{{
					ID:       "profile-1",
					Name:     "Local DB",
					DBType:   "postgres",
					Host:     "localhost",
					Port:     5432,
					Database: "app",
					Schema:   "public",
					User:     "user",
				}},
				ActiveConnectionProfileID: stringPointer("profile-1"),
				FlowStates:                map[string]any{"table": map[string]any{"name": "users"}},
			},
		},
		{
			name: "アクティブプロファイルがない場合に設定エラーを返す",
			setup: func(t *testing.T) *AppHandler {
				return newTestAppHandler(t, config.NewStore(t.TempDir()), &connectionProfileRepositoryStub{
					activeID: stringPointer("missing"),
				})
			},
			wantError: &ErrorResponse{
				Code:    string(apperr.CodeConfigBroken),
				Message: "設定ファイルが壊れています",
			},
		},
		{
			name: "設定がない場合に未検出エラーを返す",
			setup: func(t *testing.T) *AppHandler {
				return newTestAppHandler(t, config.NewStore(t.TempDir()), &connectionProfileRepositoryStub{})
			},
			wantError: &ErrorResponse{
				Code:    string(apperr.CodeConfigNotFound),
				Message: "設定ファイルが見つかりません",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setup(t)

			got := handler.GetConfig()

			if tt.wantData == nil && got.Data != nil {
				t.Errorf("GetConfig() Data = %#v, want nil", got.Data)
			}

			if tt.wantData != nil {
				if got.Data == nil {
					t.Fatal("GetConfig() Data = nil, want non-nil")
				}

				if !reflect.DeepEqual(*got.Data, *tt.wantData) {
					t.Errorf("GetConfig() Data = %#v, want %#v", *got.Data, *tt.wantData)
				}
			}

			if tt.wantError == nil && got.Error != nil {
				t.Errorf("GetConfig() Error = %#v, want nil", got.Error)
			}

			if tt.wantError != nil {
				if got.Error == nil {
					t.Fatal("GetConfig() Error = nil, want non-nil")
				}

				if *got.Error != *tt.wantError {
					t.Errorf("GetConfig() Error = %#v, want %#v", *got.Error, *tt.wantError)
				}
			}
		})
	}
}

// 設定レスポンス変換確認
func TestToConfigResponse(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want ConfigResponse
	}{
		{
			name: "すべてのフィールドを変換する",
			cfg: config.Config{
				Version: config.FileVersion,
				ConnectionProfiles: []config.ConnectionProfile{{
					ID:       "profile-1",
					Name:     "Local DB",
					DBType:   "postgres",
					Host:     "localhost",
					Port:     5432,
					Database: "app",
					Schema:   "public",
					User:     "user",
				}},
				ActiveConnectionProfileID: stringPointer("profile-1"),
				FlowStates: map[string]json.RawMessage{
					"enabled": json.RawMessage(`true`),
				},
			},
			want: ConfigResponse{
				Version: config.FileVersion,
				ConnectionProfiles: []ProfileResponse{{
					ID:       "profile-1",
					Name:     "Local DB",
					DBType:   "postgres",
					Host:     "localhost",
					Port:     5432,
					Database: "app",
					Schema:   "public",
					User:     "user",
				}},
				ActiveConnectionProfileID: stringPointer("profile-1"),
				FlowStates:                map[string]any{"enabled": true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toConfigResponse(tt.cfg)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toConfigResponse() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// フロー状態復号確認
func TestDecodeFlowStates(t *testing.T) {
	tests := []struct {
		name       string
		flowStates map[string]json.RawMessage
		want       map[string]any
	}{
		{
			name: "JSON値を復号する",
			flowStates: map[string]json.RawMessage{
				"enabled": json.RawMessage(`true`),
				"table":   json.RawMessage(`{"name":"users"}`),
			},
			want: map[string]any{
				"enabled": true,
				"table":   map[string]any{"name": "users"},
			},
		},
		{
			name: "不正なJSONを文字列として保持する",
			flowStates: map[string]json.RawMessage{
				"broken": json.RawMessage(`{`),
			},
			want: map[string]any{"broken": "{"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeFlowStates(tt.flowStates)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeFlowStates() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
