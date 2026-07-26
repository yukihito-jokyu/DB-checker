package repository

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 接続プロファイル読込検証
func TestAppRepositoryLoadProfiles(t *testing.T) {
	tests := []struct {
		name          string
		profiles      []config.ConnectionProfile
		activeID      *string
		wantProfiles  []domain.Profile
		wantActiveID  *string
		wantErrorCode apperr.Code
		withoutConfig bool
	}{
		{
			name: "パスワードを含めずに読み込む",
			profiles: []config.ConnectionProfile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			wantProfiles: []domain.Profile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   domain.DBTypePostgres,
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
		},
		{
			name: "アクティブIDを読み込む",
			profiles: []config.ConnectionProfile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			activeID: stringPointer("profile-1"),
			wantProfiles: []domain.Profile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   domain.DBTypePostgres,
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			wantActiveID: stringPointer("profile-1"),
		},
		{
			name: "不正な設定を設定エラーとして返す",
			profiles: []config.ConnectionProfile{{
				ID:       "profile-1",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			wantErrorCode: apperr.CodeConfigBroken,
		},
		{
			name:          "設定ストアエラーを返す",
			wantErrorCode: apperr.CodeConfigNotFound,
			withoutConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := config.NewStore(t.TempDir())
			if !tt.withoutConfig {
				configuration := config.Default()
				configuration.ConnectionProfiles = tt.profiles
				configuration.ActiveConnectionProfileID = tt.activeID
				if err := store.Save(configuration); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			profiles, activeID, err := NewAppRepository(store).LoadProfiles()
			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("LoadProfiles() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode != "" {
				return
			}

			if !reflect.DeepEqual(profiles, tt.wantProfiles) {
				t.Errorf("LoadProfiles() profiles = %#v, want %#v", profiles, tt.wantProfiles)
			}
			if gotFound := activeID != nil; gotFound != (tt.wantActiveID != nil) {
				t.Fatalf("LoadProfiles() active ID found = %v, want %v", gotFound, tt.wantActiveID != nil)
			}
			if tt.wantActiveID != nil && *activeID != *tt.wantActiveID {
				t.Errorf("LoadProfiles() active ID = %q, want %q", *activeID, *tt.wantActiveID)
			}
		})
	}
}

// 接続プロファイル保存検証
func TestAppRepositorySaveProfiles(t *testing.T) {
	profile, err := domain.NewProfile("profile-2", "Remote DB", domain.DBTypeMySQL, "db.example.com", 3306, "app", "", "admin")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	tests := []struct {
		name          string
		profiles      []domain.Profile
		activeID      *string
		wantProfiles  []config.ConnectionProfile
		wantActiveID  *string
		wantFlowState string
		wantErrorCode apperr.Code
	}{
		{
			name:     "プロファイルとアクティブIDを保存する",
			profiles: []domain.Profile{profile},
			activeID: stringPointer("profile-2"),
			wantProfiles: []config.ConnectionProfile{
				{
					ID:       "profile-2",
					Name:     "Remote DB",
					DBType:   "mysql",
					Host:     "db.example.com",
					Port:     3306,
					Database: "app",
					Schema:   "",
					User:     "admin",
				},
			},
			wantActiveID:  stringPointer("profile-2"),
			wantFlowState: "complete",
		},
		{
			name:          "空のプロファイルと未選択アクティブIDを保存する",
			profiles:      []domain.Profile{},
			activeID:      nil,
			wantProfiles:  []config.ConnectionProfile{},
			wantActiveID:  nil,
			wantFlowState: "complete",
		},
		{
			name:          "設定ストアエラーを返す",
			wantErrorCode: apperr.CodeConfigNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := config.NewStore(t.TempDir())
			if tt.wantErrorCode == "" {
				configuration := config.Default()
				configuration.FlowStates["wizard"] = []byte(`{"step":"complete"}`)
				if err := store.Save(configuration); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			err := NewAppRepository(store).SaveProfiles(tt.profiles, tt.activeID)
			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("SaveProfiles() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode != "" {
				return
			}

			result, err := store.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !reflect.DeepEqual(result.Config.ConnectionProfiles, tt.wantProfiles) {
				t.Errorf("SaveProfiles() profiles = %#v, want %#v", result.Config.ConnectionProfiles, tt.wantProfiles)
			}
			if !reflect.DeepEqual(result.Config.ActiveConnectionProfileID, tt.wantActiveID) {
				t.Errorf("SaveProfiles() active ID = %#v, want %#v", result.Config.ActiveConnectionProfileID, tt.wantActiveID)
			}
			var flowState map[string]string
			if err := json.Unmarshal(result.Config.FlowStates["wizard"], &flowState); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got := flowState["step"]; got != tt.wantFlowState {
				t.Errorf("SaveProfiles() flow state = %q, want %q", got, tt.wantFlowState)
			}
		})
	}
}

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}
