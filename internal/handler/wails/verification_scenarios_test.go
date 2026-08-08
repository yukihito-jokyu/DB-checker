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
	scenarios   []domain.VerificationScenarioSummary
	scenario    domain.VerificationScenario
	found       bool
	err         error
	createCalls *int
	updateCalls *int
	updateFound bool
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

// シナリオ作成再現
func (s verificationScenarioHandlerRepositoryStub) CreateVerificationScenario(_ context.Context, _ string, _ domain.VerificationScenario) error {
	if s.createCalls != nil {
		*s.createCalls++
	}

	return s.err
}

// シナリオ更新再現
func (s verificationScenarioHandlerRepositoryStub) UpdateVerificationScenario(_ context.Context, _ string, _ domain.VerificationScenario) (bool, error) {
	if s.updateCalls != nil {
		*s.updateCalls++
	}

	return s.updateFound, s.err
}

// シナリオ作成ハンドラー応答検証
func TestAppHandlerCreateVerificationScenario(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	tests := []struct {
		name            string
		useCaseNil      bool
		profiles        verificationScenarioHandlerProfilesStub
		repository      verificationScenarioHandlerRepositoryStub
		request         CreateVerificationScenarioRequest
		wantCode        apperr.Code
		wantCreateCalls int
	}{
		{
			name: "作成結果をDTOで返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			request:         validCreateVerificationScenarioRequest(),
			wantCreateCalls: 1,
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			useCaseNil: true,
			request:    validCreateVerificationScenarioRequest(),
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name: "主対象の一意生成規則不足を安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			request:  createRequestWithoutPrimaryKeyGenerator(),
			wantCode: apperr.CodePrimaryKeyRequired,
		},
		{
			name: "形式違反を安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			request: CreateVerificationScenarioRequest{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name:     "アクティブプロファイル不在を安全な失敗で返す",
			request:  validCreateVerificationScenarioRequest(),
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "保存失敗を安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			repository:      verificationScenarioHandlerRepositoryStub{err: errors.New("sqlite path=/private/secret")},
			request:         validCreateVerificationScenarioRequest(),
			wantCode:        apperr.CodeScenarioStoreFailed,
			wantCreateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			createCalls := 0
			if !tt.useCaseNil {
				repository := tt.repository
				repository.createCalls = &createCalls
				scenarioUseCase = usecase.NewVerificationScenarioUseCase(tt.profiles, repository)
			}
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil, scenarioUseCase)

			got := handler.CreateVerificationScenario(tt.request)
			if tt.wantCode == "" {
				if got.Error != nil {
					t.Fatalf("CreateVerificationScenario() Error = %#v, want nil", got.Error)
				}
				if got.Data == nil {
					t.Fatal("CreateVerificationScenario() Data = nil, want non-nil")
				}
				if got.Data.ID == "" || got.Data.WorkspaceName != nil || got.Data.LatestRun != nil {
					t.Errorf("CreateVerificationScenario() Data = %#v, want generated ID and nil workspace/run", got.Data)
				}
			} else {
				if got.Data != nil {
					t.Errorf("CreateVerificationScenario() Data = %#v, want nil", got.Data)
				}
				if got.Error == nil {
					t.Fatal("CreateVerificationScenario() Error = nil, want error response")
				}
				if got.Error.Code != string(tt.wantCode) {
					t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
				}
				if got.Error.Message == "" || bytes.Contains([]byte(got.Error.Message), []byte("secret")) {
					t.Errorf("Error.Message = %q, want safe non-empty message", got.Error.Message)
				}
				if !bytes.Contains(output.Bytes(), []byte("code="+string(tt.wantCode))) {
					t.Errorf("log = %q, want code=%s", output.String(), tt.wantCode)
				}
				if bytes.Contains(output.Bytes(), []byte("secret")) || bytes.Contains(output.Bytes(), []byte("verification.sqlite3")) {
					t.Errorf("log = %q, must not contain sensitive details", output.String())
				}
			}
			if createCalls != tt.wantCreateCalls {
				t.Errorf("CreateVerificationScenario() repository calls = %d, want %d", createCalls, tt.wantCreateCalls)
			}
		})
	}
}

// 有効な作成要求生成
func validCreateVerificationScenarioRequest() CreateVerificationScenarioRequest {
	return CreateVerificationScenarioRequest{
		Name:         "検証",
		PrimaryTable: "orders",
		Definition: map[string]any{
			"childTables": []any{"order_items"},
			"rowCounts": map[string]any{
				"orders":      10,
				"order_items": 20,
			},
			"columnGenerators": map[string]any{
				"orders":      map[string]any{"id": map[string]any{"kind": "sequence"}},
				"order_items": map[string]any{"id": map[string]any{"kind": "uuid"}},
			},
			"sql":              []any{"SELECT * FROM orders WHERE id = ?"},
			"warmupRuns":       0,
			"iterations":       1,
			"timeLimitSeconds": 1,
		},
	}
}

// 主対象生成規則なしの作成要求生成
func createRequestWithoutPrimaryKeyGenerator() CreateVerificationScenarioRequest {
	request := validCreateVerificationScenarioRequest()
	request.Definition["columnGenerators"] = map[string]any{
		"orders":      map[string]any{"id": map[string]any{"kind": "fixed"}},
		"order_items": map[string]any{"id": map[string]any{"kind": "uuid"}},
	}

	return request
}

// 有効な更新要求生成
func validUpdateVerificationScenarioRequest() UpdateVerificationScenarioRequest {
	request := validCreateVerificationScenarioRequest()

	return UpdateVerificationScenarioRequest{
		ScenarioID:   "scenario-1",
		Name:         request.Name,
		PrimaryTable: request.PrimaryTable,
		Definition:   request.Definition,
	}
}

// 主対象生成規則なしの更新要求生成
func updateRequestWithoutPrimaryKeyGenerator() UpdateVerificationScenarioRequest {
	request := validUpdateVerificationScenarioRequest()
	request.Definition["columnGenerators"] = map[string]any{
		"orders":      map[string]any{"id": map[string]any{"kind": "fixed"}},
		"order_items": map[string]any{"id": map[string]any{"kind": "uuid"}},
	}

	return request
}

// シナリオ更新ハンドラー応答検証
func TestAppHandlerUpdateVerificationScenario(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	workspaceName := "workspace-1"
	existing := domain.VerificationScenario{
		ID:            "scenario-1",
		Name:          "更新前",
		PrimaryTable:  "orders",
		Definition:    validCreateVerificationScenarioRequest().Definition,
		WorkspaceName: &workspaceName,
		CreatedAt:     time.Date(2026, time.August, 8, 11, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		UpdatedAt:     time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC),
	}
	tests := []struct {
		name            string
		useCaseNil      bool
		profiles        verificationScenarioHandlerProfilesStub
		repository      verificationScenarioHandlerRepositoryStub
		request         UpdateVerificationScenarioRequest
		wantCode        apperr.Code
		wantUpdateCalls int
	}{
		{
			name: "更新結果をDTOで返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			repository: verificationScenarioHandlerRepositoryStub{
				scenario:    existing,
				found:       true,
				updateFound: true,
			},
			request:         validUpdateVerificationScenarioRequest(),
			wantUpdateCalls: 1,
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			useCaseNil: true,
			request:    validUpdateVerificationScenarioRequest(),
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name: "形式違反を安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			request: UpdateVerificationScenarioRequest{
				ScenarioID:   "scenario-1",
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "主対象の一意生成規則不足を安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			request:  updateRequestWithoutPrimaryKeyGenerator(),
			wantCode: apperr.CodePrimaryKeyRequired,
		},
		{
			name: "他プロファイルのシナリオを安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			request:  validUpdateVerificationScenarioRequest(),
			wantCode: apperr.CodeScenarioNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			updateCalls := 0
			if !tt.useCaseNil {
				repository := tt.repository
				repository.updateCalls = &updateCalls
				scenarioUseCase = usecase.NewVerificationScenarioUseCase(tt.profiles, repository)
			}
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil, scenarioUseCase)

			got := handler.UpdateVerificationScenario(tt.request)
			if tt.wantCode == "" {
				if got.Error != nil {
					t.Fatalf("UpdateVerificationScenario() Error = %#v, want nil", got.Error)
				}
				if got.Data == nil {
					t.Fatal("UpdateVerificationScenario() Data = nil, want non-nil")
				}
				if got.Data.ID != existing.ID {
					t.Errorf("Data.ID = %q, want %q", got.Data.ID, existing.ID)
				}
				if got.Data.Name != tt.request.Name {
					t.Errorf("Data.Name = %q, want %q", got.Data.Name, tt.request.Name)
				}
				if got.Data.PrimaryTable != tt.request.PrimaryTable {
					t.Errorf("Data.PrimaryTable = %q, want %q", got.Data.PrimaryTable, tt.request.PrimaryTable)
				}
				expectedDraft, err := domain.NewVerificationScenarioDraft(tt.request.Name, tt.request.PrimaryTable, tt.request.Definition)
				if err != nil {
					t.Fatalf("NewVerificationScenarioDraft() error = %v", err)
				}
				if !reflect.DeepEqual(got.Data.Definition, expectedDraft.Definition) {
					t.Errorf("Data.Definition = %#v, want %#v", got.Data.Definition, expectedDraft.Definition)
				}
				if !reflect.DeepEqual(got.Data.WorkspaceName, existing.WorkspaceName) {
					t.Errorf("Data.WorkspaceName = %#v, want %#v", got.Data.WorkspaceName, existing.WorkspaceName)
				}
				if got.Data.CreatedAt != existing.CreatedAt.UTC().Format(time.RFC3339Nano) {
					t.Errorf("Data.CreatedAt = %q, want %q", got.Data.CreatedAt, existing.CreatedAt.UTC().Format(time.RFC3339Nano))
				}
				updatedAt, err := time.Parse(time.RFC3339Nano, got.Data.UpdatedAt)
				if err != nil {
					t.Errorf("Data.UpdatedAt = %q, want RFC3339Nano time: %v", got.Data.UpdatedAt, err)
				} else if updatedAt.Location() != time.UTC || got.Data.UpdatedAt != updatedAt.UTC().Format(time.RFC3339Nano) {
					t.Errorf("Data.UpdatedAt = %q, want UTC RFC3339Nano time", got.Data.UpdatedAt)
				}
				if got.Data.LatestRun != nil {
					t.Errorf("Data.LatestRun = %#v, want nil", got.Data.LatestRun)
				}
			} else {
				if got.Data != nil {
					t.Errorf("UpdateVerificationScenario() Data = %#v, want nil", got.Data)
				}
				if got.Error == nil {
					t.Fatal("UpdateVerificationScenario() Error = nil, want error response")
				}
				if got.Error.Code != string(tt.wantCode) {
					t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
				}
				if got.Error.Message == "" || bytes.Contains([]byte(got.Error.Message), []byte("secret")) {
					t.Errorf("Error.Message = %q, want safe non-empty message", got.Error.Message)
				}
				if !bytes.Contains(output.Bytes(), []byte("code="+string(tt.wantCode))) {
					t.Errorf("log = %q, want code=%s", output.String(), tt.wantCode)
				}
			}
			if bytes.Contains(output.Bytes(), []byte("secret")) || bytes.Contains(output.Bytes(), []byte("verification.sqlite3")) {
				t.Errorf("log = %q, must not contain sensitive details", output.String())
			}
			if updateCalls != tt.wantUpdateCalls {
				t.Errorf("UpdateVerificationScenario() repository calls = %d, want %d", updateCalls, tt.wantUpdateCalls)
			}
		})
	}
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
