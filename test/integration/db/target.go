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
	MySQLDSNEnv    = "DB_CHECKER_INTEGRATION_MYSQL_DSN"
	PostgresDSNEnv = "DB_CHECKER_INTEGRATION_POSTGRES_DSN"
)

type Kind string

type Target struct {
	Kind       Kind
	DriverName string
	DSN        string
}

// TargetsFromEnv は結合テストで使う DB 接続先を環境変数から組み立てる。
func TargetsFromEnv() ([]Target, error) {
	mysqlDSN := os.Getenv(MySQLDSNEnv)
	postgresDSN := os.Getenv(PostgresDSNEnv)

	if mysqlDSN == "" {
		return nil, fmt.Errorf("%s is required", MySQLDSNEnv)
	}
	if postgresDSN == "" {
		return nil, fmt.Errorf("%s is required", PostgresDSNEnv)
	}

	return []Target{
		{
			Kind:       MySQL,
			DriverName: "mysql",
			DSN:        mysqlDSN,
		},
		{
			Kind:       Postgres,
			DriverName: "postgres",
			DSN:        postgresDSN,
		},
	}, nil
}
