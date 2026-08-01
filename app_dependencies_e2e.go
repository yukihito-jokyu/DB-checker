//go:build e2e

package main

import (
	"fmt"
	"os"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
)

const e2eConfigDirEnv = "DB_CHECKER_E2E_CONFIG_DIR"

// E2E依存生成
func newApplicationDependencies() (*config.Store, usecase.AppRepository, error) {
	configDir := os.Getenv(e2eConfigDirEnv)
	if configDir == "" {
		return nil, nil, fmt.Errorf("%s is required", e2eConfigDirEnv)
	}

	configStore := config.NewStore(configDir)

	return configStore, repository.NewE2EAppRepository(configStore), nil
}
