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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

const verificationScenarioTestDriverName = "verification-scenario-test"

var (
	verificationScenarioTestScripts   = map[string]verificationScenarioDatabaseScript{}
	verificationScenarioTestScriptsMu sync.Mutex
)

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

type verificationScenarioTestResult struct {
	rowsAffected    int64
	rowsAffectedErr error
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
	verificationScenarioTestScriptsMu.Lock()
	script, exists := verificationScenarioTestScripts[name]
	verificationScenarioTestScriptsMu.Unlock()
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

// テスト用更新件数取得
func (r verificationScenarioTestResult) RowsAffected() (int64, error) {
	return r.rowsAffected, r.rowsAffectedErr
}

// テスト用最終ID取得
func (verificationScenarioTestResult) LastInsertId() (int64, error) {
	return 0, nil
}

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

// SQLiteシナリオ作成検証
func TestSQLiteVerificationScenarioRepositoryCreateVerificationScenario(t *testing.T) {
	createdAt := mustParseScenarioTime(t, "2026-08-08T11:00:00.123456789Z")
	scenario, err := domain.NewVerificationScenario("scenario-1", "検証", "orders", []byte(`{"z":1,"a":{"id":{"kind":"sequence"}}}`), nil, createdAt, createdAt)
	if err != nil {
		t.Fatalf("NewVerificationScenario() error = %v", err)
	}
	tests := []struct {
		name      string
		profileID string
		wantFound bool
	}{
		{
			name:      "プロファイルに紐付けてJSONと時刻を保存する",
			profileID: "profile-1",
			wantFound: true,
		},
		{
			name:      "別プロファイルからは取得できない",
			profileID: "profile-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())
			if err := repository.Initialize(context.Background()); err != nil {
				t.Fatalf("Initialize() error = %v", err)
			}
			if err := repository.CreateVerificationScenario(context.Background(), "profile-1", scenario); err != nil {
				t.Fatalf("CreateVerificationScenario() error = %v", err)
			}

			got, found, err := repository.GetVerificationScenario(context.Background(), tt.profileID, scenario.ID)
			if err != nil {
				t.Fatalf("GetVerificationScenario() error = %v", err)
			}
			if found != tt.wantFound {
				t.Fatalf("GetVerificationScenario() found = %v, want %v", found, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if !reflect.DeepEqual(got, scenario) {
				t.Errorf("GetVerificationScenario() = %#v, want %#v", got, scenario)
			}
		})
	}
}

// SQLiteシナリオ作成障害検証
func TestSQLiteVerificationScenarioRepositoryCreateVerificationScenarioFailures(t *testing.T) {
	scenario, err := domain.NewVerificationScenario("scenario-1", "検証", "orders", []byte(`{}`), nil, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		t.Fatalf("NewVerificationScenario() error = %v", err)
	}
	cyclicDefinition := map[string]any{}
	cyclicDefinition["cycle"] = cyclicDefinition
	marshalFailureScenario := domain.VerificationScenario{
		ID:           "scenario-1",
		Name:         "検証",
		PrimaryTable: "orders",
		Definition:   cyclicDefinition,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	tests := []struct {
		name     string
		open     func(string, string) (*sql.DB, error)
		scenario domain.VerificationScenario
		want     string
	}{
		{
			name: "データベースオープン失敗",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			scenario: scenario,
			want:     "open verification scenario database",
		},
		{
			name: "定義JSON符号化失敗",
			open: verificationScenarioScriptOpener("create-insert-error", verificationScenarioDatabaseScript{
				exec: func(string) (driver.Result, error) {
					return nil, errors.New("INSERT should not be called")
				},
			}),
			scenario: marshalFailureScenario,
			want:     "encode verification scenario definition",
		},
		{
			name: "INSERT失敗",
			open: verificationScenarioScriptOpener("create-insert-error", verificationScenarioDatabaseScript{
				exec: func(string) (driver.Result, error) {
					return nil, errors.New("insert failed")
				},
			}),
			scenario: scenario,
			want:     "insert verification scenario",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, tt.open)
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())

			err := repository.CreateVerificationScenario(context.Background(), "profile-1", tt.scenario)
			if !containsErrorText(err, tt.want) {
				t.Errorf("CreateVerificationScenario() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// SQLiteシナリオ更新検証
func TestSQLiteVerificationScenarioRepositoryUpdateVerificationScenario(t *testing.T) {
	workspaceName := "verification_orders"
	createdAt := mustParseScenarioTime(t, "2026-08-08T11:00:00Z")
	updatedAt := mustParseScenarioTime(t, "2026-08-08T12:00:00.123456789Z")
	updatedScenario, err := domain.NewVerificationScenario("scenario-1", "更新後", "users", []byte(`{"rowCounts":{"users":20}}`), &workspaceName, createdAt, updatedAt)
	if err != nil {
		t.Fatalf("NewVerificationScenario() error = %v", err)
	}
	tests := []struct {
		name      string
		profileID string
		seed      []verificationScenarioSeed
		wantFound bool
	}{
		{
			name:      "定義と更新日時だけを同一プロファイルで更新する",
			profileID: "profile-1",
			seed: []verificationScenarioSeed{
				{
					id:             "scenario-1",
					profileID:      "profile-1",
					name:           "更新前",
					primaryTable:   "orders",
					definitionJSON: `{"rowCounts":{"orders":10}}`,
					workspaceName:  &workspaceName,
					createdAt:      createdAt.Format(time.RFC3339Nano),
					updatedAt:      createdAt.Format(time.RFC3339Nano),
				},
			},
			wantFound: true,
		},
		{
			name:      "他プロファイルのシナリオを更新しない",
			profileID: "profile-1",
			seed: []verificationScenarioSeed{
				{
					id:             "scenario-1",
					profileID:      "profile-2",
					name:           "更新前",
					primaryTable:   "orders",
					definitionJSON: `{"rowCounts":{"orders":10}}`,
					createdAt:      createdAt.Format(time.RFC3339Nano),
					updatedAt:      createdAt.Format(time.RFC3339Nano),
				},
			},
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

			found, err := repository.UpdateVerificationScenario(context.Background(), tt.profileID, updatedScenario)
			if err != nil {
				t.Fatalf("UpdateVerificationScenario() error = %v", err)
			}
			if found != tt.wantFound {
				t.Fatalf("UpdateVerificationScenario() found = %v, want %v", found, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}

			got, found, err := repository.GetVerificationScenario(context.Background(), tt.profileID, updatedScenario.ID)
			if err != nil {
				t.Fatalf("GetVerificationScenario() error = %v", err)
			}
			if !found {
				t.Fatal("GetVerificationScenario() found = false, want true")
			}
			if !reflect.DeepEqual(got, updatedScenario) {
				t.Errorf("GetVerificationScenario() = %#v, want %#v", got, updatedScenario)
			}
		})
	}
}

// SQLiteシナリオ更新障害検証
func TestSQLiteVerificationScenarioRepositoryUpdateVerificationScenarioFailures(t *testing.T) {
	scenario, err := domain.NewVerificationScenario("scenario-1", "検証", "orders", []byte(`{}`), nil, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		t.Fatalf("NewVerificationScenario() error = %v", err)
	}
	cyclicDefinition := map[string]any{}
	cyclicDefinition["cycle"] = cyclicDefinition
	marshalFailureScenario := scenario
	marshalFailureScenario.Definition = cyclicDefinition
	tests := []struct {
		name     string
		open     func(string, string) (*sql.DB, error)
		scenario domain.VerificationScenario
		want     string
	}{
		{
			name: "データベースオープン失敗",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			scenario: scenario,
			want:     "open verification scenario database",
		},
		{
			name:     "定義JSON符号化失敗",
			open:     verificationScenarioScriptOpener("update-marshal-error", verificationScenarioDatabaseScript{}),
			scenario: marshalFailureScenario,
			want:     "encode verification scenario definition",
		},
		{
			name: "UPDATE失敗",
			open: verificationScenarioScriptOpener("update-exec-error", verificationScenarioDatabaseScript{
				exec: func(string) (driver.Result, error) {
					return nil, errors.New("update failed")
				},
			}),
			scenario: scenario,
			want:     "update verification scenario",
		},
		{
			name: "更新件数取得失敗",
			open: verificationScenarioScriptOpener("update-rows-affected-error", verificationScenarioDatabaseScript{
				exec: func(string) (driver.Result, error) {
					return verificationScenarioTestResult{rowsAffectedErr: errors.New("rows affected failed")}, nil
				},
			}),
			scenario: scenario,
			want:     "count updated verification scenarios",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, tt.open)
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())

			_, err := repository.UpdateVerificationScenario(context.Background(), "profile-1", tt.scenario)
			if !containsErrorText(err, tt.want) {
				t.Errorf("UpdateVerificationScenario() error = %v, want text %q", err, tt.want)
			}
		})
	}
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

// SQLiteシナリオ詳細取得検証
func TestSQLiteVerificationScenarioRepositoryGetVerificationScenario(t *testing.T) {
	updatedAt := "2026-08-08T12:00:00.123456789Z"
	workspaceName := "verification_orders"
	tests := []struct {
		name       string
		seed       []verificationScenarioSeed
		profileID  string
		scenarioID string
		wantFound  bool
		want       domain.VerificationScenario
	}{
		{
			name: "同一プロファイルの詳細を返す",
			seed: []verificationScenarioSeed{
				{
					id:             "scenario-1",
					profileID:      "profile-1",
					name:           "検証",
					primaryTable:   "orders",
					definitionJSON: `{"rowCounts":{"orders":10}}`,
					workspaceName:  &workspaceName,
					createdAt:      "2026-08-08T11:00:00Z",
					updatedAt:      updatedAt,
				},
			},
			profileID:  "profile-1",
			scenarioID: "scenario-1",
			wantFound:  true,
			want: domain.VerificationScenario{
				ID:           "scenario-1",
				Name:         "検証",
				PrimaryTable: "orders",
				Definition: map[string]any{
					"rowCounts": map[string]any{
						"orders": float64(10),
					},
				},
				WorkspaceName: &workspaceName,
				CreatedAt:     mustParseScenarioTime(t, "2026-08-08T11:00:00Z"),
				UpdatedAt:     mustParseScenarioTime(t, updatedAt),
			},
		},
		{
			name: "他プロファイルのシナリオを返さない",
			seed: []verificationScenarioSeed{
				{
					id:           "scenario-1",
					profileID:    "profile-2",
					name:         "検証",
					primaryTable: "orders",
					createdAt:    "2026-08-08T11:00:00Z",
					updatedAt:    updatedAt,
				},
			},
			profileID:  "profile-1",
			scenarioID: "scenario-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())
			for _, seed := range tt.seed {
				seedVerificationScenario(t, repository.databasePath, seed)
			}

			got, found, err := repository.GetVerificationScenario(context.Background(), tt.profileID, tt.scenarioID)
			if err != nil {
				t.Fatalf("GetVerificationScenario() error = %v", err)
			}
			if found != tt.wantFound {
				t.Fatalf("GetVerificationScenario() found = %v, want %v", found, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVerificationScenario() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// SQLiteシナリオ詳細取得障害検証
func TestSQLiteVerificationScenarioRepositoryGetVerificationScenarioFailures(t *testing.T) {
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
			name: "詳細クエリ失敗",
			open: verificationScenarioScriptOpener("get-query-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return nil, errors.New("get query failed")
				},
			}),
			want: "query verification scenario",
		},
		{
			name: "不正な作成日時",
			open: verificationScenarioScriptOpener("get-created-time-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return verificationScenarioDetailRows("scenario-1", "name", "table", `{}`, nil, "invalid", "2026-08-08T12:00:00Z"), nil
				},
			}),
			want: "parse verification scenario created time",
		},
		{
			name: "不正な更新日時",
			open: verificationScenarioScriptOpener("get-updated-time-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return verificationScenarioDetailRows("scenario-1", "name", "table", `{}`, nil, "2026-08-08T11:00:00Z", "invalid"), nil
				},
			}),
			want: "parse verification scenario updated time",
		},
		{
			name: "不正な定義JSON",
			open: verificationScenarioScriptOpener("get-definition-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return verificationScenarioDetailRows("scenario-1", "name", "table", `{`, nil, "2026-08-08T11:00:00Z", "2026-08-08T12:00:00Z"), nil
				},
			}),
			want: "decode verification scenario",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, tt.open)
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())

			_, _, err := repository.GetVerificationScenario(context.Background(), "profile-1", "scenario-1")
			if err == nil {
				t.Fatal("GetVerificationScenario() error = nil, want error")
			}
			if !containsErrorText(err, tt.want) {
				t.Errorf("GetVerificationScenario() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// SQLiteシナリオ削除検証
func TestSQLiteVerificationScenarioRepositoryDeleteVerificationScenario(t *testing.T) {
	tests := []struct {
		name            string
		profileID       string
		workspaceState  *string
		runState        *string
		removeWorkspace bool
		wantFound       bool
		wantRemoved     bool
		wantBusy        bool
	}{
		{
			name:           "停止済みworkspaceを残してシナリオを削除する",
			profileID:      "profile-1",
			workspaceState: stringPointer("inactive"),
			wantFound:      true,
		},
		{
			name:           "使用中workspaceを拒否する",
			profileID:      "profile-1",
			workspaceState: stringPointer("active"),
			wantBusy:       true,
		},
		{
			name:      "実行中runを拒否する",
			profileID: "profile-1",
			runState:  stringPointer("running"),
			wantBusy:  true,
		},
		{
			name:      "他プロファイルのシナリオを削除しない",
			profileID: "profile-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())
			if err := repository.Initialize(context.Background()); err != nil {
				t.Fatalf("Initialize() error = %v", err)
			}
			seedVerificationScenario(t, repository.databasePath, verificationScenarioSeed{
				id:             "scenario-1",
				profileID:      "profile-1",
				name:           "検証",
				primaryTable:   "orders",
				definitionJSON: `{}`,
				updatedAt:      "2026-08-08T12:00:00Z",
			})
			database, err := sql.Open("sqlite", repository.databasePath)
			if err != nil {
				t.Fatalf("sql.Open() error = %v", err)
			}
			defer database.Close()
			if tt.workspaceState != nil {
				if _, err := database.ExecContext(context.Background(), `INSERT INTO verification_workspaces (profile_id, scenario_id, state) VALUES (?, ?, ?)`, "profile-1", "scenario-1", *tt.workspaceState); err != nil {
					t.Fatalf("INSERT verification_workspaces error = %v", err)
				}
			}
			if tt.runState != nil {
				if _, err := database.ExecContext(context.Background(), `INSERT INTO verification_runs (id, profile_id, scenario_id, state) VALUES (?, ?, ?, ?)`, "run-1", "profile-1", "scenario-1", *tt.runState); err != nil {
					t.Fatalf("INSERT verification_runs error = %v", err)
				}
			}

			found, workspaceRemoved, busy, err := repository.DeleteVerificationScenario(context.Background(), tt.profileID, "scenario-1", tt.removeWorkspace)
			if err != nil {
				t.Fatalf("DeleteVerificationScenario() error = %v", err)
			}
			if found != tt.wantFound {
				t.Errorf("DeleteVerificationScenario() found = %v, want %v", found, tt.wantFound)
			}
			if workspaceRemoved != tt.wantRemoved {
				t.Errorf("DeleteVerificationScenario() workspace removed = %v, want %v", workspaceRemoved, tt.wantRemoved)
			}
			if busy != tt.wantBusy {
				t.Errorf("DeleteVerificationScenario() busy = %v, want %v", busy, tt.wantBusy)
			}
		})
	}
}

// SQLiteシナリオ削除失敗検証
func TestSQLiteVerificationScenarioRepositoryDeleteVerificationScenarioFailures(t *testing.T) {
	tests := []struct {
		name string
		open func(string, string) (*sql.DB, error)
		want string
	}{
		{
			name: "データベースオープン失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			want: "open verification scenario database",
		},
		{
			name: "トランザクション開始失敗を返す",
			open: verificationScenarioScriptOpener("delete-begin-error", verificationScenarioDatabaseScript{
				begin: func() (driver.Tx, error) {
					return nil, errors.New("begin failed")
				},
			}),
			want: "begin verification scenario deletion",
		},
		{
			name: "workspace状態確認失敗を返す",
			open: verificationScenarioScriptOpener("delete-workspace-query-error", verificationScenarioDatabaseScript{
				query: func(string) (driver.Rows, error) {
					return nil, errors.New("workspace query failed")
				},
			}),
			want: "check verification workspace state",
		},
		{
			name: "run状態確認失敗を返す",
			open: verificationScenarioScriptOpener("delete-run-query-error", verificationScenarioDatabaseScript{
				query: func(query string) (driver.Rows, error) {
					if containsErrorTextValue(query, "verification_workspaces") {
						return verificationScenarioExistsRows(false), nil
					}

					return nil, errors.New("run query failed")
				},
			}),
			want: "check verification run state",
		},
		{
			name: "専用検証先の物理削除未対応を返す",
			open: verificationScenarioScriptOpener("delete-workspace-removal-unsupported", verificationScenarioDatabaseScript{
				query: verificationScenarioInactiveRows,
			}),
			want: "verification workspace removal must be coordinated by usecase",
		},
		{
			name: "シナリオ削除失敗を返す",
			open: verificationScenarioScriptOpener("delete-scenario-error", verificationScenarioDatabaseScript{
				query: verificationScenarioInactiveRows,
				exec: func(string) (driver.Result, error) {
					return nil, errors.New("delete failed")
				},
			}),
			want: "delete verification scenario",
		},
		{
			name: "削除件数取得失敗を返す",
			open: verificationScenarioScriptOpener("delete-rows-affected-error", verificationScenarioDatabaseScript{
				query: verificationScenarioInactiveRows,
				exec: func(string) (driver.Result, error) {
					return verificationScenarioTestResult{rowsAffectedErr: errors.New("count failed")}, nil
				},
			}),
			want: "count deleted verification scenarios",
		},
		{
			name: "コミット失敗を返す",
			open: verificationScenarioScriptOpener("delete-commit-error", verificationScenarioDatabaseScript{
				query: verificationScenarioInactiveRows,
				begin: func() (driver.Tx, error) {
					return verificationScenarioTestTransaction{commitErr: errors.New("commit failed")}, nil
				},
			}),
			want: "commit verification scenario deletion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, tt.open)
			repository := NewSQLiteVerificationScenarioRepository(t.TempDir())

			_, _, _, err := repository.DeleteVerificationScenario(context.Background(), "profile-1", "scenario-1", tt.name == "専用検証先の物理削除未対応を返す")
			if !containsErrorText(err, tt.want) {
				t.Errorf("DeleteVerificationScenario() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// SQLite専用検証先削除未対応時の状態保持検証
func TestSQLiteVerificationScenarioRepositoryDeleteVerificationScenarioKeepsStateWhenWorkspaceRemovalIsUnsupported(t *testing.T) {
	repository := NewSQLiteVerificationScenarioRepository(t.TempDir())
	if err := repository.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	seedVerificationScenario(t, repository.databasePath, verificationScenarioSeed{
		id:             "scenario-1",
		profileID:      "profile-1",
		name:           "検証",
		primaryTable:   "orders",
		definitionJSON: `{}`,
		updatedAt:      "2026-08-08T12:00:00Z",
	})
	database, err := sql.Open("sqlite", repository.databasePath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer database.Close()
	if _, err := database.ExecContext(context.Background(), `INSERT INTO verification_workspaces (profile_id, scenario_id, state) VALUES (?, ?, ?)`, "profile-1", "scenario-1", "inactive"); err != nil {
		t.Fatalf("INSERT verification_workspaces error = %v", err)
	}

	_, _, _, err = repository.DeleteVerificationScenario(context.Background(), "profile-1", "scenario-1", true)
	if !containsErrorText(err, "verification workspace removal must be coordinated by usecase") {
		t.Errorf("DeleteVerificationScenario() error = %v, want unsupported workspace removal error", err)
	}
	for _, query := range []string{
		`SELECT COUNT(*) FROM scenarios WHERE profile_id = 'profile-1' AND id = 'scenario-1'`,
		`SELECT COUNT(*) FROM verification_workspaces WHERE profile_id = 'profile-1' AND scenario_id = 'scenario-1'`,
	} {
		var count int
		if err := database.QueryRowContext(context.Background(), query).Scan(&count); err != nil {
			t.Fatalf("state count query error = %v", err)
		}
		if count != 1 {
			t.Errorf("state count = %d, want 1", count)
		}
	}
}

// SQLite検証状態永続化検証
func TestSQLiteVerificationScenarioRepositoryVerificationStatePersistence(t *testing.T) {
	tests := []struct {
		name             string
		workspaceState   string
		runState         string
		deleteWorkspace  bool
		updateProfileID  string
		wantFound        bool
		wantUpdated      bool
		wantScenarioBusy bool
		wantRunBusy      bool
	}{
		{
			name:             "ワークスペースを更新して使用中と判定する",
			workspaceState:   "creating",
			runState:         "running",
			updateProfileID:  "profile-1",
			wantFound:        true,
			wantUpdated:      true,
			wantScenarioBusy: true,
			wantRunBusy:      true,
		},
		{
			name:             "ワークスペースを削除して実行状態を更新しない",
			workspaceState:   "inactive",
			deleteWorkspace:  true,
			updateProfileID:  "profile-2",
			wantScenarioBusy: true,
			wantRunBusy:      true,
		},
		{
			name:             "停止済み状態を使用中と判定しない",
			workspaceState:   "inactive",
			runState:         "completed",
			updateProfileID:  "profile-1",
			wantFound:        true,
			wantUpdated:      true,
			wantScenarioBusy: false,
			wantRunBusy:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newInitializedVerificationScenarioRepository(t)
			if err := repository.SaveVerificationWorkspace(context.Background(), "profile-1", "scenario-1", "verification_orders", "creating"); err != nil {
				t.Fatalf("SaveVerificationWorkspace() initial error = %v", err)
			}
			if err := repository.SaveVerificationWorkspace(context.Background(), "profile-1", "scenario-1", "verification_orders_v2", tt.workspaceState); err != nil {
				t.Fatalf("SaveVerificationWorkspace() error = %v", err)
			}
			if tt.deleteWorkspace {
				if err := repository.DeleteVerificationWorkspace(context.Background(), "profile-1", "scenario-1"); err != nil {
					t.Fatalf("DeleteVerificationWorkspace() error = %v", err)
				}
			}
			if err := repository.CreateVerificationRun(context.Background(), "profile-1", "scenario-1", "run-1"); err != nil {
				t.Fatalf("CreateVerificationRun() error = %v", err)
			}

			gotUpdated, err := repository.UpdateVerificationRunState(context.Background(), tt.updateProfileID, "run-1", tt.runState)
			if err != nil {
				t.Fatalf("UpdateVerificationRunState() error = %v", err)
			}
			if gotUpdated != tt.wantUpdated {
				t.Errorf("UpdateVerificationRunState() updated = %v, want %v", gotUpdated, tt.wantUpdated)
			}

			gotWorkspaceState, gotWorkspaceName, gotWorkspaceFound, err := repository.GetVerificationWorkspace(context.Background(), "profile-1", "scenario-1")
			if err != nil {
				t.Fatalf("GetVerificationWorkspace() error = %v", err)
			}
			if gotWorkspaceFound != !tt.deleteWorkspace {
				t.Errorf("GetVerificationWorkspace() found = %v, want %v", gotWorkspaceFound, !tt.deleteWorkspace)
			}
			if gotWorkspaceFound {
				if gotWorkspaceState != tt.workspaceState {
					t.Errorf("GetVerificationWorkspace() state = %q, want %q", gotWorkspaceState, tt.workspaceState)
				}
				if gotWorkspaceName != "verification_orders_v2" {
					t.Errorf("GetVerificationWorkspace() name = %q, want %q", gotWorkspaceName, "verification_orders_v2")
				}
			}

			gotScenarioID, gotRunState, gotRunFound, err := repository.GetVerificationRun(context.Background(), tt.updateProfileID, "run-1")
			if err != nil {
				t.Fatalf("GetVerificationRun() error = %v", err)
			}
			if gotRunFound != tt.wantFound {
				t.Fatalf("GetVerificationRun() found = %v, want %v", gotRunFound, tt.wantFound)
			}
			if gotRunFound {
				if gotScenarioID != "scenario-1" {
					t.Errorf("GetVerificationRun() scenario ID = %q, want %q", gotScenarioID, "scenario-1")
				}
				wantRunState := "prepared"
				if tt.wantUpdated {
					wantRunState = tt.runState
				}
				if gotRunState != wantRunState {
					t.Errorf("GetVerificationRun() state = %q, want %q", gotRunState, wantRunState)
				}
			}

			gotScenarioBusy, err := repository.IsVerificationScenarioBusy(context.Background(), "profile-1", "scenario-1")
			if err != nil {
				t.Fatalf("IsVerificationScenarioBusy() error = %v", err)
			}
			if gotScenarioBusy != tt.wantScenarioBusy {
				t.Errorf("IsVerificationScenarioBusy() = %v, want %v", gotScenarioBusy, tt.wantScenarioBusy)
			}
			gotRunBusy, err := repository.IsVerificationRunBusy(context.Background(), "profile-1", "scenario-1")
			if err != nil {
				t.Fatalf("IsVerificationRunBusy() error = %v", err)
			}
			if gotRunBusy != tt.wantRunBusy {
				t.Errorf("IsVerificationRunBusy() = %v, want %v", gotRunBusy, tt.wantRunBusy)
			}
		})
	}
}

// SQLite検証状態使用中分類検証
func TestSQLiteVerificationScenarioRepositoryVerificationStateBusyClassification(t *testing.T) {
	tests := []struct {
		name             string
		workspaceState   string
		runState         string
		wantScenarioBusy bool
		wantRunBusy      bool
	}{
		{
			name:             "activeワークスペースのみをシナリオ使用中と判定する",
			workspaceState:   "active",
			wantScenarioBusy: true,
		},
		{
			name:             "testワークスペースのみをシナリオ使用中と判定する",
			workspaceState:   "test",
			wantScenarioBusy: true,
		},
		{
			name:             "deletingワークスペースのみをシナリオ使用中と判定する",
			workspaceState:   "deleting",
			wantScenarioBusy: true,
		},
		{
			name:             "prepared実行のみを使用中と判定する",
			runState:         "prepared",
			wantScenarioBusy: true,
			wantRunBusy:      true,
		},
		{
			name:             "canceling実行のみを使用中と判定する",
			runState:         "canceling",
			wantScenarioBusy: true,
			wantRunBusy:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newInitializedVerificationScenarioRepository(t)
			if tt.workspaceState != "" {
				if err := repository.SaveVerificationWorkspace(context.Background(), "profile-1", "scenario-1", "verification_orders", tt.workspaceState); err != nil {
					t.Fatalf("SaveVerificationWorkspace() error = %v", err)
				}
			}
			if tt.runState != "" {
				if err := repository.CreateVerificationRun(context.Background(), "profile-1", "scenario-1", "run-1"); err != nil {
					t.Fatalf("CreateVerificationRun() error = %v", err)
				}
				if tt.runState != "prepared" {
					updated, err := repository.UpdateVerificationRunState(context.Background(), "profile-1", "run-1", tt.runState)
					if err != nil {
						t.Fatalf("UpdateVerificationRunState() error = %v", err)
					}
					if !updated {
						t.Fatal("UpdateVerificationRunState() updated = false, want true")
					}
				}
			}

			gotScenarioBusy, err := repository.IsVerificationScenarioBusy(context.Background(), "profile-1", "scenario-1")
			if err != nil {
				t.Fatalf("IsVerificationScenarioBusy() error = %v", err)
			}
			if gotScenarioBusy != tt.wantScenarioBusy {
				t.Errorf("IsVerificationScenarioBusy() = %v, want %v", gotScenarioBusy, tt.wantScenarioBusy)
			}
			gotRunBusy, err := repository.IsVerificationRunBusy(context.Background(), "profile-1", "scenario-1")
			if err != nil {
				t.Fatalf("IsVerificationRunBusy() error = %v", err)
			}
			if gotRunBusy != tt.wantRunBusy {
				t.Errorf("IsVerificationRunBusy() = %v, want %v", gotRunBusy, tt.wantRunBusy)
			}
		})
	}
}

// SQLite検証状態プロファイル分離検証
func TestSQLiteVerificationScenarioRepositoryVerificationStateProfileIsolation(t *testing.T) {
	tests := []struct {
		name                       string
		profileID                  string
		otherProfileID             string
		scenarioID                 string
		runID                      string
		otherRunID                 string
		profileWorkspaceName       string
		profileWorkspaceState      string
		otherWorkspaceName         string
		otherWorkspaceState        string
		updatedRunState            string
		wantProfileWorkspaceFound  bool
		wantOtherWorkspaceFound    bool
		wantDeletedWorkspaceFound  bool
		wantProfileRunFound        bool
		wantOtherProfileRunFound   bool
		wantProfileScenarioID      string
		wantProfileRunState        string
		wantOtherProfileScenarioID string
		wantOtherProfileRunState   string
		wantProfileScenarioBusy    bool
		wantProfileRunBusy         bool
	}{
		{
			name:                       "同一シナリオIDの別プロファイル状態から分離する",
			profileID:                  "profile-1",
			otherProfileID:             "profile-2",
			scenarioID:                 "scenario-1",
			runID:                      "run-1",
			otherRunID:                 "run-2",
			profileWorkspaceName:       "verification_profile_1",
			profileWorkspaceState:      "inactive",
			otherWorkspaceName:         "verification_profile_2",
			otherWorkspaceState:        "active",
			updatedRunState:            "completed",
			wantProfileWorkspaceFound:  true,
			wantOtherWorkspaceFound:    true,
			wantDeletedWorkspaceFound:  false,
			wantProfileRunFound:        true,
			wantOtherProfileRunFound:   false,
			wantProfileScenarioID:      "scenario-1",
			wantProfileRunState:        "completed",
			wantOtherProfileScenarioID: "",
			wantOtherProfileRunState:   "",
			wantProfileScenarioBusy:    false,
			wantProfileRunBusy:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newInitializedVerificationScenarioRepository(t)
			if err := repository.SaveVerificationWorkspace(context.Background(), tt.profileID, tt.scenarioID, tt.profileWorkspaceName, tt.profileWorkspaceState); err != nil {
				t.Fatalf("SaveVerificationWorkspace() profile-1 error = %v", err)
			}
			if err := repository.SaveVerificationWorkspace(context.Background(), tt.otherProfileID, tt.scenarioID, tt.otherWorkspaceName, tt.otherWorkspaceState); err != nil {
				t.Fatalf("SaveVerificationWorkspace() profile-2 error = %v", err)
			}
			if err := repository.CreateVerificationRun(context.Background(), tt.profileID, tt.scenarioID, tt.runID); err != nil {
				t.Fatalf("CreateVerificationRun() profile-1 error = %v", err)
			}
			updated, err := repository.UpdateVerificationRunState(context.Background(), tt.profileID, tt.runID, tt.updatedRunState)
			if err != nil {
				t.Fatalf("UpdateVerificationRunState() profile-1 error = %v", err)
			}
			if !updated {
				t.Fatal("UpdateVerificationRunState() updated = false, want true")
			}
			if err := repository.CreateVerificationRun(context.Background(), tt.otherProfileID, tt.scenarioID, tt.otherRunID); err != nil {
				t.Fatalf("CreateVerificationRun() profile-2 error = %v", err)
			}

			gotWorkspaceState, gotWorkspaceName, gotWorkspaceFound, err := repository.GetVerificationWorkspace(context.Background(), tt.profileID, tt.scenarioID)
			if err != nil {
				t.Fatalf("GetVerificationWorkspace() profile-1 error = %v", err)
			}
			if gotWorkspaceFound != tt.wantProfileWorkspaceFound {
				t.Fatalf("GetVerificationWorkspace() profile-1 found = %v, want %v", gotWorkspaceFound, tt.wantProfileWorkspaceFound)
			}
			if gotWorkspaceState != tt.profileWorkspaceState {
				t.Errorf("GetVerificationWorkspace() profile-1 state = %q, want %q", gotWorkspaceState, tt.profileWorkspaceState)
			}
			if gotWorkspaceName != tt.profileWorkspaceName {
				t.Errorf("GetVerificationWorkspace() profile-1 name = %q, want %q", gotWorkspaceName, tt.profileWorkspaceName)
			}
			if err := repository.DeleteVerificationWorkspace(context.Background(), tt.profileID, tt.scenarioID); err != nil {
				t.Fatalf("DeleteVerificationWorkspace() profile-1 error = %v", err)
			}

			_, _, gotProfile1WorkspaceFound, err := repository.GetVerificationWorkspace(context.Background(), tt.profileID, tt.scenarioID)
			if err != nil {
				t.Fatalf("GetVerificationWorkspace() deleted profile-1 error = %v", err)
			}
			if gotProfile1WorkspaceFound != tt.wantDeletedWorkspaceFound {
				t.Errorf("GetVerificationWorkspace() deleted profile-1 found = %v, want %v", gotProfile1WorkspaceFound, tt.wantDeletedWorkspaceFound)
			}
			gotProfile2WorkspaceState, gotProfile2WorkspaceName, gotProfile2WorkspaceFound, err := repository.GetVerificationWorkspace(context.Background(), tt.otherProfileID, tt.scenarioID)
			if err != nil {
				t.Fatalf("GetVerificationWorkspace() profile-2 error = %v", err)
			}
			if gotProfile2WorkspaceFound != tt.wantOtherWorkspaceFound {
				t.Fatalf("GetVerificationWorkspace() profile-2 found = %v, want %v", gotProfile2WorkspaceFound, tt.wantOtherWorkspaceFound)
			}
			if gotProfile2WorkspaceState != tt.otherWorkspaceState {
				t.Errorf("GetVerificationWorkspace() profile-2 state = %q, want %q", gotProfile2WorkspaceState, tt.otherWorkspaceState)
			}
			if gotProfile2WorkspaceName != tt.otherWorkspaceName {
				t.Errorf("GetVerificationWorkspace() profile-2 name = %q, want %q", gotProfile2WorkspaceName, tt.otherWorkspaceName)
			}

			gotScenarioID, gotRunState, gotRunFound, err := repository.GetVerificationRun(context.Background(), tt.profileID, tt.runID)
			if err != nil {
				t.Fatalf("GetVerificationRun() profile-1 error = %v", err)
			}
			if gotRunFound != tt.wantProfileRunFound {
				t.Fatalf("GetVerificationRun() profile-1 found = %v, want %v", gotRunFound, tt.wantProfileRunFound)
			}
			if gotScenarioID != tt.wantProfileScenarioID {
				t.Errorf("GetVerificationRun() profile-1 scenario ID = %q, want %q", gotScenarioID, tt.wantProfileScenarioID)
			}
			if gotRunState != tt.wantProfileRunState {
				t.Errorf("GetVerificationRun() profile-1 state = %q, want %q", gotRunState, tt.wantProfileRunState)
			}

			gotOtherScenarioID, gotOtherRunState, gotOtherRunFound, err := repository.GetVerificationRun(context.Background(), tt.otherProfileID, tt.runID)
			if err != nil {
				t.Fatalf("GetVerificationRun() other profile error = %v", err)
			}
			if gotOtherRunFound != tt.wantOtherProfileRunFound {
				t.Fatalf("GetVerificationRun() other profile found = %v, want %v", gotOtherRunFound, tt.wantOtherProfileRunFound)
			}
			if gotOtherScenarioID != tt.wantOtherProfileScenarioID {
				t.Errorf("GetVerificationRun() other profile scenario ID = %q, want %q", gotOtherScenarioID, tt.wantOtherProfileScenarioID)
			}
			if gotOtherRunState != tt.wantOtherProfileRunState {
				t.Errorf("GetVerificationRun() other profile state = %q, want %q", gotOtherRunState, tt.wantOtherProfileRunState)
			}

			gotScenarioBusy, err := repository.IsVerificationScenarioBusy(context.Background(), tt.profileID, tt.scenarioID)
			if err != nil {
				t.Fatalf("IsVerificationScenarioBusy() profile-1 error = %v", err)
			}
			if gotScenarioBusy != tt.wantProfileScenarioBusy {
				t.Errorf("IsVerificationScenarioBusy() profile-1 = %v, want %v", gotScenarioBusy, tt.wantProfileScenarioBusy)
			}
			gotRunBusy, err := repository.IsVerificationRunBusy(context.Background(), tt.profileID, tt.scenarioID)
			if err != nil {
				t.Fatalf("IsVerificationRunBusy() profile-1 error = %v", err)
			}
			if gotRunBusy != tt.wantProfileRunBusy {
				t.Errorf("IsVerificationRunBusy() profile-1 = %v, want %v", gotRunBusy, tt.wantProfileRunBusy)
			}
		})
	}
}

// SQLite検証状態永続化オープン障害検証
func TestSQLiteVerificationScenarioRepositoryVerificationStateOpenFailures(t *testing.T) {
	tests := []struct {
		name string
		call func(*SQLiteVerificationScenarioRepository) error
	}{
		{
			name: "ワークスペース取得",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, _, _, err := repository.GetVerificationWorkspace(context.Background(), "profile-1", "scenario-1")

				return err
			},
		},
		{
			name: "ワークスペース保存",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				return repository.SaveVerificationWorkspace(context.Background(), "profile-1", "scenario-1", "verification_orders", "active")
			},
		},
		{
			name: "ワークスペース削除",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				return repository.DeleteVerificationWorkspace(context.Background(), "profile-1", "scenario-1")
			},
		},
		{
			name: "実行状態作成",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				return repository.CreateVerificationRun(context.Background(), "profile-1", "scenario-1", "run-1")
			},
		},
		{
			name: "実行状態取得",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, _, _, err := repository.GetVerificationRun(context.Background(), "profile-1", "run-1")

				return err
			},
		},
		{
			name: "実行状態更新",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, err := repository.UpdateVerificationRunState(context.Background(), "profile-1", "run-1", "running")

				return err
			},
		},
		{
			name: "シナリオ使用中判定",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, err := repository.IsVerificationScenarioBusy(context.Background(), "profile-1", "scenario-1")

				return err
			},
		},
		{
			name: "実行使用中判定",
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, err := repository.IsVerificationRunBusy(context.Background(), "profile-1", "scenario-1")

				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			})

			err := tt.call(NewSQLiteVerificationScenarioRepository(t.TempDir()))
			if !containsErrorText(err, "open verification scenario database") {
				t.Errorf("operation error = %v, want text %q", err, "open verification scenario database")
			}
		})
	}
}

// SQLite検証状態永続化SQL障害検証
func TestSQLiteVerificationScenarioRepositoryVerificationStateSQLFailures(t *testing.T) {
	tests := []struct {
		name   string
		script verificationScenarioDatabaseScript
		call   func(*SQLiteVerificationScenarioRepository) error
		want   string
	}{
		{
			name: "ワークスペース取得のクエリ失敗",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioQueryFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, _, _, err := repository.GetVerificationWorkspace(context.Background(), "profile-1", "scenario-1")

				return err
			},
			want: "query verification workspace",
		},
		{
			name: "ワークスペース保存の実行失敗",
			script: verificationScenarioDatabaseScript{
				exec: verificationScenarioExecFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				return repository.SaveVerificationWorkspace(context.Background(), "profile-1", "scenario-1", "verification_orders", "active")
			},
			want: "save verification workspace",
		},
		{
			name: "ワークスペース削除の実行失敗",
			script: verificationScenarioDatabaseScript{
				exec: verificationScenarioExecFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				return repository.DeleteVerificationWorkspace(context.Background(), "profile-1", "scenario-1")
			},
			want: "delete verification workspace",
		},
		{
			name: "実行状態作成の実行失敗",
			script: verificationScenarioDatabaseScript{
				exec: verificationScenarioExecFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				return repository.CreateVerificationRun(context.Background(), "profile-1", "scenario-1", "run-1")
			},
			want: "insert verification run",
		},
		{
			name: "実行状態取得のクエリ失敗",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioQueryFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, _, _, err := repository.GetVerificationRun(context.Background(), "profile-1", "run-1")

				return err
			},
			want: "query verification run",
		},
		{
			name: "実行状態更新の実行失敗",
			script: verificationScenarioDatabaseScript{
				exec: verificationScenarioExecFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, err := repository.UpdateVerificationRunState(context.Background(), "profile-1", "run-1", "running")

				return err
			},
			want: "update verification run",
		},
		{
			name: "実行状態更新の更新件数取得失敗",
			script: verificationScenarioDatabaseScript{
				exec: func(string) (driver.Result, error) {
					return verificationScenarioTestResult{rowsAffectedErr: errors.New("rows affected failed")}, nil
				},
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, err := repository.UpdateVerificationRunState(context.Background(), "profile-1", "run-1", "running")

				return err
			},
			want: "count updated verification runs",
		},
		{
			name: "シナリオ使用中判定のクエリ失敗",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioQueryFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, err := repository.IsVerificationScenarioBusy(context.Background(), "profile-1", "scenario-1")

				return err
			},
			want: "check verification scenario busy state",
		},
		{
			name: "実行使用中判定のクエリ失敗",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioQueryFailure,
			},
			call: func(repository *SQLiteVerificationScenarioRepository) error {
				_, err := repository.IsVerificationRunBusy(context.Background(), "profile-1", "scenario-1")

				return err
			},
			want: "check verification run state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withVerificationScenarioOpener(t, verificationScenarioScriptOpener("verification-state-"+tt.name, tt.script))

			err := tt.call(NewSQLiteVerificationScenarioRepository(t.TempDir()))
			if !containsErrorText(err, tt.want) {
				t.Errorf("operation error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// SQLiteシナリオDBバージョン1移行検証
func TestSQLiteVerificationScenarioRepositoryInitializeMigratesVersion1Database(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), verificationScenarioDatabaseName)
	database, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer database.Close()

	if _, err := database.ExecContext(context.Background(), `CREATE TABLE scenarios (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		name TEXT NOT NULL,
		primary_table TEXT NOT NULL,
		definition_json TEXT NOT NULL,
		workspace_name TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("CREATE TABLE scenarios error = %v", err)
	}
	if _, err := database.ExecContext(context.Background(), `INSERT INTO scenarios (id, profile_id, name, primary_table, definition_json, workspace_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "scenario-1", "profile-1", "既存シナリオ", "orders", `{}`, "verification_orders", "2026-08-08T12:00:00Z", "2026-08-08T12:00:00Z"); err != nil {
		t.Fatalf("INSERT scenarios error = %v", err)
	}
	if _, err := database.ExecContext(context.Background(), "PRAGMA user_version = 1"); err != nil {
		t.Fatalf("PRAGMA user_version error = %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("database.Close() error = %v", err)
	}

	repository := NewSQLiteVerificationScenarioRepository(filepath.Dir(databasePath))
	if err := repository.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	database, err = sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer database.Close()
	var version int
	if err := database.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("PRAGMA user_version query error = %v", err)
	}
	if version != 3 {
		t.Errorf("user_version = %d, want 3", version)
	}
	for _, table := range []string{
		"verification_workspaces",
		"verification_runs",
	} {
		var found bool
		if err := database.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?)`, table).Scan(&found); err != nil {
			t.Fatalf("table existence query error = %v", err)
		}
		if !found {
			t.Errorf("table %q = absent, want present", table)
		}
	}
	var name string
	if err := database.QueryRowContext(context.Background(), `SELECT name FROM scenarios WHERE profile_id = ? AND id = ?`, "profile-1", "scenario-1").Scan(&name); err != nil {
		t.Fatalf("existing scenario query error = %v", err)
	}
	if name != "既存シナリオ" {
		t.Errorf("scenario name = %q, want %q", name, "既存シナリオ")
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
				query: verificationScenarioVersionRows(2),
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
			name: "既存バージョン3を維持する",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioVersionRows(3),
			},
		},
		{
			name: "未対応バージョンを拒否する",
			script: verificationScenarioDatabaseScript{
				query: verificationScenarioVersionRows(4),
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
			script: verificationScenarioMigrationExecError("PRAGMA user_version = 3", "version write failed"),
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
	id             string
	profileID      string
	name           string
	primaryTable   string
	definitionJSON string
	workspaceName  *string
	createdAt      string
	updatedAt      string
}

// 初期化済みシナリオリポジトリ生成
func newInitializedVerificationScenarioRepository(t *testing.T) *SQLiteVerificationScenarioRepository {
	t.Helper()
	repository := NewSQLiteVerificationScenarioRepository(t.TempDir())
	if err := repository.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	return repository
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
	definitionJSON := seed.definitionJSON
	if definitionJSON == "" {
		definitionJSON = "{}"
	}
	createdAt := seed.createdAt
	if createdAt == "" {
		createdAt = seed.updatedAt
	}
	if _, err := database.ExecContext(context.Background(), `INSERT INTO scenarios (id, profile_id, name, primary_table, definition_json, workspace_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, seed.id, seed.profileID, seed.name, seed.primaryTable, definitionJSON, seed.workspaceName, createdAt, seed.updatedAt); err != nil {
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
	verificationScenarioTestScriptsMu.Lock()
	verificationScenarioTestScripts[name] = script
	verificationScenarioTestScriptsMu.Unlock()
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

// EXISTS結果行生成
func verificationScenarioExistsRows(exists bool) *verificationScenarioTestRows {
	return &verificationScenarioTestRows{
		columns: []string{"exists"},
		values:  [][]driver.Value{{exists}},
	}
}

// 停止済み状態行生成
func verificationScenarioInactiveRows(query string) (driver.Rows, error) {
	if containsErrorTextValue(query, "verification_workspaces") {
		return verificationScenarioExistsRows(false), nil
	}

	return verificationScenarioExistsRows(false), nil
}

// テスト用クエリ障害
func verificationScenarioQueryFailure(string) (driver.Rows, error) {
	return nil, errors.New("query failed")
}

// テスト用実行障害
func verificationScenarioExecFailure(string) (driver.Result, error) {
	return nil, errors.New("exec failed")
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

// シナリオ詳細行生成
func verificationScenarioDetailRows(id, name, primaryTable, definitionJSON string, workspaceName *string, createdAt, updatedAt string) *verificationScenarioTestRows {
	var workspaceValue driver.Value
	if workspaceName != nil {
		workspaceValue = *workspaceName
	}

	return &verificationScenarioTestRows{
		columns: []string{
			"id",
			"name",
			"primary_table",
			"definition_json",
			"workspace_name",
			"created_at",
			"updated_at",
		},
		values: [][]driver.Value{
			{
				id,
				name,
				primaryTable,
				definitionJSON,
				workspaceValue,
				createdAt,
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

type verificationWorkspaceCredentialsStub struct {
	password        string
	found           bool
	err             error
	credentialCalls *int
}

// 資格情報取得再現
func (s verificationWorkspaceCredentialsStub) GetCredential(string) (string, bool, error) {
	if s.credentialCalls != nil {
		*s.credentialCalls++
	}

	return s.password, s.found, s.err
}

// 外部検証先DDL検証
func TestVerificationWorkspaceRepositoryDDL(t *testing.T) {
	profile := verificationWorkspaceProfile(t, domain.DBTypeMySQL)
	tests := []struct {
		name                  string
		dbType                domain.DBType
		delete                bool
		workspaceName         string
		host                  string
		credentials           verificationWorkspaceCredentialsStub
		open                  func(string, string) (*sql.DB, error)
		context               context.Context
		wantStatement         string
		wantError             string
		wantOpenCalls         int
		wantCredentialSkipped bool
	}{
		{
			name:          "MySQL databaseを作成する",
			dbType:        domain.DBTypeMySQL,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantStatement: "CREATE DATABASE IF NOT EXISTS `db_checker_v_profile_scenario`",
			wantOpenCalls: 1,
		},
		{
			name:          "PostgreSQL schemaを作成する",
			dbType:        domain.DBTypePostgres,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantStatement: `CREATE SCHEMA IF NOT EXISTS "db_checker_v_profile_scenario"`,
			wantOpenCalls: 1,
		},
		{
			name:          "MySQL databaseを削除する",
			dbType:        domain.DBTypeMySQL,
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantStatement: "DROP DATABASE IF EXISTS `db_checker_v_profile_scenario`",
			wantOpenCalls: 1,
		},
		{
			name:          "PostgreSQL schemaを削除する",
			dbType:        domain.DBTypePostgres,
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantStatement: `DROP SCHEMA IF EXISTS "db_checker_v_profile_scenario" CASCADE`,
			wantOpenCalls: 1,
		},
		{
			name:          "IPv4 loopback hostでdatabaseを作成する",
			workspaceName: "db_checker_v_profile_scenario",
			host:          "127.0.0.1",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantStatement: "CREATE DATABASE IF NOT EXISTS `db_checker_v_profile_scenario`",
			wantOpenCalls: 1,
		},
		{
			name:          "IPv6 loopback hostでdatabaseを作成する",
			workspaceName: "db_checker_v_profile_scenario",
			host:          "::1",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantStatement: "CREATE DATABASE IF NOT EXISTS `db_checker_v_profile_scenario`",
			wantOpenCalls: 1,
		},
		{
			name:          "localhostを資格情報取得前に拒否する",
			workspaceName: "db_checker_v_profile_scenario",
			host:          "localhost",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantError:             "verification workspace host is not allowed",
			wantCredentialSkipped: true,
		},
		{
			name:          "localhostの削除を資格情報取得前に拒否する",
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			host:          "localhost",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantError:             "verification workspace host is not allowed",
			wantCredentialSkipped: true,
		},
		{
			name:          "リモートhostを資格情報取得前に拒否する",
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantError:             "verification workspace host is not allowed",
			wantCredentialSkipped: true,
		},
		{
			name:          "リモートhostの削除を資格情報取得前に拒否する",
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantError:             "verification workspace host is not allowed",
			wantCredentialSkipped: true,
		},
		{
			name:          "不正な識別子では接続しない",
			workspaceName: "invalid-name",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantError: "invalid verification workspace identifier",
		},
		{
			name:          "資格情報なしでは作成接続しない",
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				found: false,
			},
			wantError: "verification credential unavailable",
		},
		{
			name:          "資格情報取得失敗を安全に返す",
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				err: errors.New("password=secret"),
			},
			wantError: "load verification credential",
		},
		{
			name:          "接続失敗を安全に返す",
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("dsn=password=secret")
			},
			wantError:     "open verification connection",
			wantOpenCalls: 1,
		},
		{
			name:          "実行失敗を安全に返す",
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			open:          verificationWorkspaceScriptOpener("workspace-exec-error", errors.New("password=secret")),
			wantError:     "create verification workspace",
			wantOpenCalls: 1,
		},
		{
			name:          "削除時に不正な識別子では接続しない",
			delete:        true,
			workspaceName: "invalid-name",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			wantError: "invalid verification workspace identifier",
		},
		{
			name:          "削除時に資格情報なしでは接続しない",
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				found: false,
			},
			wantError: "verification credential unavailable",
		},
		{
			name:          "削除時の資格情報取得失敗を安全に返す",
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				err: errors.New("password=secret"),
			},
			wantError: "load verification credential",
		},
		{
			name:          "削除時の接続失敗を安全に返す",
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("dsn=password=secret")
			},
			wantError:     "open verification connection",
			wantOpenCalls: 1,
		},
		{
			name:          "削除時の実行失敗を安全に返す",
			delete:        true,
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			open:          verificationWorkspaceScriptOpener("workspace-drop-exec-error", errors.New("password=secret")),
			wantError:     "delete verification workspace",
			wantOpenCalls: 1,
		},
		{
			name:          "キャンセル済みcontextを安全に返す",
			workspaceName: "db_checker_v_profile_scenario",
			credentials: verificationWorkspaceCredentialsStub{
				password: "secret",
				found:    true,
			},
			context:       verificationWorkspaceCanceledContext(),
			wantError:     "create verification workspace",
			wantOpenCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statement := ""
			openCalls := 0
			credentialCalls := 0
			credentials := tt.credentials
			credentials.credentialCalls = &credentialCalls
			open := tt.open
			if open == nil {
				open = func(driverName, dsn string) (*sql.DB, error) {
					openCalls++
					if strings.Contains(dsn, "secret") == false {
						t.Errorf("DSN = %q, want credential", dsn)
					}

					return openVerificationScenarioTestDatabase(nil, "workspace-"+tt.name, verificationScenarioDatabaseScript{
						exec: func(query string) (driver.Result, error) {
							statement = query

							return driver.RowsAffected(1), nil
						},
					}), nil
				}
			} else {
				baseOpen := open
				open = func(driverName, dsn string) (*sql.DB, error) {
					openCalls++

					return baseOpen(driverName, dsn)
				}
			}
			repository := &VerificationWorkspaceRepository{
				credentials: credentials,
				open:        open,
			}
			selected := profile
			if tt.dbType != "" {
				selected = verificationWorkspaceProfile(t, tt.dbType)
			}
			if tt.host != "" {
				selected.Host = tt.host
			} else if strings.Contains(tt.name, "リモートhost") {
				selected.Host = "db.example.com"
			}
			ctx := tt.context
			if ctx == nil {
				ctx = context.Background()
			}

			var err error
			if tt.delete {
				err = repository.DeleteWorkspace(ctx, selected, tt.workspaceName)
			} else {
				err = repository.CreateWorkspace(ctx, selected, tt.workspaceName)
			}
			if tt.wantError == "" && err != nil {
				t.Fatalf("workspace DDL error = %v", err)
			}
			if tt.wantError != "" && (err == nil || err.Error() != tt.wantError) {
				t.Errorf("workspace DDL error = %v, want %q", err, tt.wantError)
			}
			if err != nil && strings.Contains(err.Error(), "secret") {
				t.Errorf("workspace DDL error = %q, must not contain credential", err)
			}
			if openCalls != tt.wantOpenCalls {
				t.Errorf("open calls = %d, want %d", openCalls, tt.wantOpenCalls)
			}
			if tt.wantCredentialSkipped && credentialCalls != 0 {
				t.Errorf("credential calls = %d, want 0", credentialCalls)
			}
			if statement != tt.wantStatement {
				t.Errorf("statement = %q, want %q", statement, tt.wantStatement)
			}
		})
	}
}

// 検証先プロファイル生成
func verificationWorkspaceProfile(t *testing.T, dbType domain.DBType) domain.Profile {
	t.Helper()
	schema := "public"
	if dbType == domain.DBTypeMySQL {
		schema = ""
	}
	profile, err := domain.NewProfile("profile-1", "検証", dbType, "127.0.0.1", 3306, "app", schema, "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	return profile
}

// 実行失敗用オープン生成
func verificationWorkspaceScriptOpener(name string, execErr error) func(string, string) (*sql.DB, error) {
	return func(string, string) (*sql.DB, error) {
		return openVerificationScenarioTestDatabase(nil, name, verificationScenarioDatabaseScript{
			exec: func(string) (driver.Result, error) { return nil, execErr },
		}), nil
	}
}

// キャンセル済みコンテキスト生成
func verificationWorkspaceCanceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}
