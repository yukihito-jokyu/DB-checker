//go:build e2e

package repository

import (
	"context"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

const e2ePlaceholderCredential = "e2e-placeholder"

// E2E用リポジトリ生成
func NewE2EAppRepository(store *config.Store) *AppRepository {
	return newAppRepository(
		store,
		&e2eCredentialStore{values: make(map[string]string)},
		e2eConnectionChecker{},
	)
}

type e2eCredentialStore struct {
	values map[string]string
}

// E2E資格情報取得
func (s *e2eCredentialStore) Get(_, user string) (string, error) {
	credential, found := s.values[user]
	if !found {
		return e2ePlaceholderCredential, nil
	}

	return credential, nil
}

// E2E資格情報保存
func (s *e2eCredentialStore) Set(_, user, password string) error {
	s.values[user] = password

	return nil
}

// E2E資格情報削除
func (s *e2eCredentialStore) Delete(_, user string) error {
	delete(s.values, user)

	return nil
}

type e2eConnectionChecker struct{}

// E2E接続確認
func (e2eConnectionChecker) Check(context.Context, domain.Profile, string) error {
	return nil
}
