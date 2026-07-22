package repository

import (
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

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}
