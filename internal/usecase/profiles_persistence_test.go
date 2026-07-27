package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
)

type persistenceDeleteRepository struct {
	*repository.AppRepository
	credentialDeleteErr error
	credentialDeleteIDs []string
	connectionCalls     int
}

// 永続化検証用資格情報削除
func (r *persistenceDeleteRepository) DeleteCredential(profileID string) error {
	r.credentialDeleteIDs = append(r.credentialDeleteIDs, profileID)

	return r.credentialDeleteErr
}

// 永続化検証用接続確認
func (r *persistenceDeleteRepository) CheckConnection(context.Context, domain.Profile, string) error {
	r.connectionCalls++

	return nil
}

// 接続プロファイル削除の設定永続化
func TestAppUseCaseDeleteConnectionProfilePersistence(t *testing.T) {
	first := newSaveTestProfile(t, "profile-1")
	second := newSaveTestProfile(t, "profile-2")
	tests := []struct {
		name                 string
		profileID            string
		credentialDeleteErr  error
		wantErrorCode        apperr.Code
		wantReturnedProfiles []domain.Profile
		wantReturnedActiveID *string
		wantProfiles         []domain.Profile
		wantActiveID         *string
	}{
		{
			name:      "アクティブプロファイル削除後の設定JSONを永続化する",
			profileID: first.ID,
			wantReturnedProfiles: []domain.Profile{
				second,
			},
			wantProfiles: []domain.Profile{
				second,
			},
		},
		{
			name:      "非アクティブプロファイル削除後にアクティブIDを永続化する",
			profileID: second.ID,
			wantReturnedProfiles: []domain.Profile{
				first,
			},
			wantReturnedActiveID: stringPointer("profile-1"),
			wantProfiles: []domain.Profile{
				first,
			},
			wantActiveID: stringPointer("profile-1"),
		},
		{
			name:                "資格情報削除失敗時に設定JSONとアクティブIDを復旧する",
			profileID:           first.ID,
			credentialDeleteErr: errors.New("credential deletion failed"),
			wantErrorCode:       apperr.CodeCredentialDeleteFailed,
			wantProfiles: []domain.Profile{
				first,
				second,
			},
			wantActiveID: stringPointer("profile-1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := config.NewStore(t.TempDir())
			if err := store.Initialize(); err != nil {
				t.Fatalf("Store.Initialize() error = %v", err)
			}
			persistedRepository := repository.NewAppRepository(store)
			initialActiveID := first.ID
			if err := persistedRepository.SaveProfiles([]domain.Profile{first, second}, &initialActiveID); err != nil {
				t.Fatalf("SaveProfiles() initial state error = %v", err)
			}
			deleteRepository := &persistenceDeleteRepository{
				AppRepository:       persistedRepository,
				credentialDeleteErr: tt.credentialDeleteErr,
			}

			profiles, activeID, err := NewAppUseCase(deleteRepository).DeleteConnectionProfile(tt.profileID)

			if gotCode := saveErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("DeleteConnectionProfile() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode == "" && err != nil {
				t.Fatalf("DeleteConnectionProfile() error = %v", err)
			}
			if got := deleteRepository.credentialDeleteIDs; !reflect.DeepEqual(got, []string{tt.profileID}) {
				t.Errorf("DeleteCredential() profile IDs = %#v, want %#v", got, []string{tt.profileID})
			}
			if got := deleteRepository.connectionCalls; got != 0 {
				t.Errorf("CheckConnection() calls = %d, want 0", got)
			}
			if tt.wantErrorCode != "" {
				if profiles != nil {
					t.Errorf("DeleteConnectionProfile() profiles = %#v, want nil", profiles)
				}
				if activeID != nil {
					t.Errorf("DeleteConnectionProfile() active ID = %#v, want nil", activeID)
				}
			} else {
				if !reflect.DeepEqual(profiles, tt.wantReturnedProfiles) {
					t.Errorf("DeleteConnectionProfile() profiles = %#v, want %#v", profiles, tt.wantReturnedProfiles)
				}
				if !reflect.DeepEqual(activeID, tt.wantReturnedActiveID) {
					t.Errorf("DeleteConnectionProfile() active ID = %#v, want %#v", activeID, tt.wantReturnedActiveID)
				}
			}

			loadedProfiles, loadedActiveID, err := repository.NewAppRepository(store).LoadProfiles()
			if err != nil {
				t.Fatalf("LoadProfiles() persisted state error = %v", err)
			}
			if !reflect.DeepEqual(loadedProfiles, tt.wantProfiles) {
				t.Errorf("LoadProfiles() profiles = %#v, want %#v", loadedProfiles, tt.wantProfiles)
			}
			if !reflect.DeepEqual(loadedActiveID, tt.wantActiveID) {
				t.Errorf("LoadProfiles() active ID = %#v, want %#v", loadedActiveID, tt.wantActiveID)
			}
		})
	}
}
