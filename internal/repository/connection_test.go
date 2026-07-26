package repository

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

// データベース接続確認検証
func TestAppRepositoryCheckConnection(t *testing.T) {
	tests := []struct {
		name   string
		dbType domain.DBType
	}{
		{
			name:   "MySQLのキャンセル済みコンテキストを返す",
			dbType: domain.DBTypeMySQL,
		},
		{
			name:   "PostgreSQLのキャンセル済みコンテキストを返す",
			dbType: domain.DBTypePostgres,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := NewAppRepository(nil).CheckConnection(ctx, connectionTestProfile(t, tt.dbType), "secret")
			if !errors.Is(err, context.Canceled) {
				t.Errorf("CheckConnection() error = %v, want %v", err, context.Canceled)
			}
		})
	}
}

// DB接続文字列生成検証
func TestConnectionDSN(t *testing.T) {
	tests := []struct {
		name       string
		profile    domain.Profile
		wantDriver string
		verify     func(*testing.T, string)
	}{
		{
			name:       "MySQLはTLSなしと3秒タイムアウトを指定する",
			profile:    connectionTestProfile(t, domain.DBTypeMySQL),
			wantDriver: "mysql",
			verify: func(t *testing.T, dsn string) {
				t.Helper()
				if dsn == "" || !containsAll(dsn, []string{"tls=false", "timeout=3s", "readTimeout=3s", "writeTimeout=3s"}) {
					t.Errorf("connectionDSN() dsn = %q, want TLS disabled and 3 second timeouts", dsn)
				}
			},
		},
		{
			name:       "PostgreSQLはTLSなしと3秒タイムアウトを指定する",
			profile:    connectionTestProfile(t, domain.DBTypePostgres),
			wantDriver: "pgx",
			verify: func(t *testing.T, dsn string) {
				t.Helper()
				connectionURL, err := url.Parse(dsn)
				if err != nil {
					t.Fatalf("url.Parse() error = %v", err)
				}
				if got := connectionURL.Query().Get("sslmode"); got != "disable" {
					t.Errorf("sslmode = %q, want %q", got, "disable")
				}
				if got := connectionURL.Query().Get("connect_timeout"); got != "3" {
					t.Errorf("connect_timeout = %q, want %q", got, "3")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, dsn := connectionDSN(tt.profile, "secret")

			if driver != tt.wantDriver {
				t.Errorf("connectionDSN() driver = %q, want %q", driver, tt.wantDriver)
			}
			tt.verify(t, dsn)
		})
	}
}

// 接続文字列用プロファイル生成
func connectionTestProfile(t *testing.T, dbType domain.DBType) domain.Profile {
	t.Helper()

	schema := ""
	if dbType == domain.DBTypePostgres {
		schema = "public"
	}
	profile, err := domain.NewProfile("profile-1", "Local DB", dbType, "localhost", 5432, "app", schema, "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	return profile
}

// 必須部分文字列判定
func containsAll(value string, values []string) bool {
	for _, expected := range values {
		if !strings.Contains(value, expected) {
			return false
		}
	}

	return true
}
