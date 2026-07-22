package wails

import (
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 接続プロファイル確認
func TestAppHandlerCheckProfiles(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T) *AppHandler
		wantData  *ProfileCheckResponse
		wantError *ErrorResponse
	}{
		{
			name: "プロファイル検証結果を返す",
			setup: func(t *testing.T) *AppHandler {
				repository := &connectionProfileRepositoryStub{
					profiles: []domain.Profile{
						newTestProfile(t, "profile-1", domain.DBTypePostgres),
						newTestProfile(t, "profile-2", domain.DBTypeMySQL),
					},
					activeID: stringPointer("profile-1"),
				}

				return newTestAppHandler(t, config.NewStore(t.TempDir()), repository)
			},
			wantData: &ProfileCheckResponse{
				Valid:        true,
				ProfileCount: 2,
			},
		},
		{
			name: "アクティブプロファイルがない場合に設定エラーを返す",
			setup: func(t *testing.T) *AppHandler {
				repository := &connectionProfileRepositoryStub{
					profiles: []domain.Profile{newTestProfile(t, "profile-1", domain.DBTypePostgres)},
					activeID: stringPointer("missing"),
				}

				return newTestAppHandler(t, config.NewStore(t.TempDir()), repository)
			},
			wantError: &ErrorResponse{
				Code:    string(apperr.CodeConfigBroken),
				Message: "設定ファイルが壊れています",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setup(t)

			got := handler.CheckProfiles()

			if tt.wantData == nil && got.Data != nil {
				t.Errorf("CheckProfiles() Data = %#v, want nil", got.Data)
			}

			if tt.wantData != nil {
				if got.Data == nil {
					t.Fatal("CheckProfiles() Data = nil, want non-nil")
				}

				if !reflect.DeepEqual(*got.Data, *tt.wantData) {
					t.Errorf("CheckProfiles() Data = %#v, want %#v", *got.Data, *tt.wantData)
				}
			}

			if tt.wantError == nil && got.Error != nil {
				t.Errorf("CheckProfiles() Error = %#v, want nil", got.Error)
			}

			if tt.wantError != nil {
				if got.Error == nil {
					t.Fatal("CheckProfiles() Error = nil, want non-nil")
				}

				if *got.Error != *tt.wantError {
					t.Errorf("CheckProfiles() Error = %#v, want %#v", *got.Error, *tt.wantError)
				}
			}
		})
	}
}

// 接続プロファイル一覧取得
func TestAppHandlerListConnectionProfiles(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T) (*AppHandler, *connectionProfileRepositoryStub)
		wantData  *ConnectionProfilesResponse
		wantError *ErrorResponse
	}{
		{
			name: "複数プロファイルとアクティブIDを返す",
			setup: func(t *testing.T) (*AppHandler, *connectionProfileRepositoryStub) {
				repository := &connectionProfileRepositoryStub{
					profiles: []domain.Profile{
						newTestProfile(t, "profile-1", domain.DBTypePostgres),
						newTestProfile(t, "profile-2", domain.DBTypeMySQL),
					},
					activeID: stringPointer("profile-2"),
				}

				return newTestAppHandler(t, config.NewStore(t.TempDir()), repository), repository
			},
			wantData: &ConnectionProfilesResponse{
				Profiles: []ProfileResponse{
					{
						ID:       "profile-1",
						Name:     "Local profile-1",
						DBType:   "postgres",
						Host:     "localhost",
						Port:     5432,
						Database: "app",
						Schema:   "public",
						User:     "user",
					},
					{
						ID:       "profile-2",
						Name:     "Local profile-2",
						DBType:   "mysql",
						Host:     "localhost",
						Port:     5432,
						Database: "app",
						Schema:   "",
						User:     "user",
					},
				},
				ActiveConnectionProfileID: stringPointer("profile-2"),
			},
		},
		{
			name: "空一覧と未選択アクティブIDを返す",
			setup: func(t *testing.T) (*AppHandler, *connectionProfileRepositoryStub) {
				repository := &connectionProfileRepositoryStub{}

				return newTestAppHandler(t, config.NewStore(t.TempDir()), repository), repository
			},
			wantData: &ConnectionProfilesResponse{
				Profiles:                  []ProfileResponse{},
				ActiveConnectionProfileID: nil,
			},
		},
		{
			name: "存在しないアクティブIDを安全な設定エラーとして返す",
			setup: func(t *testing.T) (*AppHandler, *connectionProfileRepositoryStub) {
				repository := &connectionProfileRepositoryStub{
					profiles: []domain.Profile{newTestProfile(t, "profile-1", domain.DBTypePostgres)},
					activeID: stringPointer("missing"),
				}

				return newTestAppHandler(t, config.NewStore(t.TempDir()), repository), repository
			},
			wantError: &ErrorResponse{
				Code:    string(apperr.CodeConfigBroken),
				Message: "設定ファイルが壊れています",
			},
		},
		{
			name: "設定未作成を安全なエラーとして返す",
			setup: func(t *testing.T) (*AppHandler, *connectionProfileRepositoryStub) {
				repository := &connectionProfileRepositoryStub{
					err: apperr.New(apperr.CodeConfigNotFound),
				}

				return newTestAppHandler(t, config.NewStore(t.TempDir()), repository), repository
			},
			wantError: &ErrorResponse{
				Code:    string(apperr.CodeConfigNotFound),
				Message: "設定ファイルが見つかりません",
			},
		},
		{
			name: "設定読み込み失敗を安全なエラーとして返す",
			setup: func(t *testing.T) (*AppHandler, *connectionProfileRepositoryStub) {
				repository := &connectionProfileRepositoryStub{
					err: apperr.New(apperr.CodeConfigReadFailed),
				}

				return newTestAppHandler(t, config.NewStore(t.TempDir()), repository), repository
			},
			wantError: &ErrorResponse{
				Code:    string(apperr.CodeConfigReadFailed),
				Message: "設定ファイルの読み込みに失敗しました",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, repository := tt.setup(t)

			got := handler.ListConnectionProfiles()

			if gotCalls := repository.calls; gotCalls != 1 {
				t.Errorf("LoadProfiles() calls = %d, want %d", gotCalls, 1)
			}
			if tt.wantData == nil && got.Data != nil {
				t.Errorf("ListConnectionProfiles() Data = %#v, want nil", got.Data)
			}
			if tt.wantData != nil {
				if got.Data == nil {
					t.Fatal("ListConnectionProfiles() Data = nil, want non-nil")
				}
				if !reflect.DeepEqual(*got.Data, *tt.wantData) {
					t.Errorf("ListConnectionProfiles() Data = %#v, want %#v", *got.Data, *tt.wantData)
				}
			}
			if tt.wantError == nil && got.Error != nil {
				t.Errorf("ListConnectionProfiles() Error = %#v, want nil", got.Error)
			}
			if tt.wantError != nil {
				if got.Error == nil {
					t.Fatal("ListConnectionProfiles() Error = nil, want non-nil")
				}
				if *got.Error != *tt.wantError {
					t.Errorf("ListConnectionProfiles() Error = %#v, want %#v", *got.Error, *tt.wantError)
				}
			}
		})
	}
}
