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

// 資格情報保存
func (r *AppRepository) SetCredential(profileID, credential string) error {
	if err := r.credentials.Set(keyringServiceName, profileID, credential); err != nil {
		return apperr.Wrap(apperr.CodeSecureStoreFailed, err)
	}

	return nil
}

// 資格情報削除
func (r *AppRepository) DeleteCredential(profileID string) error {
	if err := r.credentials.Delete(keyringServiceName, profileID); err != nil && !stderrors.Is(err, keyring.ErrNotFound) {
		return apperr.Wrap(apperr.CodeSecureStoreFailed, err)
	}

	return nil
}
