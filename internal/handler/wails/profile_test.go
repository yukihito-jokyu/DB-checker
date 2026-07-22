package wails

import (
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

				if *got.Data != *tt.wantData {
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
