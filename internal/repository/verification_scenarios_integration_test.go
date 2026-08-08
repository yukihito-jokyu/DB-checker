//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	"github.com/yukihito-jokyu/DB-checker/test/integration/db"
)

// 外部検証先DDL結合検証
// repository境界の例外: DBMS固有DDLの物理作成・削除はUseCaseから観測できないため実DBで保証する。
func TestVerificationWorkspaceRepositoryDDLIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			profile, password := integrationWorkspaceProfile(t, target)
			workspaceName := fmt.Sprintf("db_checker_v_i%d", time.Now().UnixNano())
			repository := NewVerificationWorkspaceRepository(verificationWorkspaceCredentialsStub{
				password: password,
				found:    true,
			})
			if err := repository.CreateWorkspace(context.Background(), profile, workspaceName); err != nil {
				t.Fatalf("CreateWorkspace() error = %v", err)
			}
			t.Cleanup(func() {
				if err := repository.DeleteWorkspace(context.Background(), profile, workspaceName); err != nil {
					t.Errorf("cleanup DeleteWorkspace() error = %v", err)
				}
			})
			dsn := target.DSN
			if target.Kind == db.MySQL {
				dsn = target.AdminDSN
			}
			database, err := sql.Open(target.DriverName, dsn)
			if err != nil {
				t.Fatalf("sql.Open() error = %v", err)
			}
			defer database.Close()
			if !integrationWorkspaceExists(t, database, target.Kind, workspaceName) {
				t.Fatalf("workspace %q does not exist after create", workspaceName)
			}
			if err := repository.DeleteWorkspace(context.Background(), profile, workspaceName); err != nil {
				t.Fatalf("DeleteWorkspace() error = %v", err)
			}
			if integrationWorkspaceExists(t, database, target.Kind, workspaceName) {
				t.Errorf("workspace %q exists after delete", workspaceName)
			}
		})
	}
}

// 結合検証用プロファイル生成
func integrationWorkspaceProfile(t *testing.T, target db.Target) (domain.Profile, string) {
	t.Helper()
	if target.Kind == db.MySQL {
		config, err := mysql.ParseDSN(target.AdminDSN)
		if err != nil {
			t.Fatalf("mysql.ParseDSN() error = %v", err)
		}
		targetConfig, err := mysql.ParseDSN(target.DSN)
		if err != nil {
			t.Fatalf("mysql.ParseDSN() error = %v", err)
		}
		host, portText, err := splitIntegrationAddress(config.Addr)
		if err != nil {
			t.Fatalf("splitIntegrationAddress() error = %v", err)
		}
		port, err := strconv.Atoi(portText)
		if err != nil {
			t.Fatalf("strconv.Atoi() error = %v", err)
		}
		profile, err := domain.NewProfile("integration-mysql", "integration", domain.DBTypeMySQL, host, port, targetConfig.DBName, "", config.User)
		if err != nil {
			t.Fatalf("NewProfile() error = %v", err)
		}

		return profile, config.Passwd
	}
	parsed, err := url.Parse(target.DSN)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	host, portText, err := splitIntegrationAddress(parsed.Host)
	if err != nil {
		t.Fatalf("splitIntegrationAddress() error = %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("strconv.Atoi() error = %v", err)
	}
	password, _ := parsed.User.Password()
	profile, err := domain.NewProfile("integration-postgres", "integration", domain.DBTypePostgres, host, port, parsed.Path[1:], "public", parsed.User.Username())
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	return profile, password
}

// 接続先アドレス分解
func splitIntegrationAddress(address string) (string, string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", "", err
	}

	return host, port, nil
}

// 外部検証先存在確認
func integrationWorkspaceExists(t *testing.T, database *sql.DB, kind db.Kind, workspaceName string) bool {
	t.Helper()
	var exists bool
	var err error
	if kind == db.MySQL {
		err = database.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = ?)`, workspaceName).Scan(&exists)
	} else {
		err = database.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`, workspaceName).Scan(&exists)
	}
	if err != nil {
		t.Fatalf("workspace existence query error = %v", err)
	}

	return exists
}
