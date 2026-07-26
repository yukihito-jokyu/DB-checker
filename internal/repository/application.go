package repository

import (
	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/zalando/go-keyring"
)

// アプリケーションリポジトリ
type AppRepository struct {
	store       *config.Store
	credentials credentialStore
}

// アプリケーションリポジトリ生成
func NewAppRepository(store *config.Store) *AppRepository {
	return newAppRepository(store, systemCredentialStore{})
}

// テスト用リポジトリ生成
func newAppRepository(store *config.Store, credentials credentialStore) *AppRepository {
	return &AppRepository{
		store:       store,
		credentials: credentials,
	}
}

type credentialStore interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type systemCredentialStore struct{}

// OS資格情報取得
func (systemCredentialStore) Get(service, user string) (string, error) {
	// 単体テスト到達不可: OS の資格情報ストアへ直接アクセスする外部境界のため。
	return keyring.Get(service, user)
}

// OS資格情報保存
func (systemCredentialStore) Set(service, user, password string) error {
	// 単体テスト到達不可: OS の資格情報ストアへ直接アクセスする外部境界のため。
	return keyring.Set(service, user, password)
}

// OS資格情報削除
func (systemCredentialStore) Delete(service, user string) error {
	// 単体テスト到達不可: OS の資格情報ストアへ直接アクセスする外部境界のため。
	return keyring.Delete(service, user)
}
