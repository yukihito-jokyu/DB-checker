package wails

import (
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
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

// 接続プロファイル保存入力検証
func TestAppHandlerSaveConnectionProfileRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name    string
		request SaveConnectionProfileRequest
	}{
		{
			name: "不正な入力を安全なエラーで返す",
			request: SaveConnectionProfileRequest{
				Name:     "Local DB",
				DBType:   "postgres",
				Host:     "",
				Port:     5432,
				Database: "app",
				Schema:   stringPointer("public"),
				User:     "user",
				Password: "secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestAppHandler(t, config.NewStore(t.TempDir()), &connectionProfileRepositoryStub{})

			got := handler.SaveConnectionProfile(tt.request)

			if got.Data != nil {
				t.Errorf("SaveConnectionProfile() Data = %#v, want nil", got.Data)
			}
			if got.Error == nil {
				t.Fatal("SaveConnectionProfile() Error = nil, want non-nil")
			}
			if got.Error.Code != string(apperr.CodeValidationFailed) {
				t.Errorf("SaveConnectionProfile() Error.Code = %q, want %q", got.Error.Code, apperr.CodeValidationFailed)
			}
		})
	}
}

// 接続プロファイル保存成功応答
func TestAppHandlerSaveConnectionProfile(t *testing.T) {
	tests := []struct {
		name     string
		request  SaveConnectionProfileRequest
		profiles []domain.Profile
		activeID *string
		wantData ConnectionProfilesResponse
	}{
		{
			name: "保存後のプロファイルとアクティブIDを返す",
			request: SaveConnectionProfileRequest{
				ID:       "profile-1",
				Name:     "Updated DB",
				DBType:   "postgres",
				Host:     "db.example.com",
				Port:     5433,
				Database: "updated",
				Schema:   stringPointer("public"),
				User:     "admin",
				Password: "new-password",
			},
			profiles: []domain.Profile{
				newTestProfile(t, "profile-1", domain.DBTypePostgres),
			},
			activeID: stringPointer("profile-1"),
			wantData: ConnectionProfilesResponse{
				Profiles: []ProfileResponse{
					{
						ID:       "profile-1",
						Name:     "Updated DB",
						DBType:   "postgres",
						Host:     "db.example.com",
						Port:     5433,
						Database: "updated",
						Schema:   "public",
						User:     "admin",
					},
				},
				ActiveConnectionProfileID: stringPointer("profile-1"),
			},
		},
		{
			name: "MySQLはスキーマ未指定で保存する",
			request: SaveConnectionProfileRequest{
				ID:       "profile-2",
				Name:     "Updated MySQL",
				DBType:   "mysql",
				Host:     "mysql.example.com",
				Port:     3306,
				Database: "updated",
				User:     "admin",
				Password: "new-password",
			},
			profiles: []domain.Profile{
				newTestProfile(t, "profile-2", domain.DBTypeMySQL),
			},
			wantData: ConnectionProfilesResponse{
				Profiles: []ProfileResponse{
					{
						ID:       "profile-2",
						Name:     "Updated MySQL",
						DBType:   "mysql",
						Host:     "mysql.example.com",
						Port:     3306,
						Database: "updated",
						Schema:   "",
						User:     "admin",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestAppHandler(t, config.NewStore(t.TempDir()), &connectionProfileRepositoryStub{
				profiles: tt.profiles,
				activeID: tt.activeID,
			})

			got := handler.SaveConnectionProfile(tt.request)

			if got.Error != nil {
				t.Fatalf("SaveConnectionProfile() Error = %#v, want nil", got.Error)
			}
			if got.Data == nil {
				t.Fatal("SaveConnectionProfile() Data = nil, want non-nil")
			}
			if !reflect.DeepEqual(*got.Data, tt.wantData) {
				t.Errorf("SaveConnectionProfile() Data = %#v, want %#v", *got.Data, tt.wantData)
			}
		})
	}
}

// アクティブ接続プロファイル切替成功応答
func TestAppHandlerActivateConnectionProfile(t *testing.T) {
	tests := []struct {
		name         string
		profileID    string
		profiles     []domain.Profile
		activeID     *string
		credential   string
		found        bool
		wantData     *ConnectionProfilesResponse
		wantGetIDs   []string
		wantSaveCall int
		wantCheck    int
		wantPassword string
	}{
		{
			name:      "保存済み資格情報で切り替え結果を返す",
			profileID: "profile-2",
			profiles: []domain.Profile{
				newTestProfile(t, "profile-1", domain.DBTypePostgres),
				newTestProfile(t, "profile-2", domain.DBTypeMySQL),
			},
			activeID:   stringPointer("profile-1"),
			credential: "secret-password",
			found:      true,
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
			wantGetIDs:   []string{"profile-2"},
			wantSaveCall: 1,
			wantCheck:    1,
			wantPassword: "secret-password",
		},
		{
			name:      "使用中のプロファイルは確認せずに返す",
			profileID: "profile-1",
			profiles:  []domain.Profile{newTestProfile(t, "profile-1", domain.DBTypePostgres)},
			activeID:  stringPointer("profile-1"),
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
				},
				ActiveConnectionProfileID: stringPointer("profile-1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &connectionProfileRepositoryStub{
				profiles:        tt.profiles,
				activeID:        tt.activeID,
				credential:      tt.credential,
				credentialFound: tt.found,
			}
			handler := newTestAppHandler(t, config.NewStore(t.TempDir()), repository)

			got := handler.ActivateConnectionProfile(tt.profileID)

			if got.Error != nil {
				t.Fatalf("ActivateConnectionProfile() Error = %#v, want nil", got.Error)
			}
			if got.Data == nil {
				t.Fatal("ActivateConnectionProfile() Data = nil, want non-nil")
			}
			if !reflect.DeepEqual(*got.Data, *tt.wantData) {
				t.Errorf("ActivateConnectionProfile() Data = %#v, want %#v", *got.Data, *tt.wantData)
			}
			if !reflect.DeepEqual(repository.credentialIDs, tt.wantGetIDs) {
				t.Errorf("GetCredential() profile IDs = %#v, want %#v", repository.credentialIDs, tt.wantGetIDs)
			}
			if gotCalls := repository.saveCalls; gotCalls != tt.wantSaveCall {
				t.Errorf("SaveProfiles() calls = %d, want %d", gotCalls, tt.wantSaveCall)
			}
			if gotCalls := repository.connectionCalls; gotCalls != tt.wantCheck {
				t.Errorf("CheckConnection() calls = %d, want %d", gotCalls, tt.wantCheck)
			}
			if gotPassword := repository.password; gotPassword != tt.wantPassword {
				t.Errorf("CheckConnection() password = %q, want %q", gotPassword, tt.wantPassword)
			}
		})
	}
}

// 接続プロファイル削除応答
func TestAppHandlerDeleteConnectionProfile(t *testing.T) {
	tests := []struct {
		name        string
		profileID   string
		profiles    []domain.Profile
		activeID    *string
		deleteErr   error
		wantData    *ConnectionProfilesResponse
		wantError   *ErrorResponse
		wantDelete  []string
		wantSave    int
		wantConnect int
	}{
		{
			name:      "アクティブプロファイルを削除して未選択を返す",
			profileID: "profile-1",
			profiles: []domain.Profile{
				newTestProfile(t, "profile-1", domain.DBTypePostgres),
				newTestProfile(t, "profile-2", domain.DBTypeMySQL),
			},
			activeID: stringPointer("profile-1"),
			wantData: &ConnectionProfilesResponse{
				Profiles: []ProfileResponse{
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
				ActiveConnectionProfileID: nil,
			},
			wantDelete: []string{"profile-1"},
			wantSave:   1,
		},
		{
			name:      "資格情報削除失敗を安全なエラーとして返す",
			profileID: "profile-1",
			profiles: []domain.Profile{
				newTestProfile(t, "profile-1", domain.DBTypePostgres),
			},
			deleteErr: errors.New("credential store failure: password=secret-password"),
			wantError: &ErrorResponse{
				Code:    string(apperr.CodeCredentialDeleteFailed),
				Message: "資格情報の削除に失敗しました",
			},
			wantDelete: []string{"profile-1"},
			wantSave:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &connectionProfileRepositoryStub{
				profiles:  tt.profiles,
				activeID:  tt.activeID,
				deleteErr: tt.deleteErr,
			}
			handler := newTestAppHandler(t, config.NewStore(t.TempDir()), repository)

			got := handler.DeleteConnectionProfile(tt.profileID)

			if tt.wantData == nil && got.Data != nil {
				t.Errorf("DeleteConnectionProfile() Data = %#v, want nil", got.Data)
			}
			if tt.wantData != nil {
				if got.Data == nil {
					t.Fatal("DeleteConnectionProfile() Data = nil, want non-nil")
				}
				if !reflect.DeepEqual(*got.Data, *tt.wantData) {
					t.Errorf("DeleteConnectionProfile() Data = %#v, want %#v", *got.Data, *tt.wantData)
				}
			}
			if tt.wantError == nil && got.Error != nil {
				t.Errorf("DeleteConnectionProfile() Error = %#v, want nil", got.Error)
			}
			if tt.wantError != nil {
				if got.Error == nil {
					t.Fatal("DeleteConnectionProfile() Error = nil, want non-nil")
				}
				if *got.Error != *tt.wantError {
					t.Errorf("DeleteConnectionProfile() Error = %#v, want %#v", *got.Error, *tt.wantError)
				}
			}
			if !reflect.DeepEqual(repository.deleteIDs, tt.wantDelete) {
				t.Errorf("DeleteCredential() profile IDs = %#v, want %#v", repository.deleteIDs, tt.wantDelete)
			}
			if got := repository.saveCalls; got != tt.wantSave {
				t.Errorf("SaveProfiles() calls = %d, want %d", got, tt.wantSave)
			}
			if got := repository.connectionCalls; got != tt.wantConnect {
				t.Errorf("CheckConnection() calls = %d, want %d", got, tt.wantConnect)
			}
		})
	}
}

// 接続プロファイル削除失敗ログの秘密情報非出力
func TestAppHandlerDeleteConnectionProfileDoesNotLogCredentialErrorCause(t *testing.T) {
	tests := []struct {
		name              string
		deleteError       error
		wantErrorCode     apperr.Code
		wantLogSubstring  string
		avoidLogSubstring string
	}{
		{
			name:              "資格情報削除失敗の原因をログへ出力しない",
			deleteError:       errors.New("credential store failure: password=secret-password host=db.example.com"),
			wantErrorCode:     apperr.CodeCredentialDeleteFailed,
			wantLogSubstring:  "code=CREDENTIAL_DELETE_FAILED",
			avoidLogSubstring: "secret-password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			logger := applogger.NewWithWriter(&buffer, slog.LevelDebug)
			repository := &connectionProfileRepositoryStub{
				profiles: []domain.Profile{
					newTestProfile(t, "profile-1", domain.DBTypePostgres),
				},
				deleteErr: tt.deleteError,
			}
			handler := NewAppHandler(logger, config.NewStore(t.TempDir()), usecase.NewAppUseCase(repository), usecase.NewInspectionUseCase(repository))

			result := handler.DeleteConnectionProfile("profile-1")

			if result.Error == nil {
				t.Fatal("DeleteConnectionProfile() Error = nil, want non-nil")
			}
			if got := result.Error.Code; got != string(tt.wantErrorCode) {
				t.Errorf("DeleteConnectionProfile() Error.Code = %q, want %q", got, tt.wantErrorCode)
			}
			if got := result.Error.Message; strings.Contains(got, tt.avoidLogSubstring) {
				t.Errorf("DeleteConnectionProfile() Error.Message = %q, want no substring %q", got, tt.avoidLogSubstring)
			}

			output := buffer.String()
			if !strings.Contains(output, tt.wantLogSubstring) {
				t.Errorf("log output = %q, want substring %q", output, tt.wantLogSubstring)
			}
			if strings.Contains(output, tt.avoidLogSubstring) {
				t.Errorf("log output = %q, want no substring %q", output, tt.avoidLogSubstring)
			}
		})
	}
}

// 接続切替失敗ログの秘密情報非出力
func TestAppHandlerActivateConnectionProfileDoesNotLogConnectionErrorCause(t *testing.T) {
	tests := []struct {
		name              string
		connectionError   error
		wantErrorCode     apperr.Code
		wantLogSubstring  string
		avoidLogSubstring string
	}{
		{
			name:              "接続失敗の原因と資格情報をログへ出力しない",
			connectionError:   errors.New("connection failed: password=secret-password"),
			wantErrorCode:     apperr.CodeConnectionFailed,
			wantLogSubstring:  "code=CONNECTION_FAILED",
			avoidLogSubstring: "secret-password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			logger := applogger.NewWithWriter(&buffer, slog.LevelDebug)
			profileRepository := &connectionProfileRepositoryStub{
				profiles: []domain.Profile{
					newTestProfile(t, "profile-1", domain.DBTypePostgres),
				},
				credential:      "secret-password",
				credentialFound: true,
				connectionErr:   tt.connectionError,
			}
			appUseCase := usecase.NewAppUseCase(profileRepository)
			handler := NewAppHandler(logger, config.NewStore(t.TempDir()), appUseCase, usecase.NewInspectionUseCase(profileRepository))

			result := handler.ActivateConnectionProfile("profile-1")

			if result.Error == nil {
				t.Fatal("ActivateConnectionProfile() Error = nil, want non-nil")
			}
			if gotCode := result.Error.Code; gotCode != string(tt.wantErrorCode) {
				t.Errorf("ActivateConnectionProfile() Error.Code = %q, want %q", gotCode, tt.wantErrorCode)
			}

			output := buffer.String()
			if !strings.Contains(output, tt.wantLogSubstring) {
				t.Errorf("log output = %q, want substring %q", output, tt.wantLogSubstring)
			}
			if strings.Contains(output, tt.avoidLogSubstring) {
				t.Errorf("log output = %q, want no substring %q", output, tt.avoidLogSubstring)
			}
		})
	}
}

// 接続失敗ログの秘密情報非出力
func TestAppHandlerSaveConnectionProfileDoesNotLogConnectionErrorCause(t *testing.T) {
	tests := []struct {
		name              string
		connectionError   error
		request           SaveConnectionProfileRequest
		wantErrorCode     apperr.Code
		wantLogSubstring  string
		avoidLogSubstring string
	}{
		{
			name:            "接続失敗の原因とパスワードをログへ出力しない",
			connectionError: errors.New("connection failed: password=secret-password"),
			request: SaveConnectionProfileRequest{
				Name:     "Local DB",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   stringPointer("public"),
				User:     "user",
				Password: "secret-password",
			},
			wantErrorCode:     apperr.CodeConnectionFailed,
			wantLogSubstring:  "code=CONNECTION_FAILED",
			avoidLogSubstring: "secret-password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			logger := applogger.NewWithWriter(&buffer, slog.LevelDebug)
			profileRepository := &connectionProfileRepositoryStub{
				connectionErr: tt.connectionError,
			}
			appUseCase := usecase.NewAppUseCase(profileRepository)
			handler := NewAppHandler(logger, config.NewStore(t.TempDir()), appUseCase, usecase.NewInspectionUseCase(profileRepository))

			result := handler.SaveConnectionProfile(tt.request)

			if result.Error == nil {
				t.Fatal("SaveConnectionProfile() Error = nil, want non-nil")
			}
			if result.Error.Code != string(tt.wantErrorCode) {
				t.Errorf("SaveConnectionProfile() Error.Code = %q, want %q", result.Error.Code, tt.wantErrorCode)
			}

			output := buffer.String()
			if !strings.Contains(output, tt.wantLogSubstring) {
				t.Errorf("log output = %q, want substring %q", output, tt.wantLogSubstring)
			}
			if strings.Contains(output, tt.avoidLogSubstring) {
				t.Errorf("log output = %q, want no substring %q", output, tt.avoidLogSubstring)
			}
		})
	}
}
