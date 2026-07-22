package repository

import (
	stderrors "errors"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	"github.com/zalando/go-keyring"
)

const keyringServiceName = "DB-checker"

// 資格情報取得
func (r *AppRepository) GetCredential(profileID string) (string, bool, error) {
	credential, err := r.credentials.Get(keyringServiceName, profileID)
	if err == nil {
		return credential, true, nil
	}

	if stderrors.Is(err, keyring.ErrNotFound) {
		return "", false, nil
	}

	return "", false, apperr.Wrap(apperr.CodeSecureStoreFailed, err)
}
