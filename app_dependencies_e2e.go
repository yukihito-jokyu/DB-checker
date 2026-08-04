//go:build e2e

package main

import (
	"fmt"
	"os"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
)

const e2eConfigDirEnv = "DB_CHECKER_E2E_CONFIG_DIR"

// E2E設定ストア生成
func newApplicationConfigStore() (*config.Store, error) {
	configDir := os.Getenv(e2eConfigDirEnv)
	if configDir == "" {
		return nil, fmt.Errorf("%s is required", e2eConfigDirEnv)
	}

	return config.NewStore(configDir), nil
}

// E2Eリポジトリ生成
func newApplicationRepository(configStore *config.Store) *repository.AppRepository {
	return repository.NewE2EAppRepository(configStore)
}
