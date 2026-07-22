package repository

import (
	"errors"
	"testing"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	"github.com/zalando/go-keyring"
)

type keyringClientStub struct {
	credential string
	err        error
}

// 資格情報取得再現
func (s keyringClientStub) Get(_, _ string) (string, error) {
	return s.credential, s.err
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
			repository := newAppRepository(nil, tt.client)
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

// アプリケーションエラーコード取得
func errorCode(err error) apperr.Code {
	if appErr := apperr.As(err); appErr != nil {
		return appErr.Code
	}

	return ""
}
