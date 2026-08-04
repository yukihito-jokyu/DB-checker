package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

const (
	mysqlSchemaTableQuery = `SELECT
	table_name
FROM information_schema.tables
WHERE table_schema = ?
	AND table_type = 'BASE TABLE'
ORDER BY table_name`

	mysqlSchemaColumnQuery = `SELECT
	c.column_name,
	c.column_type,
	c.is_nullable = 'YES',
	c.column_key = 'PRI',
	EXISTS (
		SELECT 1
		FROM information_schema.key_column_usage k
		WHERE k.table_schema = c.table_schema
			AND k.table_name = c.table_name
			AND k.column_name = c.column_name
			AND k.referenced_table_name IS NOT NULL
	),
	EXISTS (
		SELECT 1
		FROM information_schema.statistics s
		WHERE s.table_schema = c.table_schema
			AND s.table_name = c.table_name
			AND s.column_name = c.column_name
			AND s.non_unique = 0
	)
FROM information_schema.columns c
WHERE c.table_schema = ?
	AND c.table_name = ?
ORDER BY c.ordinal_position`

	mysqlSchemaForeignKeyQuery = `SELECT
	k.constraint_name,
	k.table_name,
	k.column_name,
	k.referenced_table_name,
	k.referenced_column_name
FROM information_schema.key_column_usage k
WHERE k.table_schema = ?
	AND k.referenced_table_schema = k.table_schema
ORDER BY
	k.table_name,
	k.constraint_name,
	k.ordinal_position`

	postgresSchemaTableQuery = `SELECT
	table_name
FROM information_schema.tables
WHERE table_schema = $1
	AND table_type = 'BASE TABLE'
ORDER BY table_name`

	postgresSchemaColumnQuery = `SELECT
	c.column_name,
	c.udt_name,
	c.is_nullable = 'YES',
	EXISTS (
		SELECT 1
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage k
			ON tc.constraint_catalog = k.constraint_catalog
			AND tc.constraint_schema = k.constraint_schema
			AND tc.constraint_name = k.constraint_name
		WHERE tc.table_schema = c.table_schema
			AND tc.table_name = c.table_name
			AND tc.constraint_type = 'PRIMARY KEY'
			AND k.column_name = c.column_name
	),
	EXISTS (
		SELECT 1
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage k
			ON tc.constraint_catalog = k.constraint_catalog
			AND tc.constraint_schema = k.constraint_schema
			AND tc.constraint_name = k.constraint_name
		WHERE tc.table_schema = c.table_schema
			AND tc.table_name = c.table_name
			AND tc.constraint_type = 'FOREIGN KEY'
			AND k.column_name = c.column_name
	),
	EXISTS (
		SELECT 1
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage k
			ON tc.constraint_catalog = k.constraint_catalog
			AND tc.constraint_schema = k.constraint_schema
			AND tc.constraint_name = k.constraint_name
		WHERE tc.table_schema = c.table_schema
			AND tc.table_name = c.table_name
			AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
			AND k.column_name = c.column_name
	)
FROM information_schema.columns c
WHERE c.table_schema = $1
	AND c.table_name = $2
ORDER BY c.ordinal_position`

	postgresSchemaForeignKeyQuery = `SELECT
	k.constraint_name,
	k.table_name,
	k.column_name,
	ref.table_name,
	ref.column_name
FROM information_schema.key_column_usage k
JOIN information_schema.referential_constraints rc
	ON rc.constraint_catalog = k.constraint_catalog
	AND rc.constraint_schema = k.constraint_schema
	AND rc.constraint_name = k.constraint_name
JOIN information_schema.key_column_usage ref
	ON ref.constraint_catalog = rc.unique_constraint_catalog
	AND ref.constraint_schema = rc.unique_constraint_schema
	AND ref.constraint_name = rc.unique_constraint_name
	AND ref.ordinal_position = k.position_in_unique_constraint
WHERE k.table_schema = $1
	AND ref.table_schema = $1
ORDER BY
	k.table_name,
	k.constraint_name,
	k.ordinal_position`
)

// スキーマメタデータ取得
func (r *AppRepository) InspectSchema(ctx context.Context, profile domain.Profile, password string) (domain.Schema, error) {
	driverName, dsn := connectionDSN(profile, password)
	database, err := sql.Open(driverName, dsn)
	if err != nil {
		return domain.Schema{}, err
	}
	defer database.Close()

	namespace := profile.Schema
	if profile.DBType == domain.DBTypeMySQL {
		namespace = profile.Database
	}

	if err := database.PingContext(ctx); err != nil {
		return domain.Schema{}, err
	}

	return inspectSchema(ctx, database, profile.DBType, namespace)
}

// スキーマ問い合わせ実行
func inspectSchema(ctx context.Context, database *sql.DB, dbType domain.DBType, namespace string) (domain.Schema, error) {
	tableQuery, columnQuery, foreignKeyQuery := schemaQueries(dbType)

	tableRows, err := database.QueryContext(ctx, tableQuery, namespace)
	if err != nil {
		return domain.Schema{}, err
	}
	defer tableRows.Close()

	schema := domain.Schema{}
	for tableRows.Next() {
		var name string
		if err := tableRows.Scan(&name); err != nil {
			return domain.Schema{}, err
		}

		columns, err := inspectColumns(ctx, database, columnQuery, namespace, name)
		if err != nil {
			return domain.Schema{}, err
		}

		schema.Tables = append(schema.Tables, domain.Table{Namespace: namespace, Name: name, Columns: columns})
	}
	if err := tableRows.Err(); err != nil {
		return domain.Schema{}, err
	}

	foreignKeys, err := inspectForeignKeys(ctx, database, foreignKeyQuery, namespace)
	if err != nil {
		return domain.Schema{}, err
	}

	schema.ForeignKeys = foreignKeys

	return schema, nil
}

// カラムメタデータ取得
func inspectColumns(ctx context.Context, database *sql.DB, query, namespace, table string) ([]domain.Column, error) {
	rows, err := database.QueryContext(ctx, query, namespace, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := []domain.Column{}
	for rows.Next() {
		var column domain.Column
		if err := rows.Scan(&column.Name, &column.DataType, &column.Nullable, &column.IsPrimaryKey, &column.IsForeignKey, &column.IsUnique); err != nil {
			return nil, err
		}

		columns = append(columns, column)
	}

	return columns, rows.Err()
}

type foreignKeyRow struct {
	name       string
	fromTable  string
	fromColumn string
	toTable    string
	toColumn   string
}

// 外部キーメタデータ取得
func inspectForeignKeys(ctx context.Context, database *sql.DB, query, namespace string) ([]domain.ForeignKey, error) {
	rows, err := database.QueryContext(ctx, query, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := map[string]*domain.ForeignKey{}
	for rows.Next() {
		var row foreignKeyRow
		if err := rows.Scan(&row.name, &row.fromTable, &row.fromColumn, &row.toTable, &row.toColumn); err != nil {
			return nil, err
		}

		key := row.name + "\x00" + row.fromTable
		foreignKey := grouped[key]
		if foreignKey == nil {
			foreignKey = &domain.ForeignKey{Name: row.name, FromTable: row.fromTable, ToTable: row.toTable}
			grouped[key] = foreignKey
		}

		foreignKey.FromColumns = append(foreignKey.FromColumns, row.fromColumn)
		foreignKey.ToColumns = append(foreignKey.ToColumns, row.toColumn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	foreignKeys := make([]domain.ForeignKey, 0, len(grouped))
	for _, foreignKey := range grouped {
		foreignKeys = append(foreignKeys, *foreignKey)
	}

	sort.Slice(foreignKeys, func(left, right int) bool {
		return fmt.Sprintf("%s\x00%s", foreignKeys[left].FromTable, foreignKeys[left].Name) < fmt.Sprintf("%s\x00%s", foreignKeys[right].FromTable, foreignKeys[right].Name)
	})

	return foreignKeys, nil
}

// DB種別別問い合わせ取得
func schemaQueries(dbType domain.DBType) (string, string, string) {
	if dbType == domain.DBTypeMySQL {
		return mysqlSchemaTableQuery, mysqlSchemaColumnQuery, mysqlSchemaForeignKeyQuery
	}

	return postgresSchemaTableQuery, postgresSchemaColumnQuery, postgresSchemaForeignKeyQuery
}
