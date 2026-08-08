//go:build integration

package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
	"github.com/yukihito-jokyu/DB-checker/test/integration/db"
)

// データベーススキーマ取得結合検証
func TestAppUseCaseGetDatabaseSchemaIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			database := integrationDatabase(t, target)
			defer database.Close()
			adminDatabase := integrationAdminDatabase(t, target)
			if adminDatabase != nil {
				defer adminDatabase.Close()
			}
			defer integrationSchemaCleanup(t, database, adminDatabase, target.Kind)
			integrationSchemaSeed(t, database, adminDatabase, target.Kind)

			appRepository, profile, activeID := integrationInspectionRepository(t, target)
			gotProfile, schema, err := NewAppUseCase(appRepository).GetDatabaseSchema(context.Background())
			if err != nil {
				t.Fatalf("GetDatabaseSchema() error = %v", err)
			}
			if gotProfile != profile {
				t.Errorf("GetDatabaseSchema() profile = %#v, want %#v", gotProfile, profile)
			}
			assertIntegrationSchema(t, schema, profile)

			invalidProfile := profile
			invalidProfile.Port = 1
			if err := appRepository.SaveProfiles([]domain.Profile{invalidProfile}, &activeID); err != nil {
				t.Fatalf("SaveProfiles() error = %v", err)
			}
			_, _, err = NewAppUseCase(appRepository).GetDatabaseSchema(context.Background())
			if !apperr.IsCode(err, apperr.CodeSchemaLoadFailed) {
				t.Errorf("GetDatabaseSchema() error = %v, want code %q", err, apperr.CodeSchemaLoadFailed)
			}
		})
	}
}

// テーブル構造取得結合検証
func TestAppUseCaseGetTableStructureIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			database := integrationDatabase(t, target)
			defer database.Close()
			adminDatabase := integrationAdminDatabase(t, target)
			if adminDatabase != nil {
				defer adminDatabase.Close()
			}
			defer integrationSchemaCleanup(t, database, adminDatabase, target.Kind)
			integrationSchemaSeed(t, database, adminDatabase, target.Kind)

			appRepository, _, _ := integrationInspectionRepository(t, target)
			structure, err := NewAppUseCase(appRepository).GetTableStructure(context.Background(), "schema_child")
			if err != nil {
				t.Fatalf("GetTableStructure() error = %v", err)
			}
			if structure.Table.Name != "schema_child" || len(structure.Table.Columns) == 0 {
				t.Errorf("Table = %#v, want schema_child with columns", structure.Table)
			}
			assertIntegrationForeignKey(t, structure.ForeignKeys, "fk_schema_child_parent", "schema_child", []string{"parent_a", "parent_b"}, "schema_parent", []string{"part_a", "part_b"})
			assertIntegrationIndexes(t, target.Kind, structure.Indexes)
			isolated, err := NewAppUseCase(appRepository).GetTableStructure(context.Background(), "schema_isolated")
			if err != nil {
				t.Fatalf("GetTableStructure() isolated error = %v", err)
			}
			if !integrationDetailedColumnsFound(isolated.Table.Columns) {
				t.Errorf("Columns = %#v, want default and generated columns", isolated.Table.Columns)
			}
			_, err = NewAppUseCase(appRepository).GetTableStructure(context.Background(), "missing_table")
			if !apperr.IsCode(err, apperr.CodeSchemaLoadFailed) {
				t.Errorf("GetTableStructure() missing error = %v, want code %q", err, apperr.CodeSchemaLoadFailed)
			}
		})
	}
}

// テーブル統計取得結合検証
func TestAppUseCaseGetTableStatisticsIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			database := integrationDatabase(t, target)
			defer database.Close()
			adminDatabase := integrationAdminDatabase(t, target)
			if adminDatabase != nil {
				defer adminDatabase.Close()
			}
			defer integrationSchemaCleanup(t, database, adminDatabase, target.Kind)
			integrationSchemaSeed(t, database, adminDatabase, target.Kind)
			integrationStatisticsSeed(t, database, target.Kind)

			appRepository, _, _ := integrationInspectionRepository(t, target)
			statistics, err := NewAppUseCase(appRepository).GetTableStatistics(context.Background(), "schema_child")
			if err != nil {
				t.Fatalf("GetTableStatistics() error = %v", err)
			}
			if statistics.Status != domain.StatisticsStatusComplete {
				t.Errorf("Status = %q, want %q", statistics.Status, domain.StatisticsStatusComplete)
			}
			if statistics.RowCount.Value == nil || *statistics.RowCount.Value != 3 || statistics.ColumnCount != 6 {
				t.Errorf("statistics = %#v, want rowCount 3 and columnCount 6", statistics)
			}
			if len(statistics.Columns) != 6 || len(statistics.ForeignKeys) != 1 {
				t.Fatalf("statistics = %#v, want 6 columns and 1 foreign key", statistics)
			}
			columns := integrationColumnStatisticsByName(statistics.Columns)
			status := columns["status"]
			if status.DistinctCount.Value == nil || *status.DistinctCount.Value != 2 || status.DuplicateCount.Value == nil || *status.DuplicateCount.Value != 1 || status.Min.Value == nil || *status.Min.Value != "closed" || status.Max.Value == nil || *status.Max.Value != "open" {
				t.Errorf("status statistics = %#v, want duplicate and min/max values", status)
			}
			note := columns["note"]
			if note.NullCount.Value == nil || *note.NullCount.Value != 1 {
				t.Errorf("note statistics = %#v, want one NULL", note)
			}
			metadata := columns["metadata"]
			if metadata.Min.Status != domain.StatisticsStatusUnavailable || metadata.Min.Reason == nil || *metadata.Min.Reason != "unsupported data type" || metadata.Max.Status != domain.StatisticsStatusUnavailable || metadata.Max.Reason == nil || *metadata.Max.Reason != "unsupported data type" {
				t.Errorf("metadata statistics = %#v, want unavailable min/max", metadata)
			}
			foreignKey := statistics.ForeignKeys[0]
			if foreignKey.SourceRowCount.Value == nil || *foreignKey.SourceRowCount.Value != 3 || foreignKey.NullCount.Value == nil || *foreignKey.NullCount.Value != 0 || foreignKey.ReferencedRowCount.Value == nil || *foreignKey.ReferencedRowCount.Value != 3 || foreignKey.MissingReferenceCount.Value == nil || *foreignKey.MissingReferenceCount.Value != 0 {
				t.Errorf("ForeignKey = %#v, want three valid references", foreignKey)
			}
		})
	}
}

// テーブル行一覧結合検証
func TestAppUseCaseListTableRowsIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			database := integrationDatabase(t, target)
			defer database.Close()
			adminDatabase := integrationAdminDatabase(t, target)
			if adminDatabase != nil {
				defer adminDatabase.Close()
			}
			defer integrationSchemaCleanup(t, database, adminDatabase, target.Kind)
			integrationSchemaSeed(t, database, adminDatabase, target.Kind)
			integrationStatisticsSeed(t, database, target.Kind)

			appRepository, _, _ := integrationInspectionRepository(t, target)
			useCase := NewAppUseCase(appRepository)
			tests := []struct {
				name        string
				query       domain.TableQuery
				wantCount   int64
				wantRows    int
				wantCode    apperr.Code
				verifyTypes bool
			}{
				{
					name: "主キーの降順を返す",
					query: domain.TableQuery{
						Table: domain.TableRef{
							Name: "schema_child",
						},
						Page: 1,
						Sort: &domain.TableSort{
							Column:    "id",
							Direction: domain.SortDirectionDescending,
						},
					},
					wantCount: 3,
					wantRows:  3,
				},
				{
					name: "ANDフィルターを返す",
					query: domain.TableQuery{
						Table: domain.TableRef{
							Name: "schema_child",
						},
						Page: 1,
						Filter: &domain.FilterGroup{
							Operator: domain.FilterGroupOperatorAnd,
							Filters: []domain.TableFilter{
								{
									Column:   "status",
									Operator: domain.FilterOperatorEqual,
									Values:   []string{"open"},
								},
								{
									Column:   "id",
									Operator: domain.FilterOperatorGreater,
									Values:   []string{"1"},
								},
							},
						},
					},
					wantCount: 1,
					wantRows:  1,
				},
				{
					name: "ORフィルターを返す",
					query: domain.TableQuery{
						Table: domain.TableRef{
							Name: "schema_child",
						},
						Page: 1,
						Filter: &domain.FilterGroup{
							Operator: domain.FilterGroupOperatorOr,
							Filters: []domain.TableFilter{
								{
									Column:   "status",
									Operator: domain.FilterOperatorEqual,
									Values:   []string{"closed"},
								},
								{
									Column:   "note",
									Operator: domain.FilterOperatorIsNull,
								},
							},
						},
					},
					wantCount: 2,
					wantRows:  2,
				},
				{
					name: "主キーなしの100件ページを返す",
					query: domain.TableQuery{
						Table: domain.TableRef{
							Name: "schema_row_values",
						},
						Page: 1,
					},
					wantCount: 101,
					wantRows:  100,
				},
				{
					name: "NULLとJSONとバイナリーと日時を返す",
					query: domain.TableQuery{
						Table: domain.TableRef{
							Name: "schema_row_values",
						},
						Page: 1,
						Sort: &domain.TableSort{
							Column:    "id",
							Direction: domain.SortDirectionAscending,
						},
					},
					wantCount:   101,
					wantRows:    100,
					verifyTypes: true,
				},
				{
					name: "最終ページ超過に空行を返す",
					query: domain.TableQuery{
						Table: domain.TableRef{
							Name: "schema_row_values",
						},
						Page: 3,
						Sort: &domain.TableSort{
							Column:    "id",
							Direction: domain.SortDirectionAscending,
						},
					},
					wantCount: 101,
					wantRows:  0,
				},
				{
					name: "JSONのLIKEを適用失敗として返す",
					query: domain.TableQuery{
						Table: domain.TableRef{
							Name: "schema_row_values",
						},
						Page: 1,
						Filter: &domain.FilterGroup{
							Operator: domain.FilterGroupOperatorAnd,
							Filters: []domain.TableFilter{
								{
									Column:   "metadata",
									Operator: domain.FilterOperatorLike,
									Values:   []string{"%key%"},
								},
							},
						},
					},
					wantCode: apperr.CodeFilterApplyFailed,
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					rows, err := useCase.ListTableRows(context.Background(), tt.query)
					if tt.wantCode != "" {
						if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
							t.Errorf("ListTableRows() error code = %q, want %q", gotCode, tt.wantCode)
						}
						return
					}
					if err != nil {
						t.Fatalf("ListTableRows() error = %v", err)
					}
					if rows.TotalCount != tt.wantCount || len(rows.Rows) != tt.wantRows || rows.PageSize != domain.TablePageSize {
						t.Errorf("rows = %#v, want count %d rows %d and page size %d", rows, tt.wantCount, tt.wantRows, domain.TablePageSize)
					}
					if tt.verifyTypes {
						assertIntegrationRowValueTypes(t, rows)
					}
				})
			}
		})
	}
}

// テーブル行追加結合検証
func TestAppUseCaseInsertTableRowIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			database := integrationDatabase(t, target)
			defer database.Close()
			adminDatabase := integrationAdminDatabase(t, target)
			if adminDatabase != nil {
				defer adminDatabase.Close()
			}
			defer integrationSchemaCleanup(t, database, adminDatabase, target.Kind)
			integrationSchemaSeed(t, database, adminDatabase, target.Kind)

			appRepository, _, _ := integrationInspectionRepository(t, target)
			useCase := NewAppUseCase(appRepository)

			name := "Inserted User"
			insertedID := "3"
			row := domain.InsertRow{
				Table: domain.TableRef{Name: "schema_child"},
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindValue,
						Value:  &insertedID,
					},
					{
						Column: "parent_a",
						Kind:   domain.CellKindValue,
						Value:  integrationStringPtr("1"),
					},
					{
						Column: "parent_b",
						Kind:   domain.CellKindValue,
						Value:  integrationStringPtr("10"),
					},
					{
						Column: "status",
						Kind:   domain.CellKindValue,
						Value:  &name,
					},
					{
						Column: "note",
						Kind:   domain.CellKindNull,
					},
					{
						Column: "metadata",
						Kind:   domain.CellKindValue,
						Value:  integrationStringPtr(`{"new": true}`),
					},
				},
			}

			// parent 行を事前に投入
			if _, err := database.Exec("INSERT INTO schema_parent (part_a, part_b, code) VALUES (1, 10, 'parent-a')"); err != nil {
				t.Fatalf("Exec parent seed error = %v", err)
			}

			affected, err := useCase.InsertTableRow(context.Background(), row)
			if err != nil {
				t.Fatalf("InsertTableRow() error = %v", err)
			}
			if affected.AffectedRows != 1 {
				t.Errorf("AffectedRows = %d, want 1", affected.AffectedRows)
			}

		})
	}
}

// テーブル行削除結合検証
func TestAppUseCaseDeleteTableRowIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			database := integrationDatabase(t, target)
			defer database.Close()
			adminDatabase := integrationAdminDatabase(t, target)
			if adminDatabase != nil {
				defer adminDatabase.Close()
			}
			defer integrationSchemaCleanup(t, database, adminDatabase, target.Kind)
			integrationSchemaSeed(t, database, adminDatabase, target.Kind)

			if _, err := database.Exec("INSERT INTO schema_parent (part_a, part_b, code) VALUES (1, 10, 'delete-parent')"); err != nil {
				t.Fatalf("Exec parent seed error = %v", err)
			}
			if _, err := database.Exec("INSERT INTO schema_child (id, parent_a, parent_b, status, note, metadata) VALUES (1, 1, 10, 'delete', NULL, '{}'), (2, 1, 10, 'keep', 'note', '{}')"); err != nil {
				t.Fatalf("Exec child seed error = %v", err)
			}
			if _, err := database.Exec("CREATE TABLE delete_no_pk (id INTEGER NOT NULL, note VARCHAR(32) NULL)"); err != nil {
				t.Fatalf("CREATE TABLE delete_no_pk error = %v", err)
			}
			defer func() {
				if _, err := database.Exec("DROP TABLE IF EXISTS delete_no_pk"); err != nil {
					t.Errorf("DROP TABLE delete_no_pk error = %v", err)
				}
			}()
			if _, err := database.Exec("INSERT INTO delete_no_pk (id, note) VALUES (1, NULL), (2, 'keep')"); err != nil {
				t.Fatalf("Exec delete_no_pk seed error = %v", err)
			}

			appRepository, _, _ := integrationInspectionRepository(t, target)
			useCase := NewAppUseCase(appRepository)
			id := "1"
			affected, err := useCase.DeleteTableRow(context.Background(), "schema_child", domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindValue,
						Value:  &id,
					},
				},
			})
			if err != nil {
				t.Fatalf("DeleteTableRow() primary key error = %v", err)
			}
			if affected.AffectedRows != 1 {
				t.Errorf("primary key affected rows = %d, want 1", affected.AffectedRows)
			}

			var childCount int
			if err := database.QueryRow("SELECT COUNT(*) FROM schema_child WHERE id = 1").Scan(&childCount); err != nil {
				t.Fatalf("QueryRow deleted child error = %v", err)
			}
			if childCount != 0 {
				t.Errorf("deleted child count = %d, want 0", childCount)
			}

			rowID := "1"
			nullLocator := domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindValue,
						Value:  &rowID,
					},
					{
						Column: "note",
						Kind:   domain.CellKindNull,
					},
				},
			}
			affected, err = useCase.DeleteTableRow(context.Background(), "delete_no_pk", nullLocator)
			if err != nil {
				t.Fatalf("DeleteTableRow() NULL locator error = %v", err)
			}
			if affected.AffectedRows != 1 {
				t.Errorf("NULL locator affected rows = %d, want 1", affected.AffectedRows)
			}

			affected, err = useCase.DeleteTableRow(context.Background(), "schema_child", domain.RowLocator{
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindValue,
						Value:  integrationStringPtr("999"),
					},
				},
			})
			if err != nil {
				t.Fatalf("DeleteTableRow() missing row error = %v", err)
			}
			if affected.AffectedRows != 0 {
				t.Errorf("missing row affected rows = %d, want 0", affected.AffectedRows)
			}
		})
	}
}

// テーブルセル更新結合検証
func TestAppUseCaseUpdateTableCellIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			database := integrationDatabase(t, target)
			defer database.Close()
			adminDatabase := integrationAdminDatabase(t, target)
			if adminDatabase != nil {
				defer adminDatabase.Close()
			}
			defer integrationSchemaCleanup(t, database, adminDatabase, target.Kind)
			integrationSchemaSeed(t, database, adminDatabase, target.Kind)
			integrationStatisticsSeed(t, database, target.Kind)

			appRepository, _, _ := integrationInspectionRepository(t, target)
			id := "1"
			change := domain.CellUpdate{
				Table:   domain.TableRef{Name: "schema_child"},
				Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}},
				Column:  "status",
				Value:   domain.CellValue{Kind: domain.CellKindValue, Value: "updated"},
			}

			affected, err := NewAppUseCase(appRepository).UpdateTableCell(context.Background(), change)
			if err != nil {
				t.Fatalf("UpdateTableCell() error = %v", err)
			}
			if affected.AffectedRows != 1 {
				t.Errorf("AffectedRows = %d, want 1", affected.AffectedRows)
			}

			var status string
			if err := database.QueryRow("SELECT status FROM schema_child WHERE id = 1").Scan(&status); err != nil {
				t.Fatalf("QueryRow updated status error = %v", err)
			}
			if status != "updated" {
				t.Errorf("status = %q, want %q", status, "updated")
			}

			if _, err := database.Exec("INSERT INTO schema_isolated (id, optional_note) VALUES (1, 'custom-note')"); err != nil {
				t.Fatalf("Exec default update seed error = %v", err)
			}
			defaultChange := domain.CellUpdate{
				Table:   domain.TableRef{Name: "schema_isolated"},
				Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{Column: "id", Kind: domain.CellKindValue, Value: &id}}},
				Column:  "optional_note",
				Value:   domain.CellValue{Kind: domain.CellKindDefault},
			}
			affected, err = NewAppUseCase(appRepository).UpdateTableCell(context.Background(), defaultChange)
			if err != nil {
				t.Fatalf("UpdateTableCell() default error = %v", err)
			}
			if affected.AffectedRows != 1 {
				t.Errorf("default AffectedRows = %d, want 1", affected.AffectedRows)
			}

			var optionalNote string
			if err := database.QueryRow("SELECT optional_note FROM schema_isolated WHERE id = 1").Scan(&optionalNote); err != nil {
				t.Fatalf("QueryRow default updated optional_note error = %v", err)
			}
			if optionalNote != "isolated-default" {
				t.Errorf("optional_note = %q, want %q", optionalNote, "isolated-default")
			}
		})
	}
}

// 検証用文字列ポインター取得
func integrationStringPtr(value string) *string {
	return &value
}

// 行値種別検証
func assertIntegrationRowValueTypes(t *testing.T, rows domain.TableRows) {
	t.Helper()

	if len(rows.Rows) == 0 || len(rows.Rows[0].Cells) != 6 {
		t.Fatalf("Rows = %#v, want first row with six cells", rows)
	}
	cells := rows.Rows[0].Cells
	if cells[0].Value != "1" || cells[2].Kind != domain.CellKindNull || cells[3].Value != `{"key": 1}` || cells[4].Value != "AQI=" || cells[5].Value != "2026-08-05T12:34:56Z" {
		t.Errorf("Rows[0].Cells = %#v, want converted NULL, JSON, binary and RFC3339 time", cells)
	}
}

// 統計検証データ投入
func integrationStatisticsSeed(t *testing.T, database *sql.DB, kind db.Kind) {
	t.Helper()

	statements := []string{
		"INSERT INTO schema_parent (part_a, part_b, code) VALUES (1, 10, 'parent-a'), (2, 20, 'parent-b')",
		"INSERT INTO schema_child (id, parent_a, parent_b, status, note, metadata) VALUES (1, 1, 10, 'open', NULL, '{\"key\": \"one\"}'), (2, 2, 20, 'open', 'same', '{\"key\": \"two\"}'), (3, 1, 10, 'closed', 'same', '{\"key\": \"three\"}')",
	}
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("statistics seed %s Exec() error = %v", kind, err)
		}
	}
	for id := 1; id <= 101; id++ {
		statement := fmt.Sprintf("INSERT INTO schema_row_values (id, name, note, metadata, binary_data, occurred_at) VALUES (%d, 'row-%d', NULL, '{\"key\": %d}', %s, '2026-08-05 12:34:56')", id, id, id, integrationBinaryLiteral(kind))
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("row values seed %s Exec() error = %v", kind, err)
		}
	}
}

// バイナリー値リテラル取得
func integrationBinaryLiteral(kind db.Kind) string {
	if kind == db.MySQL {
		return "X'0102'"
	}

	return "decode('0102', 'hex')"
}

// カラム統計名前別取得
func integrationColumnStatisticsByName(columns []domain.ColumnStatistics) map[string]domain.ColumnStatistics {
	result := make(map[string]domain.ColumnStatistics, len(columns))
	for _, column := range columns {
		result[column.Name] = column
	}

	return result
}

// 詳細カラム属性検出
func integrationDetailedColumnsFound(columns []domain.Column) bool {
	defaultFound := false
	generatedFound := false
	for _, column := range columns {
		if column.DefaultValue != nil {
			defaultFound = true
		}
		if column.IsGenerated {
			generatedFound = true
		}
	}

	return defaultFound && generatedFound
}

// スキーマ検証用リポジトリ生成
func integrationInspectionRepository(t *testing.T, target db.Target) (*integrationAppRepository, domain.Profile, string) {
	t.Helper()

	store := config.NewStore(t.TempDir())
	if err := store.Initialize(); err != nil {
		t.Fatalf("Store.Initialize() error = %v", err)
	}
	credentials := &integrationCredentialRepository{credentials: map[string]string{}}
	appRepository := &integrationAppRepository{
		AppRepository: repository.NewAppRepository(store),
		credentials:   credentials,
	}
	draft := integrationProfileDraft(t, target)
	profiles, _, err := NewAppUseCase(appRepository).SaveConnectionProfile(context.Background(), draft)
	if err != nil {
		t.Fatalf("SaveConnectionProfile() error = %v", err)
	}
	activeID := profiles[0].ID
	if err := appRepository.SaveProfiles(profiles, &activeID); err != nil {
		t.Fatalf("SaveProfiles() error = %v", err)
	}

	return appRepository, profiles[0], activeID
}

// 取得スキーマ全項目検証
func assertIntegrationSchema(t *testing.T, schema domain.Schema, profile domain.Profile) {
	t.Helper()

	wantNamespace := profile.Schema
	if profile.DBType == domain.DBTypeMySQL {
		wantNamespace = profile.Database
	}
	tables := integrationTablesByName(schema.Tables)
	for _, name := range []string{"schema_parent", "schema_child", "schema_cycle_left", "schema_cycle_right", "schema_isolated"} {
		if _, found := tables[name]; !found {
			t.Errorf("Tables[%q] = missing", name)
		}
	}
	for _, table := range schema.Tables {
		if table.Namespace != wantNamespace {
			t.Errorf("Table.Namespace = %q, want %q", table.Namespace, wantNamespace)
		}
	}
	if _, found := tables["schema_outside"]; found {
		t.Error("Tables contains target namespace outside table schema_outside")
	}
	assertIntegrationColumn(t, tables, "schema_parent", "part_a", false, true, false, true)
	assertIntegrationColumn(t, tables, "schema_parent", "code", false, false, false, true)
	assertIntegrationColumn(t, tables, "schema_parent", "optional_note", true, false, false, false)
	assertIntegrationColumn(t, tables, "schema_child", "parent_a", false, false, true, false)
	assertIntegrationColumn(t, tables, "schema_isolated", "optional_note", true, false, false, false)
	assertIntegrationColumn(t, tables, "schema_cycle_left", "right_id", false, false, true, false)
	assertIntegrationColumn(t, tables, "schema_cycle_right", "left_id", false, false, true, false)
	if profile.DBType == domain.DBTypeMySQL {
		assertIntegrationColumn(t, tables, "schema_child", "status", false, false, false, false)
	}
	for _, table := range schema.Tables {
		for _, column := range table.Columns {
			if column.Name == "" || column.DataType == "" {
				t.Errorf("Column = %#v, want name and data type", column)
			}
		}
	}
	assertIntegrationForeignKey(t, schema.ForeignKeys, "fk_schema_child_parent", "schema_child", []string{"parent_a", "parent_b"}, "schema_parent", []string{"part_a", "part_b"})
	assertIntegrationForeignKey(t, schema.ForeignKeys, "fk_schema_cycle_left_right", "schema_cycle_left", []string{"right_id"}, "schema_cycle_right", []string{"id"})
	assertIntegrationForeignKey(t, schema.ForeignKeys, "fk_schema_cycle_right_left", "schema_cycle_right", []string{"left_id"}, "schema_cycle_left", []string{"id"})
}

// テーブル名別索引生成
func integrationTablesByName(tables []domain.Table) map[string]domain.Table {
	result := make(map[string]domain.Table, len(tables))
	for _, table := range tables {
		result[table.Name] = table
	}

	return result
}

// カラム属性検証
func assertIntegrationColumn(t *testing.T, tables map[string]domain.Table, tableName, columnName string, nullable, primaryKey, foreignKey, unique bool) {
	t.Helper()

	table, found := tables[tableName]
	if !found {
		t.Errorf("Table %q = missing", tableName)

		return
	}
	for _, column := range table.Columns {
		if column.Name != columnName {
			continue
		}
		if column.Nullable != nullable || column.IsPrimaryKey != primaryKey || column.IsForeignKey != foreignKey || column.IsUnique != unique {
			t.Errorf("Column %s.%s = %#v, want nullable=%v primaryKey=%v foreignKey=%v unique=%v", tableName, columnName, column, nullable, primaryKey, foreignKey, unique)
		}

		return
	}
	t.Errorf("Column %s.%s = missing", tableName, columnName)
}

// 外部キー属性検証
func assertIntegrationForeignKey(t *testing.T, foreignKeys []domain.ForeignKey, name, fromTable string, fromColumns []string, toTable string, toColumns []string) {
	t.Helper()

	for _, foreignKey := range foreignKeys {
		if foreignKey.Name != name {
			continue
		}
		if foreignKey.FromTable != fromTable || foreignKey.ToTable != toTable || !equalIntegrationStrings(foreignKey.FromColumns, fromColumns) || !equalIntegrationStrings(foreignKey.ToColumns, toColumns) {
			t.Errorf("ForeignKey %q = %#v, want %s.%v -> %s.%v", name, foreignKey, fromTable, fromColumns, toTable, toColumns)
		}

		return
	}
	t.Errorf("ForeignKey %q = missing", name)
}

// インデックス属性検証
func assertIntegrationIndexes(t *testing.T, kind db.Kind, indexes []domain.Index) {
	t.Helper()

	primaryKeyName := "schema_child_pkey"
	if kind == db.MySQL {
		primaryKeyName = "PRIMARY"
	}
	assertIntegrationIndex(t, indexes, primaryKeyName, []string{"id"}, true, "btree")
	assertIntegrationIndex(t, indexes, "idx_schema_child_status", []string{"status"}, false, "btree")
	assertIntegrationIndex(t, indexes, "idx_schema_child_parent_status", []string{"parent_b", "status"}, false, "btree")
}

// 個別インデックス属性検証
func assertIntegrationIndex(t *testing.T, indexes []domain.Index, name string, columns []string, unique bool, kind string) {
	t.Helper()

	for _, index := range indexes {
		if index.Name != name {
			continue
		}
		if !equalIntegrationStrings(index.Columns, columns) || index.Unique != unique || index.Kind != kind {
			t.Errorf("Index %q = %#v, want columns=%v unique=%v kind=%q", name, index, columns, unique, kind)
		}

		return
	}
	t.Errorf("Index %q = missing", name)
}

// 文字列スライス一致判定
func equalIntegrationStrings(left, right []string) bool {
	return reflect.DeepEqual(left, right)
}

// 結合テストDB接続
func integrationDatabase(t *testing.T, target db.Target) *sql.DB {
	t.Helper()
	database, err := sql.Open(target.DriverName, target.DSN)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if err := database.Ping(); err != nil {
		t.Fatalf("database.Ping() error = %v", err)
	}

	return database
}

// MySQL管理接続
func integrationAdminDatabase(t *testing.T, target db.Target) *sql.DB {
	t.Helper()

	if target.Kind != db.MySQL {
		return nil
	}
	database, err := sql.Open(target.DriverName, target.AdminDSN)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if err := database.Ping(); err != nil {
		t.Fatalf("database.Ping() error = %v", err)
	}

	return database
}

// 結合テストスキーマ投入
func integrationSchemaSeed(t *testing.T, database, adminDatabase *sql.DB, kind db.Kind) {
	t.Helper()

	if kind == db.MySQL {
		if _, err := adminDatabase.Exec("CREATE DATABASE integration_outside"); err != nil {
			t.Fatalf("CREATE DATABASE integration_outside error = %v", err)
		}
		if _, err := adminDatabase.Exec("CREATE TABLE integration_outside.schema_outside (id INT PRIMARY KEY)"); err != nil {
			t.Fatalf("CREATE TABLE integration_outside.schema_outside error = %v", err)
		}
	}
	for _, statement := range integrationSchemaStatements(kind) {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed statement %q error = %v", statement, err)
		}
	}
}

// 結合テストスキーマ削除
func integrationSchemaCleanup(t *testing.T, database, adminDatabase *sql.DB, kind db.Kind) {
	t.Helper()
	for _, statement := range integrationCleanupStatements(kind) {
		if _, err := database.Exec(statement); err != nil {
			t.Errorf("cleanup statement %q error = %v", statement, err)
		}
	}
	if kind == db.MySQL {
		if _, err := adminDatabase.Exec("DROP DATABASE IF EXISTS integration_outside"); err != nil {
			t.Errorf("DROP DATABASE integration_outside error = %v", err)
		}
	}
}

// 結合テストDDL取得
func integrationSchemaStatements(kind db.Kind) []string {
	if kind == db.MySQL {
		return []string{
			"CREATE TABLE schema_parent (part_a INT NOT NULL, part_b INT NOT NULL, code VARCHAR(32) NOT NULL UNIQUE, optional_note VARCHAR(32) NULL DEFAULT 'parent-default', PRIMARY KEY (part_a, part_b))",
			"CREATE TABLE schema_child (id INT NOT NULL PRIMARY KEY, parent_a INT NOT NULL, parent_b INT NOT NULL, status VARCHAR(32) NOT NULL, note VARCHAR(32) NULL, metadata JSON NOT NULL, INDEX idx_schema_child_status (status), INDEX idx_schema_child_parent_status (parent_b, status), CONSTRAINT fk_schema_child_parent FOREIGN KEY (parent_a, parent_b) REFERENCES schema_parent (part_a, part_b))",
			"CREATE TABLE schema_cycle_left (id INT NOT NULL PRIMARY KEY, right_id INT NOT NULL)",
			"CREATE TABLE schema_cycle_right (id INT NOT NULL PRIMARY KEY, left_id INT NOT NULL, CONSTRAINT fk_schema_cycle_right_left FOREIGN KEY (left_id) REFERENCES schema_cycle_left (id))",
			"ALTER TABLE schema_cycle_left ADD CONSTRAINT fk_schema_cycle_left_right FOREIGN KEY (right_id) REFERENCES schema_cycle_right (id)",
			"CREATE TABLE schema_isolated (id INT NOT NULL PRIMARY KEY, optional_note VARCHAR(32) NULL DEFAULT 'isolated-default', generated_value INT GENERATED ALWAYS AS (id + 1) STORED)",
			"CREATE TABLE schema_row_values (id INT NOT NULL, name VARCHAR(32) NOT NULL, note VARCHAR(32) NULL, metadata JSON NOT NULL, binary_data BLOB NOT NULL, occurred_at DATETIME NOT NULL)",
		}
	}
	return []string{
		"CREATE SCHEMA integration_outside",
		"CREATE TABLE integration_outside.schema_outside (id INTEGER PRIMARY KEY)",
		"CREATE TABLE schema_parent (part_a INTEGER NOT NULL, part_b INTEGER NOT NULL, code VARCHAR(32) NOT NULL UNIQUE, optional_note VARCHAR(32) NULL DEFAULT 'parent-default', PRIMARY KEY (part_a, part_b))",
		"CREATE TABLE schema_child (id INTEGER PRIMARY KEY, parent_a INTEGER NOT NULL, parent_b INTEGER NOT NULL, status VARCHAR(32) NOT NULL, note VARCHAR(32) NULL, metadata JSON NOT NULL, CONSTRAINT fk_schema_child_parent FOREIGN KEY (parent_a, parent_b) REFERENCES schema_parent (part_a, part_b))",
		"CREATE INDEX idx_schema_child_status ON schema_child (status)",
		"CREATE INDEX idx_schema_child_parent_status ON schema_child (parent_b, status)",
		"CREATE TABLE schema_cycle_left (id INTEGER PRIMARY KEY, right_id INTEGER NOT NULL)",
		"CREATE TABLE schema_cycle_right (id INTEGER PRIMARY KEY, left_id INTEGER NOT NULL, CONSTRAINT fk_schema_cycle_right_left FOREIGN KEY (left_id) REFERENCES schema_cycle_left (id))",
		"ALTER TABLE schema_cycle_left ADD CONSTRAINT fk_schema_cycle_left_right FOREIGN KEY (right_id) REFERENCES schema_cycle_right (id)",
		"CREATE TABLE schema_isolated (id INTEGER PRIMARY KEY, optional_note VARCHAR(32) NULL DEFAULT 'isolated-default', generated_value INTEGER GENERATED ALWAYS AS (id + 1) STORED)",
		"CREATE TABLE schema_row_values (id INTEGER NOT NULL, name VARCHAR(32) NOT NULL, note VARCHAR(32) NULL, metadata JSON NOT NULL, binary_data BYTEA NOT NULL, occurred_at TIMESTAMP NOT NULL)",
	}
}

// 結合テスト削除DDL取得
func integrationCleanupStatements(kind db.Kind) []string {
	if kind == db.MySQL {
		return []string{
			"SET FOREIGN_KEY_CHECKS = 0",
			"DROP TABLE IF EXISTS schema_child",
			"DROP TABLE IF EXISTS schema_cycle_right",
			"DROP TABLE IF EXISTS schema_cycle_left",
			"DROP TABLE IF EXISTS schema_isolated",
			"DROP TABLE IF EXISTS schema_row_values",
			"DROP TABLE IF EXISTS schema_parent",
			"SET FOREIGN_KEY_CHECKS = 1",
		}
	}
	statements := []string{
		"DROP TABLE IF EXISTS schema_child",
		"DROP TABLE IF EXISTS schema_cycle_left CASCADE",
		"DROP TABLE IF EXISTS schema_cycle_right CASCADE",
		"DROP TABLE IF EXISTS schema_isolated",
		"DROP TABLE IF EXISTS schema_row_values",
		"DROP TABLE IF EXISTS schema_parent",
	}

	return append(statements, "DROP SCHEMA IF EXISTS integration_outside CASCADE")
}
