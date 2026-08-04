//go:build integration

package usecase

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
	"github.com/yukihito-jokyu/DB-checker/test/integration/db"
)

// データベーススキーマ取得結合検証
func TestInspectionUseCaseGetDatabaseSchemaIntegration(t *testing.T) {
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
			gotProfile, schema, err := NewInspectionUseCase(appRepository).GetDatabaseSchema(context.Background())
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
			_, _, err = NewInspectionUseCase(appRepository).GetDatabaseSchema(context.Background())
			if !apperr.IsCode(err, apperr.CodeSchemaLoadFailed) {
				t.Errorf("GetDatabaseSchema() error = %v, want code %q", err, apperr.CodeSchemaLoadFailed)
			}
		})
	}
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
			"CREATE TABLE schema_child (id INT NOT NULL PRIMARY KEY, parent_a INT NOT NULL, parent_b INT NOT NULL, status VARCHAR(32) NOT NULL, INDEX idx_schema_child_status (status), CONSTRAINT fk_schema_child_parent FOREIGN KEY (parent_a, parent_b) REFERENCES schema_parent (part_a, part_b))",
			"CREATE TABLE schema_cycle_left (id INT NOT NULL PRIMARY KEY, right_id INT NOT NULL)",
			"CREATE TABLE schema_cycle_right (id INT NOT NULL PRIMARY KEY, left_id INT NOT NULL, CONSTRAINT fk_schema_cycle_right_left FOREIGN KEY (left_id) REFERENCES schema_cycle_left (id))",
			"ALTER TABLE schema_cycle_left ADD CONSTRAINT fk_schema_cycle_left_right FOREIGN KEY (right_id) REFERENCES schema_cycle_right (id)",
			"CREATE TABLE schema_isolated (id INT NOT NULL PRIMARY KEY, optional_note VARCHAR(32) NULL DEFAULT 'isolated-default')",
		}
	}
	return []string{
		"CREATE SCHEMA integration_outside",
		"CREATE TABLE integration_outside.schema_outside (id INTEGER PRIMARY KEY)",
		"CREATE TABLE schema_parent (part_a INTEGER NOT NULL, part_b INTEGER NOT NULL, code VARCHAR(32) NOT NULL UNIQUE, optional_note VARCHAR(32) NULL DEFAULT 'parent-default', PRIMARY KEY (part_a, part_b))",
		"CREATE TABLE schema_child (id INTEGER PRIMARY KEY, parent_a INTEGER NOT NULL, parent_b INTEGER NOT NULL, CONSTRAINT fk_schema_child_parent FOREIGN KEY (parent_a, parent_b) REFERENCES schema_parent (part_a, part_b))",
		"CREATE TABLE schema_cycle_left (id INTEGER PRIMARY KEY, right_id INTEGER NOT NULL)",
		"CREATE TABLE schema_cycle_right (id INTEGER PRIMARY KEY, left_id INTEGER NOT NULL, CONSTRAINT fk_schema_cycle_right_left FOREIGN KEY (left_id) REFERENCES schema_cycle_left (id))",
		"ALTER TABLE schema_cycle_left ADD CONSTRAINT fk_schema_cycle_left_right FOREIGN KEY (right_id) REFERENCES schema_cycle_right (id)",
		"CREATE TABLE schema_isolated (id INTEGER PRIMARY KEY, optional_note VARCHAR(32) NULL DEFAULT 'isolated-default')",
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
			"DROP TABLE IF EXISTS schema_parent",
			"SET FOREIGN_KEY_CHECKS = 1",
		}
	}
	statements := []string{
		"DROP TABLE IF EXISTS schema_child",
		"DROP TABLE IF EXISTS schema_cycle_left CASCADE",
		"DROP TABLE IF EXISTS schema_cycle_right CASCADE",
		"DROP TABLE IF EXISTS schema_isolated",
		"DROP TABLE IF EXISTS schema_parent",
	}

	return append(statements, "DROP SCHEMA IF EXISTS integration_outside CASCADE")
}
