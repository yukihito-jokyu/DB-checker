package wails

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
)

type verificationScenarioHandlerProfilesStub struct {
	profiles []domain.Profile
	activeID *string
	err      error
}

type verificationScenarioHandlerRepositoryStub struct {
	scenarios []domain.VerificationScenarioSummary
	scenario  domain.VerificationScenario
	found     bool
	err       error
}

// プロファイル読込再現
func (s verificationScenarioHandlerProfilesStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return s.profiles, s.activeID, s.err
}

// シナリオ一覧取得再現
func (s verificationScenarioHandlerRepositoryStub) ListVerificationScenarios(_ context.Context, _ string) ([]domain.VerificationScenarioSummary, error) {
	return s.scenarios, s.err
}

// シナリオ詳細取得再現
func (s verificationScenarioHandlerRepositoryStub) GetVerificationScenario(_ context.Context, _, _ string) (domain.VerificationScenario, bool, error) {
	return s.scenario, s.found, s.err
}

// シナリオ詳細ハンドラー応答検証
func TestAppHandlerGetVerificationScenario(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	createdAt := time.Date(2026, time.August, 8, 11, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	updatedAt := time.Date(2026, time.August, 8, 12, 0, 0, 123456789, time.FixedZone("JST", 9*60*60))
	scenario, err := domain.NewVerificationScenario("scenario-1", "検証", "orders", []byte(`{"rowCounts":{"orders":10}}`), nil, createdAt, updatedAt)
	if err != nil {
		t.Fatalf("NewVerificationScenario() error = %v", err)
	}
	tests := []struct {
		name       string
		useCaseNil bool
		profiles   verificationScenarioHandlerProfilesStub
		repository verificationScenarioHandlerRepositoryStub
		wantData   VerificationScenarioResponse
		wantCode   apperr.Code
		wantLog    bool
	}{
		{
			name: "定義と時刻を含むDTOを返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{
					profile,
				},
				activeID: stringPointer(profile.ID),
			},
			repository: verificationScenarioHandlerRepositoryStub{
				scenario: scenario,
				found:    true,
			},
			wantData: VerificationScenarioResponse{
				ID:           "scenario-1",
				Name:         "検証",
				PrimaryTable: "orders",
				Definition: map[string]any{
					"rowCounts": map[string]any{
						"orders": float64(10),
					},
				},
				CreatedAt: "2026-08-08T02:00:00Z",
				UpdatedAt: "2026-08-08T03:00:00.123456789Z",
			},
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
			wantLog:    true,
		},
		{
			name: "他プロファイルのシナリオを安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{
					profile,
				},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeScenarioNotFound,
			wantLog:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			if !tt.useCaseNil {
				scenarioUseCase = usecase.NewVerificationScenarioUseCase(tt.profiles, tt.repository)
			}
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil, scenarioUseCase)

			got := handler.GetVerificationScenario("scenario-1")
			if tt.wantCode == "" {
				if got.Error != nil {
					t.Fatalf("GetVerificationScenario() Error = %#v, want nil", got.Error)
				}
				if got.Data == nil {
					t.Fatal("GetVerificationScenario() Data = nil, want non-nil")
				}
				if !reflect.DeepEqual(*got.Data, tt.wantData) {
					t.Errorf("GetVerificationScenario() Data = %#v, want %#v", *got.Data, tt.wantData)
				}

				return
			}
			if got.Data != nil {
				t.Errorf("GetVerificationScenario() Data = %#v, want nil", got.Data)
			}
			if got.Error == nil {
				t.Fatal("GetVerificationScenario() Error = nil, want error response")
			}
			if got.Error.Code != string(tt.wantCode) {
				t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if got.Error.Message == "" || bytes.Contains([]byte(got.Error.Message), []byte("secret")) {
				t.Errorf("Error.Message = %q, want safe non-empty message", got.Error.Message)
			}
			if tt.wantLog && !bytes.Contains(output.Bytes(), []byte("code="+string(tt.wantCode))) {
				t.Errorf("log = %q, want code=%s", output.String(), tt.wantCode)
			}
			if bytes.Contains(output.Bytes(), []byte("secret")) || bytes.Contains(output.Bytes(), []byte("verification.sqlite3")) {
				t.Errorf("log = %q, must not contain sensitive details", output.String())
			}
		})
	}
}

// シナリオ一覧ハンドラー応答検証
func TestAppHandlerListVerificationScenarios(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	updatedAt := time.Date(2026, time.August, 8, 12, 0, 0, 123456789, time.FixedZone("JST", 9*60*60))
	tests := []struct {
		name       string
		useCaseNil bool
		profiles   verificationScenarioHandlerProfilesStub
		repository verificationScenarioHandlerRepositoryStub
		wantData   []VerificationScenarioSummaryResponse
		wantCode   apperr.Code
		wantLog    bool
	}{
		{
			name: "小数秒を含むDTOを返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			repository: verificationScenarioHandlerRepositoryStub{
				scenarios: []domain.VerificationScenarioSummary{
					{
						ID:           "scenario-1",
						Name:         "検証",
						PrimaryTable: "orders",
						UpdatedAt:    updatedAt,
					},
				},
			},
			wantData: []VerificationScenarioSummaryResponse{
				{
					ID:           "scenario-1",
					Name:         "検証",
					PrimaryTable: "orders",
					UpdatedAt:    "2026-08-08T03:00:00.123456789Z",
				},
			},
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
			wantLog:    true,
		},
		{
			name: "プロファイル読込失敗を安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				err: errors.New("config path=/private/secret/config.json failed"),
			},
			wantCode: apperr.CodeUnexpected,
			wantLog:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			if !tt.useCaseNil {
				scenarioUseCase = usecase.NewVerificationScenarioUseCase(tt.profiles, tt.repository)
			}
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil, scenarioUseCase)

			got := handler.ListVerificationScenarios()
			if tt.wantCode == "" {
				if got.Error != nil {
					t.Fatalf("ListVerificationScenarios() Error = %#v, want nil", got.Error)
				}
				if got.Data == nil {
					t.Fatal("ListVerificationScenarios() Data = nil, want non-nil")
				}
				if !reflect.DeepEqual(*got.Data, tt.wantData) {
					t.Errorf("ListVerificationScenarios() Data = %#v, want %#v", *got.Data, tt.wantData)
				}

				return
			}
			if got.Data != nil {
				t.Errorf("ListVerificationScenarios() Data = %#v, want nil", got.Data)
			}
			if got.Error == nil {
				t.Fatal("ListVerificationScenarios() Error = nil, want error response")
			}
			if got.Error.Code != string(tt.wantCode) {
				t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if got.Error.Message == "" || bytes.Contains([]byte(got.Error.Message), []byte("secret")) {
				t.Errorf("Error.Message = %q, want safe non-empty message", got.Error.Message)
			}
			if tt.wantLog && !bytes.Contains(output.Bytes(), []byte("code="+string(tt.wantCode))) {
				t.Errorf("log = %q, want code=%s", output.String(), tt.wantCode)
			}
			if bytes.Contains(output.Bytes(), []byte("secret")) || bytes.Contains(output.Bytes(), []byte("verification.sqlite3")) {
				t.Errorf("log = %q, must not contain sensitive details", output.String())
			}
		})
	}
}
