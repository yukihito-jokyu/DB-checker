package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

const inspectionTestDriverName = "repository-inspection-test"

var (
	inspectionTestScenarios   = map[string]*inspectionTestScenario{}
	inspectionTestScenariosMu sync.Mutex
	inspectionTestSequence    uint64
)

func init() {
	sql.Register(inspectionTestDriverName, inspectionTestDriver{})
}

type inspectionTestDriver struct{}

// テスト用接続生成
func (inspectionTestDriver) Open(name string) (driver.Conn, error) {
	inspectionTestScenariosMu.Lock()
	defer inspectionTestScenariosMu.Unlock()

	scenario := inspectionTestScenarios[name]
	if scenario == nil {
		return nil, fmt.Errorf("unknown inspection test scenario: %s", name)
	}

	return &inspectionTestConnection{scenario: scenario}, nil
}

type inspectionTestConnection struct {
	scenario *inspectionTestScenario
}

// テスト用接続終了
func (*inspectionTestConnection) Close() error { return nil }

// テスト用トランザクション開始
func (*inspectionTestConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

// テスト用文準備
func (*inspectionTestConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements are not supported")
}

// テスト用問い合わせ実行
func (c *inspectionTestConnection) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.scenario.mu.Lock()
	defer c.scenario.mu.Unlock()

	if len(c.scenario.responses) == 0 {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}

	response := c.scenario.responses[0]
	c.scenario.responses = c.scenario.responses[1:]
	if response.query != query {
		return nil, fmt.Errorf("query = %q, want %q", query, response.query)
	}
	if response.err != nil {
		return nil, response.err
	}
	if response.after != nil {
		response.after()
	}

	return &inspectionTestRows{response: response, errorAt: response.errorAt}, nil
}

// テスト用更新実行
func (c *inspectionTestConnection) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.scenario.mu.Lock()
	defer c.scenario.mu.Unlock()

	if len(c.scenario.responses) == 0 {
		return nil, fmt.Errorf("unexpected exec: %s", query)
	}

	response := c.scenario.responses[0]
	c.scenario.responses = c.scenario.responses[1:]
	if response.query != query {
		return nil, fmt.Errorf("query = %q, want %q", query, response.query)
	}
	if response.args != nil && !reflect.DeepEqual(args, response.args) {
		return nil, fmt.Errorf("args = %#v, want %#v", args, response.args)
	}
	if response.err != nil {
		return nil, response.err
	}
	if response.after != nil {
		response.after()
	}

	return inspectionTestResult{affected: response.execAffected, err: response.rowsAffectedErr}, nil
}

// テスト用接続確認
func (*inspectionTestConnection) Ping(context.Context) error { return nil }

type inspectionTestScenario struct {
	mu        sync.Mutex
	responses []inspectionTestResponse
}

type inspectionTestResponse struct {
	query           string
	columns         []string
	values          [][]driver.Value
	execAffected    int64
	rowsAffectedErr error
	err             error
	rowErr          error
	errorAt         int
	after           func()
	args            []driver.NamedValue
}

type inspectionTestResult struct {
	affected int64
	err      error
}

// テスト用更新件数取得
func (r inspectionTestResult) RowsAffected() (int64, error) {
	return r.affected, r.err
}

// テスト用最終挿入ID取得
func (inspectionTestResult) LastInsertId() (int64, error) {
	return 0, errors.New("last insert ID is not supported")
}

type inspectionTestRows struct {
	response inspectionTestResponse
	index    int
	errorAt  int
}

// テスト用列名取得
func (r *inspectionTestRows) Columns() []string { return r.response.columns }

// テスト用行終了
func (*inspectionTestRows) Close() error { return nil }

// テスト用行読込
func (r *inspectionTestRows) Next(dest []driver.Value) error {
	if r.errorAt >= 0 && r.index == r.errorAt {
		return r.response.rowErr
	}
	if r.index >= len(r.response.values) {
		return io.EOF
	}

	copy(dest, r.response.values[r.index])
	r.index++

	return nil
}

// テスト用データベース生成
func newInspectionTestDatabase(t *testing.T, responses []inspectionTestResponse) (*sql.DB, *inspectionTestScenario) {
	t.Helper()

	id := fmt.Sprintf("scenario-%d", atomic.AddUint64(&inspectionTestSequence, 1))
	scenario := &inspectionTestScenario{responses: responses}
	inspectionTestScenariosMu.Lock()
	inspectionTestScenarios[id] = scenario
	inspectionTestScenariosMu.Unlock()

	database, err := sql.Open(inspectionTestDriverName, id)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		inspectionTestScenariosMu.Lock()
		delete(inspectionTestScenarios, id)
		inspectionTestScenariosMu.Unlock()
	})

	return database, scenario
}

// テスト用問い合わせ消費検証
func assertInspectionQueriesConsumed(t *testing.T, scenario *inspectionTestScenario) {
	t.Helper()

	scenario.mu.Lock()
	defer scenario.mu.Unlock()
	if got := len(scenario.responses); got != 0 {
		t.Errorf("unconsumed queries = %d, want 0", got)
	}
}

// 最小最大値対応型判定検証
func TestSupportsMinMax(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		want     bool
	}{
		{
			name:     "MySQL数値型を許可する",
			dataType: "decimal(10,2)",
			want:     true,
		},
		{
			name:     "PostgreSQL日時型を許可する",
			dataType: "timestamptz",
			want:     true,
		},
		{
			name:     "空白を含む型名を許可する",
			dataType: "double precision",
			want:     true,
		},
		{
			name:     "PostgreSQL点型を拒否する",
			dataType: "point",
			want:     false,
		},
		{
			name:     "PostgreSQL間隔型を拒否する",
			dataType: "interval",
			want:     false,
		},
		{
			name:     "バイナリ型を拒否する",
			dataType: "bytea",
			want:     false,
		},
		{
			name:     "JSON型を拒否する",
			dataType: "jsonb",
			want:     false,
		},
		{
			name:     "配列型を拒否する",
			dataType: "_int4",
			want:     false,
		},
		{
			name:     "範囲型を拒否する",
			dataType: "int4range",
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := supportsMinMax(tt.dataType); got != tt.want {
				t.Errorf("supportsMinMax(%q) = %v, want %v", tt.dataType, got, tt.want)
			}
		})
	}
}

// 構造取得前タイムアウト統計検証
func TestInspectTableStatisticsReturnsUnavailableDataBeforeStructure(t *testing.T) {
	database, scenario := newInspectionTestDatabase(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ref := domain.TableRef{Namespace: "public", Name: "items"}

	statistics, err := inspectTableStatistics(ctx, database, domain.DBTypePostgres, ref)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("inspectTableStatistics() error = %v, want %v", err, context.Canceled)
	}
	if statistics.Table != ref {
		t.Errorf("Table = %#v, want %#v", statistics.Table, ref)
	}
	if statistics.RowCount.Status != domain.StatisticsStatusUnavailable || statistics.RowCount.Reason == nil || *statistics.RowCount.Reason != "not collected" {
		t.Errorf("RowCount = %#v, want unavailable not collected", statistics.RowCount)
	}
	if !statistics.CollectedAt.IsZero() {
		t.Errorf("CollectedAt = %v, want zero time", statistics.CollectedAt)
	}
	assertInspectionQueriesConsumed(t, scenario)
}

// 統計取得成功応答生成
func tableStatisticsSuccessResponses() []inspectionTestResponse {
	return []inspectionTestResponse{
		{
			query:   postgresTableStructureColumnQuery,
			columns: []string{"name", "type", "nullable", "default", "generated", "primary", "foreign", "unique"},
			values: [][]driver.Value{{
				"id",
				"int4",
				false,
				nil,
				false,
				true,
				false,
				true,
			}},
			errorAt: -1,
		},
		{
			query:   postgresTableStructureForeignKeyQuery,
			columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
			errorAt: -1,
		},
		{
			query:   postgresTableStructureIndexQuery,
			columns: []string{"name", "column", "unique", "kind"},
			errorAt: -1,
		},
		{
			query:   `SELECT COUNT(*) FROM "public"."items"`,
			columns: []string{"count"},
			values:  [][]driver.Value{{int64(3)}},
			errorAt: -1,
		},
		{
			query:   `SELECT COUNT(*) - COUNT("id") FROM "public"."items"`,
			columns: []string{"null_count"},
			values:  [][]driver.Value{{int64(1)}},
			errorAt: -1,
		},
		{
			query:   `SELECT COUNT(DISTINCT "id"), MIN("id"), MAX("id") FROM "public"."items"`,
			columns: []string{"distinct_count", "min", "max"},
			values:  [][]driver.Value{{int64(2), "1", "3"}},
			errorAt: -1,
		},
		{
			query:   `SELECT COUNT("id") FROM "public"."items"`,
			columns: []string{"non_null_count"},
			values:  [][]driver.Value{{int64(2)}},
			errorAt: -1,
		},
	}
}

// 外部キーを含むテーブル統計成功応答生成
func tableStatisticsWithForeignKeyResponses() []inspectionTestResponse {
	responses := tableStatisticsSuccessResponses()
	responses[0].values = [][]driver.Value{{"parent_id", "int4", false, nil, false, false, true, false}}
	responses[1].values = [][]driver.Value{{"fk_items_parent", "items", "parent_id", "parents", "id"}}
	responses[4].query = `SELECT COUNT(*) - COUNT("parent_id") FROM "public"."items"`
	responses[5].query = `SELECT COUNT(DISTINCT "parent_id"), MIN("parent_id"), MAX("parent_id") FROM "public"."items"`
	responses[6].query = `SELECT COUNT("parent_id") FROM "public"."items"`
	responses = append(responses, inspectionTestResponse{
		query:   `SELECT COUNT(*), COALESCE(SUM(CASE WHEN src."parent_id" IS NULL THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN src."parent_id" IS NULL THEN 0 WHEN dst."id" IS NULL THEN 0 ELSE 1 END), 0) FROM "public"."items" src LEFT JOIN "public"."parents" dst ON src."parent_id" = dst."id"`,
		columns: []string{"source", "nulls", "referenced"},
		values:  [][]driver.Value{{int64(3), int64(1), int64(2)}},
		errorAt: -1,
	})

	return responses
}

// テーブル統計問い合わせ成功検証
func TestInspectTableStatistics(t *testing.T) {
	database, scenario := newInspectionTestDatabase(t, tableStatisticsSuccessResponses())
	ref := domain.TableRef{Namespace: "public", Name: "items"}

	statistics, err := inspectTableStatistics(context.Background(), database, domain.DBTypePostgres, ref)
	if err != nil {
		t.Fatalf("inspectTableStatistics() error = %v", err)
	}
	if statistics.Status != domain.StatisticsStatusComplete {
		t.Errorf("Status = %q, want %q", statistics.Status, domain.StatisticsStatusComplete)
	}
	if statistics.RowCount.Value == nil || *statistics.RowCount.Value != 3 {
		t.Errorf("RowCount = %#v, want 3", statistics.RowCount)
	}
	if len(statistics.Columns) != 1 {
		t.Fatalf("Columns length = %d, want 1", len(statistics.Columns))
	}
	column := statistics.Columns[0]
	if column.DuplicateCount.Value == nil || *column.DuplicateCount.Value != 0 {
		t.Errorf("DuplicateCount = %#v, want 0", column.DuplicateCount)
	}
	if column.Min.Value == nil || *column.Min.Value != "1" {
		t.Errorf("Min = %#v, want 1", column.Min)
	}
	if column.Max.Value == nil || *column.Max.Value != "3" {
		t.Errorf("Max = %#v, want 3", column.Max)
	}
	if statistics.CollectedAt.IsZero() {
		t.Error("CollectedAt is zero, want collected time")
	}
	assertInspectionQueriesConsumed(t, scenario)
}

// 未対応型カラム統計検証
func TestInspectColumnStatisticsForUnsupportedType(t *testing.T) {
	database, scenario := newInspectionTestDatabase(t, []inspectionTestResponse{{
		query:   `SELECT COUNT(*) - COUNT("payload") FROM "public"."items"`,
		columns: []string{"null_count"},
		values:  [][]driver.Value{{int64(1)}},
		errorAt: -1,
	}})

	statistics, err := inspectColumnStatistics(context.Background(), database, domain.DBTypePostgres, `"public"."items"`, domain.Column{Name: "payload", DataType: "jsonb"})
	if err != nil {
		t.Fatalf("inspectColumnStatistics() error = %v", err)
	}
	if statistics.Min.Reason == nil || *statistics.Min.Reason != "unsupported data type" {
		t.Errorf("Min = %#v, want unsupported data type", statistics.Min)
	}
	if statistics.DistinctCount.Status != domain.StatisticsStatusUnavailable {
		t.Errorf("DistinctCount.Status = %q, want %q", statistics.DistinctCount.Status, domain.StatisticsStatusUnavailable)
	}
	assertInspectionQueriesConsumed(t, scenario)
}

// テーブル統計メタデータ接続検証
func TestAppRepositoryInspectTableStatistics(t *testing.T) {
	profile := connectionTestProfile(t, domain.DBTypePostgres)
	ref := domain.TableRef{Namespace: "public", Name: "items"}
	originalOpenDatabase := openDatabase
	t.Cleanup(func() { openDatabase = originalOpenDatabase })

	tests := []struct {
		name      string
		open      func(string, string) (*sql.DB, error)
		ctx       context.Context
		wantError string
		wantCount *int64
	}{
		{
			name: "データベース接続生成失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			ctx:       context.Background(),
			wantError: "open failed",
		},
		{
			name: "Ping失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, nil)

				return database, nil
			},
			ctx:       canceledInspectionContext(),
			wantError: context.Canceled.Error(),
		},
		{
			name: "統計を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, tableStatisticsSuccessResponses())

				return database, nil
			},
			ctx:       context.Background(),
			wantCount: int64Pointer(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openDatabase = tt.open

			statistics, err := NewAppRepository(nil).InspectTableStatistics(tt.ctx, profile, "secret", ref)
			if tt.wantError != "" {
				if err == nil || err.Error() != tt.wantError {
					t.Errorf("InspectTableStatistics() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("InspectTableStatistics() error = %v", err)
			}
			if statistics.RowCount.Value == nil {
				t.Fatal("RowCount.Value = nil, want value")
			}
			if *statistics.RowCount.Value != *tt.wantCount {
				t.Errorf("RowCount.Value = %d, want %d", *statistics.RowCount.Value, *tt.wantCount)
			}
		})
	}
}

// int64ポインター生成
func int64Pointer(value int64) *int64 {
	return &value
}

// 統計取得失敗経路検証
func TestInspectTableStatisticsErrors(t *testing.T) {
	ref := domain.TableRef{Namespace: "public", Name: "items"}
	tests := []struct {
		name      string
		responses []inspectionTestResponse
		ctx       context.Context
		wantError string
	}{
		{
			name: "行数取得失敗を返す",
			responses: func() []inspectionTestResponse {
				responses := tableStatisticsSuccessResponses()
				responses[3].err = errors.New("row count failed")

				return responses[:4]
			}(),
			ctx:       context.Background(),
			wantError: "row count failed",
		},
		{
			name: "カラム統計取得失敗を返す",
			responses: func() []inspectionTestResponse {
				responses := tableStatisticsSuccessResponses()
				responses[4].err = errors.New("column statistics failed")

				return responses[:5]
			}(),
			ctx:       context.Background(),
			wantError: "column statistics failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, _ := newInspectionTestDatabase(t, tt.responses)

			_, err := inspectTableStatistics(tt.ctx, database, domain.DBTypePostgres, ref)
			if err == nil || err.Error() != tt.wantError {
				t.Errorf("inspectTableStatistics() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

// テーブル統計カラム開始前キャンセル検証
func TestInspectTableStatisticsStopsBeforeColumn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responses := tableStatisticsSuccessResponses()
	responses[3].after = cancel
	database, _ := newInspectionTestDatabase(t, responses[:4])

	_, err := inspectTableStatistics(ctx, database, domain.DBTypePostgres, domain.TableRef{Namespace: "public", Name: "items"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("inspectTableStatistics() error = %v, want %v", err, context.Canceled)
	}
}

// 外部キーを含むテーブル統計検証
func TestInspectTableStatisticsWithForeignKey(t *testing.T) {
	database, scenario := newInspectionTestDatabase(t, tableStatisticsWithForeignKeyResponses())

	statistics, err := inspectTableStatistics(context.Background(), database, domain.DBTypePostgres, domain.TableRef{Namespace: "public", Name: "items"})
	if err != nil {
		t.Fatalf("inspectTableStatistics() error = %v", err)
	}
	if len(statistics.ForeignKeys) != 1 {
		t.Fatalf("ForeignKeys length = %d, want 1", len(statistics.ForeignKeys))
	}
	if statistics.ForeignKeys[0].MissingReferenceCount.Value == nil || *statistics.ForeignKeys[0].MissingReferenceCount.Value != 0 {
		t.Errorf("MissingReferenceCount = %#v, want 0", statistics.ForeignKeys[0].MissingReferenceCount)
	}
	assertInspectionQueriesConsumed(t, scenario)
}

// テーブル統計外部キー開始前キャンセル検証
func TestInspectTableStatisticsStopsBeforeForeignKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responses := tableStatisticsWithForeignKeyResponses()
	responses[6].after = cancel
	database, _ := newInspectionTestDatabase(t, responses[:7])

	_, err := inspectTableStatistics(ctx, database, domain.DBTypePostgres, domain.TableRef{Namespace: "public", Name: "items"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("inspectTableStatistics() error = %v, want %v", err, context.Canceled)
	}
}

// テーブル統計外部キー取得失敗検証
func TestInspectTableStatisticsReturnsForeignKeyError(t *testing.T) {
	responses := tableStatisticsWithForeignKeyResponses()
	responses[7].err = errors.New("foreign key statistics failed")
	database, _ := newInspectionTestDatabase(t, responses)

	_, err := inspectTableStatistics(context.Background(), database, domain.DBTypePostgres, domain.TableRef{Namespace: "public", Name: "items"})
	if err == nil || err.Error() != "foreign key statistics failed" {
		t.Errorf("inspectTableStatistics() error = %v, want %q", err, "foreign key statistics failed")
	}
}

// カラム統計問い合わせ失敗検証
func TestInspectColumnStatisticsReturnsQueryErrors(t *testing.T) {
	column := domain.Column{Name: "id", DataType: "int4"}
	tests := []struct {
		name      string
		responses []inspectionTestResponse
		wantError string
	}{
		{
			name: "NULL件数取得失敗を返す",
			responses: []inspectionTestResponse{{
				query: `SELECT COUNT(*) - COUNT("id") FROM "public"."items"`, err: errors.New("null count failed"),
			}},
			wantError: "null count failed",
		},
		{
			name: "最小最大値取得失敗を返す",
			responses: []inspectionTestResponse{
				{query: `SELECT COUNT(*) - COUNT("id") FROM "public"."items"`, columns: []string{"null_count"}, values: [][]driver.Value{{int64(0)}}, errorAt: -1},
				{query: `SELECT COUNT(DISTINCT "id"), MIN("id"), MAX("id") FROM "public"."items"`, err: errors.New("min max failed")},
			},
			wantError: "min max failed",
		},
		{
			name: "非NULL件数取得失敗を返す",
			responses: []inspectionTestResponse{
				{query: `SELECT COUNT(*) - COUNT("id") FROM "public"."items"`, columns: []string{"null_count"}, values: [][]driver.Value{{int64(0)}}, errorAt: -1},
				{query: `SELECT COUNT(DISTINCT "id"), MIN("id"), MAX("id") FROM "public"."items"`, columns: []string{"distinct_count", "min", "max"}, values: [][]driver.Value{{int64(1), nil, nil}}, errorAt: -1},
				{query: `SELECT COUNT("id") FROM "public"."items"`, err: errors.New("non-null count failed")},
			},
			wantError: "non-null count failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, _ := newInspectionTestDatabase(t, tt.responses)

			_, err := inspectColumnStatistics(context.Background(), database, domain.DBTypePostgres, `"public"."items"`, column)
			if err == nil || err.Error() != tt.wantError {
				t.Errorf("inspectColumnStatistics() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

// 外部キー統計問い合わせ失敗検証
func TestInspectForeignKeyStatisticsReturnsQueryError(t *testing.T) {
	foreignKey := domain.ForeignKey{Name: "fk_items_parent", FromColumns: []string{"parent_id"}, ToTable: "parents", ToColumns: []string{"id"}}
	database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{{
		query: `SELECT COUNT(*), COALESCE(SUM(CASE WHEN src."parent_id" IS NULL THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN src."parent_id" IS NULL THEN 0 WHEN dst."id" IS NULL THEN 0 ELSE 1 END), 0) FROM "public"."items" src LEFT JOIN "public"."parents" dst ON src."parent_id" = dst."id"`,
		err:   errors.New("foreign key query failed"),
	}})

	_, err := inspectForeignKeyStatistics(context.Background(), database, domain.DBTypePostgres, "public", "items", foreignKey)
	if err == nil || err.Error() != "foreign key query failed" {
		t.Errorf("inspectForeignKeyStatistics() error = %v, want %q", err, "foreign key query failed")
	}
}

// 統計補助関数検証
func TestStatisticHelpers(t *testing.T) {
	tests := []struct {
		name       string
		dbType     domain.DBType
		identifier string
		wantQuoted string
		value      sql.NullString
		wantValue  *string
	}{
		{
			name:       "MySQL識別子を引用する",
			dbType:     domain.DBTypeMySQL,
			identifier: "table`name",
			wantQuoted: "`table``name`",
			value:      sql.NullString{String: "value", Valid: true},
			wantValue:  stringPointer("value"),
		},
		{
			name:       "NULL最小値を値なしで返す",
			dbType:     domain.DBTypePostgres,
			identifier: "table",
			wantQuoted: `"table"`,
			value:      sql.NullString{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteIdentifier(tt.dbType, tt.identifier); got != tt.wantQuoted {
				t.Errorf("quoteIdentifier() = %q, want %q", got, tt.wantQuoted)
			}
			if got := availableValue(tt.value); !reflect.DeepEqual(got.Value, tt.wantValue) {
				t.Errorf("availableValue() = %#v, want value %#v", got, tt.wantValue)
			}
		})
	}
	if got := unavailableValue("reason"); got.Reason == nil || *got.Reason != "reason" {
		t.Errorf("unavailableValue() = %#v, want reason", got)
	}
}

// 外部キー欠損参照集計検証
func TestInspectForeignKeyStatisticsCountsMissingReferences(t *testing.T) {
	foreignKey := domain.ForeignKey{
		Name:        "fk_child_parent",
		FromColumns: []string{"parent_id"},
		ToTable:     "parents",
		ToColumns:   []string{"id"},
	}
	query := "SELECT COUNT(*), COALESCE(SUM(CASE WHEN src.\"parent_id\" IS NULL THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN src.\"parent_id\" IS NULL THEN 0 WHEN dst.\"id\" IS NULL THEN 0 ELSE 1 END), 0) FROM \"public\".\"children\" src LEFT JOIN \"public\".\"parents\" dst ON src.\"parent_id\" = dst.\"id\""
	database, scenario := newInspectionTestDatabase(t, []inspectionTestResponse{{
		query:   query,
		columns: []string{"source", "nulls", "referenced"},
		errorAt: -1,
		values: [][]driver.Value{{
			int64(3),
			int64(1),
			int64(1),
		}},
	}})

	statistics, err := inspectForeignKeyStatistics(context.Background(), database, domain.DBTypePostgres, "public", "children", foreignKey)
	if err != nil {
		t.Fatalf("inspectForeignKeyStatistics() error = %v", err)
	}
	if statistics.MissingReferenceCount.Value == nil || *statistics.MissingReferenceCount.Value != 1 {
		t.Errorf("MissingReferenceCount = %#v, want 1", statistics.MissingReferenceCount)
	}
	assertInspectionQueriesConsumed(t, scenario)
}

// スキーマメタデータ取得検証
func TestInspectSchema(t *testing.T) {
	tests := []struct {
		name      string
		dbType    domain.DBType
		namespace string
		responses []inspectionTestResponse
		want      domain.Schema
		wantErr   error
	}{
		{
			name:      "PostgreSQLのテーブルと複合外部キーを返す",
			dbType:    domain.DBTypePostgres,
			namespace: "public",
			responses: []inspectionTestResponse{
				{
					query:   postgresSchemaTableQuery,
					columns: []string{"table_name"},
					values:  [][]driver.Value{{"children"}, {"parents"}},
					errorAt: -1,
				},
				{
					query:   postgresSchemaColumnQuery,
					columns: []string{"column_name", "udt_name", "nullable", "primary", "foreign", "unique"},
					values:  [][]driver.Value{{"parent_id", "int4", false, false, true, false}},
					errorAt: -1,
				},
				{
					query:   postgresSchemaColumnQuery,
					columns: []string{"column_name", "udt_name", "nullable", "primary", "foreign", "unique"},
					values:  [][]driver.Value{{"id", "int4", false, true, false, true}},
					errorAt: -1,
				},
				{
					query:   postgresSchemaForeignKeyQuery,
					columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
					values:  [][]driver.Value{{"children_parent_fkey", "children", "parent_id", "parents", "id"}},
					errorAt: -1,
				},
			},
			want: domain.Schema{
				Tables: []domain.Table{
					{Namespace: "public", Name: "children", Columns: []domain.Column{{Name: "parent_id", DataType: "int4", IsForeignKey: true}}},
					{Namespace: "public", Name: "parents", Columns: []domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true, IsUnique: true}}},
				},
				ForeignKeys: []domain.ForeignKey{{Name: "children_parent_fkey", FromTable: "children", FromColumns: []string{"parent_id"}, ToTable: "parents", ToColumns: []string{"id"}}},
			},
		},
		{
			name:      "MySQLの名前空間で問い合わせる",
			dbType:    domain.DBTypeMySQL,
			namespace: "app",
			responses: []inspectionTestResponse{
				{
					query:   mysqlSchemaTableQuery,
					columns: []string{"table_name"},
					errorAt: -1,
				},
				{
					query:   mysqlSchemaForeignKeyQuery,
					columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
					errorAt: -1,
				},
			},
			want: domain.Schema{ForeignKeys: []domain.ForeignKey{}},
		},
		{
			name:      "テーブル問い合わせ失敗を返す",
			dbType:    domain.DBTypePostgres,
			namespace: "public",
			responses: []inspectionTestResponse{{
				query: postgresSchemaTableQuery,
				err:   errors.New("table query failed"),
			}},
			wantErr: errors.New("table query failed"),
		},
		{
			name:      "テーブル行読込失敗を返す",
			dbType:    domain.DBTypePostgres,
			namespace: "public",
			responses: []inspectionTestResponse{{
				query:   postgresSchemaTableQuery,
				columns: []string{"table_name"},
				values:  [][]driver.Value{{nil}},
				errorAt: -1,
			}},
			wantErr: errors.New("converting NULL to string is unsupported"),
		},
		{
			name:      "テーブル行反復失敗を返す",
			dbType:    domain.DBTypePostgres,
			namespace: "public",
			responses: []inspectionTestResponse{{
				query:   postgresSchemaTableQuery,
				columns: []string{"table_name"},
				rowErr:  errors.New("table rows failed"),
				errorAt: 0,
			}},
			wantErr: errors.New("table rows failed"),
		},
		{
			name:      "カラム問い合わせ失敗を返す",
			dbType:    domain.DBTypePostgres,
			namespace: "public",
			responses: []inspectionTestResponse{
				{
					query:   postgresSchemaTableQuery,
					columns: []string{"table_name"},
					values:  [][]driver.Value{{"users"}},
					errorAt: -1,
				},
				{
					query: postgresSchemaColumnQuery,
					err:   errors.New("column query failed"),
				},
			},
			wantErr: errors.New("column query failed"),
		},
		{
			name:      "外部キー問い合わせ失敗を返す",
			dbType:    domain.DBTypePostgres,
			namespace: "public",
			responses: []inspectionTestResponse{
				{
					query:   postgresSchemaTableQuery,
					columns: []string{"table_name"},
					errorAt: -1,
				},
				{
					query: postgresSchemaForeignKeyQuery,
					err:   errors.New("foreign key query failed"),
				},
			},
			wantErr: errors.New("foreign key query failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, scenario := newInspectionTestDatabase(t, tt.responses)

			got, err := inspectSchema(context.Background(), database, tt.dbType, tt.namespace)
			if tt.wantErr != nil {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("inspectSchema() error = %v, want %v", err, tt.wantErr)
				}

				return
			}
			if err != nil {
				t.Fatalf("inspectSchema() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inspectSchema() = %#v, want %#v", got, tt.want)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// テーブル構造メタデータ取得検証
func TestInspectTableStructure(t *testing.T) {
	defaultValue := "nextval('items_id_seq')"
	tests := []struct {
		name      string
		responses []inspectionTestResponse
		want      domain.TableStructure
		wantError string
	}{
		{
			name: "PostgreSQLの詳細属性と索引を返す",
			responses: []inspectionTestResponse{
				{
					query:   postgresTableStructureColumnQuery,
					columns: []string{"name", "type", "nullable", "default", "generated", "primary", "foreign", "unique"},
					values:  [][]driver.Value{{"id", "int4", false, defaultValue, true, true, false, true}},
					errorAt: -1,
				},
				{
					query:   postgresTableStructureForeignKeyQuery,
					columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
					errorAt: -1,
				},
				{
					query:   postgresTableStructureIndexQuery,
					columns: []string{"name", "column", "unique", "kind"},
					values: [][]driver.Value{
						{"items_pkey", "id", true, "btree"},
						{"items_name_key", "name", true, "btree"},
					},
					errorAt: -1,
				},
			},
			want: domain.TableStructure{
				Table: domain.Table{
					Namespace: "public",
					Name:      "items",
					Columns: []domain.Column{
						{
							Name:         "id",
							DataType:     "int4",
							DefaultValue: &defaultValue,
							IsGenerated:  true,
							IsPrimaryKey: true,
							IsUnique:     true,
						},
					},
				},
				ForeignKeys: []domain.ForeignKey{},
				Indexes: []domain.Index{
					{
						Name:    "items_name_key",
						Columns: []string{"name"},
						Unique:  true,
						Kind:    "btree",
					},
					{
						Name:    "items_pkey",
						Columns: []string{"id"},
						Unique:  true,
						Kind:    "btree",
					},
				},
			},
		},
		{
			name: "存在しないテーブルを返す",
			responses: []inspectionTestResponse{{
				query:   postgresTableStructureColumnQuery,
				columns: []string{"name"},
				errorAt: -1,
			}},
			wantError: "no rows",
		},
		{
			name: "カラム問い合わせ失敗を返す",
			responses: []inspectionTestResponse{{
				query: postgresTableStructureColumnQuery,
				err:   errors.New("column query failed"),
			}},
			wantError: "column query failed",
		},
		{
			name: "外部キー問い合わせ失敗を返す",
			responses: []inspectionTestResponse{
				{
					query:   postgresTableStructureColumnQuery,
					columns: []string{"name", "type", "nullable", "default", "generated", "primary", "foreign", "unique"},
					values:  [][]driver.Value{{"id", "int4", false, nil, false, true, false, true}},
					errorAt: -1,
				},
				{
					query: postgresTableStructureForeignKeyQuery,
					err:   errors.New("foreign key query failed"),
				},
			},
			wantError: "foreign key query failed",
		},
		{
			name: "インデックス問い合わせ失敗を返す",
			responses: []inspectionTestResponse{
				{
					query:   postgresTableStructureColumnQuery,
					columns: []string{"name", "type", "nullable", "default", "generated", "primary", "foreign", "unique"},
					values:  [][]driver.Value{{"id", "int4", false, nil, false, true, false, true}},
					errorAt: -1,
				},
				{
					query:   postgresTableStructureForeignKeyQuery,
					columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
					errorAt: -1,
				},
				{
					query: postgresTableStructureIndexQuery,
					err:   errors.New("index query failed"),
				},
			},
			wantError: "index query failed",
		},
	}

	ref, err := domain.NewTableRef("public", "items")
	if err != nil {
		t.Fatalf("NewTableRef() error = %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, scenario := newInspectionTestDatabase(t, tt.responses)

			got, err := inspectTableStructure(context.Background(), database, domain.DBTypePostgres, ref)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("inspectTableStructure() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("inspectTableStructure() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inspectTableStructure() = %#v, want %#v", got, tt.want)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// 詳細カラムメタデータ取得検証
func TestInspectDetailedColumns(t *testing.T) {
	tests := []struct {
		name      string
		response  inspectionTestResponse
		wantError string
	}{
		{
			name: "カラム行読込失敗を返す",
			response: inspectionTestResponse{
				query:   "detailed-columns",
				columns: []string{"name", "type", "nullable", "default", "generated", "primary", "foreign", "unique"},
				values:  [][]driver.Value{{nil, "int4", false, nil, false, false, false, false}},
				errorAt: -1,
			},
			wantError: "converting NULL to string is unsupported",
		},
		{
			name: "カラム行反復失敗を返す",
			response: inspectionTestResponse{
				query:   "detailed-columns",
				columns: []string{"name"},
				rowErr:  errors.New("detailed column rows failed"),
				errorAt: 0,
			},
			wantError: "detailed column rows failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{tt.response})

			_, err := inspectDetailedColumns(context.Background(), database, tt.response.query, "public", "items")
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("inspectDetailedColumns() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

// テーブル外部キーメタデータ取得検証
func TestInspectTableForeignKeys(t *testing.T) {
	database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{{
		query: "table-foreign-keys",
		err:   errors.New("foreign key query failed"),
	}})

	_, err := inspectTableForeignKeys(context.Background(), database, "table-foreign-keys", "public", "items")
	if err == nil || !strings.Contains(err.Error(), "foreign key query failed") {
		t.Errorf("inspectTableForeignKeys() error = %v, want %q", err, "foreign key query failed")
	}
}

// インデックスメタデータ取得検証
func TestInspectIndexes(t *testing.T) {
	tests := []struct {
		name      string
		response  inspectionTestResponse
		wantError string
	}{
		{
			name: "インデックス行読込失敗を返す",
			response: inspectionTestResponse{
				query:   "indexes",
				columns: []string{"name", "column", "unique", "kind"},
				values:  [][]driver.Value{{nil, "id", true, "btree"}},
				errorAt: -1,
			},
			wantError: "converting NULL to string is unsupported",
		},
		{
			name: "インデックス行反復失敗を返す",
			response: inspectionTestResponse{
				query:   "indexes",
				columns: []string{"name"},
				rowErr:  errors.New("index rows failed"),
				errorAt: 0,
			},
			wantError: "index rows failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{tt.response})

			_, err := inspectIndexes(context.Background(), database, tt.response.query, "public", "items")
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("inspectIndexes() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

// カラムメタデータ取得検証
func TestInspectColumns(t *testing.T) {
	tests := []struct {
		name      string
		response  inspectionTestResponse
		want      []domain.Column
		wantError string
	}{
		{
			name: "カラム行を返す",
			response: inspectionTestResponse{
				query:   "columns",
				columns: []string{"name", "type", "nullable", "primary", "foreign", "unique"},
				values:  [][]driver.Value{{"id", "int4", false, true, false, true}},
				errorAt: -1,
			},
			want: []domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true, IsUnique: true}},
		},
		{
			name: "問い合わせ失敗を返す",
			response: inspectionTestResponse{
				query: "columns",
				err:   errors.New("columns failed"),
			},
			wantError: "columns failed",
		},
		{
			name: "行読込失敗を返す",
			response: inspectionTestResponse{
				query:   "columns",
				columns: []string{"name", "type", "nullable", "primary", "foreign", "unique"},
				values:  [][]driver.Value{{nil, "int4", false, false, false, false}},
				errorAt: -1,
			},
			wantError: "converting NULL to string is unsupported",
		},
		{
			name: "行反復失敗を返す",
			response: inspectionTestResponse{
				query:   "columns",
				columns: []string{"name"},
				rowErr:  errors.New("column rows failed"),
				errorAt: 0,
			},
			wantError: "column rows failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, scenario := newInspectionTestDatabase(t, []inspectionTestResponse{tt.response})

			got, err := inspectColumns(context.Background(), database, "columns", "public", "users")
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("inspectColumns() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("inspectColumns() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inspectColumns() = %#v, want %#v", got, tt.want)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// 外部キーメタデータ取得検証
func TestInspectForeignKeys(t *testing.T) {
	tests := []struct {
		name      string
		response  inspectionTestResponse
		want      []domain.ForeignKey
		wantError string
	}{
		{
			name: "複合外部キーをグループ化してソートする",
			response: inspectionTestResponse{
				query:   "foreign-keys",
				columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
				values: [][]driver.Value{
					{"fk_orders_users", "orders", "tenant_id", "users", "tenant_id"},
					{"fk_orders_users", "orders", "user_id", "users", "id"},
					{"fk_items_orders", "items", "order_id", "orders", "id"},
				},
				errorAt: -1,
			},
			want: []domain.ForeignKey{
				{
					Name:        "fk_items_orders",
					FromTable:   "items",
					FromColumns: []string{"order_id"},
					ToTable:     "orders",
					ToColumns:   []string{"id"},
				},
				{
					Name:        "fk_orders_users",
					FromTable:   "orders",
					FromColumns: []string{"tenant_id", "user_id"},
					ToTable:     "users",
					ToColumns:   []string{"tenant_id", "id"},
				},
			},
		},
		{
			name: "問い合わせ失敗を返す",
			response: inspectionTestResponse{
				query: "foreign-keys",
				err:   errors.New("foreign keys failed"),
			},
			wantError: "foreign keys failed",
		},
		{
			name: "行読込失敗を返す",
			response: inspectionTestResponse{
				query:   "foreign-keys",
				columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
				values:  [][]driver.Value{{nil, "items", "order_id", "orders", "id"}},
				errorAt: -1,
			},
			wantError: "converting NULL to string is unsupported",
		},
		{
			name: "行反復失敗を返す",
			response: inspectionTestResponse{
				query:   "foreign-keys",
				columns: []string{"name"},
				rowErr:  errors.New("foreign key rows failed"),
				errorAt: 0,
			},
			wantError: "foreign key rows failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, scenario := newInspectionTestDatabase(t, []inspectionTestResponse{tt.response})

			got, err := inspectForeignKeys(context.Background(), database, "foreign-keys", "public")
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("inspectForeignKeys() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("inspectForeignKeys() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inspectForeignKeys() = %#v, want %#v", got, tt.want)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// スキーマメタデータ接続検証
func TestAppRepositoryInspectSchema(t *testing.T) {
	profile := connectionTestProfile(t, domain.DBTypePostgres)
	originalOpenDatabase := openDatabase
	t.Cleanup(func() { openDatabase = originalOpenDatabase })

	tests := []struct {
		name      string
		open      func(string, string) (*sql.DB, error)
		ctx       context.Context
		wantError string
	}{
		{
			name: "データベース接続生成失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			ctx:       context.Background(),
			wantError: "open failed",
		},
		{
			name: "Ping失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, nil)

				return database, nil
			},
			ctx:       canceledInspectionContext(),
			wantError: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openDatabase = tt.open

			_, err := NewAppRepository(nil).InspectSchema(tt.ctx, profile, "secret")
			if err == nil || err.Error() != tt.wantError {
				t.Errorf("InspectSchema() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

// MySQLスキーマメタデータ接続検証
func TestAppRepositoryInspectSchemaMySQL(t *testing.T) {
	database, scenario := newInspectionTestDatabase(t, []inspectionTestResponse{
		{
			query:   mysqlSchemaTableQuery,
			columns: []string{"table_name"},
			errorAt: -1,
		},
		{
			query:   mysqlSchemaForeignKeyQuery,
			columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
			errorAt: -1,
		},
	})
	originalOpenDatabase := openDatabase
	openDatabase = func(string, string) (*sql.DB, error) { return database, nil }
	t.Cleanup(func() { openDatabase = originalOpenDatabase })

	profile := connectionTestProfile(t, domain.DBTypeMySQL)
	got, err := NewAppRepository(nil).InspectSchema(context.Background(), profile, "secret")
	if err != nil {
		t.Fatalf("InspectSchema() error = %v", err)
	}
	want := domain.Schema{ForeignKeys: []domain.ForeignKey{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("InspectSchema() = %#v, want %#v", got, want)
	}
	assertInspectionQueriesConsumed(t, scenario)
}

// テーブル構造メタデータ接続検証
func TestAppRepositoryInspectTableStructure(t *testing.T) {
	profile := connectionTestProfile(t, domain.DBTypePostgres)
	ref, err := domain.NewTableRef("public", "items")
	if err != nil {
		t.Fatalf("NewTableRef() error = %v", err)
	}
	originalOpenDatabase := openDatabase
	t.Cleanup(func() { openDatabase = originalOpenDatabase })

	tests := []struct {
		name      string
		open      func(string, string) (*sql.DB, error)
		ctx       context.Context
		wantError string
	}{
		{
			name: "データベース接続生成失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			ctx:       context.Background(),
			wantError: "open failed",
		},
		{
			name: "接続確認失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, nil)

				return database, nil
			},
			ctx:       canceledInspectionContext(),
			wantError: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openDatabase = tt.open

			_, err := NewAppRepository(nil).InspectTableStructure(tt.ctx, profile, "secret", ref)
			if err == nil || err.Error() != tt.wantError {
				t.Errorf("InspectTableStructure() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

// MySQLテーブル構造メタデータ接続検証
func TestAppRepositoryInspectTableStructureMySQL(t *testing.T) {
	database, scenario := newInspectionTestDatabase(t, []inspectionTestResponse{
		{
			query:   mysqlTableStructureColumnQuery,
			columns: []string{"name", "type", "nullable", "default", "generated", "primary", "foreign", "unique"},
			values:  [][]driver.Value{{"id", "int", false, nil, false, true, false, true}},
			errorAt: -1,
		},
		{
			query:   mysqlTableStructureForeignKeyQuery,
			columns: []string{"name", "from_table", "from_column", "to_table", "to_column"},
			errorAt: -1,
		},
		{
			query:   mysqlTableStructureIndexQuery,
			columns: []string{"name", "column", "unique", "kind"},
			errorAt: -1,
		},
	})
	originalOpenDatabase := openDatabase
	openDatabase = func(string, string) (*sql.DB, error) { return database, nil }
	t.Cleanup(func() { openDatabase = originalOpenDatabase })

	profile := connectionTestProfile(t, domain.DBTypeMySQL)
	ref, err := domain.NewTableRef(profile.Database, "items")
	if err != nil {
		t.Fatalf("NewTableRef() error = %v", err)
	}
	got, err := NewAppRepository(nil).InspectTableStructure(context.Background(), profile, "secret", ref)
	if err != nil {
		t.Fatalf("InspectTableStructure() error = %v", err)
	}
	want := domain.TableStructure{
		Table: domain.Table{
			Namespace: profile.Database,
			Name:      "items",
			Columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int",
					IsPrimaryKey: true,
					IsUnique:     true,
				},
			},
		},
		ForeignKeys: []domain.ForeignKey{},
		Indexes:     []domain.Index{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("InspectTableStructure() = %#v, want %#v", got, want)
	}
	assertInspectionQueriesConsumed(t, scenario)
}

// キャンセル済みコンテキスト生成
func canceledInspectionContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

// DB種別別問い合わせ取得検証
func TestSchemaQueries(t *testing.T) {
	tests := []struct {
		name      string
		dbType    domain.DBType
		wantTable string
	}{
		{name: "MySQLの問い合わせを返す", dbType: domain.DBTypeMySQL, wantTable: mysqlSchemaTableQuery},
		{name: "PostgreSQLの問い合わせを返す", dbType: domain.DBTypePostgres, wantTable: postgresSchemaTableQuery},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableQuery, columnQuery, foreignKeyQuery := schemaQueries(tt.dbType)
			if tableQuery != tt.wantTable {
				t.Errorf("schemaQueries() table query = %q, want %q", tableQuery, tt.wantTable)
			}
			if columnQuery == "" {
				t.Error("schemaQueries() column query = empty, want non-empty")
			}
			if foreignKeyQuery == "" {
				t.Error("schemaQueries() foreign key query = empty, want non-empty")
			}
		})
	}
}

// テーブル構造問い合わせ取得検証
func TestTableStructureQueries(t *testing.T) {
	tests := []struct {
		name       string
		dbType     domain.DBType
		wantColumn string
	}{
		{
			name:       "MySQLの問い合わせを返す",
			dbType:     domain.DBTypeMySQL,
			wantColumn: mysqlTableStructureColumnQuery,
		},
		{
			name:       "PostgreSQLの問い合わせを返す",
			dbType:     domain.DBTypePostgres,
			wantColumn: postgresTableStructureColumnQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columnQuery, foreignKeyQuery, indexQuery := tableStructureQueries(tt.dbType)
			if columnQuery != tt.wantColumn {
				t.Errorf("tableStructureQueries() column query = %q, want %q", columnQuery, tt.wantColumn)
			}
			if foreignKeyQuery == "" {
				t.Error("tableStructureQueries() foreign key query = empty, want non-empty")
			}
			if indexQuery == "" {
				t.Error("tableStructureQueries() index query = empty, want non-empty")
			}
		})
	}
}

// テーブル行問い合わせ検証
func TestListRows(t *testing.T) {
	query := domain.TableQuery{
		Table: domain.TableRef{
			Namespace: "public",
			Name:      "users",
		},
		Page: 2,
		Columns: []domain.Column{
			{
				Name:         "id",
				DataType:     "int",
				IsPrimaryKey: true,
			},
			{
				Name:     "name",
				DataType: "varchar",
			},
			{
				Name:     "payload",
				DataType: "json",
			},
			{
				Name:     "binary",
				DataType: "bytea",
			},
		},
	}
	tests := []struct {
		name      string
		dbType    domain.DBType
		query     domain.TableQuery
		responses []inspectionTestResponse
		want      domain.TableRows
		wantError string
	}{
		{
			name:   "PostgreSQLの固定ページを返す",
			dbType: domain.DBTypePostgres,
			responses: []inspectionTestResponse{
				{
					query:   `SELECT COUNT(*) FROM "public"."users"`,
					columns: []string{"count"},
					values: [][]driver.Value{
						{int64(101)},
					},
					errorAt: -1,
				},
				{
					query: `SELECT "id", "name", "payload", "binary" FROM "public"."users" ORDER BY "id" ASC LIMIT $1 OFFSET $2`,
					columns: []string{
						"id",
						"name",
						"payload",
						"binary",
					},
					values: [][]driver.Value{
						{
							int64(101),
							"Ada",
							`{"enabled":true}`,
							[]byte{
								1,
								2,
							},
						},
					},
					errorAt: -1,
				},
			},
			want: domain.TableRows{
				Rows: []domain.TableRow{
					{
						Cells: []domain.CellValue{
							{
								Kind:  domain.CellKindValue,
								Value: "101",
							},
							{
								Kind:  domain.CellKindValue,
								Value: "Ada",
							},
							{
								Kind:  domain.CellKindValue,
								Value: `{"enabled":true}`,
							},
							{
								Kind:  domain.CellKindValue,
								Value: "AQI=",
							},
						},
					},
				},
				TotalCount: 101,
				Page:       2,
				PageSize:   100,
			},
		},
		{
			name:   "不正問い合わせを返す",
			dbType: domain.DBTypePostgres,
			query: domain.TableQuery{
				Table: domain.TableRef{
					Namespace: "public",
					Name:      "users",
				},
			},
			wantError: domain.ErrInvalidTableQuery.Error(),
		},
		{
			name:   "件数問い合わせ失敗を返す",
			dbType: domain.DBTypePostgres,
			responses: []inspectionTestResponse{
				{
					query: `SELECT COUNT(*) FROM "public"."users"`,
					err:   errors.New("count failed"),
				},
			},
			wantError: "count failed",
		},
		{
			name:   "行問い合わせ失敗を返す",
			dbType: domain.DBTypePostgres,
			responses: []inspectionTestResponse{
				{
					query:   `SELECT COUNT(*) FROM "public"."users"`,
					columns: []string{"count"},
					values: [][]driver.Value{
						{int64(1)},
					},
					errorAt: -1,
				},
				{
					query: `SELECT "id", "name", "payload", "binary" FROM "public"."users" ORDER BY "id" ASC LIMIT $1 OFFSET $2`,
					err:   errors.New("rows failed"),
				},
			},
			wantError: "rows failed",
		},
		{
			name:   "行走査失敗を返す",
			dbType: domain.DBTypePostgres,
			responses: []inspectionTestResponse{
				{
					query:   `SELECT COUNT(*) FROM "public"."users"`,
					columns: []string{"count"},
					values: [][]driver.Value{
						{int64(1)},
					},
					errorAt: -1,
				},
				{
					query: `SELECT "id", "name", "payload", "binary" FROM "public"."users" ORDER BY "id" ASC LIMIT $1 OFFSET $2`,
					columns: []string{
						"id",
						"name",
						"payload",
					},
					values: [][]driver.Value{
						{
							int64(1),
							"Ada",
							"{}",
						},
					},
					errorAt: -1,
				},
			},
			wantError: "expected 3 destination arguments in Scan, not 4",
		},
		{
			name:   "行反復失敗を返す",
			dbType: domain.DBTypePostgres,
			responses: []inspectionTestResponse{
				{
					query:   `SELECT COUNT(*) FROM "public"."users"`,
					columns: []string{"count"},
					values: [][]driver.Value{
						{int64(1)},
					},
					errorAt: -1,
				},
				{
					query: `SELECT "id", "name", "payload", "binary" FROM "public"."users" ORDER BY "id" ASC LIMIT $1 OFFSET $2`,
					columns: []string{
						"id",
						"name",
						"payload",
						"binary",
					},
					rowErr:  errors.New("iteration failed"),
					errorAt: 0,
				},
			},
			wantError: "iteration failed",
		},
		{
			name:   "MySQLの固定ページを返す",
			dbType: domain.DBTypeMySQL,
			responses: []inspectionTestResponse{
				{
					query:   "SELECT COUNT(*) FROM `public`.`users`",
					columns: []string{"count"},
					values: [][]driver.Value{
						{int64(0)},
					},
					errorAt: -1,
				},
				{
					query: "SELECT `id`, `name`, `payload`, `binary` FROM `public`.`users` ORDER BY `id` ASC LIMIT ? OFFSET ?",
					columns: []string{
						"id",
						"name",
						"payload",
						"binary",
					},
					errorAt: -1,
				},
			},
			want: domain.TableRows{
				Rows:       []domain.TableRow{},
				TotalCount: 0,
				Page:       2,
				PageSize:   domain.TablePageSize,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentQuery := query
			if tt.query.Table.Name != "" {
				currentQuery = tt.query
			}
			database, scenario := newInspectionTestDatabase(t, tt.responses)
			got, err := listRows(context.Background(), database, tt.dbType, currentQuery)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("listRows() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("listRows() error = %v", err)
			}
			if !reflect.DeepEqual(got.Rows, tt.want.Rows) {
				t.Errorf("Rows = %#v, want %#v", got.Rows, tt.want.Rows)
			}
			if got.TotalCount != tt.want.TotalCount {
				t.Errorf("TotalCount = %d, want %d", got.TotalCount, tt.want.TotalCount)
			}
			if got.Page != tt.want.Page {
				t.Errorf("Page = %d, want %d", got.Page, tt.want.Page)
			}
			if got.PageSize != tt.want.PageSize {
				t.Errorf("PageSize = %d, want %d", got.PageSize, tt.want.PageSize)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// セル値変換検証
func TestTableCellValue(t *testing.T) {
	timestamp := time.Date(2026, time.August, 5, 12, 34, 56, 0, time.UTC)
	tests := []struct {
		name   string
		value  any
		column domain.Column
		want   domain.CellValue
	}{
		{
			name:  "MySQL日時をRFC3339で返す",
			value: timestamp,
			column: domain.Column{
				Name:     "created_at",
				DataType: "datetime",
			},
			want: domain.CellValue{
				Kind:  domain.CellKindValue,
				Value: "2026-08-05T12:34:56Z",
			},
		},
		{
			name: "NULLをnullセルで返す",
			column: domain.Column{
				Name:     "note",
				DataType: "varchar",
			},
			want: domain.CellValue{
				Kind: domain.CellKindNull,
			},
		},
		{
			name: "バイナリーをbase64で返す",
			value: []byte{
				1,
				2,
			},
			column: domain.Column{
				Name:     "payload",
				DataType: "blob",
			},
			want: domain.CellValue{
				Kind:  domain.CellKindValue,
				Value: "AQI=",
			},
		},
		{
			name:  "テキストバイト列を文字列で返す",
			value: []byte("Ada"),
			column: domain.Column{
				Name:     "name",
				DataType: "varchar",
			},
			want: domain.CellValue{
				Kind:  domain.CellKindValue,
				Value: "Ada",
			},
		},
		{
			name:  "有効なJSONを文字列で返す",
			value: `{"enabled":true}`,
			column: domain.Column{
				Name:     "payload",
				DataType: "json",
			},
			want: domain.CellValue{
				Kind:  domain.CellKindValue,
				Value: `{"enabled":true}`,
			},
		},
		{
			name:  "不正なJSONを文字列で返す",
			value: "{",
			column: domain.Column{
				Name:     "payload",
				DataType: "json",
			},
			want: domain.CellValue{
				Kind:  domain.CellKindValue,
				Value: "{",
			},
		},
		{
			name:  "数値を文字列で返す",
			value: int64(42),
			column: domain.Column{
				Name:     "id",
				DataType: "int",
			},
			want: domain.CellValue{
				Kind:  domain.CellKindValue,
				Value: "42",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tableCellValue(tt.value, tt.column); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tableCellValue() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// テーブル行並び替え生成検証
func TestTableRowsOrder(t *testing.T) {
	columns := []domain.Column{
		{
			Name:         "tenant_id",
			IsPrimaryKey: true,
		},
		{
			Name:         "id",
			IsPrimaryKey: true,
		},
		{Name: "name"},
	}
	tests := []struct {
		name   string
		dbType domain.DBType
		query  domain.TableQuery
		want   string
	}{
		{
			name:   "指定並び替えを返す",
			dbType: domain.DBTypePostgres,
			query: domain.TableQuery{
				Sort: &domain.TableSort{
					Column:    "name",
					Direction: domain.SortDirectionDescending,
				},
			},
			want: ` ORDER BY "name" DESC`,
		},
		{
			name:   "主キーの昇順を返す",
			dbType: domain.DBTypeMySQL,
			query: domain.TableQuery{
				Columns: columns,
			},
			want: " ORDER BY `tenant_id` ASC, `id` ASC",
		},
		{
			name:   "主キーなしは空文字を返す",
			dbType: domain.DBTypePostgres,
			query: domain.TableQuery{
				Columns: []domain.Column{
					{Name: "name"},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tableRowsOrder(tt.dbType, tt.query); got != tt.want {
				t.Errorf("tableRowsOrder() = %q, want %q", got, tt.want)
			}
		})
	}
}

// テーブル行条件生成検証
func TestTableRowsWhere(t *testing.T) {
	filter := &domain.FilterGroup{
		Operator: domain.FilterGroupOperatorAnd,
		Filters: []domain.TableFilter{
			{
				Column:   "status",
				Operator: domain.FilterOperatorIn,
				Values: []string{
					"active",
					"pending",
				},
			},
			{
				Column:   "created_at",
				Operator: domain.FilterOperatorBetween,
				Values: []string{
					"2026-01-01",
					"2026-01-31",
				},
			},
			{
				Column:   "deleted_at",
				Operator: domain.FilterOperatorIsNull,
			},
		},
		Groups: []domain.FilterGroup{
			{
				Operator: domain.FilterGroupOperatorOr,
				Filters: []domain.TableFilter{
					{
						Column:   "name",
						Operator: domain.FilterOperatorLike,
						Values:   []string{"A%"},
					},
					{
						Column:   "role",
						Operator: domain.FilterOperatorNotEqual,
						Values:   []string{"guest"},
					},
				},
			},
		},
	}
	multipleGroupsFilter := &domain.FilterGroup{
		Operator: domain.FilterGroupOperatorAnd,
		Filters: []domain.TableFilter{
			{
				Column:   "tenant_id",
				Operator: domain.FilterOperatorEqual,
				Values:   []string{"tenant-1"},
			},
		},
		Groups: []domain.FilterGroup{
			{
				Operator: domain.FilterGroupOperatorOr,
				Filters: []domain.TableFilter{
					{
						Column:   "status",
						Operator: domain.FilterOperatorEqual,
						Values:   []string{"active"},
					},
				},
				Groups: []domain.FilterGroup{
					{
						Operator: domain.FilterGroupOperatorAnd,
						Filters: []domain.TableFilter{
							{
								Column:   "role",
								Operator: domain.FilterOperatorEqual,
								Values:   []string{"admin"},
							},
						},
					},
				},
			},
			{
				Operator: domain.FilterGroupOperatorAnd,
				Filters: []domain.TableFilter{
					{
						Column:   "enabled",
						Operator: domain.FilterOperatorEqual,
						Values:   []string{"true"},
					},
				},
			},
		},
	}
	tests := []struct {
		name      string
		dbType    domain.DBType
		filter    *domain.FilterGroup
		wantWhere string
		wantArgs  []any
	}{
		{
			name:      "フィルターなしは空条件を返す",
			dbType:    domain.DBTypePostgres,
			wantWhere: "",
		},
		{
			name:      "PostgreSQL条件と引数を返す",
			dbType:    domain.DBTypePostgres,
			filter:    filter,
			wantWhere: ` WHERE "status" IN ($1, $2) AND "created_at" BETWEEN $3 AND $4 AND "deleted_at" IS NULL AND ("name" LIKE $5 OR "role" != $6)`,
			wantArgs: []any{
				"active",
				"pending",
				"2026-01-01",
				"2026-01-31",
				"A%",
				"guest",
			},
		},
		{
			name:      "MySQL条件と引数を返す",
			dbType:    domain.DBTypeMySQL,
			filter:    filter,
			wantWhere: " WHERE `status` IN (?, ?) AND `created_at` BETWEEN ? AND ? AND `deleted_at` IS NULL AND (`name` LIKE ? OR `role` != ?)",
			wantArgs: []any{
				"active",
				"pending",
				"2026-01-01",
				"2026-01-31",
				"A%",
				"guest",
			},
		},
		{
			name:      "PostgreSQLの複数子グループを連番で返す",
			dbType:    domain.DBTypePostgres,
			filter:    multipleGroupsFilter,
			wantWhere: ` WHERE "tenant_id" = $1 AND ("status" = $2 OR ("role" = $3)) AND ("enabled" = $4)`,
			wantArgs: []any{
				"tenant-1",
				"active",
				"admin",
				"true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWhere, gotArgs := tableRowsWhere(tt.dbType, tt.filter)
			if gotWhere != tt.wantWhere {
				t.Errorf("tableRowsWhere() where = %q, want %q", gotWhere, tt.wantWhere)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("tableRowsWhere() args = %#v, want %#v", gotArgs, tt.wantArgs)
			}
		})
	}
}

// テーブル行プレースホルダー生成検証
func TestTableRowsPlaceholder(t *testing.T) {
	tests := []struct {
		name      string
		dbType    domain.DBType
		parameter int
		want      string
	}{
		{
			name:      "MySQLは疑問符を返す",
			dbType:    domain.DBTypeMySQL,
			parameter: 3,
			want:      "?",
		},
		{
			name:      "PostgreSQLは番号付き記号を返す",
			dbType:    domain.DBTypePostgres,
			parameter: 3,
			want:      "$3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tableRowsPlaceholder(tt.dbType, tt.parameter); got != tt.want {
				t.Errorf("tableRowsPlaceholder() = %q, want %q", got, tt.want)
			}
		})
	}
}

// テーブル行取得メタデータ接続検証
func TestAppRepositoryListRows(t *testing.T) {
	profile := connectionTestProfile(t, domain.DBTypePostgres)
	query := domain.TableQuery{
		Table: domain.TableRef{
			Namespace: "public",
			Name:      "items",
		},
		Page: 1,
		Columns: []domain.Column{
			{
				Name:         "id",
				DataType:     "int",
				IsPrimaryKey: true,
			},
		},
	}
	originalOpenDatabase := openDatabase
	t.Cleanup(func() { openDatabase = originalOpenDatabase })
	tests := []struct {
		name      string
		open      func(string, string) (*sql.DB, error)
		ctx       context.Context
		wantError string
		wantCount int64
	}{
		{
			name: "接続生成失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			ctx:       context.Background(),
			wantError: "open failed",
		},
		{
			name: "接続確認失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, nil)

				return database, nil
			},
			ctx:       canceledInspectionContext(),
			wantError: context.Canceled.Error(),
		},
		{
			name: "行一覧を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   `SELECT COUNT(*) FROM "public"."items"`,
						columns: []string{"count"},
						values: [][]driver.Value{
							{int64(0)},
						},
						errorAt: -1,
					},
					{
						query:   `SELECT "id" FROM "public"."items" ORDER BY "id" ASC LIMIT $1 OFFSET $2`,
						columns: []string{"id"},
						errorAt: -1,
					},
				})

				return database, nil
			},
			ctx:       context.Background(),
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openDatabase = tt.open
			got, err := NewAppRepository(nil).ListRows(tt.ctx, profile, "secret", query)
			if tt.wantError != "" {
				if err == nil || err.Error() != tt.wantError {
					t.Errorf("ListRows() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("ListRows() error = %v", err)
			}
			if got.TotalCount != tt.wantCount {
				t.Errorf("ListRows() TotalCount = %d, want %d", got.TotalCount, tt.wantCount)
			}
		})
	}
}

// 入力値変換テスト
func TestConvertInputValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		dataType string
		wantErr  bool
	}{
		{
			name:     "テキスト値をそのまま返す",
			value:    "hello",
			dataType: "varchar(255)",
			wantErr:  false,
		},
		{
			name:     "正常なBase64バイナリー値を変換する",
			value:    "SGVsbG8=",
			dataType: "blob",
			wantErr:  false,
		},
		{
			name:     "不正なBase64バイナリー値を拒否する",
			value:    "invalid-base64!!!",
			dataType: "bytea",
			wantErr:  true,
		},
		{
			name:     "正常なJSON値を検証・返却する",
			value:    `{"a": 1}`,
			dataType: "json",
			wantErr:  false,
		},
		{
			name:     "不正なJSON値を拒否する",
			value:    `{invalid json}`,
			dataType: "json",
			wantErr:  true,
		},
		{
			name:     "RFC3339日時をtime.Timeへ変換する",
			value:    "2026-08-05T12:34:56+09:00",
			dataType: "timestamp with time zone",
			wantErr:  false,
		},
		{
			name:     "RFC3339ではない日時を拒否する",
			value:    "2026-08-05 12:34:56",
			dataType: "datetime(6)",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := convertInputValue(tt.value, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("convertInputValue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (strings.Contains(tt.dataType, "timestamp") || strings.Contains(tt.dataType, "datetime")) {
				if _, ok := got.(time.Time); !ok {
					t.Errorf("convertInputValue() = %T, want time.Time", got)
				}
			}
		})
	}
}

// テーブル行追加リポジトリ検証
func TestAppRepositoryInsertRow(t *testing.T) {
	profile := connectionTestProfile(t, domain.DBTypePostgres)
	ref := domain.TableRef{Namespace: "public", Name: "items"}
	valStr := "Item-1"
	valInvalid := "invalid-base64!!!"
	row := domain.InsertRow{
		Table: ref,
		Values: []domain.ColumnValueInput{
			{
				Column: "id",
				Kind:   domain.CellKindDefault,
			},
			{
				Column: "name",
				Kind:   domain.CellKindValue,
				Value:  &valStr,
			},
		},
	}
	originalOpenDatabase := openDatabase
	t.Cleanup(func() { openDatabase = originalOpenDatabase })

	tests := []struct {
		name      string
		open      func(string, string) (*sql.DB, error)
		useMySQL  bool
		customRow *domain.InsertRow
		ctx       context.Context
		wantError string
		want      domain.AffectedRows
	}{
		{
			name: "DB接続失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			ctx:       context.Background(),
			wantError: "open failed",
		},
		{
			name: "Ping失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, nil)

				return database, nil
			},
			ctx:       canceledInspectionContext(),
			wantError: context.Canceled.Error(),
		},
		{
			name: "正常に行を追加して影響行数を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   postgresTableStructureColumnQuery,
						columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
						values: [][]driver.Value{
							{"id", "int4", false, nil, true, true, false, true},
							{"name", "text", false, nil, false, false, false, false},
						},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureIndexQuery,
						columns: []string{"relname", "attname", "indisunique", "amname"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:        `INSERT INTO "public"."items" ("name") VALUES ($1)`,
						execAffected: 1,
						errorAt:      -1,
					},
				})

				return database, nil
			},
			ctx:  context.Background(),
			want: domain.AffectedRows{AffectedRows: 1},
		},
		{
			name: "テーブル構造取得失敗エラーを返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query: postgresTableStructureColumnQuery,
						err:   errors.New("table structure query error"),
					},
				})

				return database, nil
			},
			ctx:       context.Background(),
			wantError: "table structure query error",
		},
		{
			name: "未知のカラム指定エラーを返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   postgresTableStructureColumnQuery,
						columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
						values: [][]driver.Value{
							{"id", "int4", false, nil, true, true, false, true},
						},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureIndexQuery,
						columns: []string{"relname", "attname", "indisunique", "amname"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
				})

				return database, nil
			},
			ctx:       context.Background(),
			wantError: "unknown column",
		},
		{
			name: "CellKindValueでのnilポインターエラーを返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   postgresTableStructureColumnQuery,
						columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
						values: [][]driver.Value{
							{"id", "int4", false, nil, true, true, false, true},
							{"name", "text", false, nil, false, false, false, false},
						},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureIndexQuery,
						columns: []string{"relname", "attname", "indisunique", "amname"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
				})

				return database, nil
			},
			customRow: &domain.InsertRow{
				Table: ref,
				Values: []domain.ColumnValueInput{
					{Column: "name", Kind: domain.CellKindValue, Value: nil},
				},
			},
			ctx:       context.Background(),
			wantError: "missing value for column",
		},
		{
			name: "PostgreSQL全デフォルト値INSERT文を生成する",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   postgresTableStructureColumnQuery,
						columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
						values: [][]driver.Value{
							{"id", "int4", false, nil, true, true, false, true},
						},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureIndexQuery,
						columns: []string{"relname", "attname", "indisunique", "amname"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:        `INSERT INTO "public"."items" DEFAULT VALUES`,
						execAffected: 1,
						errorAt:      -1,
					},
				})

				return database, nil
			},
			customRow: &domain.InsertRow{
				Table: ref,
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindDefault,
					},
				},
			},
			ctx:  context.Background(),
			want: domain.AffectedRows{AffectedRows: 1},
		},
		{
			name: "MySQL全デフォルト値INSERT文を生成する",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   mysqlTableStructureColumnQuery,
						columns: []string{"column_name", "column_type", "is_nullable", "column_default", "extra", "column_key", "is_foreign", "is_unique"},
						values: [][]driver.Value{
							{"id", "int", false, nil, true, true, false, true},
						},
						errorAt: -1,
					},
					{
						query:   mysqlTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   mysqlTableStructureIndexQuery,
						columns: []string{"index_name", "column_name", "non_unique", "index_type"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:        "INSERT INTO `app`.`items` () VALUES ()",
						execAffected: 1,
						errorAt:      -1,
					},
				})

				return database, nil
			},
			useMySQL: true,
			customRow: &domain.InsertRow{
				Table: domain.TableRef{
					Namespace: "app",
					Name:      "items",
				},
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindDefault,
					},
				},
			},
			ctx:  context.Background(),
			want: domain.AffectedRows{AffectedRows: 1},
		},
		{
			name: "CellKindNullの入力値を扱える",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   postgresTableStructureColumnQuery,
						columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
						values: [][]driver.Value{
							{"id", "int4", false, nil, true, true, false, true},
							{"name", "text", true, nil, false, false, false, false},
						},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureIndexQuery,
						columns: []string{"relname", "attname", "indisunique", "amname"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:        `INSERT INTO "public"."items" ("name") VALUES ($1)`,
						execAffected: 1,
						errorAt:      -1,
					},
				})

				return database, nil
			},
			customRow: &domain.InsertRow{
				Table: ref,
				Values: []domain.ColumnValueInput{
					{
						Column: "name",
						Kind:   domain.CellKindNull,
					},
				},
			},
			ctx:  context.Background(),
			want: domain.AffectedRows{AffectedRows: 1},
		},
		{
			name: "入力値変換失敗エラーを返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   postgresTableStructureColumnQuery,
						columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
						values: [][]driver.Value{
							{"id", "int4", false, nil, true, true, false, true},
							{"data", "bytea", false, nil, false, false, false, false},
						},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureIndexQuery,
						columns: []string{"relname", "attname", "indisunique", "amname"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
				})

				return database, nil
			},
			customRow: &domain.InsertRow{
				Table: ref,
				Values: []domain.ColumnValueInput{
					{
						Column: "data",
						Kind:   domain.CellKindValue,
						Value:  &valInvalid,
					},
				},
			},
			ctx:       context.Background(),
			wantError: "invalid base64 value for binary column",
		},
		{
			name: "Exec実行エラーを返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
					{
						query:   postgresTableStructureColumnQuery,
						columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
						values: [][]driver.Value{
							{"id", "int4", false, nil, true, true, false, true},
							{"name", "text", false, nil, false, false, false, false},
						},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureForeignKeyQuery,
						columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query:   postgresTableStructureIndexQuery,
						columns: []string{"relname", "attname", "indisunique", "amname"},
						values:  [][]driver.Value{},
						errorAt: -1,
					},
					{
						query: `INSERT INTO "public"."items" ("name") VALUES ($1)`,
						err:   errors.New("exec error"),
					},
				})

				return database, nil
			},
			ctx:       context.Background(),
			wantError: "exec error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openDatabase = tt.open
			targetProfile := profile
			if tt.useMySQL {
				targetProfile = connectionTestProfile(t, domain.DBTypeMySQL)
			}
			targetRow := row
			if tt.customRow != nil {
				targetRow = *tt.customRow
			}
			got, err := NewAppRepository(nil).InsertRow(tt.ctx, targetProfile, "secret", targetRow.Table, targetRow)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("InsertRow() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("InsertRow() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InsertRow() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// テーブルセル更新リポジトリ検証
func TestUpdateCell(t *testing.T) {
	id := "1"
	name := "updated"
	ref := domain.TableRef{Namespace: "public", Name: "items"}
	change := domain.CellUpdate{
		Table:   ref,
		Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}},
		Column:  "name",
		Value:   domain.CellValue{Kind: domain.CellKindValue, Value: name},
	}
	database, _ := newInspectionTestDatabase(t, []inspectionTestResponse{
		{
			query:   postgresTableStructureColumnQuery,
			columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"},
			values:  [][]driver.Value{{"id", "int4", false, nil, false, true, false, true}, {"name", "text", false, nil, false, false, false, false}},
			errorAt: -1,
		},
		{query: postgresTableStructureForeignKeyQuery, columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"}, values: [][]driver.Value{}, errorAt: -1},
		{query: postgresTableStructureIndexQuery, columns: []string{"relname", "attname", "indisunique", "amname"}, values: [][]driver.Value{}, errorAt: -1},
		{query: `UPDATE "public"."items" SET "name" = $1 WHERE "id" = $2`, execAffected: 1, errorAt: -1},
	})
	defer database.Close()

	got, err := updateCell(context.Background(), database, domain.DBTypePostgres, ref, change)
	if err != nil {
		t.Fatalf("updateCell() error = %v", err)
	}
	if want := (domain.AffectedRows{AffectedRows: 1}); !reflect.DeepEqual(got, want) {
		t.Errorf("updateCell() = %#v, want %#v", got, want)
	}
}

// テーブルセル更新接続検証
func TestAppRepositoryUpdateCell(t *testing.T) {
	profile := connectionTestProfile(t, domain.DBTypePostgres)
	id := "1"
	ref := domain.TableRef{Namespace: "public", Name: "items"}
	change := domain.CellUpdate{
		Table:   ref,
		Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}},
		Column:  "name",
		Value:   domain.CellValue{Kind: domain.CellKindValue, Value: "updated"},
	}
	originalOpenDatabase := openDatabase
	t.Cleanup(func() { openDatabase = originalOpenDatabase })
	tests := []struct {
		name      string
		open      func(string, string) (*sql.DB, error)
		ctx       context.Context
		wantError string
	}{
		{
			name: "接続生成失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			ctx:       context.Background(),
			wantError: "open failed",
		},
		{
			name: "接続確認失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, nil)

				return database, nil
			},
			ctx:       canceledInspectionContext(),
			wantError: context.Canceled.Error(),
		},
		{
			name: "セルを更新する",
			open: func(string, string) (*sql.DB, error) {
				responses := updateCellStructureResponses(domain.DBTypePostgres, []domain.Column{
					{Name: "id", DataType: "int4", IsPrimaryKey: true},
					{Name: "name", DataType: "text"},
				})
				responses = append(responses, inspectionTestResponse{
					query:        `UPDATE "public"."items" SET "name" = $1 WHERE "id" = $2`,
					execAffected: 1,
					errorAt:      -1,
				})
				database, _ := newInspectionTestDatabase(t, responses)

				return database, nil
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openDatabase = tt.open

			got, err := NewAppRepository(nil).UpdateCell(tt.ctx, profile, "secret", ref, change)
			if tt.wantError != "" {
				if err == nil || err.Error() != tt.wantError {
					t.Errorf("UpdateCell() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("UpdateCell() error = %v", err)
			}
			if got.AffectedRows != 1 {
				t.Errorf("AffectedRows = %d, want 1", got.AffectedRows)
			}
		})
	}
}

// テーブルセル更新入力エラー検証
func TestUpdateCellInputErrors(t *testing.T) {
	id := "1"
	invalidBinary := "not-base64"
	ref := domain.TableRef{Namespace: "public", Name: "items"}
	standardColumns := []domain.Column{
		{Name: "id", DataType: "int4", IsPrimaryKey: true},
		{Name: "name", DataType: "text"},
	}
	tests := []struct {
		name      string
		responses []inspectionTestResponse
		change    domain.CellUpdate
		wantError string
	}{
		{
			name: "構造取得失敗を返す",
			responses: []inspectionTestResponse{{
				query: postgresTableStructureColumnQuery,
				err:   errors.New("structure failed"),
			}},
			change:    domain.CellUpdate{},
			wantError: "structure failed",
		},
		{
			name:      "未知更新列を拒否する",
			responses: updateCellStructureResponses(domain.DBTypePostgres, standardColumns),
			change: domain.CellUpdate{
				Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}},
				Column:  "unknown",
				Value:   domain.CellValue{Kind: domain.CellKindValue, Value: "updated"},
			},
			wantError: "unknown column unknown",
		},
		{
			name: "更新値変換失敗を返す",
			responses: updateCellStructureResponses(domain.DBTypePostgres, []domain.Column{
				{Name: "id", DataType: "int4", IsPrimaryKey: true},
				{Name: "binary_data", DataType: "bytea"},
			}),
			change: domain.CellUpdate{
				Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}},
				Column:  "binary_data",
				Value:   domain.CellValue{Kind: domain.CellKindValue, Value: invalidBinary},
			},
			wantError: "invalid base64",
		},
		{
			name:      "未知位置指定列を拒否する",
			responses: updateCellStructureResponses(domain.DBTypePostgres, standardColumns),
			change: domain.CellUpdate{
				Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "unknown", Kind: domain.CellKindValue, Value: &id}}},
				Column:  "name",
				Value:   domain.CellValue{Kind: domain.CellKindValue, Value: "updated"},
			},
			wantError: "unknown locator column unknown",
		},
		{
			name:      "値なし位置指定列を拒否する",
			responses: updateCellStructureResponses(domain.DBTypePostgres, standardColumns),
			change: domain.CellUpdate{
				Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue}}},
				Column:  "name",
				Value:   domain.CellValue{Kind: domain.CellKindValue, Value: "updated"},
			},
			wantError: "missing locator value",
		},
		{
			name: "位置指定値変換失敗を返す",
			responses: updateCellStructureResponses(domain.DBTypePostgres, []domain.Column{
				{Name: "binary_data", DataType: "bytea", IsPrimaryKey: true},
				{Name: "name", DataType: "text"},
			}),
			change: domain.CellUpdate{
				Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "binary_data", Kind: domain.CellKindValue, Value: &invalidBinary}}},
				Column:  "name",
				Value:   domain.CellValue{Kind: domain.CellKindValue, Value: "updated"},
			},
			wantError: "invalid base64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, scenario := newInspectionTestDatabase(t, tt.responses)
			defer database.Close()

			_, err := updateCell(context.Background(), database, domain.DBTypePostgres, ref, tt.change)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("updateCell() error = %v, want %q", err, tt.wantError)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// 日時型判定検証
func TestIsDateTimeDataType(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		want     bool
	}{
		{
			name:     "空文字を拒否する",
			dataType: "",
			want:     false,
		},
		{
			name:     "日時型を許可する",
			dataType: "timestamp without time zone",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDateTimeDataType(tt.dataType); got != tt.want {
				t.Errorf("isDateTimeDataType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// テーブルセル更新SQL分岐検証
func TestUpdateCellSQLVariants(t *testing.T) {
	id := "1"
	partA := "10"
	partB := "20"
	name := "after"
	note := "before"
	encodedOld := "b2xk"
	encodedNew := "bmV3"
	metadata := `{"before":true}`
	occurredAt := "2026-08-05T12:34:56+09:00"
	parsedOccurredAt, err := time.Parse(time.RFC3339, occurredAt)
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}

	tests := []struct {
		name            string
		dbType          domain.DBType
		ref             domain.TableRef
		columns         []domain.Column
		change          domain.CellUpdate
		query           string
		args            []driver.NamedValue
		execError       error
		rowsAffectedErr error
		wantError       string
	}{
		{
			name:   "PostgreSQL既定値更新はプレースホルダーを使わない",
			dbType: domain.DBTypePostgres,
			ref: domain.TableRef{
				Namespace: "public",
				Name:      "items",
			},
			columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					IsPrimaryKey: true,
				},
				{Name: "name", DataType: "text", DefaultValue: &name},
			},
			change: domain.CellUpdate{Table: domain.TableRef{Namespace: "public", Name: "items"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}}, Column: "name", Value: domain.CellValue{Kind: domain.CellKindDefault}},
			query:  `UPDATE "public"."items" SET "name" = DEFAULT WHERE "id" = $1`,
			args:   []driver.NamedValue{{Ordinal: 1, Value: "1"}},
		},
		{
			name:   "MySQL既定値更新は引用符と疑問符を使う",
			dbType: domain.DBTypeMySQL,
			ref:    domain.TableRef{Namespace: "app", Name: "items"},
			columns: []domain.Column{
				{Name: "id", DataType: "int", IsPrimaryKey: true},
				{Name: "name", DataType: "varchar(32)", DefaultValue: &name},
			},
			change: domain.CellUpdate{Table: domain.TableRef{Namespace: "app", Name: "items"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}}, Column: "name", Value: domain.CellValue{Kind: domain.CellKindDefault}},
			query:  "UPDATE `app`.`items` SET `name` = DEFAULT WHERE `id` = ?",
			args:   []driver.NamedValue{{Ordinal: 1, Value: "1"}},
		},
		{
			name:   "主キーなしは全カラムとNULL位置指定を使う",
			dbType: domain.DBTypePostgres,
			ref:    domain.TableRef{Namespace: "public", Name: "logs"},
			columns: []domain.Column{
				{Name: "name", DataType: "text"},
				{Name: "note", DataType: "text", Nullable: true},
			},
			change: domain.CellUpdate{Table: domain.TableRef{Namespace: "public", Name: "logs"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "name", Kind: domain.CellKindValue, Value: &note}, {Column: "note", Kind: domain.CellKindNull}}}, Column: "name", Value: domain.CellValue{Kind: domain.CellKindValue, Value: name}},
			query:  `UPDATE "public"."logs" SET "name" = $1 WHERE "name" = $2 AND "note" IS NULL`,
			args:   []driver.NamedValue{{Ordinal: 1, Value: "after"}, {Ordinal: 2, Value: "before"}},
		},
		{
			name:   "複合主キー更新は編集前の全主キー値を使う",
			dbType: domain.DBTypePostgres,
			ref:    domain.TableRef{Namespace: "public", Name: "pairs"},
			columns: []domain.Column{
				{Name: "part_a", DataType: "int4", IsPrimaryKey: true},
				{Name: "part_b", DataType: "int4", IsPrimaryKey: true},
				{Name: "name", DataType: "text"},
			},
			change: domain.CellUpdate{Table: domain.TableRef{Namespace: "public", Name: "pairs"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "part_a", Kind: domain.CellKindValue, Value: &partA}, {Column: "part_b", Kind: domain.CellKindValue, Value: &partB}}}, Column: "part_a", Value: domain.CellValue{Kind: domain.CellKindValue, Value: "11"}},
			query:  `UPDATE "public"."pairs" SET "part_a" = $1 WHERE "part_a" = $2 AND "part_b" = $3`,
			args:   []driver.NamedValue{{Ordinal: 1, Value: "11"}, {Ordinal: 2, Value: "10"}, {Ordinal: 3, Value: "20"}},
		},
		{
			name:   "バイナリーJSON日時を更新値と位置指定値へ変換する",
			dbType: domain.DBTypePostgres,
			ref:    domain.TableRef{Namespace: "public", Name: "row_values"},
			columns: []domain.Column{
				{Name: "binary_data", DataType: "bytea"},
				{Name: "metadata", DataType: "json"},
				{Name: "occurred_at", DataType: "timestamp"},
			},
			change: domain.CellUpdate{Table: domain.TableRef{Namespace: "public", Name: "row_values"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "binary_data", Kind: domain.CellKindValue, Value: &encodedOld}, {Column: "metadata", Kind: domain.CellKindValue, Value: &metadata}, {Column: "occurred_at", Kind: domain.CellKindValue, Value: &occurredAt}}}, Column: "binary_data", Value: domain.CellValue{Kind: domain.CellKindValue, Value: encodedNew}},
			query:  `UPDATE "public"."row_values" SET "binary_data" = $1 WHERE "binary_data" = $2 AND "metadata" = $3 AND "occurred_at" = $4`,
			args:   []driver.NamedValue{{Ordinal: 1, Value: []byte("new")}, {Ordinal: 2, Value: []byte("old")}, {Ordinal: 3, Value: metadata}, {Ordinal: 4, Value: parsedOccurredAt}},
		},
		{
			name:      "Exec失敗を返す",
			dbType:    domain.DBTypePostgres,
			ref:       domain.TableRef{Namespace: "public", Name: "items"},
			columns:   []domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "name", DataType: "text"}},
			change:    domain.CellUpdate{Table: domain.TableRef{Namespace: "public", Name: "items"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}}, Column: "name", Value: domain.CellValue{Kind: domain.CellKindValue, Value: name}},
			query:     `UPDATE "public"."items" SET "name" = $1 WHERE "id" = $2`,
			execError: errors.New("password=secret"),
			wantError: "password=secret",
		},
		{
			name:            "更新件数取得失敗を返す",
			dbType:          domain.DBTypePostgres,
			ref:             domain.TableRef{Namespace: "public", Name: "items"},
			columns:         []domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "name", DataType: "text"}},
			change:          domain.CellUpdate{Table: domain.TableRef{Namespace: "public", Name: "items"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}}, Column: "name", Value: domain.CellValue{Kind: domain.CellKindValue, Value: name}},
			query:           `UPDATE "public"."items" SET "name" = $1 WHERE "id" = $2`,
			rowsAffectedErr: errors.New("affected rows failed"),
			wantError:       "affected rows failed",
		},
		{
			name:   "NULL更新を実行する",
			dbType: domain.DBTypePostgres,
			ref: domain.TableRef{
				Namespace: "public",
				Name:      "items",
			},
			columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					IsPrimaryKey: true,
				},
				{Name: "note", DataType: "text", Nullable: true},
			},
			change: domain.CellUpdate{Table: domain.TableRef{Namespace: "public", Name: "items"}, Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}}, Column: "note", Value: domain.CellValue{Kind: domain.CellKindNull}},
			query:  `UPDATE "public"."items" SET "note" = $1 WHERE "id" = $2`,
			args:   []driver.NamedValue{{Ordinal: 1, Value: nil}, {Ordinal: 2, Value: "1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := updateCellStructureResponses(tt.dbType, tt.columns)
			responses = append(responses, inspectionTestResponse{query: tt.query, args: tt.args, execAffected: 1, rowsAffectedErr: tt.rowsAffectedErr, err: tt.execError, errorAt: -1})
			database, scenario := newInspectionTestDatabase(t, responses)

			got, err := updateCell(context.Background(), database, tt.dbType, tt.ref, tt.change)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("updateCell() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("updateCell() error = %v", err)
			}
			if got.AffectedRows != 1 {
				t.Errorf("AffectedRows = %d, want 1", got.AffectedRows)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// テーブルセル更新構造取得応答
func updateCellStructureResponses(dbType domain.DBType, columns []domain.Column) []inspectionTestResponse {
	if dbType == domain.DBTypeMySQL {
		values := make([][]driver.Value, 0, len(columns))
		for _, column := range columns {
			values = append(values, []driver.Value{column.Name, column.DataType, column.Nullable, column.DefaultValue, column.IsGenerated, column.IsPrimaryKey, false, false})
		}

		return []inspectionTestResponse{
			{query: mysqlTableStructureColumnQuery, columns: []string{"column_name", "column_type", "is_nullable", "column_default", "extra", "column_key", "is_foreign", "is_unique"}, values: values, errorAt: -1},
			{query: mysqlTableStructureForeignKeyQuery, columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"}, values: [][]driver.Value{}, errorAt: -1},
			{query: mysqlTableStructureIndexQuery, columns: []string{"index_name", "column_name", "non_unique", "index_type"}, values: [][]driver.Value{}, errorAt: -1},
		}
	}

	values := make([][]driver.Value, 0, len(columns))
	for _, column := range columns {
		values = append(values, []driver.Value{column.Name, column.DataType, column.Nullable, column.DefaultValue, column.IsGenerated, column.IsPrimaryKey, false, false})
	}

	return []inspectionTestResponse{
		{query: postgresTableStructureColumnQuery, columns: []string{"column_name", "udt_name", "is_nullable", "column_default", "is_generated", "is_primary_key", "is_foreign_key", "is_unique"}, values: values, errorAt: -1},
		{query: postgresTableStructureForeignKeyQuery, columns: []string{"constraint_name", "table_name", "column_name", "referenced_table_name", "referenced_column_name"}, values: [][]driver.Value{}, errorAt: -1},
		{query: postgresTableStructureIndexQuery, columns: []string{"relname", "attname", "indisunique", "amname"}, values: [][]driver.Value{}, errorAt: -1},
	}
}

// テーブル行削除SQL検証
func TestDeleteRow(t *testing.T) {
	id := "1"
	note := "before"
	tests := []struct {
		name    string
		dbType  domain.DBType
		ref     domain.TableRef
		columns []domain.Column
		locator domain.RowLocator
		query   string
		args    []driver.NamedValue
		want    domain.AffectedRows
	}{
		{
			name:   "PostgreSQL主キー行を削除する",
			dbType: domain.DBTypePostgres,
			ref: domain.TableRef{
				Namespace: "public",
				Name:      "items",
			},
			columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					IsPrimaryKey: true,
				},
			},
			locator: domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindValue,
						Value:  &id,
					},
				},
			},
			query: `DELETE FROM "public"."items" WHERE "id" = $1`,
			args: []driver.NamedValue{
				{
					Ordinal: 1,
					Value:   "1",
				},
			},
			want: domain.AffectedRows{
				AffectedRows: 2,
			},
		},
		{
			name:   "MySQL主キーなし行をNULL条件で削除する",
			dbType: domain.DBTypeMySQL,
			ref: domain.TableRef{
				Namespace: "app",
				Name:      "logs",
			},
			columns: []domain.Column{
				{
					Name:     "note",
					DataType: "varchar(32)",
				},
				{
					Name:     "deleted_at",
					DataType: "timestamp",
					Nullable: true,
				},
			},
			locator: domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "note",
						Kind:   domain.CellKindValue,
						Value:  &note,
					},
					{
						Column: "deleted_at",
						Kind:   domain.CellKindNull,
					},
				},
			},
			query: "DELETE FROM `app`.`logs` WHERE `note` = ? AND `deleted_at` IS NULL",
			args: []driver.NamedValue{
				{
					Ordinal: 1,
					Value:   "before",
				},
			},
			want: domain.AffectedRows{
				AffectedRows: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := updateCellStructureResponses(tt.dbType, tt.columns)
			responses = append(responses, inspectionTestResponse{
				query:        tt.query,
				args:         tt.args,
				execAffected: tt.want.AffectedRows,
				errorAt:      -1,
			})
			database, scenario := newInspectionTestDatabase(t, responses)
			defer database.Close()

			got, err := deleteRow(context.Background(), database, tt.dbType, tt.ref, tt.locator)
			if err != nil {
				t.Fatalf("deleteRow() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deleteRow() = %#v, want %#v", got, tt.want)
			}
			assertInspectionQueriesConsumed(t, scenario)
		})
	}
}

// テーブル行削除失敗検証
func TestDeleteRowFailures(t *testing.T) {
	id := "1"
	invalidBinary := "not-base64"
	ref := domain.TableRef{
		Namespace: "public",
		Name:      "items",
	}
	tests := []struct {
		name      string
		columns   []domain.Column
		locator   domain.RowLocator
		responses []inspectionTestResponse
		wantError string
	}{
		{
			name: "構造取得失敗を返す",
			responses: []inspectionTestResponse{
				{
					query:   postgresTableStructureColumnQuery,
					err:     errors.New("structure failed"),
					errorAt: -1,
				},
			},
			wantError: "structure failed",
		},
		{
			name: "不正位置指定を返す",
			columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					IsPrimaryKey: true,
				},
			},
			locator:   domain.RowLocator{},
			wantError: domain.ErrInvalidRowInput.Error(),
		},
		{
			name: "位置指定値の変換失敗を返す",
			columns: []domain.Column{
				{
					Name:         "binary_data",
					DataType:     "bytea",
					IsPrimaryKey: true,
				},
			},
			locator: domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "binary_data",
						Kind:   domain.CellKindValue,
						Value:  &invalidBinary,
					},
				},
			},
			wantError: "invalid base64 value for binary column",
		},
		{
			name: "削除実行失敗を返す",
			columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					IsPrimaryKey: true,
				},
			},
			locator: domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindValue,
						Value:  &id,
					},
				},
			},
			responses: []inspectionTestResponse{
				{
					query:   `DELETE FROM "public"."items" WHERE "id" = $1`,
					err:     errors.New("delete failed"),
					errorAt: -1,
				},
			},
			wantError: "delete failed",
		},
		{
			name: "影響行数取得失敗を返す",
			columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					IsPrimaryKey: true,
				},
			},
			locator: domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindValue,
						Value:  &id,
					},
				},
			},
			responses: []inspectionTestResponse{
				{
					query:           `DELETE FROM "public"."items" WHERE "id" = $1`,
					rowsAffectedErr: errors.New("affected rows failed"),
					errorAt:         -1,
				},
			},
			wantError: "affected rows failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := tt.responses
			if tt.name != "構造取得失敗を返す" {
				responses = updateCellStructureResponses(domain.DBTypePostgres, tt.columns)
				responses = append(responses, tt.responses...)
			}
			database, _ := newInspectionTestDatabase(t, responses)
			defer database.Close()

			_, err := deleteRow(context.Background(), database, domain.DBTypePostgres, ref, tt.locator)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("deleteRow() error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

// テーブル行削除接続検証
func TestAppRepositoryDeleteRow(t *testing.T) {
	profile := connectionTestProfile(t, domain.DBTypePostgres)
	id := "1"
	ref := domain.TableRef{
		Namespace: "public",
		Name:      "items",
	}
	locator := domain.RowLocator{
		Values: []domain.ColumnValueInput{
			{
				Column: "id",
				Kind:   domain.CellKindValue,
				Value:  &id,
			},
		},
	}
	originalOpenDatabase := openDatabase
	t.Cleanup(func() { openDatabase = originalOpenDatabase })
	tests := []struct {
		name      string
		open      func(string, string) (*sql.DB, error)
		ctx       context.Context
		wantError string
	}{
		{
			name: "接続生成失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				return nil, errors.New("open failed")
			},
			ctx:       context.Background(),
			wantError: "open failed",
		},
		{
			name: "接続確認失敗を返す",
			open: func(string, string) (*sql.DB, error) {
				database, _ := newInspectionTestDatabase(t, nil)

				return database, nil
			},
			ctx:       canceledInspectionContext(),
			wantError: context.Canceled.Error(),
		},
		{
			name: "行を削除する",
			open: func(string, string) (*sql.DB, error) {
				responses := updateCellStructureResponses(domain.DBTypePostgres, []domain.Column{
					{
						Name:         "id",
						DataType:     "int4",
						IsPrimaryKey: true,
					},
				})
				responses = append(responses, inspectionTestResponse{
					query:        `DELETE FROM "public"."items" WHERE "id" = $1`,
					execAffected: 1,
					errorAt:      -1,
				})
				database, _ := newInspectionTestDatabase(t, responses)

				return database, nil
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openDatabase = tt.open

			got, err := NewAppRepository(nil).DeleteRow(tt.ctx, profile, "secret", ref, locator)
			if tt.wantError != "" {
				if err == nil || err.Error() != tt.wantError {
					t.Errorf("DeleteRow() error = %v, want %q", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("DeleteRow() error = %v", err)
			}
			if got.AffectedRows != 1 {
				t.Errorf("AffectedRows = %d, want 1", got.AffectedRows)
			}
		})
	}
}
