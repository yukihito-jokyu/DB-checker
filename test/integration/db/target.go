package db

import (
	"fmt"
	"os"
)

const (
	MySQL    Kind = "mysql"
	Postgres Kind = "postgres"
)

const (
	MySQLDSNEnv      = "DB_CHECKER_INTEGRATION_MYSQL_DSN"
	MySQLAdminDSNEnv = "DB_CHECKER_INTEGRATION_MYSQL_ADMIN_DSN"
	PostgresDSNEnv   = "DB_CHECKER_INTEGRATION_POSTGRES_DSN"
)

type Kind string

type Target struct {
	Kind       Kind
	DriverName string
	DSN        string
	AdminDSN   string
}

// 結合テスト接続先生成
func TargetsFromEnv() ([]Target, error) {
	mysqlDSN := os.Getenv(MySQLDSNEnv)
	mysqlAdminDSN := os.Getenv(MySQLAdminDSNEnv)
	postgresDSN := os.Getenv(PostgresDSNEnv)

	if mysqlDSN == "" {
		return nil, fmt.Errorf("%s is required", MySQLDSNEnv)
	}
	if mysqlAdminDSN == "" {
		return nil, fmt.Errorf("%s is required", MySQLAdminDSNEnv)
	}
	if postgresDSN == "" {
		return nil, fmt.Errorf("%s is required", PostgresDSNEnv)
	}

	return []Target{
		{
			Kind:       MySQL,
			DriverName: "mysql",
			DSN:        mysqlDSN,
			AdminDSN:   mysqlAdminDSN,
		},
		{
			Kind:       Postgres,
			DriverName: "pgx",
			DSN:        postgresDSN,
		},
	}, nil
}
