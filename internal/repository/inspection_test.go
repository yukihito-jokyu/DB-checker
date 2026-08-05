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

	return &inspectionTestRows{response: response, errorAt: response.errorAt}, nil
}

// テスト用接続確認
func (*inspectionTestConnection) Ping(context.Context) error { return nil }

type inspectionTestScenario struct {
	mu        sync.Mutex
	responses []inspectionTestResponse
}

type inspectionTestResponse struct {
	query   string
	columns []string
	values  [][]driver.Value
	err     error
	rowErr  error
	errorAt int
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
				Table:       domain.Table{Namespace: "public", Name: "items", Columns: []domain.Column{{Name: "id", DataType: "int4", DefaultValue: &defaultValue, IsGenerated: true, IsPrimaryKey: true, IsUnique: true}}},
				ForeignKeys: []domain.ForeignKey{},
				Indexes: []domain.Index{
					{Name: "items_name_key", Columns: []string{"name"}, Unique: true, Kind: "btree"},
					{Name: "items_pkey", Columns: []string{"id"}, Unique: true, Kind: "btree"},
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
				{Name: "fk_items_orders", FromTable: "items", FromColumns: []string{"order_id"}, ToTable: "orders", ToColumns: []string{"id"}},
				{Name: "fk_orders_users", FromTable: "orders", FromColumns: []string{"tenant_id", "user_id"}, ToTable: "users", ToColumns: []string{"tenant_id", "id"}},
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
