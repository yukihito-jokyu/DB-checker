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
	profiles        []domain.Profile
	activeID        *string
	err             error
	credential      string
	credentialFound bool
	credentialErr   error
	schema          domain.Schema
	schemaErr       error
}

type verificationScenarioHandlerRepositoryStub struct {
	scenarios              []domain.VerificationScenarioSummary
	scenario               domain.VerificationScenario
	found                  bool
	err                    error
	createErr              error
	createCalls            *int
	updateCalls            *int
	updateFound            bool
	deleteFound            bool
	deleteWorkspaceRemoved bool
	deleteBusy             bool
	deleteCalls            *int
	workspaceState         string
	workspaceName          string
	workspaceFound         bool
	workspaceErr           error
	saveWorkspaceErr       error
	runScenarioID          string
	runState               string
	runFound               bool
	runErr                 error
	createRunErr           error
	updateRunFound         bool
	updateRunErr           error
	runBusy                bool
	runBusyErr             error
}

type verificationScenarioHandlerWorkspaceStub struct {
	createErr error
	deleteErr error
}

// 検証実行プレビュー要求検証
func TestAppHandlerPreviewVerificationRunRejectsAmbiguousInput(t *testing.T) {
	handler := NewAppHandler(
		applogger.NewWithWriter(&bytes.Buffer{}, slog.LevelDebug),
		config.NewStore(t.TempDir()),
		nil,
		usecase.NewVerificationScenarioUseCaseWithPreview(verificationScenarioHandlerProfilesStub{}, verificationScenarioHandlerRepositoryStub{}, verificationScenarioHandlerProfilesStub{}),
	)

	tests := []struct {
		name    string
		request PreviewVerificationRunRequest
	}{
		{
			name:    "保存済みと下書きが未指定",
			request: PreviewVerificationRunRequest{},
		},
		{
			name: "保存済みと下書きを同時指定",
			request: PreviewVerificationRunRequest{
				ScenarioID: "scenario-1",
				Draft:      &VerificationScenarioDraftRequest{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.PreviewVerificationRun(tt.request)
			if got.Data != nil {
				t.Errorf("PreviewVerificationRun() Data = %#v, want nil", got.Data)
			}
			if got.Error == nil || got.Error.Code != string(apperr.CodeValidationFailed) {
				t.Errorf("PreviewVerificationRun() Error = %#v, want VALIDATION_FAILED", got.Error)
			}
		})
	}
}

// 検証実行プレビュー応答分岐検証
func TestAppHandlerPreviewVerificationRunBranches(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	validRequest := validCreateVerificationScenarioRequest()
	validDraft := VerificationScenarioDraftRequest(validRequest)
	validSchema := domain.Schema{Tables: []domain.Table{
		{
			Namespace: "app",
			Name:      "orders",
			Columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "bigint",
					IsPrimaryKey: true,
				},
			},
		},
		{
			Namespace: "app",
			Name:      "order_items",
			Columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "bigint",
					IsPrimaryKey: true,
				},
			},
		},
	}}

	tests := []struct {
		name       string
		useCaseNil bool
		profiles   verificationScenarioHandlerProfilesStub
		request    PreviewVerificationRunRequest
		wantCode   apperr.Code
		wantReady  bool
	}{
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			useCaseNil: true,
			request:    PreviewVerificationRunRequest{Draft: &validDraft},
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name: "主キー生成規則不足を安全な失敗で返す",
			request: PreviewVerificationRunRequest{Draft: &VerificationScenarioDraftRequest{
				Name:         createRequestWithoutPrimaryKeyGenerator().Name,
				PrimaryTable: createRequestWithoutPrimaryKeyGenerator().PrimaryTable,
				Definition:   createRequestWithoutPrimaryKeyGenerator().Definition,
			}},
			wantCode: apperr.CodePrimaryKeyRequired,
		},
		{
			name: "形式違反を安全な失敗で返す",
			request: PreviewVerificationRunRequest{Draft: &VerificationScenarioDraftRequest{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			}},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "ユースケース失敗を安全な失敗で返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{profile},
			},
			request:  PreviewVerificationRunRequest{Draft: &validDraft},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "下書きプレビューをDTOで返す",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema:          validSchema,
			},
			request:   PreviewVerificationRunRequest{Draft: &validDraft},
			wantReady: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			if !tt.useCaseNil {
				scenarioUseCase = usecase.NewVerificationScenarioUseCaseWithPreview(tt.profiles, verificationScenarioHandlerRepositoryStub{}, tt.profiles)
			}
			handler := NewAppHandler(
				applogger.NewWithWriter(&bytes.Buffer{}, slog.LevelDebug),
				config.NewStore(t.TempDir()),
				nil,
				scenarioUseCase,
			)

			got := handler.PreviewVerificationRun(tt.request)
			if tt.wantCode != "" {
				if got.Error == nil {
					t.Fatal("Error = nil, want error response")
				}
				if got.Error.Code != string(tt.wantCode) {
					t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
				}

				return
			}
			if got.Error != nil {
				t.Fatalf("Error = %#v, want nil", got.Error)
			}
			if got.Data == nil {
				t.Fatal("Data = nil, want preview response")
			}
			if got.Data.Ready != tt.wantReady {
				t.Errorf("Data.Ready = %v, want %v", got.Data.Ready, tt.wantReady)
			}
		})
	}
}

// 検証実行プレビュー成功DTO検証
func TestAppHandlerPreviewVerificationRunResponse(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	request := validCreateVerificationScenarioRequest()
	draft := VerificationScenarioDraftRequest(request)
	schema := domain.Schema{
		Tables: []domain.Table{
			{
				Namespace: "app",
				Name:      "customers",
				Columns: []domain.Column{{
					Name:         "id",
					DataType:     "bigint",
					IsPrimaryKey: true,
				}},
			},
			{
				Namespace: "app",
				Name:      "orders",
				Columns: []domain.Column{{
					Name:         "id",
					DataType:     "bigint",
					IsPrimaryKey: true,
				}},
			},
			{
				Namespace: "app",
				Name:      "order_items",
				Columns: []domain.Column{{
					Name:         "id",
					DataType:     "bigint",
					IsPrimaryKey: true,
				}},
			},
		},
		ForeignKeys: []domain.ForeignKey{
			{
				Name:        "orders_customer",
				FromTable:   "orders",
				FromColumns: []string{"customer_id"},
				ToTable:     "customers",
				ToColumns:   []string{"id"},
			},
			{
				Name:        "order_items_order",
				FromTable:   "order_items",
				FromColumns: []string{"order_id"},
				ToTable:     "orders",
				ToColumns:   []string{"id"},
			},
		},
	}
	tests := []struct {
		name string
		want VerificationRunPreviewResponse
	}{
		{
			name: "全フィールドをプレビューDTOへ変換する",
			want: VerificationRunPreviewResponse{
				Ready: true,
				InsertOrder: []VerificationRunPreviewTableResponse{
					{
						Name:               "customers",
						RowCount:           10,
						AutomaticallyAdded: true,
						GeneratedColumns:   []string{},
					},
					{
						Name:               "orders",
						RowCount:           10,
						AutomaticallyAdded: false,
						GeneratedColumns:   []string{"id"},
					},
					{
						Name:               "order_items",
						RowCount:           20,
						AutomaticallyAdded: false,
						GeneratedColumns:   []string{"id"},
					},
				},
				DeleteOrder: []VerificationRunPreviewTableResponse{
					{
						Name:               "order_items",
						RowCount:           20,
						AutomaticallyAdded: false,
						GeneratedColumns:   []string{"id"},
					},
					{
						Name:               "orders",
						RowCount:           10,
						AutomaticallyAdded: false,
						GeneratedColumns:   []string{"id"},
					},
					{
						Name:               "customers",
						RowCount:           10,
						AutomaticallyAdded: true,
						GeneratedColumns:   []string{},
					},
				},
				Warnings: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiles := verificationScenarioHandlerProfilesStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema:          schema,
			}
			handler := NewAppHandler(applogger.NewWithWriter(&bytes.Buffer{}, slog.LevelDebug), config.NewStore(t.TempDir()), nil, usecase.NewVerificationScenarioUseCaseWithPreview(profiles, verificationScenarioHandlerRepositoryStub{}, profiles))

			got := handler.PreviewVerificationRun(PreviewVerificationRunRequest{Draft: &draft})
			if got.Error != nil {
				t.Fatalf("PreviewVerificationRun() Error = %#v, want nil", got.Error)
			}
			if got.Data == nil {
				t.Fatal("PreviewVerificationRun() Data = nil, want preview")
			}
			if !reflect.DeepEqual(*got.Data, tt.want) {
				t.Errorf("PreviewVerificationRun() Data = %#v, want %#v", *got.Data, tt.want)
			}
		})
	}
}

// 機密情報を含むプレビュー障害の非公開検証
func TestAppHandlerPreviewVerificationRunDoesNotExposeSensitiveErrors(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	request := validCreateVerificationScenarioRequest()
	draft := VerificationScenarioDraftRequest(request)
	tests := []struct {
		name     string
		profiles verificationScenarioHandlerProfilesStub
		wantCode apperr.Code
	}{
		{
			name: "資格情報取得エラーを公開しない",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles:      []domain.Profile{profile},
				activeID:      stringPointer(profile.ID),
				credentialErr: errors.New("password=super-secret"),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "DB接続エラーを公開しない",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "super-secret",
				credentialFound: true,
				schemaErr:       errors.New("mysql://user:super-secret@host/app"),
			},
			wantCode: apperr.CodeSchemaLoadFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil, usecase.NewVerificationScenarioUseCaseWithPreview(tt.profiles, verificationScenarioHandlerRepositoryStub{}, tt.profiles))

			got := handler.PreviewVerificationRun(PreviewVerificationRunRequest{Draft: &draft})
			if got.Data != nil {
				t.Errorf("PreviewVerificationRun() Data = %#v, want nil", got.Data)
			}
			if got.Error == nil {
				t.Fatal("PreviewVerificationRun() Error = nil, want error response")
			}
			if got.Error.Code != string(tt.wantCode) {
				t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if bytes.Contains([]byte(got.Error.Message), []byte("super-secret")) {
				t.Errorf("Error.Message = %q, must not contain sensitive detail", got.Error.Message)
			}
			if bytes.Contains(output.Bytes(), []byte("super-secret")) {
				t.Errorf("log = %q, must not contain sensitive detail", output.String())
			}
		})
	}
}

// プロファイル読込再現
func (s verificationScenarioHandlerProfilesStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return s.profiles, s.activeID, s.err
}

// 資格情報取得再現
func (s verificationScenarioHandlerProfilesStub) GetCredential(string) (string, bool, error) {
	return s.credential, s.credentialFound, s.credentialErr
}

// スキーマ取得再現
func (s verificationScenarioHandlerProfilesStub) InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error) {
	return s.schema, s.schemaErr
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

	if s.createErr != nil {
		return s.createErr
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

// シナリオ削除再現
func (s verificationScenarioHandlerRepositoryStub) DeleteVerificationScenario(_ context.Context, _, _ string, _ bool) (bool, bool, bool, error) {
	if s.deleteCalls != nil {
		*s.deleteCalls++
	}

	return s.deleteFound, s.deleteWorkspaceRemoved, s.deleteBusy, s.err
}

// ワークスペース状態取得再現
func (s verificationScenarioHandlerRepositoryStub) GetVerificationWorkspace(context.Context, string, string) (string, string, bool, error) {
	return s.workspaceState, s.workspaceName, s.workspaceFound, s.workspaceErr
}

// ワークスペース状態保存再現
func (s verificationScenarioHandlerRepositoryStub) SaveVerificationWorkspace(context.Context, string, string, string, string) error {
	return s.saveWorkspaceErr
}

// ワークスペース状態削除再現
func (verificationScenarioHandlerRepositoryStub) DeleteVerificationWorkspace(context.Context, string, string) error {
	return nil
}

// 実行状態作成再現
func (s verificationScenarioHandlerRepositoryStub) CreateVerificationRun(context.Context, string, string, string) error {
	return s.createRunErr
}

// 実行状態取得再現
func (s verificationScenarioHandlerRepositoryStub) GetVerificationRun(context.Context, string, string) (string, string, bool, error) {
	return s.runScenarioID, s.runState, s.runFound, s.runErr
}

// 実行状態更新再現
func (s verificationScenarioHandlerRepositoryStub) UpdateVerificationRunState(context.Context, string, string, string) (bool, error) {
	return s.updateRunFound, s.updateRunErr
}

// シナリオ使用中判定再現
func (verificationScenarioHandlerRepositoryStub) IsVerificationScenarioBusy(context.Context, string, string) (bool, error) {
	return false, nil
}

// 実行使用中判定再現
func (s verificationScenarioHandlerRepositoryStub) IsVerificationRunBusy(context.Context, string, string) (bool, error) {
	return s.runBusy, s.runBusyErr
}

// 外部ワークスペース作成再現
func (s verificationScenarioHandlerWorkspaceStub) CreateWorkspace(context.Context, domain.Profile, string) error {
	return s.createErr
}

// 外部ワークスペース削除再現
func (s verificationScenarioHandlerWorkspaceStub) DeleteWorkspace(context.Context, domain.Profile, string) error {
	return s.deleteErr
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

// シナリオ複製ハンドラー応答検証
func TestAppHandlerDuplicateVerificationScenario(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	workspaceName := "verification_orders"
	scenario := domain.VerificationScenario{
		ID:            "scenario-1",
		Name:          "検証",
		PrimaryTable:  "orders",
		Definition:    validCreateVerificationScenarioRequest().Definition,
		WorkspaceName: &workspaceName,
		CreatedAt:     time.Date(2026, time.August, 8, 11, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC),
	}
	tests := []struct {
		name            string
		useCaseNil      bool
		profiles        verificationScenarioHandlerProfilesStub
		repository      verificationScenarioHandlerRepositoryStub
		wantCode        apperr.Code
		wantCreateCalls int
		sensitiveTexts  []string
	}{
		{
			name: "複製結果をDTOで返す",
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
			wantCreateCalls: 1,
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
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
		},
		{
			name: "保存失敗時に機密情報を応答とログへ出さない",
			profiles: verificationScenarioHandlerProfilesStub{
				profiles: []domain.Profile{
					profile,
				},
				activeID: stringPointer(profile.ID),
			},
			repository: verificationScenarioHandlerRepositoryStub{
				scenario: scenario,
				found:    true,
				createErr: errors.New(
					"postgres://db-user:db-password@db.example.test:5432/verification",
				),
			},
			wantCode:        apperr.CodeScenarioStoreFailed,
			wantCreateCalls: 1,
			sensitiveTexts: []string{
				"db-user",
				"db-password",
				"db.example.test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			createCalls := 0
			tt.repository.createCalls = &createCalls
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			if !tt.useCaseNil {
				scenarioUseCase = usecase.NewVerificationScenarioUseCase(tt.profiles, tt.repository)
			}
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil, scenarioUseCase)

			got := handler.DuplicateVerificationScenario("scenario-1")
			if tt.wantCode == "" {
				if got.Error != nil {
					t.Fatalf("DuplicateVerificationScenario() Error = %#v, want nil", got.Error)
				}
				if got.Data == nil {
					t.Fatal("DuplicateVerificationScenario() Data = nil, want non-nil")
				}
				if got.Data.ID == scenario.ID || got.Data.ID == "" || got.Data.Name != scenario.Name || got.Data.WorkspaceName != nil {
					t.Errorf("DuplicateVerificationScenario() Data = %#v, want new ID, copied definition, and nil workspace", got.Data)
				}
			} else {
				if got.Data != nil {
					t.Errorf("DuplicateVerificationScenario() Data = %#v, want nil", got.Data)
				}
				if got.Error == nil {
					t.Fatal("DuplicateVerificationScenario() Error = nil, want error response")
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
				for _, sensitiveText := range tt.sensitiveTexts {
					if bytes.Contains([]byte(got.Error.Message), []byte(sensitiveText)) {
						t.Errorf("Error.Message = %q, must not contain %q", got.Error.Message, sensitiveText)
					}
					if bytes.Contains(output.Bytes(), []byte(sensitiveText)) {
						t.Errorf("log = %q, must not contain %q", output.String(), sensitiveText)
					}
				}
			}
			if createCalls != tt.wantCreateCalls {
				t.Errorf("CreateVerificationScenario() calls = %d, want %d", createCalls, tt.wantCreateCalls)
			}
		})
	}
}

// シナリオ削除ハンドラー応答検証
func TestAppHandlerDeleteVerificationScenario(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	tests := []struct {
		name       string
		useCaseNil bool
		repository verificationScenarioHandlerRepositoryStub
		wantData   *DeleteScenarioResponse
		wantCode   apperr.Code
	}{
		{
			name: "workspace削除結果を返す",
			repository: verificationScenarioHandlerRepositoryStub{
				deleteFound:            true,
				deleteWorkspaceRemoved: true,
			},
			wantData: &DeleteScenarioResponse{
				ScenarioID:       "scenario-1",
				WorkspaceRemoved: true,
			},
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name: "使用中シナリオを安全な失敗で返す",
			repository: verificationScenarioHandlerRepositoryStub{
				deleteBusy: true,
			},
			wantCode: apperr.CodeScenarioBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			if !tt.useCaseNil {
				scenarioUseCase = usecase.NewVerificationScenarioUseCase(verificationScenarioHandlerProfilesStub{
					profiles: []domain.Profile{profile},
					activeID: stringPointer(profile.ID),
				}, tt.repository)
			}
			handler := NewAppHandler(applogger.NewWithWriter(&bytes.Buffer{}, slog.LevelDebug), config.NewStore(t.TempDir()), nil, scenarioUseCase)

			got := handler.DeleteVerificationScenario(DeleteVerificationScenarioRequest{
				ScenarioID:      "scenario-1",
				RemoveWorkspace: true,
			})
			if tt.wantCode != "" {
				if got.Data != nil {
					t.Errorf("DeleteVerificationScenario() Data = %#v, want nil", got.Data)
				}
				if got.Error == nil {
					t.Fatal("DeleteVerificationScenario() Error = nil, want error response")
				}
				if got.Error.Code != string(tt.wantCode) {
					t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
				}

				return
			}
			if got.Error != nil {
				t.Fatalf("DeleteVerificationScenario() Error = %#v, want nil", got.Error)
			}
			if got.Data == nil {
				t.Fatal("DeleteVerificationScenario() Data = nil, want non-nil")
			}
			if !reflect.DeepEqual(*got.Data, *tt.wantData) {
				t.Errorf("DeleteVerificationScenario() Data = %#v, want %#v", *got.Data, *tt.wantData)
			}
		})
	}
}

// ワークスペースと実行状態ハンドラー応答検証
func TestAppHandlerVerificationWorkspaceAndRunOperations(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypeMySQL)
	tests := []struct {
		name       string
		operation  string
		useCaseNil bool
		repository verificationScenarioHandlerRepositoryStub
		wantCode   apperr.Code
	}{
		{
			name:      "ワークスペース開始成功を返す",
			operation: "enter",
			repository: verificationScenarioHandlerRepositoryStub{
				found: true,
			},
		},
		{
			name:       "ワークスペース開始の未注入を安全な失敗で返す",
			operation:  "enter",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name:      "ワークスペース開始のユースケース失敗を安全に返す",
			operation: "enter",
			wantCode:  apperr.CodeScenarioNotFound,
		},
		{
			name:      "ワークスペース終了成功を返す",
			operation: "exit",
			repository: verificationScenarioHandlerRepositoryStub{
				found:          true,
				workspaceFound: true,
				workspaceState: "test",
				workspaceName:  "db_checker_v_profile_scenario",
			},
		},
		{
			name:       "ワークスペース終了の未注入を安全な失敗で返す",
			operation:  "exit",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name:      "ワークスペース終了のユースケース失敗を安全に返す",
			operation: "exit",
			wantCode:  apperr.CodeScenarioNotFound,
		},
		{
			name:      "実行準備成功を返す",
			operation: "prepare",
			repository: verificationScenarioHandlerRepositoryStub{
				found:          true,
				workspaceFound: true,
				workspaceState: "test",
			},
		},
		{
			name:       "実行準備の未注入を安全な失敗で返す",
			operation:  "prepare",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name:      "実行準備のユースケース失敗を安全に返す",
			operation: "prepare",
			wantCode:  apperr.CodeScenarioNotFound,
		},
		{
			name:      "実行状態更新成功を返す",
			operation: "update",
			repository: verificationScenarioHandlerRepositoryStub{
				runScenarioID:  "scenario-1",
				runState:       "prepared",
				runFound:       true,
				updateRunFound: true,
			},
		},
		{
			name:       "実行状態更新の未注入を安全な失敗で返す",
			operation:  "update",
			useCaseNil: true,
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name:      "実行状態更新のユースケース失敗を安全に返す",
			operation: "update",
			wantCode:  apperr.CodeScenarioNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scenarioUseCase *usecase.VerificationScenarioUseCase
			if !tt.useCaseNil {
				scenarioUseCase = usecase.NewVerificationScenarioUseCase(
					verificationScenarioHandlerProfilesStub{
						profiles: []domain.Profile{profile},
						activeID: stringPointer(profile.ID),
					},
					tt.repository,
					verificationScenarioHandlerWorkspaceStub{},
				)
			}
			handler := NewAppHandler(applogger.NewWithWriter(&bytes.Buffer{}, slog.LevelDebug), config.NewStore(t.TempDir()), nil, scenarioUseCase)

			var errorCode string
			var success bool
			switch tt.operation {
			case "enter":
				got := handler.EnterVerificationWorkspace("scenario-1")
				success = got.Data != nil && got.Error == nil && got.Data.ScenarioID == "scenario-1" && got.Data.WorkspaceName != "" && got.Data.Mode == "test"
				if got.Error != nil {
					errorCode = got.Error.Code
				}
			case "exit":
				got := handler.ExitVerificationWorkspace("scenario-1")
				success = got.Data != nil && got.Error == nil
				if got.Error != nil {
					errorCode = got.Error.Code
				}
			case "prepare":
				got := handler.PrepareVerificationRun(PrepareVerificationRunRequest{
					ScenarioID: "scenario-1",
					RunID:      "run-1",
				})
				success = got.Data != nil && got.Error == nil
				if got.Error != nil {
					errorCode = got.Error.Code
				}
			case "update":
				got := handler.UpdateVerificationRunState(UpdateVerificationRunStateRequest{
					RunID: "run-1",
					State: "running",
				})
				success = got.Data != nil && got.Error == nil
				if got.Error != nil {
					errorCode = got.Error.Code
				}
			}
			if tt.wantCode != "" {
				if success {
					t.Errorf("%s handler succeeded, want error response", tt.operation)
				}
				if errorCode != string(tt.wantCode) {
					t.Errorf("Error.Code = %q, want %q", errorCode, tt.wantCode)
				}

				return
			}
			if !success {
				t.Errorf("%s handler success = %v, want true; error code = %q", tt.operation, success, errorCode)
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
