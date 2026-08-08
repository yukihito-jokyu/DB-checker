package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

const verificationScenarioTestDriverName = "verification-scenario-test"

var verificationScenarioTestScripts = map[string]verificationScenarioDatabaseScript{}

type verificationScenarioDatabaseScript struct {
	query func(string) (driver.Rows, error)
	exec  func(string) (driver.Result, error)
	begin func() (driver.Tx, error)
}

type verificationScenarioTestDriver struct{}

type verificationScenarioTestConnection struct {
	script verificationScenarioDatabaseScript
}

type verificationScenarioTestTransaction struct {
	commitErr error
}

type verificationScenarioTestRows struct {
	columns []string
	values  [][]driver.Value
	nextErr error
	index   int
}

// テスト用SQLiteドライバ登録
func init() {
	sql.Register(verificationScenarioTestDriverName, verificationScenarioTestDriver{})
}

// テスト用SQLite接続生成
func (verificationScenarioTestDriver) Open(name string) (driver.Conn, error) {
	script, exists := verificationScenarioTestScripts[name]
	if !exists {
		return nil, fmt.Errorf("test database script not found: %s", name)
	}

	return verificationScenarioTestConnection{script: script}, nil
}

// テスト用接続終了
func (verificationScenarioTestConnection) Close() error { return nil }

// テスト用接続開始
func (verificationScenarioTestConnection) Begin() (driver.Tx, error) {
	return verificationScenarioTestTransaction{}, nil
}

// テスト用接続実行
func (c verificationScenarioTestConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

// テスト用クエリ実行
func (c verificationScenarioTestConnection) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.script.query == nil {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}

	return c.script.query(query)
}

// テスト用SQL実行
func (c verificationScenarioTestConnection) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if c.script.exec == nil {
		return driver.RowsAffected(1), nil
	}

	return c.script.exec(query)
}

// テスト用トランザクション開始
func (c verificationScenarioTestConnection) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if c.script.begin == nil {
		return verificationScenarioTestTransaction{}, nil
	}

	return c.script.begin()
}

// テスト用トランザクション確定
func (t verificationScenarioTestTransaction) Commit() error { return t.commitErr }

// テスト用トランザクション取消
func (verificationScenarioTestTransaction) Rollback() error { return nil }

// テスト用行列名取得
func (r *verificationScenarioTestRows) Columns() []string { return r.columns }

// テスト用行列終了
func (*verificationScenarioTestRows) Close() error { return nil }

// テスト用行走査
func (r *verificationScenarioTestRows) Next(dest []driver.Value) error {
	if r.index < len(r.values) {
		copy(dest, r.values[r.index])
		r.index++

		return nil
	}
	if r.nextErr != nil {
		return r.nextErr
	}

	return io.EOF
}

// SQLiteシナリオ一覧取得検証
func TestSQLiteVerificationScenarioRepositoryListVerificationScenarios(t *testing.T) {
	tests := []struct {
		name    string
		seed    []verificationScenarioSeed
		want    []domain.VerificationScenarioSummary
		profile string
	}{
		{
			name: "プロファイルで絞り込み更新日時とIDで並べる",
			seed: []verificationScenarioSeed{
				{
					id:           "scenario-2",
					profileID:    "profile-1",
					name:         "新しい",
					primaryTable: "orders",
					updatedAt:    "2026-08-08T12:00:00.123456789Z",
				},
				{
					id:           "scenario-1",
					profileID:    "profile-1",
					name:         "同時刻",
					primaryTable: "users",
					updatedAt:    "2026-08-08T12:00:00.123456789Z",
				},
				{
					id:           "scenario-other",
					profileID:    "profile-2",
					name:         "別プロファイル",
					primaryTable: "payments",
					updatedAt:    "2026-08-09T12:00:00Z",
				},
			},
			profile: "profile-1",
			want: []domain.VerificationScenarioSummary{
				{
					ID:           "scenario-1",
					Name:         "同時刻",
					PrimaryTable: "users",
					UpdatedAt:    mustParseScenarioTime(t, "2026-08-08T12:00:00.123456789Z"),
				},
				{
					ID:           "scenario-2",
					Name:         "新しい",
					PrimaryTable: "orders",
					UpdatedAt:    mustParseScenarioTime(t, "2026-08-08T12:00:00.123456789Z"),
				},
			},
		},
		{
			name:    "空一覧を空スライスで返す",
			profile: "profile-1",
			want:    []domain.VerificationScenarioSummary{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())
			if err := repository.Initialize(context.Background()); err != nil {
				t.Fatalf("Initialize() error = %v", err)
			}
			for _, seed := range tt.seed {
				seedVerificationScenario(t, repository.databasePath, seed)
			}

			got, err := repository.ListVerificationScenarios(context.Background(), tt.profile)
			if err != nil {
				t.Fatalf("ListVerificationScenarios() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListVerificationScenarios() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// SQLiteシナリオ一覧障害検証
func TestSQLiteVerificationScenarioRepositoryListVerificationScenariosFailures(t *testing.T) {
	tests := []struct {
		name string
		open func(string, string) (*sql.DB, error)
		want string
	}{
		{
			name: "データベースオープン失敗",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			want: "open verification scenario database",
		},
		{
			name: "一覧クエリ失敗",
			open: verificationScenarioScriptOpener("list-query-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return nil, errors.New("list query failed")
				},
			}),
			want: "query verification scenarios",
		},
		{
			name: "行走査失敗",
			open: verificationScenarioScriptOpener("scan-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return &verificationScenarioTestRows{
						columns: []string{
							"id",
							"name",
							"primary_table",
							"updated_at",
						},
						values: [][]driver.Value{
							{
								nil,
								"name",
								"table",
								"2026-08-08T12:00:00Z",
							},
						},
					}, nil
				},
			}),
			want: "scan verification scenario",
		},
		{
			name: "不正な更新日時",
			open: verificationScenarioScriptOpener("time-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return verificationScenarioRows("scenario-1", "name", "table", "invalid"), nil
				},
			}),
			want: "parse verification scenario updated time",
		},
		{
			name: "行反復失敗",
			open: verificationScenarioScriptOpener("rows-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					rows := verificationScenarioRows("scenario-1", "name", "table", "2026-08-08T12:00:00Z")
					rows.nextErr = errors.New("rows failed")

					return rows, nil
				},
			}),
			want: "iterate verification scenarios",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, tt.open)
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())

			_, err := repository.ListVerificationScenarios(context.Background(), "profile-1")
			if err == nil {
				t.Fatal("ListVerificationScenarios() error = nil, want error")
			}
			if !containsErrorText(err, tt.want) {
				t.Errorf("ListVerificationScenarios() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// SQLiteシナリオDB初期化検証
func TestSQLiteVerificationScenarioRepositoryInitialize(t *testing.T) {
	tests := []struct {
		name string
		open func(string, string) (*sql.DB, error)
		want string
	}{
		{
			name: "初期化に成功する",
			open: verificationScenarioScriptOpener("initialize-success", verificationScenarioDatabaseScript{
				query: verificationScenarioVersionRows(1),
			}),
		},
		{
			name: "データベースオープン失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			want: "open verification scenario database",
		},
		{
			name: "マイグレーション読込失敗を返す",
			open: verificationScenarioScriptOpener("initialize-migration-query-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return nil, errors.New("version read failed")
				},
			}),
			want: "read verification scenario schema version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, tt.open)
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())

			err := repository.Initialize(context.Background())
			if tt.want == "" {
				if err != nil {
					t.Errorf("Initialize() error = %v, want nil", err)
				}

				return
			}
			if !containsErrorText(err, tt.want) {
				t.Errorf("Initialize() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// 並行一覧取得時migration非実行検証
func TestSQLiteVerificationScenarioRepositoryListVerificationScenariosDoesNotMigrateConcurrently(t *testing.T) {
	var execCalls atomic.Int32
	withVerificationScenarioOpener(t, verificationScenarioScriptOpener("concurrent-list", verificationScenarioDatabaseScript{
		query: func(string) (driver.Rows, error) {
			return &verificationScenarioTestRows{
				columns: []string{
					"id",
					"name",
					"primary_table",
					"updated_at",
				},
			}, nil
		},
		exec: func(string) (driver.Result, error) {
			execCalls.Add(1)

			return driver.RowsAffected(1), nil
		},
	}))
	repository := NewSQLiteVerificationScenarioRepository(t.TempDir())
	errors := make(chan error, 2)

	for range 2 {
		go func() {
			_, err := repository.ListVerificationScenarios(context.Background(), "profile-1")
			errors <- err
		}()
	}
	for range 2 {
		if err := <-errors; err != nil {
			t.Errorf("ListVerificationScenarios() error = %v, want nil", err)
		}
	}
	if got := execCalls.Load(); got != 0 {
		t.Errorf("migration SQL execution count = %d, want 0", got)
	}
}

// シナリオDBマイグレーション分岐検証
func TestMigrateVerificationScenarioDatabase(t *testing.T) {
	tests := []struct {
		name   string
		script verificationScenarioDatabaseScript
		want   string
	}{
		{
			name: "既存バージョン1を維持する",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioVersionRows(1),
			},
		},
		{
			name: "未対応バージョンを拒否する",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioVersionRows(2),
			},
			want: "unsupported verification scenario schema version",
		},
		{
			name: "トランザクション開始失敗",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioVersionRows(0),
				begin: func() (driver.Tx, error) {
					return nil, errors.New("begin failed")
				},
			},
			want: "begin verification scenario migration",
		},
		{
			name:   "テーブル作成失敗",
			script: verificationScenarioMigrationExecError("CREATE TABLE", "create failed"),
			want:   "create verification scenarios table",
		},
		{
			name:   "インデックス作成失敗",
			script: verificationScenarioMigrationExecError("CREATE INDEX", "index failed"),
			want:   "create verification scenario index",
		},
		{
			name:   "バージョン保存失敗",
			script: verificationScenarioMigrationExecError("PRAGMA user_version = 1", "version write failed"),
			want:   "write verification scenario schema version",
		},
		{
			name: "コミット失敗",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioVersionRows(0),
				begin: func() (driver.Tx, error) {
					return verificationScenarioTestTransaction{commitErr: errors.New("commit failed")}, nil
				},
			},
			want: "commit verification scenario migration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database := openVerificationScenarioTestDatabase(t, tt.name, tt.script)
			defer database.Close()

			err := migrateVerificationScenarioDatabase(context.Background(), database)
			if tt.want == "" {
				if err != nil {
					t.Errorf("migrateVerificationScenarioDatabase() error = %v, want nil", err)
				}

				return
			}
			if !containsErrorText(err, tt.want) {
				t.Errorf("migrateVerificationScenarioDatabase() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// SQLiteシナリオDBパス検証
func TestNewSQLiteVerificationScenarioRepository(t *testing.T) {
	tests := []struct {
		name      string
		directory string
	}{
		{
			name:      "設定ディレクトリ配下へ配置する",
			directory: t.TempDir(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := NewSQLiteVerificationScenarioRepository(tt.directory)
			want := filepath.Join(tt.directory, verificationScenarioDatabaseName)
			if repository.databasePath != want {
				t.Errorf("database path = %q, want %q", repository.databasePath, want)
			}
		})
	}
}

type verificationScenarioSeed struct {
	id           string
	profileID    string
	name         string
	primaryTable string
	updatedAt    string
}

// シナリオDBシード
func seedVerificationScenario(t *testing.T, databasePath string, seed verificationScenarioSeed) {
	t.Helper()
	repository := &SQLiteVerificationScenarioRepository{databasePath: databasePath}
	if err := repository.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer database.Close()
	if _, err := database.ExecContext(context.Background(), `INSERT INTO scenarios (id, profile_id, name, primary_table, definition_json, created_at, updated_at) VALUES (?, ?, ?, ?, '{}', ?, ?)`, seed.id, seed.profileID, seed.name, seed.primaryTable, seed.updatedAt, seed.updatedAt); err != nil {
		t.Fatalf("INSERT scenarios error = %v", err)
	}
}

// シナリオ時刻変換
func mustParseScenarioTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}

	return parsed
}

// テスト用DBオープン差替
func withVerificationScenarioOpener(t *testing.T, opener func(string, string) (*sql.DB, error)) {
	t.Helper()
	previous := openVerificationScenarioDatabase
	openVerificationScenarioDatabase = opener
	t.Cleanup(func() { openVerificationScenarioDatabase = previous })
}

// テスト用スクリプトオープン生成
func verificationScenarioScriptOpener(name string, script verificationScenarioDatabaseScript) func(string, string) (*sql.DB, error) {
	return func(string, string) (*sql.DB, error) {
		return openVerificationScenarioTestDatabase(nil, name, script), nil
	}
}

// テスト用DB生成
func openVerificationScenarioTestDatabase(t *testing.T, name string, script verificationScenarioDatabaseScript) *sql.DB {
	if t != nil {
		t.Helper()
	}
	verificationScenarioTestScripts[name] = script
	database, err := sql.Open(verificationScenarioTestDriverName, name)
	if err != nil {
		if t != nil {
			t.Fatalf("sql.Open() error = %v", err)
		}

		panic(err)
	}

	return database
}

// バージョン行生成
func verificationScenarioVersionRows(version int64) func(string) (driver.Rows, error) {
	return func(string) (driver.Rows, error) {
		return &verificationScenarioTestRows{
			columns: []string{"user_version"},
			values:  [][]driver.Value{{version}},
		}, nil
	}
}

// シナリオ行生成
func verificationScenarioRows(id, name, primaryTable, updatedAt string) *verificationScenarioTestRows {
	return &verificationScenarioTestRows{
		columns: []string{
			"id",
			"name",
			"primary_table",
			"updated_at",
		},
		values: [][]driver.Value{
			{
				id,
				name,
				primaryTable,
				updatedAt,
			},
		},
	}
}

// マイグレーションSQL障害生成
func verificationScenarioMigrationExecError(target, message string) verificationScenarioDatabaseScript {
	return verificationScenarioDatabaseScript{
		query: verificationScenarioVersionRows(0),
		exec: func(query string) (driver.Result, error) {
			if query == target || containsErrorText(errors.New(query), target) {
				return nil, errors.New(message)
			}

			return driver.RowsAffected(1), nil
		},
	}
}

// エラー文言包含判定
func containsErrorText(err error, want string) bool {
	return err != nil && want != "" && containsErrorTextValue(err.Error(), want)
}

// 文字列包含判定
func containsErrorTextValue(value, want string) bool {
	return len(want) <= len(value) && (value == want || containsSubstring(value, want))
}

// 部分文字列判定
func containsSubstring(value, want string) bool {
	for index := 0; index+len(want) <= len(value); index++ {
		if value[index:index+len(want)] == want {
			return true
		}
	}

	return false
}
