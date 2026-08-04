//go:build !e2e

package main

import (
	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
)

// 本番設定ストア生成
func newApplicationConfigStore() (*config.Store, error) {
	return config.NewDefaultStore()
}

// 本番リポジトリ生成
func newApplicationRepository(configStore *config.Store) *repository.AppRepository {
	return repository.NewAppRepository(configStore)
}
