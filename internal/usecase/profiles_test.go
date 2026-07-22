package usecase

import (
	"errors"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type connectionProfileRepositoryStub struct {
	profiles []domain.Profile
	activeID *string
	err      error
}

// 接続プロファイル読込再現
func (s connectionProfileRepositoryStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return s.profiles, s.activeID, s.err
}

// 接続プロファイル読込
func TestAppUseCaseLoadProfiles(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local DB", domain.DBTypePostgres, "localhost", 5432, "app", "public", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	repositoryErr := errors.New("repository error")
	tests := []struct {
		name          string
		repository    connectionProfileRepositoryStub
		wantProfiles  []domain.Profile
		wantFound     bool
		wantCause     error
		wantErrorCode apperr.Code
	}{
		{
			name: "プロファイルを返す",
			repository: connectionProfileRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer("profile-1"),
			},
			wantProfiles: []domain.Profile{profile},
		},
		{
			name: "リポジトリエラーを返す",
			repository: connectionProfileRepositoryStub{
				err: repositoryErr,
			},
			wantFound: true,
			wantCause: repositoryErr,
		},
		{
			name: "存在しないアクティブIDを設定エラーとして返す",
			repository: connectionProfileRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer("missing"),
			},
			wantFound:     true,
			wantErrorCode: apperr.CodeConfigBroken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := NewAppUseCase(tt.repository, nil)

			gotProfiles, err := useCase.LoadProfiles()

			if !reflect.DeepEqual(gotProfiles, tt.wantProfiles) {
				t.Errorf("LoadProfiles() profiles = %#v, want %#v", gotProfiles, tt.wantProfiles)
			}
			if gotFound := err != nil; gotFound != tt.wantFound {
				t.Fatalf("LoadProfiles() error found = %v, want %v", gotFound, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if tt.wantErrorCode != "" && !apperr.IsCode(err, tt.wantErrorCode) {
				t.Errorf("LoadProfiles() error = %v, want code %v", err, tt.wantErrorCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("LoadProfiles() error = %v, want cause %v", err, tt.wantCause)
			}
		})
	}
}

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}
