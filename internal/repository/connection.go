package repository

import (
	"context"
	"database/sql"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

const connectionTimeout = 3 * time.Second

type connectionChecker interface {
	Check(context.Context, domain.Profile, string) error
}

type databaseConnectionChecker struct{}

// データベース接続確認
func (r *AppRepository) CheckConnection(ctx context.Context, profile domain.Profile, password string) error {
	return r.connectionChecker.Check(ctx, profile, password)
}

// 実DB接続確認
func (databaseConnectionChecker) Check(ctx context.Context, profile domain.Profile, password string) error {
	driverName, dsn := connectionDSN(profile, password)
	database, err := sql.Open(driverName, dsn)
	if err != nil {
		// 単体テスト到達不可: 登録済みドライバーと有効なDSNを固定で生成するため。
		return err
	}
	defer database.Close()

	timeoutContext, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	return database.PingContext(timeoutContext)
}

// DB接続文字列生成
func connectionDSN(profile domain.Profile, password string) (string, string) {
	if profile.DBType == domain.DBTypeMySQL {
		config := mysql.NewConfig()
		config.User = profile.User
		config.Passwd = password
		config.Net = "tcp"
		config.Addr = net.JoinHostPort(profile.Host, strconv.Itoa(profile.Port))
		config.DBName = profile.Database
		config.TLSConfig = "false"
		config.ParseTime = true
		config.Loc = time.UTC
		config.Timeout = connectionTimeout
		config.ReadTimeout = connectionTimeout
		config.WriteTimeout = connectionTimeout

		return "mysql", config.FormatDSN()
	}

	connectionURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(profile.User, password),
		Host:   net.JoinHostPort(profile.Host, strconv.Itoa(profile.Port)),
		Path:   profile.Database,
	}
	query := connectionURL.Query()
	query.Set("sslmode", "disable")
	query.Set("connect_timeout", strconv.Itoa(int(connectionTimeout.Seconds())))
	connectionURL.RawQuery = query.Encode()

	return "pgx", connectionURL.String()
}
