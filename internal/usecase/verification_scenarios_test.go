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
	profiles []domain.Profile
	activeID *string
	err      error
}

type verificationScenarioRepositoryStub struct {
	scenarios   []domain.VerificationScenarioSummary
	scenario    domain.VerificationScenario
	found       bool
	err         error
	profileID   string
	created     domain.VerificationScenario
	createCalls int
	updated     domain.VerificationScenario
	updateCalls int
	updateFound bool
	updateErr   error
}

// プロファイル読込再現
func (s verificationScenarioProfilesStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return s.profiles, s.activeID, s.err
}

// シナリオ一覧取得再現
func (s *verificationScenarioRepositoryStub) ListVerificationScenarios(_ context.Context, profileID string) ([]domain.VerificationScenarioSummary, error) {
	s.profileID = profileID

	return s.scenarios, s.err
}

// シナリオ詳細取得再現
func (s *verificationScenarioRepositoryStub) GetVerificationScenario(_ context.Context, profileID, _ string) (domain.VerificationScenario, bool, error) {
	s.profileID = profileID

	return s.scenario, s.found, s.err
}

// シナリオ作成再現
func (s *verificationScenarioRepositoryStub) CreateVerificationScenario(_ context.Context, profileID string, scenario domain.VerificationScenario) error {
	s.profileID = profileID
	s.created = scenario
	s.createCalls++

	return s.err
}

// シナリオ更新再現
func (s *verificationScenarioRepositoryStub) UpdateVerificationScenario(_ context.Context, profileID string, scenario domain.VerificationScenario) (bool, error) {
	s.profileID = profileID
	s.updated = scenario
	s.updateCalls++

	return s.updateFound, s.updateErr
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

// シナリオ一覧ユースケース検証
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
