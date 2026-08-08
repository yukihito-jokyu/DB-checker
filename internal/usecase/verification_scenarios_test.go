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
	scenarios []domain.VerificationScenarioSummary
	err       error
	profileID string
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
