package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type verificationScenarioProfilesStub struct {
	profiles        []domain.Profile
	activeID        *string
	err             error
	credential      string
	credentialFound bool
	credentialErr   error
	schema          domain.Schema
	schemaErr       error
}

type verificationScenarioRepositoryStub struct {
	scenarios              []domain.VerificationScenarioSummary
	scenario               domain.VerificationScenario
	found                  bool
	err                    error
	getErr                 error
	createErr              error
	profileID              string
	created                domain.VerificationScenario
	createCalls            int
	updated                domain.VerificationScenario
	updateCalls            int
	updateFound            bool
	updateErr              error
	deleteFound            bool
	deleteWorkspaceRemoved bool
	deleteBusy             bool
	deleteErr              error
	deleteCalls            int
	workspaceState         string
	workspaceName          string
	workspaceFound         bool
	workspaceErr           error
	savedWorkspaceStates   []string
	saveWorkspaceErrors    []error
	deleteWorkspaceCalls   int
	deleteWorkspaceErr     error
	runBusy                bool
	runBusyErr             error
	createRunErr           error
	createRunCalls         int
	runScenarioID          string
	runState               string
	runFound               bool
	runErr                 error
	updateRunFound         bool
	updateRunErr           error
	updateRunCalls         int
}

type verificationWorkspaceStub struct {
	createCalls int
	createErr   error
	deleteCalls int
	deleteErr   error
}

// プロファイル読込再現
func (s verificationScenarioProfilesStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return s.profiles, s.activeID, s.err
}

// 資格情報取得再現
func (s verificationScenarioProfilesStub) GetCredential(string) (string, bool, error) {
	return s.credential, s.credentialFound, s.credentialErr
}

// スキーマ取得再現
func (s verificationScenarioProfilesStub) InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error) {
	return s.schema, s.schemaErr
}

// シナリオ一覧取得再現
func (s *verificationScenarioRepositoryStub) ListVerificationScenarios(_ context.Context, profileID string) ([]domain.VerificationScenarioSummary, error) {
	s.profileID = profileID

	return s.scenarios, s.err
}

// シナリオ詳細取得再現
func (s *verificationScenarioRepositoryStub) GetVerificationScenario(_ context.Context, profileID, _ string) (domain.VerificationScenario, bool, error) {
	s.profileID = profileID

	if s.getErr != nil {
		return domain.VerificationScenario{}, false, s.getErr
	}

	return s.scenario, s.found, s.err
}

// シナリオ作成再現
func (s *verificationScenarioRepositoryStub) CreateVerificationScenario(_ context.Context, profileID string, scenario domain.VerificationScenario) error {
	s.profileID = profileID
	s.created = scenario
	s.createCalls++

	if s.createErr != nil {
		return s.createErr
	}

	return s.err
}

// シナリオ更新再現
func (s *verificationScenarioRepositoryStub) UpdateVerificationScenario(_ context.Context, profileID string, scenario domain.VerificationScenario) (bool, error) {
	s.profileID = profileID
	s.updated = scenario
	s.updateCalls++

	return s.updateFound, s.updateErr
}

// シナリオ削除再現
func (s *verificationScenarioRepositoryStub) DeleteVerificationScenario(_ context.Context, profileID, _ string, _ bool) (bool, bool, bool, error) {
	s.profileID = profileID
	s.deleteCalls++

	return s.deleteFound, s.deleteWorkspaceRemoved, s.deleteBusy, s.deleteErr
}

// ワークスペース状態取得再現
func (s *verificationScenarioRepositoryStub) GetVerificationWorkspace(_ context.Context, profileID, _ string) (string, string, bool, error) {
	s.profileID = profileID

	return s.workspaceState, s.workspaceName, s.workspaceFound, s.workspaceErr
}

// ワークスペース状態保存再現
func (s *verificationScenarioRepositoryStub) SaveVerificationWorkspace(_ context.Context, profileID, _ string, _ string, state string) error {
	s.profileID = profileID
	s.savedWorkspaceStates = append(s.savedWorkspaceStates, state)
	if len(s.saveWorkspaceErrors) == 0 {
		return nil
	}
	err := s.saveWorkspaceErrors[0]
	s.saveWorkspaceErrors = s.saveWorkspaceErrors[1:]

	return err
}

// ワークスペース状態削除再現
func (s *verificationScenarioRepositoryStub) DeleteVerificationWorkspace(context.Context, string, string) error {
	s.deleteWorkspaceCalls++

	return s.deleteWorkspaceErr
}

// 実行状態作成再現
func (s *verificationScenarioRepositoryStub) CreateVerificationRun(_ context.Context, _ string, scenarioID, _ string) error {
	s.createRunCalls++
	s.runScenarioID = scenarioID

	return s.createRunErr
}

// 実行状態取得再現
func (s *verificationScenarioRepositoryStub) GetVerificationRun(context.Context, string, string) (string, string, bool, error) {
	return s.runScenarioID, s.runState, s.runFound, s.runErr
}

// 実行状態更新再現
func (s *verificationScenarioRepositoryStub) UpdateVerificationRunState(context.Context, string, string, string) (bool, error) {
	s.updateRunCalls++

	return s.updateRunFound, s.updateRunErr
}

// シナリオ使用中判定再現
func (*verificationScenarioRepositoryStub) IsVerificationScenarioBusy(context.Context, string, string) (bool, error) {
	return false, nil
}

// 実行使用中判定再現
func (s *verificationScenarioRepositoryStub) IsVerificationRunBusy(context.Context, string, string) (bool, error) {
	return s.runBusy, s.runBusyErr
}

// 外部ワークスペース作成再現
func (s *verificationWorkspaceStub) CreateWorkspace(context.Context, domain.Profile, string) error {
	s.createCalls++

	return s.createErr
}

// 外部ワークスペース削除再現
func (s *verificationWorkspaceStub) DeleteWorkspace(context.Context, domain.Profile, string) error {
	s.deleteCalls++

	return s.deleteErr
}

// 検証実行プレビューユースケース検証
func TestVerificationScenarioUseCasePreviewVerificationRun(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	draft := newVerificationScenarioDraft(t)
	profiles := verificationScenarioProfilesStub{
		profiles:        []domain.Profile{profile},
		activeID:        stringPointer("profile-1"),
		credential:      "secret",
		credentialFound: true,
		schema: domain.Schema{
			Tables: []domain.Table{
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
			},
			ForeignKeys: []domain.ForeignKey{
				{
					Name:        "order_items_order",
					FromTable:   "order_items",
					FromColumns: []string{"order_id"},
					ToTable:     "orders",
					ToColumns:   []string{"id"},
				},
			},
		},
	}
	repository := &verificationScenarioRepositoryStub{}
	useCase := NewVerificationScenarioUseCaseWithPreview(profiles, repository, profiles)

	got, err := useCase.PreviewVerificationRun(context.Background(), "", &draft)
	if err != nil {
		t.Fatalf("PreviewVerificationRun() error = %v", err)
	}
	if !got.Ready || len(got.InsertOrder) != 2 || got.InsertOrder[0].Name != "orders" || got.InsertOrder[1].Name != "order_items" {
		t.Errorf("PreviewVerificationRun() = %#v, want ready parent-to-child preview", got)
	}
	if repository.createCalls != 0 || repository.updateCalls != 0 || repository.deleteCalls != 0 {
		t.Errorf("repository mutations = create:%d update:%d delete:%d, want all zero", repository.createCalls, repository.updateCalls, repository.deleteCalls)
	}

	_, err = useCase.PreviewVerificationRun(context.Background(), "scenario-1", &draft)
	if !apperr.IsCode(err, apperr.CodeValidationFailed) {
		t.Errorf("PreviewVerificationRun() error code = %q, want %q", apperr.As(err).Code, apperr.CodeValidationFailed)
	}
}

// 検証実行プレビューユースケース分岐検証
func TestVerificationScenarioUseCasePreviewVerificationRunBranches(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	draft := newVerificationScenarioDraft(t)
	scenario, err := draft.NewVerificationScenario("scenario-1", time.Now())
	if err != nil {
		t.Fatalf("NewVerificationScenario() error = %v", err)
	}
	schema := domain.Schema{Tables: []domain.Table{
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
		name          string
		scenarioID    string
		draft         *domain.VerificationScenarioDraft
		profiles      verificationScenarioProfilesStub
		repository    verificationScenarioRepositoryStub
		wantCode      apperr.Code
		wantSourceErr error
		wantReady     bool
	}{
		{
			name:       "保存済みと下書きの同時指定を拒否する",
			scenarioID: "scenario-1",
			draft:      &draft,
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:          "プロファイル読込失敗を返す",
			draft:         &draft,
			profiles:      verificationScenarioProfilesStub{err: errors.New("profiles failed")},
			wantSourceErr: errors.New("profiles failed"),
		},
		{
			name:     "アクティブプロファイルなしを拒否する",
			draft:    &draft,
			profiles: verificationScenarioProfilesStub{profiles: []domain.Profile{profile}},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name:  "下書き形式違反を拒否する",
			draft: &domain.VerificationScenarioDraft{},
			profiles: verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name:       "保存済み読込失敗を変換する",
			scenarioID: "scenario-1",
			profiles: verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			repository: verificationScenarioRepositoryStub{getErr: errors.New("sqlite failed")},
			wantCode:   apperr.CodeScenarioStoreFailed,
		},
		{
			name:       "保存済み未発見を返す",
			scenarioID: "scenario-1",
			profiles: verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			repository: verificationScenarioRepositoryStub{},
			wantCode:   apperr.CodeScenarioNotFound,
		},
		{
			name:       "保存済み形式違反を変換する",
			scenarioID: "scenario-1",
			profiles: verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			repository: verificationScenarioRepositoryStub{
				found: true,
				scenario: domain.VerificationScenario{
					Name:         "検証",
					PrimaryTable: "orders",
					Definition:   map[string]any{},
				},
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name:  "資格情報取得失敗を変換する",
			draft: &draft,
			profiles: verificationScenarioProfilesStub{
				profiles:      []domain.Profile{profile},
				activeID:      stringPointer(profile.ID),
				credentialErr: errors.New("credential failed"),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name:  "資格情報未発見を返す",
			draft: &draft,
			profiles: verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name:  "スキーマ取得失敗を変換する",
			draft: &draft,
			profiles: verificationScenarioProfilesStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schemaErr:       errors.New("schema failed"),
			},
			wantCode: apperr.CodeSchemaLoadFailed,
		},
		{
			name:  "不正なスキーマを変換する",
			draft: &draft,
			profiles: verificationScenarioProfilesStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema: domain.Schema{Tables: []domain.Table{
					{
						Namespace: "other",
						Name:      "orders",
					},
				}},
			},
			wantCode: apperr.CodeSchemaLoadFailed,
		},
		{
			name:  "下書きプレビューを返す",
			draft: &draft,
			profiles: verificationScenarioProfilesStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema:          schema,
			},
			wantReady: true,
		},
		{
			name:       "保存済みプレビューを返す",
			scenarioID: "scenario-1",
			profiles: verificationScenarioProfilesStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema:          schema,
			},
			repository: verificationScenarioRepositoryStub{
				found:    true,
				scenario: scenario,
			},
			wantReady: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			useCase := NewVerificationScenarioUseCaseWithPreview(tt.profiles, &repository, tt.profiles)

			got, err := useCase.PreviewVerificationRun(context.Background(), tt.scenarioID, tt.draft)
			if tt.wantSourceErr != nil {
				if err == nil || err.Error() != tt.wantSourceErr.Error() {
					t.Errorf("PreviewVerificationRun() error = %v, want %v", err, tt.wantSourceErr)
				}

				return
			}
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("PreviewVerificationRun() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}

				return
			}
			if err != nil {
				t.Fatalf("PreviewVerificationRun() error = %v", err)
			}
			if got.Ready != tt.wantReady {
				t.Errorf("Ready = %v, want %v", got.Ready, tt.wantReady)
			}
			if repository.createCalls != 0 || repository.updateCalls != 0 || repository.deleteCalls != 0 {
				t.Errorf("repository mutations = create:%d update:%d delete:%d, want all zero", repository.createCalls, repository.updateCalls, repository.deleteCalls)
			}
		})
	}
}

// プレビュー入力エラー変換検証
func TestPreviewValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode apperr.Code
	}{
		{
			name:     "主キー生成規則不足を専用コードへ変換する",
			err:      domain.ErrPrimaryKeyRequired,
			wantCode: apperr.CodePrimaryKeyRequired,
		},
		{
			name:     "その他の形式違反を検証失敗へ変換する",
			err:      domain.ErrInvalidVerificationScenarioDraft,
			wantCode: apperr.CodeValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := previewValidationError(tt.err); !apperr.IsCode(got, tt.wantCode) {
				t.Errorf("previewValidationError() code = %v, want %v", apperr.As(got), tt.wantCode)
			}
		})
	}
}

// プレビュー依存未注入検証
func TestVerificationScenarioUseCasePreviewVerificationRunWithoutPreviewRepository(t *testing.T) {
	draft := newVerificationScenarioDraft(t)
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name string
	}{
		{
			name: "プレビュー専用依存未注入を安全に拒否する",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := NewVerificationScenarioUseCase(verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			}, &verificationScenarioRepositoryStub{})

			_, got := useCase.PreviewVerificationRun(context.Background(), "", &draft)
			if !apperr.IsCode(got, apperr.CodeSchemaLoadFailed) {
				t.Errorf("PreviewVerificationRun() error code = %v, want %v", apperr.As(got), apperr.CodeSchemaLoadFailed)
			}
		})
	}
}

// シナリオ作成ユースケース検証
func TestVerificationScenarioUseCaseCreateVerificationScenario(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	draft := newVerificationScenarioDraft(t)
	tests := []struct {
		name            string
		profiles        []domain.Profile
		activeID        *string
		profilesErr     error
		repositoryErr   error
		draft           domain.VerificationScenarioDraft
		wantCode        apperr.Code
		wantCreateCalls int
	}{
		{
			name: "作成内容をアクティブプロファイルへ渡す",
			profiles: []domain.Profile{
				profile,
			},
			activeID:        stringPointer(profile.ID),
			draft:           draft,
			wantCreateCalls: 1,
		},
		{
			name:        "プロファイル読込失敗",
			profilesErr: errors.New("read profiles failed"),
			draft:       draft,
		},
		{
			name: "アクティブプロファイルなし",
			profiles: []domain.Profile{
				profile,
			},
			wantCode: apperr.CodeProfileNotFound,
			draft:    draft,
		},
		{
			name: "保存失敗",
			profiles: []domain.Profile{
				profile,
			},
			activeID:        stringPointer(profile.ID),
			repositoryErr:   errors.New("sqlite path=/private/secret"),
			wantCode:        apperr.CodeScenarioStoreFailed,
			draft:           draft,
			wantCreateCalls: 1,
		},
		{
			name: "下書き形式違反",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			draft: domain.VerificationScenarioDraft{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "主対象の一意生成規則不足",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			draft: domain.VerificationScenarioDraft{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition: map[string]any{
					"childTables": []any{},
					"rowCounts": map[string]any{
						"orders": float64(1),
					},
					"columnGenerators": map[string]any{
						"orders": map[string]any{"id": map[string]any{"kind": "fixed"}},
					},
					"sql":              []any{"SELECT 1"},
					"warmupRuns":       float64(0),
					"iterations":       float64(1),
					"timeLimitSeconds": float64(1),
				},
			},
			wantCode: apperr.CodePrimaryKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &verificationScenarioRepositoryStub{err: tt.repositoryErr}
			profiles := verificationScenarioProfilesStub{
				profiles: tt.profiles,
				activeID: tt.activeID,
				err:      tt.profilesErr,
			}
			useCase := NewVerificationScenarioUseCase(profiles, repository)

			got, err := useCase.CreateVerificationScenario(context.Background(), tt.draft)
			if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("CreateVerificationScenario() error = %v, want wrapped %v", err, tt.profilesErr)
				}

				return
			}
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("CreateVerificationScenario() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}
			} else if err != nil {
				t.Fatalf("CreateVerificationScenario() error = %v", err)
			} else {
				if got.ID == "" {
					t.Error("CreateVerificationScenario() ID is empty, want UUID")
				}
				if got.Name != tt.draft.Name || got.PrimaryTable != tt.draft.PrimaryTable {
					t.Errorf("CreateVerificationScenario() = %#v, want draft name and primary table", got)
				}
				if got.WorkspaceName != nil {
					t.Errorf("WorkspaceName = %v, want nil", got.WorkspaceName)
				}
				if got.CreatedAt.Location() != time.UTC || !got.CreatedAt.Equal(got.UpdatedAt) {
					t.Errorf("timestamps = %v, %v, want equal UTC timestamps", got.CreatedAt, got.UpdatedAt)
				}
				if repository.profileID != profile.ID {
					t.Errorf("CreateVerificationScenario() profile ID = %q, want %q", repository.profileID, profile.ID)
				}
				if !reflect.DeepEqual(repository.created, got) {
					t.Errorf("CreateVerificationScenario() repository scenario = %#v, want %#v", repository.created, got)
				}
			}
			if repository.createCalls != tt.wantCreateCalls {
				t.Errorf("CreateVerificationScenario() repository calls = %d, want %d", repository.createCalls, tt.wantCreateCalls)
			}
		})
	}
}

// 有効なシナリオ下書き生成
func newVerificationScenarioDraft(t *testing.T) domain.VerificationScenarioDraft {
	t.Helper()
	draft, err := domain.NewVerificationScenarioDraft("検証", "orders", verificationScenarioDefinition())
	if err != nil {
		t.Fatalf("NewVerificationScenarioDraft() error = %v", err)
	}

	return draft
}

// 有効なシナリオ定義生成
func verificationScenarioDefinition() map[string]any {
	return map[string]any{
		"childTables": []string{"order_items"},
		"rowCounts": map[string]int{
			"orders":      10,
			"order_items": 20,
		},
		"columnGenerators": map[string]map[string]map[string]string{
			"orders": {
				"id": {"kind": "sequence"},
			},
			"order_items": {
				"id": {"kind": "uuid"},
			},
		},
		"sql":              []string{"SELECT * FROM orders WHERE id = ?"},
		"warmupRuns":       0,
		"iterations":       1,
		"timeLimitSeconds": 1,
	}
}

// シナリオ詳細ユースケース検証
func TestVerificationScenarioUseCaseGetVerificationScenario(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	scenario, err := domain.NewVerificationScenario("scenario-1", "検証", "orders", []byte(`{"rowCounts":{"orders":10}}`), nil, time.Date(2026, time.August, 8, 11, 0, 0, 0, time.UTC), time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewVerificationScenario() error = %v", err)
	}
	tests := []struct {
		name        string
		profiles    []domain.Profile
		activeID    *string
		profilesErr error
		repository  verificationScenarioRepositoryStub
		want        domain.VerificationScenario
		wantCode    apperr.Code
	}{
		{
			name: "詳細返却",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario: scenario,
				found:    true,
			},
			want: scenario,
		},
		{
			name:        "プロファイル読込失敗",
			profilesErr: errors.New("read profiles failed"),
		},
		{
			name: "アクティブプロファイルなし",
			profiles: []domain.Profile{
				profile,
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "他プロファイルのシナリオ",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			wantCode: apperr.CodeScenarioNotFound,
		},
		{
			name: "ストア障害",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				err: errors.New("sqlite path=/private/secret"),
			},
			wantCode: apperr.CodeScenarioStoreFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			profiles := verificationScenarioProfilesStub{
				profiles: tt.profiles,
				activeID: tt.activeID,
				err:      tt.profilesErr,
			}
			useCase := NewVerificationScenarioUseCase(profiles, &repository)

			got, err := useCase.GetVerificationScenario(context.Background(), "scenario-1")
			if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("GetVerificationScenario() error = %v, want wrapped %v", err, tt.profilesErr)
				}

				return
			}
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("GetVerificationScenario() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}

				return
			}
			if err != nil {
				t.Fatalf("GetVerificationScenario() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVerificationScenario() = %#v, want %#v", got, tt.want)
			}
			if repository.profileID != profile.ID {
				t.Errorf("GetVerificationScenario() profile ID = %q, want %q", repository.profileID, profile.ID)
			}
		})
	}
}

// シナリオ複製ユースケース検証
func TestVerificationScenarioUseCaseDuplicateVerificationScenario(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	workspaceName := "verification_orders"
	existing := domain.VerificationScenario{
		ID:            "scenario-1",
		Name:          "検証",
		PrimaryTable:  "orders",
		Definition:    verificationScenarioDefinition(),
		WorkspaceName: &workspaceName,
		CreatedAt:     time.Date(2026, time.August, 8, 11, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC),
	}
	expectedDefinition := newVerificationScenarioDraft(t).Definition
	tests := []struct {
		name            string
		profiles        []domain.Profile
		activeID        *string
		profilesErr     error
		repository      verificationScenarioRepositoryStub
		wantCode        apperr.Code
		wantCreateCalls int
	}{
		{
			name: "同一アクティブプロファイルへ定義だけを複製する",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario: existing,
				found:    true,
			},
			wantCreateCalls: 1,
		},
		{
			name:        "プロファイル読込失敗",
			profilesErr: errors.New("read profiles failed"),
		},
		{
			name: "他プロファイルのシナリオを見つけない",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			wantCode: apperr.CodeScenarioNotFound,
		},
		{
			name: "詳細取得失敗",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				getErr: errors.New("sqlite path=/private/secret"),
			},
			wantCode: apperr.CodeScenarioStoreFailed,
		},
		{
			name: "アクティブプロファイルなし",
			profiles: []domain.Profile{
				profile,
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "保存失敗",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario:  existing,
				found:     true,
				createErr: errors.New("sqlite path=/private/secret"),
			},
			wantCode:        apperr.CodeScenarioStoreFailed,
			wantCreateCalls: 1,
		},
		{
			name: "保存済みシナリオの形式違反",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario: domain.VerificationScenario{
					Name:         "検証",
					PrimaryTable: "orders",
					Definition:   map[string]any{},
				},
				found: true,
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "保存済みシナリオの主対象PK不足",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario: domain.VerificationScenario{
					Name:         "検証",
					PrimaryTable: "orders",
					Definition: map[string]any{
						"childTables": []any{},
						"rowCounts": map[string]any{
							"orders": float64(1),
						},
						"columnGenerators": map[string]any{
							"orders": map[string]any{"id": map[string]any{"kind": "fixed"}},
						},
						"sql":              []any{"SELECT 1"},
						"warmupRuns":       float64(0),
						"iterations":       float64(1),
						"timeLimitSeconds": float64(1),
					},
				},
				found: true,
			},
			wantCode: apperr.CodePrimaryKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			profiles := verificationScenarioProfilesStub{
				profiles: tt.profiles,
				activeID: tt.activeID,
				err:      tt.profilesErr,
			}
			useCase := NewVerificationScenarioUseCase(profiles, &repository)

			got, err := useCase.DuplicateVerificationScenario(context.Background(), "scenario-1")
			if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("DuplicateVerificationScenario() error = %v, want wrapped %v", err, tt.profilesErr)
				}

				return
			}
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("DuplicateVerificationScenario() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}
			} else if err != nil {
				t.Fatalf("DuplicateVerificationScenario() error = %v", err)
			} else {
				if got.ID == existing.ID || got.ID == "" {
					t.Errorf("ID = %q, want new non-empty ID", got.ID)
				}
				if got.Name != existing.Name || got.PrimaryTable != existing.PrimaryTable || !reflect.DeepEqual(got.Definition, expectedDefinition) {
					t.Errorf("DuplicateVerificationScenario() = %#v, want copied definition", got)
				}
				if got.WorkspaceName != nil {
					t.Errorf("WorkspaceName = %v, want nil", got.WorkspaceName)
				}
				if !reflect.DeepEqual(repository.created, got) {
					t.Errorf("CreateVerificationScenario() scenario = %#v, want %#v", repository.created, got)
				}
				if repository.profileID != profile.ID {
					t.Errorf("CreateVerificationScenario() profile ID = %q, want %q", repository.profileID, profile.ID)
				}
			}
			if repository.createCalls != tt.wantCreateCalls {
				t.Errorf("CreateVerificationScenario() calls = %d, want %d", repository.createCalls, tt.wantCreateCalls)
			}
		})
	}
}

// シナリオ更新ユースケース検証
func TestVerificationScenarioUseCaseUpdateVerificationScenario(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	workspaceName := "verification_orders"
	existing := domain.VerificationScenario{
		ID:            "scenario-1",
		Name:          "更新前",
		PrimaryTable:  "orders",
		Definition:    verificationScenarioDefinition(),
		WorkspaceName: &workspaceName,
		CreatedAt:     time.Date(2026, time.August, 8, 11, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC),
	}
	draft := newVerificationScenarioDraft(t)
	tests := []struct {
		name            string
		profiles        []domain.Profile
		activeID        *string
		profilesErr     error
		repository      verificationScenarioRepositoryStub
		draft           domain.VerificationScenarioDraft
		wantCode        apperr.Code
		wantUpdateCalls int
	}{
		{
			name: "ID作成日時ワークスペースを保持して更新する",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario:    existing,
				found:       true,
				updateFound: true,
			},
			draft:           draft,
			wantUpdateCalls: 1,
		},
		{
			name:        "プロファイル読込失敗",
			profilesErr: errors.New("read profiles failed"),
			draft:       draft,
		},
		{
			name: "他プロファイルのシナリオを見つけない",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			draft:    draft,
			wantCode: apperr.CodeScenarioNotFound,
		},
		{
			name: "アクティブプロファイルなし",
			profiles: []domain.Profile{
				profile,
			},
			draft:    draft,
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "主対象の一意生成規則不足",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			draft: domain.VerificationScenarioDraft{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition: map[string]any{
					"childTables": []any{},
					"rowCounts": map[string]any{
						"orders": float64(1),
					},
					"columnGenerators": map[string]any{
						"orders": map[string]any{"id": map[string]any{"kind": "fixed"}},
					},
					"sql":              []any{"SELECT 1"},
					"warmupRuns":       float64(0),
					"iterations":       float64(1),
					"timeLimitSeconds": float64(1),
				},
			},
			wantCode: apperr.CodePrimaryKeyRequired,
		},
		{
			name: "詳細取得のストア障害",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				err: errors.New("sqlite path=/private/secret"),
			},
			draft:    draft,
			wantCode: apperr.CodeScenarioStoreFailed,
		},
		{
			name: "不正な保存済みIDを拒否する",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario:    domain.VerificationScenario{},
				found:       true,
				updateFound: true,
			},
			draft:    draft,
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "更新競合で対象が消えた場合は見つからない",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario: existing,
				found:    true,
			},
			draft:           draft,
			wantCode:        apperr.CodeScenarioNotFound,
			wantUpdateCalls: 1,
		},
		{
			name: "ストア障害",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				scenario:  existing,
				found:     true,
				updateErr: errors.New("sqlite path=/private/secret"),
			},
			draft:           draft,
			wantCode:        apperr.CodeScenarioStoreFailed,
			wantUpdateCalls: 1,
		},
		{
			name: "下書き形式違反",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			draft: domain.VerificationScenarioDraft{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			},
			wantCode: apperr.CodeValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			profiles := verificationScenarioProfilesStub{
				profiles: tt.profiles,
				activeID: tt.activeID,
				err:      tt.profilesErr,
			}
			useCase := NewVerificationScenarioUseCase(profiles, &repository)

			got, err := useCase.UpdateVerificationScenario(context.Background(), "scenario-1", tt.draft)
			if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("UpdateVerificationScenario() error = %v, want wrapped %v", err, tt.profilesErr)
				}

				return
			}
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("UpdateVerificationScenario() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}
			} else if err != nil {
				t.Fatalf("UpdateVerificationScenario() error = %v", err)
			} else {
				if got.ID != existing.ID || !got.CreatedAt.Equal(existing.CreatedAt) || got.WorkspaceName != existing.WorkspaceName {
					t.Errorf("UpdateVerificationScenario() = %#v, want preserved identity and workspace", got)
				}
				if got.UpdatedAt.Location() != time.UTC || got.UpdatedAt.IsZero() {
					t.Errorf("UpdatedAt = %v, want non-zero UTC time", got.UpdatedAt)
				}
				if !reflect.DeepEqual(repository.updated, got) {
					t.Errorf("UpdateVerificationScenario() repository scenario = %#v, want %#v", repository.updated, got)
				}
			}
			if repository.updateCalls != tt.wantUpdateCalls {
				t.Errorf("UpdateVerificationScenario() repository calls = %d, want %d", repository.updateCalls, tt.wantUpdateCalls)
			}
		})
	}
}

// シナリオ削除ユースケース検証
func TestVerificationScenarioUseCaseDeleteVerificationScenario(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name                 string
		profiles             []domain.Profile
		activeID             *string
		profilesErr          error
		repository           verificationScenarioRepositoryStub
		wantWorkspaceRemoved bool
		wantCode             apperr.Code
		wantErr              string
		wantDeleteCalls      int
	}{
		{
			name: "workspace削除結果を返す",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				deleteFound:            true,
				deleteWorkspaceRemoved: true,
			},
			wantWorkspaceRemoved: true,
			wantDeleteCalls:      1,
		},
		{
			name: "使用中シナリオを拒否する",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				deleteBusy: true,
			},
			wantCode:        apperr.CodeScenarioBusy,
			wantDeleteCalls: 1,
		},
		{
			name: "存在しないシナリオを返す",
			profiles: []domain.Profile{
				profile,
			},
			activeID:        stringPointer(profile.ID),
			wantCode:        apperr.CodeScenarioNotFound,
			wantDeleteCalls: 1,
		},
		{
			name: "ストア障害を安全に変換する",
			profiles: []domain.Profile{
				profile,
			},
			activeID: stringPointer(profile.ID),
			repository: verificationScenarioRepositoryStub{
				deleteErr: errors.New("sqlite path=/private/secret"),
			},
			wantCode:        apperr.CodeScenarioStoreFailed,
			wantDeleteCalls: 1,
		},
		{
			name:            "プロファイル読込失敗を返す",
			profilesErr:     errors.New("profile load failed"),
			wantErr:         "profile load failed",
			wantDeleteCalls: 0,
		},
		{
			name: "アクティブプロファイル不在を拒否する",
			profiles: []domain.Profile{
				profile,
			},
			wantCode:        apperr.CodeProfileNotFound,
			wantDeleteCalls: 0,
		},
		{
			name: "アクティブプロファイル不一致を拒否する",
			profiles: []domain.Profile{
				profile,
			},
			activeID:        stringPointer("profile-2"),
			wantCode:        apperr.CodeProfileNotFound,
			wantDeleteCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			useCase := NewVerificationScenarioUseCase(verificationScenarioProfilesStub{
				profiles: tt.profiles,
				activeID: tt.activeID,
				err:      tt.profilesErr,
			}, &repository)

			got, err := useCase.DeleteVerificationScenario(context.Background(), "scenario-1", true)
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("DeleteVerificationScenario() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}
			} else if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("DeleteVerificationScenario() error = %v, want text %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("DeleteVerificationScenario() error = %v", err)
			} else if got != tt.wantWorkspaceRemoved {
				t.Errorf("DeleteVerificationScenario() = %v, want %v", got, tt.wantWorkspaceRemoved)
			}
			if repository.deleteCalls != tt.wantDeleteCalls {
				t.Errorf("DeleteVerificationScenario() repository calls = %d, want %d", repository.deleteCalls, tt.wantDeleteCalls)
			}
		})
	}
}

// シナリオ削除の実行中判定と作成中補償検証
func TestVerificationScenarioUseCaseDeleteVerificationScenarioWorkspaceCoordination(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name                    string
		workspaceState          string
		workspaceErr            error
		runBusy                 bool
		runBusyErr              error
		keepWorkspace           bool
		workspaceNil            bool
		saveWorkspaceErr        error
		workspaceDeleteErr      error
		workspaceName           string
		stateDeleteErr          error
		wantCode                apperr.Code
		wantRemoved             bool
		wantSavedStates         []string
		wantExternalDeleteCalls int
		wantStateDeleteCalls    int
		wantScenarioDeleteCalls int
	}{
		{
			name:                    "想定外workspace名は副作用なしで拒否する",
			workspaceState:          "inactive",
			workspaceName:           "db_checker_v_profile_scenario",
			wantCode:                apperr.CodeVerificationNamespaceFailed,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "ワークスペース状態取得失敗は副作用なしで返す",
			workspaceErr:            errors.New("sqlite failed"),
			wantCode:                apperr.CodeScenarioStoreFailed,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "prepared実行中は副作用なしで拒否する",
			workspaceState:          "inactive",
			runBusy:                 true,
			wantCode:                apperr.CodeScenarioBusy,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "running実行中は副作用なしで拒否する",
			workspaceState:          "inactive",
			runBusy:                 true,
			wantCode:                apperr.CodeScenarioBusy,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "canceling実行中は副作用なしで拒否する",
			workspaceState:          "inactive",
			runBusy:                 true,
			wantCode:                apperr.CodeScenarioBusy,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "実行状態取得失敗は副作用なしで返す",
			workspaceState:          "inactive",
			runBusyErr:              errors.New("sqlite failed"),
			wantCode:                apperr.CodeScenarioStoreFailed,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "workspaceを残す指定では副作用なしで拒否する",
			workspaceState:          "inactive",
			keepWorkspace:           true,
			wantCode:                apperr.CodeScenarioBusy,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "active workspaceは副作用なしで拒否する",
			workspaceState:          "active",
			wantCode:                apperr.CodeScenarioBusy,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "外部workspace未注入では副作用なしで返す",
			workspaceState:          "inactive",
			workspaceNil:            true,
			wantCode:                apperr.CodeVerificationNamespaceFailed,
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "deleting保存失敗では外部削除しない",
			workspaceState:          "inactive",
			saveWorkspaceErr:        errors.New("sqlite failed"),
			wantCode:                apperr.CodeScenarioStoreFailed,
			wantSavedStates:         []string{"deleting"},
			wantExternalDeleteCalls: 0,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "creating状態を補償削除してシナリオを削除する",
			workspaceState:          "creating",
			wantRemoved:             true,
			wantSavedStates:         []string{"deleting"},
			wantExternalDeleteCalls: 1,
			wantStateDeleteCalls:    1,
			wantScenarioDeleteCalls: 1,
		},
		{
			name:                    "workspace状態削除失敗ではシナリオを残す",
			workspaceState:          "creating",
			stateDeleteErr:          errors.New("sqlite failed"),
			wantCode:                apperr.CodeScenarioStoreFailed,
			wantSavedStates:         []string{"deleting"},
			wantExternalDeleteCalls: 1,
			wantStateDeleteCalls:    1,
			wantScenarioDeleteCalls: 0,
		},
		{
			name:                    "creating状態の外部削除失敗ではdeletingを保持する",
			workspaceState:          "creating",
			workspaceDeleteErr:      errors.New("drop failed"),
			wantCode:                apperr.CodeVerificationNamespaceFailed,
			wantSavedStates:         []string{"deleting"},
			wantExternalDeleteCalls: 1,
			wantStateDeleteCalls:    0,
			wantScenarioDeleteCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := verificationScenarioRepositoryStub{
				deleteFound:        true,
				workspaceFound:     true,
				workspaceState:     tt.workspaceState,
				workspaceName:      verificationWorkspaceName("profile-1", "scenario-1"),
				workspaceErr:       tt.workspaceErr,
				runBusy:            tt.runBusy,
				runBusyErr:         tt.runBusyErr,
				deleteWorkspaceErr: tt.stateDeleteErr,
			}
			if tt.saveWorkspaceErr != nil {
				repository.saveWorkspaceErrors = []error{tt.saveWorkspaceErr}
			}
			if tt.workspaceName != "" {
				repository.workspaceName = tt.workspaceName
			}
			workspace := &verificationWorkspaceStub{deleteErr: tt.workspaceDeleteErr}
			if tt.workspaceNil {
				workspace = nil
			}
			profiles := verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			}
			var useCase *VerificationScenarioUseCase
			if tt.workspaceNil {
				useCase = NewVerificationScenarioUseCase(profiles, &repository)
			} else {
				useCase = NewVerificationScenarioUseCase(profiles, &repository, workspace)
			}

			removeWorkspace := !tt.keepWorkspace
			got, err := useCase.DeleteVerificationScenario(context.Background(), "scenario-1", removeWorkspace)
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("DeleteVerificationScenario() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}
			} else if err != nil {
				t.Fatalf("DeleteVerificationScenario() error = %v", err)
			}
			if got != tt.wantRemoved {
				t.Errorf("DeleteVerificationScenario() = %v, want %v", got, tt.wantRemoved)
			}
			if !reflect.DeepEqual(repository.savedWorkspaceStates, tt.wantSavedStates) {
				t.Errorf("saved states = %#v, want %#v", repository.savedWorkspaceStates, tt.wantSavedStates)
			}
			workspaceDeleteCalls := 0
			if workspace != nil {
				workspaceDeleteCalls = workspace.deleteCalls
			}
			if workspaceDeleteCalls != tt.wantExternalDeleteCalls {
				t.Errorf("DeleteWorkspace() calls = %d, want %d", workspaceDeleteCalls, tt.wantExternalDeleteCalls)
			}
			if repository.deleteWorkspaceCalls != tt.wantStateDeleteCalls {
				t.Errorf("DeleteVerificationWorkspace() calls = %d, want %d", repository.deleteWorkspaceCalls, tt.wantStateDeleteCalls)
			}
			if repository.deleteCalls != tt.wantScenarioDeleteCalls {
				t.Errorf("DeleteVerificationScenario() calls = %d, want %d", repository.deleteCalls, tt.wantScenarioDeleteCalls)
			}
		})
	}
}

// 検証ワークスペース開始の補償状態検証
func TestVerificationScenarioUseCaseEnterVerificationWorkspaceKeepsCreatingState(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name       string
		repository verificationScenarioRepositoryStub
		wantCode   apperr.Code
		wantStates []string
		wantCreate int
	}{
		{
			name: "想定外workspace名は副作用なしで拒否する",
			repository: verificationScenarioRepositoryStub{
				found:          true,
				workspaceFound: true,
				workspaceState: "creating",
				workspaceName:  "db_checker_v_profile_scenario",
			},
			wantCode: apperr.CodeVerificationNamespaceFailed,
		},
		{
			name: "初期状態保存に失敗した場合は外部作成しない",
			repository: verificationScenarioRepositoryStub{
				found:               true,
				saveWorkspaceErrors: []error{errors.New("sqlite failed")},
			},
			wantCode:   apperr.CodeScenarioStoreFailed,
			wantStates: []string{"creating"},
		},
		{
			name: "作成後の状態保存失敗ではcreatingを残す",
			repository: verificationScenarioRepositoryStub{
				found: true,
				saveWorkspaceErrors: []error{
					nil,
					errors.New("sqlite failed"),
				},
			},
			wantCode: apperr.CodeScenarioStoreFailed,
			wantStates: []string{
				"creating",
				"test",
			},
			wantCreate: 1,
		},
		{
			name: "creating状態から再試行して復旧する",
			repository: verificationScenarioRepositoryStub{
				found:          true,
				workspaceFound: true,
				workspaceState: "creating",
				workspaceName:  verificationWorkspaceName("profile-1", "scenario-1"),
			},
			wantStates: []string{"test"},
			wantCreate: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			workspace := &verificationWorkspaceStub{}
			useCase := NewVerificationScenarioUseCase(verificationScenarioProfilesStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			}, &repository, workspace)
			_, err := useCase.EnterVerificationWorkspace(context.Background(), "scenario-1")
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("EnterVerificationWorkspace() error = %v, want code %q", err, tt.wantCode)
				}
			} else if err != nil {
				t.Fatalf("EnterVerificationWorkspace() error = %v", err)
			}
			if !reflect.DeepEqual(repository.savedWorkspaceStates, tt.wantStates) {
				t.Errorf("saved states = %#v, want %#v", repository.savedWorkspaceStates, tt.wantStates)
			}
			if workspace.createCalls != tt.wantCreate {
				t.Errorf("CreateWorkspace() calls = %d, want %d", workspace.createCalls, tt.wantCreate)
			}
		})
	}
}

// 検証先内部名の衝突回避検証
func TestVerificationWorkspaceName(t *testing.T) {
	tests := []struct {
		name       string
		profileID  string
		scenarioID string
		wantName   string
	}{
		{
			name:       "同じIDの組合せは同じ名前になる",
			profileID:  "profile-123456789012-a",
			scenarioID: "scenario-123456789012-a",
			wantName:   "db_checker_v_1316d63feae08ab6f190d7e63c33449d9f0209cee3d9212d",
		},
		{
			name:       "先頭12文字が同じでも別シナリオは別名になる",
			profileID:  "profile-123456789012-a",
			scenarioID: "scenario-123456789012-b",
			wantName:   "db_checker_v_68cf3ef972343d6e508f4d9693f2b08d7ebfd7cc5bc3bef3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verificationWorkspaceName(tt.profileID, tt.scenarioID); got != tt.wantName {
				t.Errorf("verificationWorkspaceName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

// ワークスペース状態非対応リポジトリ
type verificationScenarioRepositoryWithoutWorkspaceState struct{}

// シナリオ作成再現
func (verificationScenarioRepositoryWithoutWorkspaceState) CreateVerificationScenario(context.Context, string, domain.VerificationScenario) error {
	return nil
}

// シナリオ一覧取得再現
func (verificationScenarioRepositoryWithoutWorkspaceState) ListVerificationScenarios(context.Context, string) ([]domain.VerificationScenarioSummary, error) {
	return nil, nil
}

// シナリオ詳細取得再現
func (verificationScenarioRepositoryWithoutWorkspaceState) GetVerificationScenario(context.Context, string, string) (domain.VerificationScenario, bool, error) {
	return domain.VerificationScenario{}, false, nil
}

// シナリオ更新再現
func (verificationScenarioRepositoryWithoutWorkspaceState) UpdateVerificationScenario(context.Context, string, domain.VerificationScenario) (bool, error) {
	return false, nil
}

// シナリオ削除再現
func (verificationScenarioRepositoryWithoutWorkspaceState) DeleteVerificationScenario(context.Context, string, string, bool) (bool, bool, bool, error) {
	return false, false, false, nil
}

// ワークスペース終了ユースケース検証
func TestVerificationScenarioUseCaseExitVerificationWorkspace(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "127.0.0.1", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name          string
		repository    verificationScenarioRepositoryStub
		profilesErr   error
		activeID      *string
		withoutStates bool
		wantCode      apperr.Code
		wantStates    []string
	}{
		{name: "プロファイル読込失敗",

			profilesErr: errors.New("profiles failed")},

		{name: "アクティブプロファイルなし",

			wantCode: apperr.CodeProfileNotFound},

		{name: "状態リポジトリ未対応",

			activeID: stringPointer(profile.ID),

			withoutStates: true,

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "シナリオ取得失敗",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{getErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "シナリオなし",

			activeID: stringPointer(profile.ID),

			wantCode: apperr.CodeScenarioNotFound},

		{name: "workspace取得失敗",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "test状態でない",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "inactive"},

			wantCode: apperr.CodeScenarioBusy},

		{name: "workspaceなし",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true},

			wantCode: apperr.CodeScenarioBusy},

		{name: "実行状態取得失敗",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test",

				runBusyErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "実行中",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test",

				runBusy: true},

			wantCode: apperr.CodeScenarioBusy},

		{name: "状態保存失敗",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test",

				saveWorkspaceErrors: []error{errors.New("sqlite failed")}},

			wantCode: apperr.CodeScenarioStoreFailed,

			wantStates: []string{"inactive"}},

		{name: "終了成功",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test"},

			wantStates: []string{"inactive"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			profiles := verificationScenarioProfilesStub{profiles: []domain.Profile{profile},

				activeID: tt.activeID,

				err: tt.profilesErr}
			var useCase *VerificationScenarioUseCase
			if tt.withoutStates {
				useCase = NewVerificationScenarioUseCase(profiles, verificationScenarioRepositoryWithoutWorkspaceState{})
			} else {
				useCase = NewVerificationScenarioUseCase(profiles, &repository)
			}

			err := useCase.ExitVerificationWorkspace(context.Background(), "scenario-1")
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("ExitVerificationWorkspace() error = %v, want code %q", err, tt.wantCode)
				}
			} else if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("ExitVerificationWorkspace() error = %v, want wrapped %v", err, tt.profilesErr)
				}
			} else if err != nil {
				t.Fatalf("ExitVerificationWorkspace() error = %v", err)
			}
			if !reflect.DeepEqual(repository.savedWorkspaceStates, tt.wantStates) {
				t.Errorf("saved states = %#v, want %#v", repository.savedWorkspaceStates, tt.wantStates)
			}
		})
	}
}

// 実行状態遷移判定検証
func TestIsAllowedVerificationRunTransition(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		want    bool
	}{
		{name: "preparedからrunning",

			current: "prepared",

			next: "running",

			want: true},

		{name: "preparedからcanceling",

			current: "prepared",

			next: "canceling",

			want: true},

		{name: "preparedからcanceled",

			current: "prepared",

			next: "canceled",

			want: true},

		{name: "preparedから成功は不可",

			current: "prepared",

			next: "succeeded"},

		{name: "runningからcanceling",

			current: "running",

			next: "canceling",

			want: true},

		{name: "runningから成功",

			current: "running",

			next: "succeeded",

			want: true},

		{name: "runningから失敗",

			current: "running",

			next: "failed",

			want: true},

		{name: "runningからcanceledは不可",

			current: "running",

			next: "canceled"},

		{name: "cancelingからcanceled",

			current: "canceling",

			next: "canceled",

			want: true},

		{name: "cancelingから失敗",

			current: "canceling",

			next: "failed",

			want: true},

		{name: "cancelingから成功は不可",

			current: "canceling",

			next: "succeeded"},

		{name: "不明状態は不可",

			current: "succeeded",

			next: "running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllowedVerificationRunTransition(tt.current, tt.next); got != tt.want {
				t.Errorf("isAllowedVerificationRunTransition() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ワークスペース開始の境界条件検証
func TestVerificationScenarioUseCaseEnterVerificationWorkspaceBoundaries(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "127.0.0.1", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name          string
		repository    verificationScenarioRepositoryStub
		profiles      []domain.Profile
		activeID      *string
		profilesErr   error
		withoutStates bool
		workspaceNil  bool
		createErr     error
		wantCode      apperr.Code
		wantName      string
		wantStates    []string
	}{
		{name: "プロファイル読込失敗",

			profilesErr: errors.New("profiles failed")},

		{name: "アクティブプロファイルなし",

			profiles: []domain.Profile{profile},

			wantCode: apperr.CodeProfileNotFound},

		{name: "アクティブプロファイル不一致",

			profiles: []domain.Profile{profile},

			activeID: stringPointer("other"),

			wantCode: apperr.CodeProfileNotFound},

		{name: "外部workspace未注入",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			workspaceNil: true,

			wantCode: apperr.CodeVerificationNamespaceFailed},

		{name: "状態リポジトリ未対応",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			withoutStates: true,

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "シナリオ取得失敗",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{getErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "シナリオなし",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			wantCode: apperr.CodeScenarioNotFound},

		{name: "workspace取得失敗",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "active状態は拒否",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "active",

				workspaceName: verificationWorkspaceName(profile.ID, "scenario-1")},

			wantCode: apperr.CodeScenarioBusy},

		{name: "test状態は拒否",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test",

				workspaceName: verificationWorkspaceName(profile.ID, "scenario-1")},

			wantCode: apperr.CodeScenarioBusy},

		{name: "deleting状態は拒否",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "deleting",

				workspaceName: verificationWorkspaceName(profile.ID, "scenario-1")},

			wantCode: apperr.CodeScenarioBusy},

		{name: "初回作成失敗ではcreatingを残す",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true},

			createErr: errors.New("create failed"),

			wantCode: apperr.CodeVerificationNamespaceFailed,

			wantStates: []string{"creating"}},

		{name: "inactive状態の保存失敗では外部作成しない",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "inactive",

				workspaceName: verificationWorkspaceName(profile.ID, "scenario-1"),

				saveWorkspaceErrors: []error{errors.New("sqlite failed")}},

			wantCode: apperr.CodeScenarioStoreFailed,

			wantStates: []string{"creating"}},

		{name: "inactiveから開始",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "inactive",

				workspaceName: verificationWorkspaceName(profile.ID, "scenario-1")},

			wantName: verificationWorkspaceName(profile.ID, "scenario-1"),

			wantStates: []string{"creating",

				"test"}},

		{name: "初回開始",

			profiles: []domain.Profile{profile},

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{found: true},

			wantName: verificationWorkspaceName(profile.ID, "scenario-1"),

			wantStates: []string{"creating",

				"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			profiles := verificationScenarioProfilesStub{profiles: tt.profiles,

				activeID: tt.activeID,

				err: tt.profilesErr}
			workspace := &verificationWorkspaceStub{createErr: tt.createErr}
			var useCase *VerificationScenarioUseCase
			switch {
			case tt.withoutStates:
				useCase = NewVerificationScenarioUseCase(profiles, verificationScenarioRepositoryWithoutWorkspaceState{}, workspace)
			case tt.workspaceNil:
				useCase = NewVerificationScenarioUseCase(profiles, &repository)
			default:
				useCase = NewVerificationScenarioUseCase(profiles, &repository, workspace)
			}

			got, err := useCase.EnterVerificationWorkspace(context.Background(), "scenario-1")
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("EnterVerificationWorkspace() error = %v, want code %q", err, tt.wantCode)
				}
			} else if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("EnterVerificationWorkspace() error = %v, want wrapped %v", err, tt.profilesErr)
				}
			} else if err != nil {
				t.Fatalf("EnterVerificationWorkspace() error = %v", err)
			}
			if got != tt.wantName {
				t.Errorf("EnterVerificationWorkspace() = %q, want %q", got, tt.wantName)
			}
			if !reflect.DeepEqual(repository.savedWorkspaceStates, tt.wantStates) {
				t.Errorf("saved states = %#v, want %#v", repository.savedWorkspaceStates, tt.wantStates)
			}
		})
	}
}

// 実行状態作成ユースケース検証
func TestVerificationScenarioUseCasePrepareVerificationRun(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "127.0.0.1", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name          string
		repository    verificationScenarioRepositoryStub
		profilesErr   error
		activeID      *string
		withoutStates bool
		scenarioID    string
		runID         string
		wantCode      apperr.Code
		wantCalls     int
	}{
		{name: "プロファイル読込失敗",

			profilesErr: errors.New("profiles failed")},

		{name: "アクティブプロファイルなし",

			wantCode: apperr.CodeProfileNotFound},

		{name: "状態リポジトリ未対応",

			activeID: stringPointer(profile.ID),

			withoutStates: true,

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "シナリオIDなし",

			activeID: stringPointer(profile.ID),

			wantCode: apperr.CodeValidationFailed},

		{name: "runIDなし",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			wantCode: apperr.CodeValidationFailed},

		{name: "シナリオ取得失敗",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{getErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "シナリオなし",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			wantCode: apperr.CodeScenarioNotFound},

		{name: "workspace取得失敗",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "workspace未開始",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{found: true},

			wantCode: apperr.CodeScenarioBusy},

		{name: "workspaceがtest以外",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "inactive"},

			wantCode: apperr.CodeScenarioBusy},

		{name: "run取得失敗",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test",

				runErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "run重複",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test",

				runFound: true},

			wantCode: apperr.CodeValidationFailed},

		{name: "run保存失敗",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test",

				createRunErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed,

			wantCalls: 1},

		{name: "run作成",

			activeID: stringPointer(profile.ID),

			scenarioID: "scenario-1",

			runID: "run-1",

			repository: verificationScenarioRepositoryStub{found: true,

				workspaceFound: true,

				workspaceState: "test"},

			wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			profiles := verificationScenarioProfilesStub{profiles: []domain.Profile{profile},

				activeID: tt.activeID,

				err: tt.profilesErr}
			var useCase *VerificationScenarioUseCase
			if tt.withoutStates {
				useCase = NewVerificationScenarioUseCase(profiles, verificationScenarioRepositoryWithoutWorkspaceState{})
			} else {
				useCase = NewVerificationScenarioUseCase(profiles, &repository)
			}
			err := useCase.PrepareVerificationRun(context.Background(), tt.scenarioID, tt.runID)
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("PrepareVerificationRun() error = %v, want code %q", err, tt.wantCode)
				}
			} else if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("PrepareVerificationRun() error = %v, want wrapped %v", err, tt.profilesErr)
				}
			} else if err != nil {
				t.Fatalf("PrepareVerificationRun() error = %v", err)
			}
			if repository.createRunCalls != tt.wantCalls {
				t.Errorf("CreateVerificationRun() calls = %d, want %d", repository.createRunCalls, tt.wantCalls)
			}
		})
	}
}

// 実行状態更新ユースケース検証
func TestVerificationScenarioUseCaseUpdateVerificationRunState(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "127.0.0.1", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name          string
		repository    verificationScenarioRepositoryStub
		profilesErr   error
		activeID      *string
		withoutStates bool
		state         string
		wantCode      apperr.Code
		wantCalls     int
	}{
		{name: "不正な遷移先",

			state: "unknown",

			wantCode: apperr.CodeValidationFailed},

		{name: "プロファイル読込失敗",

			state: "running",

			profilesErr: errors.New("profiles failed")},

		{name: "アクティブプロファイルなし",

			state: "running",

			wantCode: apperr.CodeProfileNotFound},

		{name: "状態リポジトリ未対応",

			state: "running",

			activeID: stringPointer(profile.ID),

			withoutStates: true,

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "run取得失敗",

			state: "running",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{runErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed},

		{name: "runなし",

			state: "running",

			activeID: stringPointer(profile.ID),

			wantCode: apperr.CodeScenarioNotFound},

		{name: "不許可遷移",

			state: "succeeded",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{runFound: true,

				runState: "prepared"},

			wantCode: apperr.CodeValidationFailed},

		{name: "更新失敗",

			state: "running",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{runFound: true,

				runState: "prepared",

				updateRunErr: errors.New("sqlite failed")},

			wantCode: apperr.CodeScenarioStoreFailed,

			wantCalls: 1},

		{name: "更新競合でrunなし",

			state: "running",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{runFound: true,

				runState: "prepared"},

			wantCode: apperr.CodeScenarioNotFound,

			wantCalls: 1},

		{name: "preparedからrunning",

			state: "running",

			activeID: stringPointer(profile.ID),

			repository: verificationScenarioRepositoryStub{runFound: true,

				runState: "prepared",

				updateRunFound: true},

			wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			profiles := verificationScenarioProfilesStub{profiles: []domain.Profile{profile},

				activeID: tt.activeID,

				err: tt.profilesErr}
			var useCase *VerificationScenarioUseCase
			if tt.withoutStates {
				useCase = NewVerificationScenarioUseCase(profiles, verificationScenarioRepositoryWithoutWorkspaceState{})
			} else {
				useCase = NewVerificationScenarioUseCase(profiles, &repository)
			}
			err := useCase.UpdateVerificationRunState(context.Background(), "run-1", tt.state)
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("UpdateVerificationRunState() error = %v, want code %q", err, tt.wantCode)
				}
			} else if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("UpdateVerificationRunState() error = %v, want wrapped %v", err, tt.profilesErr)
				}
			} else if err != nil {
				t.Fatalf("UpdateVerificationRunState() error = %v", err)
			}
			if repository.updateRunCalls != tt.wantCalls {
				t.Errorf("UpdateVerificationRunState() calls = %d, want %d", repository.updateRunCalls, tt.wantCalls)
			}
		})
	}
}

// シナリオ一覧取得検証
func TestVerificationScenarioUseCaseListVerificationScenarios(t *testing.T) {
	updatedAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	profile, err := domain.NewProfile("profile-1", "Local", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	tests := []struct {
		name          string
		profiles      []domain.Profile
		activeID      *string
		profilesErr   error
		repositoryErr error
		want          []domain.VerificationScenarioSummary
		wantCode      apperr.Code
		wantProfileID string
	}{
		{
			name:          "空一覧",
			profiles:      []domain.Profile{profile},
			activeID:      stringPointer(profile.ID),
			want:          []domain.VerificationScenarioSummary{},
			wantProfileID: profile.ID,
		},
		{
			name:        "プロファイル読込失敗",
			profilesErr: errors.New("read profiles failed"),
		},
		{
			name:     "アクティブプロファイルなし",
			profiles: []domain.Profile{profile},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name:     "アクティブプロファイル未発見",
			profiles: []domain.Profile{profile},
			activeID: stringPointer("missing"),
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name:          "ストア障害",
			profiles:      []domain.Profile{profile},
			activeID:      stringPointer(profile.ID),
			repositoryErr: errors.New("sqlite path=/private/secret"),
			wantCode:      apperr.CodeScenarioStoreFailed,
			wantProfileID: profile.ID,
		},
		{
			name:     "一覧返却",
			profiles: []domain.Profile{profile},
			activeID: stringPointer(profile.ID),
			want: []domain.VerificationScenarioSummary{{
				ID:           "scenario-1",
				Name:         "検証",
				PrimaryTable: "orders",
				UpdatedAt:    updatedAt,
			}},
			wantProfileID: profile.ID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &verificationScenarioRepositoryStub{
				scenarios: tt.want,
				err:       tt.repositoryErr,
			}
			profiles := verificationScenarioProfilesStub{
				profiles: tt.profiles,
				activeID: tt.activeID,
				err:      tt.profilesErr,
			}
			useCase := NewVerificationScenarioUseCase(profiles, repository)

			got, err := useCase.ListVerificationScenarios(context.Background())
			if tt.profilesErr != nil {
				if !errors.Is(err, tt.profilesErr) {
					t.Errorf("ListVerificationScenarios() error = %v, want wrapped %v", err, tt.profilesErr)
				}

				return
			}
			if tt.wantCode != "" {
				if !apperr.IsCode(err, tt.wantCode) {
					t.Errorf("ListVerificationScenarios() error code = %v, want %v", apperr.As(err), tt.wantCode)
				}

				return
			}
			if err != nil {
				t.Fatalf("ListVerificationScenarios() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListVerificationScenarios() = %#v, want %#v", got, tt.want)
			}
			if repository.profileID != tt.wantProfileID {
				t.Errorf("ListVerificationScenarios() profile ID = %q, want %q", repository.profileID, tt.wantProfileID)
			}
		})
	}
}
