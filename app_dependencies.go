//go:build !e2e

package main

import (
	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
)

// 本番依存生成
func newApplicationDependencies() (*config.Store, usecase.AppRepository, error) {
	configStore, err := config.NewDefaultStore()
	if err != nil {
		return nil, nil, err
	}

	return configStore, repository.NewAppRepository(configStore), nil
}
