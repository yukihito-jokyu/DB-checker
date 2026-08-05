package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
)

var openDatabase = sql.Open

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

	mysqlTableStructureColumnQuery = `SELECT
	c.column_name,
	c.column_type,
	c.is_nullable = 'YES',
	c.column_default,
	c.extra LIKE '%GENERATED%',
	c.column_key = 'PRI',
	EXISTS (
		SELECT 1 FROM information_schema.key_column_usage k
		WHERE k.table_schema = c.table_schema AND k.table_name = c.table_name AND k.column_name = c.column_name AND k.referenced_table_name IS NOT NULL
	),
	EXISTS (
		SELECT 1 FROM information_schema.statistics s
		WHERE s.table_schema = c.table_schema AND s.table_name = c.table_name AND s.column_name = c.column_name AND s.non_unique = 0
	)
FROM information_schema.columns c
WHERE c.table_schema = ? AND c.table_name = ?
ORDER BY c.ordinal_position`

	mysqlTableStructureForeignKeyQuery = `SELECT
	k.constraint_name, k.table_name, k.column_name, k.referenced_table_name, k.referenced_column_name
FROM information_schema.key_column_usage k
WHERE k.table_schema = ? AND k.table_name = ? AND k.referenced_table_schema = k.table_schema
ORDER BY k.constraint_name, k.ordinal_position`

	mysqlTableStructureIndexQuery = `SELECT
	s.index_name, s.column_name, s.non_unique = 0, LOWER(s.index_type)
FROM information_schema.statistics s
WHERE s.table_schema = ? AND s.table_name = ?
ORDER BY s.index_name, s.seq_in_index`

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

	postgresTableStructureColumnQuery = `SELECT
	c.column_name,
	c.udt_name,
	c.is_nullable = 'YES',
	c.column_default,
	c.is_generated = 'ALWAYS',
	EXISTS (
		SELECT 1 FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage k ON tc.constraint_catalog = k.constraint_catalog AND tc.constraint_schema = k.constraint_schema AND tc.constraint_name = k.constraint_name
		WHERE tc.table_schema = c.table_schema AND tc.table_name = c.table_name AND tc.constraint_type = 'PRIMARY KEY' AND k.column_name = c.column_name
	),
	EXISTS (
		SELECT 1 FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage k ON tc.constraint_catalog = k.constraint_catalog AND tc.constraint_schema = k.constraint_schema AND tc.constraint_name = k.constraint_name
		WHERE tc.table_schema = c.table_schema AND tc.table_name = c.table_name AND tc.constraint_type = 'FOREIGN KEY' AND k.column_name = c.column_name
	),
	EXISTS (
		SELECT 1 FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage k ON tc.constraint_catalog = k.constraint_catalog AND tc.constraint_schema = k.constraint_schema AND tc.constraint_name = k.constraint_name
		WHERE tc.table_schema = c.table_schema AND tc.table_name = c.table_name AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE') AND k.column_name = c.column_name
	)
FROM information_schema.columns c
WHERE c.table_schema = $1 AND c.table_name = $2
ORDER BY c.ordinal_position`

	postgresTableStructureForeignKeyQuery = `SELECT
	k.constraint_name, k.table_name, k.column_name, ref.table_name, ref.column_name
FROM information_schema.key_column_usage k
JOIN information_schema.referential_constraints rc ON rc.constraint_catalog = k.constraint_catalog AND rc.constraint_schema = k.constraint_schema AND rc.constraint_name = k.constraint_name
JOIN information_schema.key_column_usage ref ON ref.constraint_catalog = rc.unique_constraint_catalog AND ref.constraint_schema = rc.unique_constraint_schema AND ref.constraint_name = rc.unique_constraint_name AND ref.ordinal_position = k.position_in_unique_constraint
WHERE k.table_schema = $1 AND k.table_name = $2 AND ref.table_schema = $1
ORDER BY k.constraint_name, k.ordinal_position`

	postgresTableStructureIndexQuery = `SELECT
	i.relname, a.attname, ix.indisunique, am.amname
FROM pg_index ix
JOIN pg_class t ON t.oid = ix.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN pg_class i ON i.oid = ix.indexrelid
JOIN pg_am am ON am.oid = i.relam
JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY key(attnum, ordinality) ON true
JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = key.attnum
WHERE n.nspname = $1 AND t.relname = $2 AND ix.indpred IS NULL AND ix.indexprs IS NULL
ORDER BY i.relname, key.ordinality`
)

// スキーマメタデータ取得
func (r *AppRepository) InspectSchema(ctx context.Context, profile domain.Profile, password string) (domain.Schema, error) {
	driverName, dsn := connectionDSN(profile, password)
	database, err := openDatabase(driverName, dsn)
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

// テーブル構造メタデータ取得
func (r *AppRepository) InspectTableStructure(ctx context.Context, profile domain.Profile, password string, ref domain.TableRef) (domain.TableStructure, error) {
	driverName, dsn := connectionDSN(profile, password)
	database, err := openDatabase(driverName, dsn)
	if err != nil {
		return domain.TableStructure{}, err
	}
	defer database.Close()

	if err := database.PingContext(ctx); err != nil {
		return domain.TableStructure{}, err
	}

	return inspectTableStructure(ctx, database, profile.DBType, ref)
}

// テーブル統計取得
func (r *AppRepository) InspectTableStatistics(ctx context.Context, profile domain.Profile, password string, ref domain.TableRef) (domain.TableStatistics, error) {
	statistics := unavailableTableStatistics(ref)
	driverName, dsn := connectionDSN(profile, password)
	database, err := openDatabase(driverName, dsn)
	if err != nil {
		return statistics, err
	}
	defer database.Close()

	if err := database.PingContext(ctx); err != nil {
		return statistics, err
	}

	return inspectTableStatistics(ctx, database, profile.DBType, ref)
}

// テーブル統計問い合わせ実行
func inspectTableStatistics(ctx context.Context, database *sql.DB, dbType domain.DBType, ref domain.TableRef) (domain.TableStatistics, error) {
	statistics := unavailableTableStatistics(ref)
	structure, err := inspectTableStructure(ctx, database, dbType, ref)
	if err != nil {
		return statistics, err
	}

	statistics = newTableStatistics(ref, structure)
	quotedTable := qualifiedIdentifier(dbType, ref.Namespace, ref.Name)
	rowCount, err := queryCount(ctx, database, "SELECT COUNT(*) FROM "+quotedTable)
	if err != nil {
		return statistics, err
	}
	statistics.RowCount = availableCount(rowCount)

	for index, column := range structure.Table.Columns {
		if err := ctx.Err(); err != nil {
			return statistics, err
		}

		columnStatistics, err := inspectColumnStatistics(ctx, database, dbType, quotedTable, column)
		if err != nil {
			return statistics, err
		}
		statistics.Columns[index] = columnStatistics
	}

	for index, foreignKey := range structure.ForeignKeys {
		if err := ctx.Err(); err != nil {
			return statistics, err
		}

		foreignKeyStatistics, err := inspectForeignKeyStatistics(ctx, database, dbType, ref.Namespace, ref.Name, foreignKey)
		if err != nil {
			return statistics, err
		}
		statistics.ForeignKeys[index] = foreignKeyStatistics
	}

	statistics.Status = domain.StatisticsStatusComplete
	statistics.CollectedAt = time.Now().UTC()

	return statistics, nil
}

// 未取得テーブル統計初期化
func unavailableTableStatistics(ref domain.TableRef) domain.TableStatistics {
	return domain.TableStatistics{
		Table:    ref,
		RowCount: unavailableCount(),
		Status:   domain.StatisticsStatusTimeout,
	}
}

// テーブル統計初期化
func newTableStatistics(ref domain.TableRef, structure domain.TableStructure) domain.TableStatistics {
	columns := make([]domain.ColumnStatistics, 0, len(structure.Table.Columns))
	for _, column := range structure.Table.Columns {
		columns = append(columns, domain.ColumnStatistics{
			Name:           column.Name,
			NullCount:      unavailableCount(),
			DistinctCount:  unavailableCount(),
			DuplicateCount: unavailableCount(),
			Min:            unavailableValue("not collected"),
			Max:            unavailableValue("not collected"),
		})
	}
	foreignKeys := make([]domain.ForeignKeyStatistics, 0, len(structure.ForeignKeys))
	for _, foreignKey := range structure.ForeignKeys {
		foreignKeys = append(foreignKeys, domain.ForeignKeyStatistics{
			Name:                  foreignKey.Name,
			FromColumns:           foreignKey.FromColumns,
			ToTable:               foreignKey.ToTable,
			ToColumns:             foreignKey.ToColumns,
			SourceRowCount:        unavailableCount(),
			NullCount:             unavailableCount(),
			ReferencedRowCount:    unavailableCount(),
			MissingReferenceCount: unavailableCount(),
		})
	}

	return domain.TableStatistics{
		Table:       ref,
		RowCount:    unavailableCount(),
		ColumnCount: len(structure.Table.Columns),
		Status:      domain.StatisticsStatusTimeout,
		Columns:     columns,
		ForeignKeys: foreignKeys,
	}
}

// カラム統計取得
func inspectColumnStatistics(ctx context.Context, database *sql.DB, dbType domain.DBType, quotedTable string, column domain.Column) (domain.ColumnStatistics, error) {
	quotedColumn := quoteIdentifier(dbType, column.Name)
	var nullCount int64
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) - COUNT("+quotedColumn+") FROM "+quotedTable).Scan(&nullCount); err != nil {
		return domain.ColumnStatistics{}, err
	}
	if !supportsMinMax(column.DataType) {
		return domain.ColumnStatistics{
			Name:           column.Name,
			NullCount:      availableCount(nullCount),
			DistinctCount:  unavailableCount(),
			DuplicateCount: unavailableCount(),
			Min:            unavailableValue("unsupported data type"),
			Max:            unavailableValue("unsupported data type"),
		}, nil
	}

	query := "SELECT COUNT(DISTINCT " + quotedColumn + "), MIN(" + quotedColumn + "), MAX(" + quotedColumn + ") FROM " + quotedTable
	var distinctCount int64
	var minValue, maxValue sql.NullString
	if err := database.QueryRowContext(ctx, query).Scan(&distinctCount, &minValue, &maxValue); err != nil {
		return domain.ColumnStatistics{}, err
	}

	duplicateCount := int64(0)
	if distinctCount > 0 {
		// 重複数は NULL を除く行数から distinct 数を引く。
		var nonNullCount int64
		if err := database.QueryRowContext(ctx, "SELECT COUNT("+quotedColumn+") FROM "+quotedTable).Scan(&nonNullCount); err != nil {
			return domain.ColumnStatistics{}, err
		}
		duplicateCount = nonNullCount - distinctCount
	}

	statistics := domain.ColumnStatistics{Name: column.Name, NullCount: availableCount(nullCount), DistinctCount: availableCount(distinctCount), DuplicateCount: availableCount(duplicateCount)}
	statistics.Min = availableValue(minValue)
	statistics.Max = availableValue(maxValue)

	return statistics, nil
}

// 外部キー統計取得
func inspectForeignKeyStatistics(ctx context.Context, database *sql.DB, dbType domain.DBType, namespace, table string, foreignKey domain.ForeignKey) (domain.ForeignKeyStatistics, error) {
	source := qualifiedIdentifier(dbType, namespace, table)
	target := qualifiedIdentifier(dbType, namespace, foreignKey.ToTable)
	joins := make([]string, 0, len(foreignKey.FromColumns))
	nulls := make([]string, 0, len(foreignKey.FromColumns))
	for index, fromColumn := range foreignKey.FromColumns {
		from := "src." + quoteIdentifier(dbType, fromColumn)
		joins = append(joins, from+" = dst."+quoteIdentifier(dbType, foreignKey.ToColumns[index]))
		nulls = append(nulls, from+" IS NULL")
	}
	query := "SELECT COUNT(*), COALESCE(SUM(CASE WHEN " + strings.Join(nulls, " OR ") + " THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN " + strings.Join(nulls, " OR ") + " THEN 0 WHEN dst." + quoteIdentifier(dbType, foreignKey.ToColumns[0]) + " IS NULL THEN 0 ELSE 1 END), 0) FROM " + source + " src LEFT JOIN " + target + " dst ON " + strings.Join(joins, " AND ")
	var sourceCount, nullCount, referencedCount int64
	if err := database.QueryRowContext(ctx, query).Scan(&sourceCount, &nullCount, &referencedCount); err != nil {
		return domain.ForeignKeyStatistics{}, err
	}

	return domain.ForeignKeyStatistics{Name: foreignKey.Name, FromColumns: foreignKey.FromColumns, ToTable: foreignKey.ToTable, ToColumns: foreignKey.ToColumns, SourceRowCount: availableCount(sourceCount), NullCount: availableCount(nullCount), ReferencedRowCount: availableCount(referencedCount), MissingReferenceCount: availableCount(sourceCount - nullCount - referencedCount)}, nil
}

// 件数問い合わせ実行
func queryCount(ctx context.Context, database *sql.DB, query string) (int64, error) {
	var count int64
	if err := database.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// 利用可能件数生成
func availableCount(value int64) domain.StatisticCount {
	return domain.StatisticCount{Value: &value, Status: domain.StatisticsStatusComplete}
}

// 未取得件数生成
func unavailableCount() domain.StatisticCount {
	reason := "not collected"

	return domain.StatisticCount{Status: domain.StatisticsStatusUnavailable, Reason: &reason}
}

// 利用可能値生成
func availableValue(value sql.NullString) domain.StatisticValue {
	if !value.Valid {
		return domain.StatisticValue{Status: domain.StatisticsStatusComplete}
	}

	return domain.StatisticValue{Value: &value.String, Status: domain.StatisticsStatusComplete}
}

// 未取得値生成
func unavailableValue(reason string) domain.StatisticValue {
	return domain.StatisticValue{Status: domain.StatisticsStatusUnavailable, Reason: &reason}
}

// 最小最大値対応判定
func supportsMinMax(dataType string) bool {
	typeName := strings.ToLower(strings.TrimSpace(dataType))
	if index := strings.IndexByte(typeName, '('); index >= 0 {
		typeName = strings.TrimSpace(typeName[:index])
	}
	if index := strings.IndexByte(typeName, ' '); index >= 0 {
		typeName = strings.TrimSpace(typeName[:index])
	}

	switch typeName {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint", "int2", "int4", "int8", "serial", "bigserial", "decimal", "numeric", "real", "float", "float4", "float8", "double", "date", "time", "timetz", "timestamp", "timestamptz", "datetime", "year", "char", "varchar", "text", "tinytext", "mediumtext", "longtext", "enum", "set", "citext", "name":
		return true
	default:
		return false
	}
}

// 識別子引用
func quoteIdentifier(dbType domain.DBType, identifier string) string {
	if dbType == domain.DBTypeMySQL {
		return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
	}

	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// 修飾識別子生成
func qualifiedIdentifier(dbType domain.DBType, namespace, table string) string {
	return quoteIdentifier(dbType, namespace) + "." + quoteIdentifier(dbType, table)
}

// テーブル構造問い合わせ実行
func inspectTableStructure(ctx context.Context, database *sql.DB, dbType domain.DBType, ref domain.TableRef) (domain.TableStructure, error) {
	columnQuery, foreignKeyQuery, indexQuery := tableStructureQueries(dbType)
	columns, err := inspectDetailedColumns(ctx, database, columnQuery, ref.Namespace, ref.Name)
	if err != nil {
		return domain.TableStructure{}, err
	}
	if len(columns) == 0 {
		return domain.TableStructure{}, sql.ErrNoRows
	}
	foreignKeys, err := inspectTableForeignKeys(ctx, database, foreignKeyQuery, ref.Namespace, ref.Name)
	if err != nil {
		return domain.TableStructure{}, err
	}
	indexes, err := inspectIndexes(ctx, database, indexQuery, ref.Namespace, ref.Name)
	if err != nil {
		return domain.TableStructure{}, err
	}

	return domain.TableStructure{Table: domain.Table{Namespace: ref.Namespace, Name: ref.Name, Columns: columns}, ForeignKeys: foreignKeys, Indexes: indexes}, nil
}

// 詳細カラムメタデータ取得
func inspectDetailedColumns(ctx context.Context, database *sql.DB, query, namespace, table string) ([]domain.Column, error) {
	rows, err := database.QueryContext(ctx, query, namespace, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := []domain.Column{}
	for rows.Next() {
		var column domain.Column
		if err := rows.Scan(&column.Name, &column.DataType, &column.Nullable, &column.DefaultValue, &column.IsGenerated, &column.IsPrimaryKey, &column.IsForeignKey, &column.IsUnique); err != nil {
			return nil, err
		}

		columns = append(columns, column)
	}

	return columns, rows.Err()
}

// テーブル外部キーメタデータ取得
func inspectTableForeignKeys(ctx context.Context, database *sql.DB, query, namespace, table string) ([]domain.ForeignKey, error) {
	rows, err := database.QueryContext(ctx, query, namespace, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectForeignKeys(rows)
}

type indexRow struct {
	name   string
	column string
	unique bool
	kind   string
}

// インデックスメタデータ取得
func inspectIndexes(ctx context.Context, database *sql.DB, query, namespace, table string) ([]domain.Index, error) {
	rows, err := database.QueryContext(ctx, query, namespace, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := map[string]*domain.Index{}
	for rows.Next() {
		var row indexRow
		if err := rows.Scan(&row.name, &row.column, &row.unique, &row.kind); err != nil {
			return nil, err
		}
		index := grouped[row.name]
		if index == nil {
			index = &domain.Index{Name: row.name, Unique: row.unique, Kind: row.kind}
			grouped[row.name] = index
		}
		index.Columns = append(index.Columns, row.column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	indexes := make([]domain.Index, 0, len(grouped))
	for _, index := range grouped {
		indexes = append(indexes, *index)
	}
	sort.Slice(indexes, func(left, right int) bool { return indexes[left].Name < indexes[right].Name })

	return indexes, nil
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

	return collectForeignKeys(rows)
}

// 外部キー行集約
func collectForeignKeys(rows *sql.Rows) ([]domain.ForeignKey, error) {
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

// テーブル構造問い合わせ取得
func tableStructureQueries(dbType domain.DBType) (string, string, string) {
	if dbType == domain.DBTypeMySQL {
		return mysqlTableStructureColumnQuery, mysqlTableStructureForeignKeyQuery, mysqlTableStructureIndexQuery
	}

	return postgresTableStructureColumnQuery, postgresTableStructureForeignKeyQuery, postgresTableStructureIndexQuery
}

// テーブル行取得
func (r *AppRepository) ListRows(ctx context.Context, profile domain.Profile, password string, query domain.TableQuery) (domain.TableRows, error) {
	driverName, dsn := connectionDSN(profile, password)
	database, err := openDatabase(driverName, dsn)
	if err != nil {
		return domain.TableRows{}, err
	}
	defer database.Close()

	if err := database.PingContext(ctx); err != nil {
		return domain.TableRows{}, err
	}

	return listRows(ctx, database, profile.DBType, query)
}

// テーブル行問い合わせ実行
func listRows(ctx context.Context, database *sql.DB, dbType domain.DBType, query domain.TableQuery) (domain.TableRows, error) {
	if err := query.Validate(); err != nil {
		return domain.TableRows{}, err
	}

	where, args := tableRowsWhere(dbType, query.Filter)
	table := qualifiedIdentifier(dbType, query.Table.Namespace, query.Table.Name)
	countQuery := "SELECT COUNT(*) FROM " + table + where
	var totalCount int64
	if err := database.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return domain.TableRows{}, err
	}

	columns := make([]string, 0, len(query.Columns))
	for _, column := range query.Columns {
		columns = append(columns, quoteIdentifier(dbType, column.Name))
	}
	//nolint:gosec // 識別子はTableQuery.Validateで許可済み列に限定し、値はプレースホルダーで渡す。
	rowQuery := "SELECT " + strings.Join(columns, ", ") + " FROM " + table + where + tableRowsOrder(dbType, query)
	offset := (query.Page - 1) * domain.TablePageSize
	if dbType == domain.DBTypeMySQL {
		rowQuery += " LIMIT ? OFFSET ?"
		args = append(args, domain.TablePageSize, offset)
	} else {
		rowQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
		args = append(args, domain.TablePageSize, offset)
	}
	rows, err := database.QueryContext(ctx, rowQuery, args...)
	if err != nil {
		return domain.TableRows{}, err
	}
	defer rows.Close()

	result := domain.TableRows{Rows: []domain.TableRow{}, TotalCount: totalCount, Page: query.Page, PageSize: domain.TablePageSize, Sort: query.Sort, Filter: query.Filter}
	for rows.Next() {
		values := make([]any, len(query.Columns))
		destinations := make([]any, len(values))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			return domain.TableRows{}, err
		}

		row := domain.TableRow{Cells: make([]domain.CellValue, 0, len(values))}
		for index, value := range values {
			row.Cells = append(row.Cells, tableCellValue(value, query.Columns[index]))
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return domain.TableRows{}, err
	}

	return result, nil
}

// テーブル行並び替え生成
func tableRowsOrder(dbType domain.DBType, query domain.TableQuery) string {
	if query.Sort != nil {
		return " ORDER BY " + quoteIdentifier(dbType, query.Sort.Column) + " " + strings.ToUpper(string(query.Sort.Direction))
	}

	primaryKeys := make([]string, 0)
	for _, column := range query.Columns {
		if column.IsPrimaryKey {
			primaryKeys = append(primaryKeys, quoteIdentifier(dbType, column.Name)+" ASC")
		}
	}
	if len(primaryKeys) == 0 {
		return ""
	}

	return " ORDER BY " + strings.Join(primaryKeys, ", ")
}

// テーブル行条件生成
func tableRowsWhere(dbType domain.DBType, filter *domain.FilterGroup) (string, []any) {
	if filter == nil {
		return "", nil
	}

	fragment, args := tableRowsFilter(dbType, *filter, 1)

	return " WHERE " + fragment, args
}

// フィルターSQL生成
func tableRowsFilter(dbType domain.DBType, group domain.FilterGroup, parameter int) (string, []any) {
	parts := make([]string, 0, len(group.Filters)+len(group.Groups))
	args := []any{}
	for _, filter := range group.Filters {
		column := quoteIdentifier(dbType, filter.Column)
		switch filter.Operator {
		case domain.FilterOperatorIsNull, domain.FilterOperatorIsNotNull:
			parts = append(parts, column+" "+string(filter.Operator))
		case domain.FilterOperatorIn:
			placeholders := make([]string, 0, len(filter.Values))
			for _, value := range filter.Values {
				placeholders = append(placeholders, tableRowsPlaceholder(dbType, parameter))
				parameter++
				args = append(args, value)
			}
			parts = append(parts, column+" IN ("+strings.Join(placeholders, ", ")+")")
		case domain.FilterOperatorBetween:
			left := tableRowsPlaceholder(dbType, parameter)
			parameter++
			right := tableRowsPlaceholder(dbType, parameter)
			parameter++
			parts = append(parts, column+" BETWEEN "+left+" AND "+right)
			args = append(args, filter.Values[0], filter.Values[1])
		default:
			parts = append(parts, column+" "+string(filter.Operator)+" "+tableRowsPlaceholder(dbType, parameter))
			parameter++
			args = append(args, filter.Values[0])
		}
	}
	for _, child := range group.Groups {
		fragment, childArgs := tableRowsFilter(dbType, child, parameter)
		parts = append(parts, "("+fragment+")")
		args = append(args, childArgs...)
		parameter += len(childArgs)
	}

	return strings.Join(parts, " "+strings.ToUpper(string(group.Operator))+" "), args
}

// プレースホルダー生成
func tableRowsPlaceholder(dbType domain.DBType, parameter int) string {
	if dbType == domain.DBTypeMySQL {
		return "?"
	}

	return "$" + strconv.Itoa(parameter)
}

// セル値変換
func tableCellValue(value any, column domain.Column) domain.CellValue {
	if value == nil {
		return domain.CellValue{Kind: domain.CellKindNull}
	}

	dataType := strings.ToLower(column.DataType)
	if bytes, found := value.([]byte); found {
		if strings.Contains(dataType, "blob") || strings.Contains(dataType, "binary") || strings.Contains(dataType, "bytea") {
			return domain.CellValue{Kind: domain.CellKindValue, Value: base64.StdEncoding.EncodeToString(bytes)}
		}

		return domain.CellValue{Kind: domain.CellKindValue, Value: string(bytes)}
	}
	if timestamp, found := value.(time.Time); found {
		return domain.CellValue{Kind: domain.CellKindValue, Value: timestamp.Format(time.RFC3339Nano)}
	}

	text := fmt.Sprint(value)
	if strings.Contains(dataType, "json") && json.Valid([]byte(text)) {
		return domain.CellValue{Kind: domain.CellKindValue, Value: text}
	}

	return domain.CellValue{Kind: domain.CellKindValue, Value: text}
}
