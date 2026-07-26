package repository

import (
	"errors"
	"testing"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	"github.com/zalando/go-keyring"
)

type keyringClientStub struct {
	credential    string
	err           error
	setErr        error
	deleteErr     error
	setService    string
	setID         string
	setValue      string
	deleteService string
	deleteID      string
}

// 資格情報取得再現
func (s *keyringClientStub) Get(_, _ string) (string, error) {
	return s.credential, s.err
}

// 資格情報保存再現
func (s *keyringClientStub) Set(service, id, credential string) error {
	s.setService = service
	s.setID = id
	s.setValue = credential

	return s.setErr
}

// 資格情報削除再現
func (s *keyringClientStub) Delete(service, id string) error {
	s.deleteService = service
	s.deleteID = id

	return s.deleteErr
}

// 資格情報取得検証
func TestAppRepositoryGetCredential(t *testing.T) {
	tests := []struct {
		name           string
		client         keyringClientStub
		wantCredential string
		wantFound      bool
		wantErrorCode  apperr.Code
	}{
		{
			name: "取得できる",
			client: keyringClientStub{
				credential: "secret",
			},
			wantCredential: "secret",
			wantFound:      true,
		},
		{
			name: "未登録",
			client: keyringClientStub{
				err: keyring.ErrNotFound,
			},
		},
		{
			name: "ストア障害",
			client: keyringClientStub{
				err: errors.New("keychain is unavailable"),
			},
			wantErrorCode: apperr.CodeSecureStoreFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.client
			repository := newAppRepository(nil, &client)
			credential, found, err := repository.GetCredential("profile-1")
			if credential != tt.wantCredential {
				t.Errorf("GetCredential() credential = %q, want %q", credential, tt.wantCredential)
			}
			if found != tt.wantFound {
				t.Errorf("GetCredential() found = %v, want %v", found, tt.wantFound)
			}
			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("GetCredential() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
		})
	}
}

// 資格情報保存検証
func TestAppRepositorySetCredential(t *testing.T) {
	tests := []struct {
		name          string
		client        keyringClientStub
		wantErrorCode apperr.Code
	}{
		{
			name: "資格情報を保存する",
		},
		{
			name: "資格情報ストア障害を返す",
			client: keyringClientStub{
				setErr: errors.New("keychain is unavailable"),
			},
			wantErrorCode: apperr.CodeSecureStoreFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.client
			err := newAppRepository(nil, &client).SetCredential("profile-1", "secret")

			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("SetCredential() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if got := client.setService; got != keyringServiceName {
				t.Errorf("SetCredential() service = %q, want %q", got, keyringServiceName)
			}
			if got := client.setID; got != "profile-1" {
				t.Errorf("SetCredential() profile ID = %q, want %q", got, "profile-1")
			}
			if got := client.setValue; got != "secret" {
				t.Errorf("SetCredential() credential = %q, want %q", got, "secret")
			}
		})
	}
}

// 資格情報削除検証
func TestAppRepositoryDeleteCredential(t *testing.T) {
	tests := []struct {
		name          string
		client        keyringClientStub
		wantErrorCode apperr.Code
	}{
		{
			name: "資格情報を削除する",
		},
		{
			name: "未登録の資格情報を削除しても成功する",
			client: keyringClientStub{
				deleteErr: keyring.ErrNotFound,
			},
		},
		{
			name: "資格情報ストア障害を返す",
			client: keyringClientStub{
				deleteErr: errors.New("keychain is unavailable"),
			},
			wantErrorCode: apperr.CodeSecureStoreFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.client
			err := newAppRepository(nil, &client).DeleteCredential("profile-1")

			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("DeleteCredential() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if got := client.deleteService; got != keyringServiceName {
				t.Errorf("DeleteCredential() service = %q, want %q", got, keyringServiceName)
			}
			if got := client.deleteID; got != "profile-1" {
				t.Errorf("DeleteCredential() profile ID = %q, want %q", got, "profile-1")
			}
		})
	}
}

// アプリケーションエラーコード取得
func errorCode(err error) apperr.Code {
	if appErr := apperr.As(err); appErr != nil {
		return appErr.Code
	}

	return ""
}
