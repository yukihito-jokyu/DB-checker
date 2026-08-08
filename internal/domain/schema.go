package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidSchema = errors.New("invalid schema")

var ErrInvalidTableQuery = errors.New("invalid table query")

var ErrInvalidTableFilter = errors.New("invalid table filter")

var ErrInvalidRowInput = errors.New("invalid row input")

const TablePageSize = 100

type SortDirection string

const (
	SortDirectionAscending  SortDirection = "asc"
	SortDirectionDescending SortDirection = "desc"
)

type FilterOperator string

const (
	FilterOperatorEqual     FilterOperator = "="
	FilterOperatorNotEqual  FilterOperator = "!="
	FilterOperatorGreater   FilterOperator = ">"
	FilterOperatorGreaterEq FilterOperator = ">="
	FilterOperatorLess      FilterOperator = "<"
	FilterOperatorLessEq    FilterOperator = "<="
	FilterOperatorLike      FilterOperator = "LIKE"
	FilterOperatorIn        FilterOperator = "IN"
	FilterOperatorBetween   FilterOperator = "BETWEEN"
	FilterOperatorIsNull    FilterOperator = "IS NULL"
	FilterOperatorIsNotNull FilterOperator = "IS NOT NULL"
)

type FilterGroupOperator string

const (
	FilterGroupOperatorAnd FilterGroupOperator = "and"
	FilterGroupOperatorOr  FilterGroupOperator = "or"
)

type TableSort struct {
	Column    string
	Direction SortDirection
}

type TableFilter struct {
	Column   string
	Operator FilterOperator
	Values   []string
}

type FilterGroup struct {
	Operator FilterGroupOperator
	Filters  []TableFilter
	Groups   []FilterGroup
}

type TableQuery struct {
	Table   TableRef
	Page    int
	Sort    *TableSort
	Filter  *FilterGroup
	Columns []Column
}

type CellKind string

const (
	CellKindNull    CellKind = "null"
	CellKindValue   CellKind = "value"
	CellKindDefault CellKind = "default"
)

type CellValue struct {
	Kind  CellKind
	Value string
}

type ColumnValueInput struct {
	Column string
	Kind   CellKind
	Value  *string
}

type InsertRow struct {
	Table  TableRef
	Values []ColumnValueInput
}

// 行位置指定
type RowLocator struct {
	Values []ColumnValueInput
}

// セル更新
type CellUpdate struct {
	Table   TableRef
	Locator RowLocator
	Column  string
	Value   CellValue
}

type AffectedRows struct {
	AffectedRows int64
}

type TableRow struct{ Cells []CellValue }

type TableRows struct {
	Rows       []TableRow
	TotalCount int64
	Page       int
	PageSize   int
	Sort       *TableSort
	Filter     *FilterGroup
}

type Table struct {
	Namespace string
	Name      string
	Columns   []Column
}

type Column struct {
	Name         string
	DataType     string
	Nullable     bool
	DefaultValue *string
	IsPrimaryKey bool
	IsForeignKey bool
	IsUnique     bool
	IsGenerated  bool
}

type TableRef struct {
	Namespace string
	Name      string
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
	Kind    string
}

type TableStructure struct {
	Table       Table
	ForeignKeys []ForeignKey
	Indexes     []Index
}

type StatisticsStatus string

const (
	StatisticsStatusComplete    StatisticsStatus = "complete"
	StatisticsStatusTimeout     StatisticsStatus = "timeout"
	StatisticsStatusUnavailable StatisticsStatus = "unavailable"
)

type StatisticCount struct {
	Value  *int64
	Status StatisticsStatus
	Reason *string
}

type StatisticValue struct {
	Value  *string
	Status StatisticsStatus
	Reason *string
}

type ColumnStatistics struct {
	Name           string
	NullCount      StatisticCount
	DistinctCount  StatisticCount
	DuplicateCount StatisticCount
	Min            StatisticValue
	Max            StatisticValue
}

type ForeignKeyStatistics struct {
	Name                  string
	FromColumns           []string
	ToTable               string
	ToColumns             []string
	SourceRowCount        StatisticCount
	NullCount             StatisticCount
	ReferencedRowCount    StatisticCount
	MissingReferenceCount StatisticCount
}

type TableStatistics struct {
	Table       TableRef
	RowCount    StatisticCount
	ColumnCount int
	CollectedAt time.Time
	Status      StatisticsStatus
	Columns     []ColumnStatistics
	ForeignKeys []ForeignKeyStatistics
}

type ForeignKey struct {
	Name        string
	FromTable   string
	FromColumns []string
	ToTable     string
	ToColumns   []string
}

type Schema struct {
	Tables      []Table
	ForeignKeys []ForeignKey
}

// スキーマ検証
func (s Schema) Validate(namespace string) error {
	tables := make(map[string]struct{}, len(s.Tables))
	for _, table := range s.Tables {
		if table.Namespace != namespace || table.Name == "" {
			return ErrInvalidSchema
		}
		if _, found := tables[table.Name]; found {
			return ErrInvalidSchema
		}
		tables[table.Name] = struct{}{}
		if err := validateColumns(table.Columns); err != nil {
			return err
		}
	}
	for _, foreignKey := range s.ForeignKeys {
		if foreignKey.Name == "" || foreignKey.FromTable == "" || foreignKey.ToTable == "" || len(foreignKey.FromColumns) == 0 || len(foreignKey.FromColumns) != len(foreignKey.ToColumns) {
			return ErrInvalidSchema
		}
		if _, found := tables[foreignKey.FromTable]; !found {
			return ErrInvalidSchema
		}
		if _, found := tables[foreignKey.ToTable]; !found {
			return ErrInvalidSchema
		}
	}

	return nil
}

// テーブル参照生成
func NewTableRef(namespace, name string) (TableRef, error) {
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		return TableRef{}, ErrInvalidSchema
	}

	return TableRef{Namespace: namespace, Name: name}, nil
}

// テーブル問い合わせ検証
func (q TableQuery) Validate() error {
	if _, err := NewTableRef(q.Table.Namespace, q.Table.Name); err != nil || q.Page < 1 || len(q.Columns) == 0 {
		return ErrInvalidTableQuery
	}

	columns := make(map[string]Column, len(q.Columns))
	for _, column := range q.Columns {
		if column.Name == "" || column.DataType == "" {
			return ErrInvalidTableQuery
		}
		columns[column.Name] = column
	}
	if q.Sort != nil {
		if _, found := columns[q.Sort.Column]; !found || (q.Sort.Direction != SortDirectionAscending && q.Sort.Direction != SortDirectionDescending) {
			return ErrInvalidTableQuery
		}
	}
	if q.Filter != nil {
		if err := validateFilterGroup(*q.Filter, columns, 1, new(int)); err != nil {
			return err
		}
	}

	return nil
}

// フィルターグループ検証
func validateFilterGroup(group FilterGroup, columns map[string]Column, depth int, count *int) error {
	if depth > 3 || (group.Operator != FilterGroupOperatorAnd && group.Operator != FilterGroupOperatorOr) || (len(group.Filters) == 0 && len(group.Groups) == 0) {
		return ErrInvalidTableQuery
	}
	for _, filter := range group.Filters {
		*count++
		if *count > 20 {
			return ErrInvalidTableQuery
		}
		if err := validateTableFilter(filter, columns); err != nil {
			return err
		}
	}
	for _, child := range group.Groups {
		if err := validateFilterGroup(child, columns, depth+1, count); err != nil {
			return err
		}
	}

	return nil
}

// テーブルフィルター検証
func validateTableFilter(filter TableFilter, columns map[string]Column) error {
	column, found := columns[filter.Column]
	if !found {
		return ErrInvalidTableFilter
	}
	values := len(filter.Values)
	switch filter.Operator {
	case FilterOperatorIsNull, FilterOperatorIsNotNull:
		if values != 0 {
			return ErrInvalidTableQuery
		}

		return nil
	case FilterOperatorIn:
		if values < 1 || values > 100 {
			return ErrInvalidTableQuery
		}
	case FilterOperatorBetween:
		if values != 2 {
			return ErrInvalidTableQuery
		}
	case FilterOperatorEqual, FilterOperatorNotEqual, FilterOperatorGreater, FilterOperatorGreaterEq, FilterOperatorLess, FilterOperatorLessEq, FilterOperatorLike:
		if values != 1 {
			return ErrInvalidTableQuery
		}
	default:
		return ErrInvalidTableFilter
	}

	if !filterOperatorAllowed(column.DataType, filter.Operator) {
		return ErrInvalidTableFilter
	}

	return nil
}

// フィルター演算子適用可否
func filterOperatorAllowed(dataType string, operator FilterOperator) bool {
	if isBinaryOrJSONColumn(dataType) {
		return operator == FilterOperatorEqual || operator == FilterOperatorNotEqual || operator == FilterOperatorIn
	}
	if isTextColumn(dataType) {
		return operator == FilterOperatorEqual || operator == FilterOperatorNotEqual || operator == FilterOperatorLike || operator == FilterOperatorIn
	}
	if isOrderedColumn(dataType) {
		return operator == FilterOperatorEqual || operator == FilterOperatorNotEqual || operator == FilterOperatorGreater || operator == FilterOperatorGreaterEq || operator == FilterOperatorLess || operator == FilterOperatorLessEq || operator == FilterOperatorIn || operator == FilterOperatorBetween
	}

	return operator == FilterOperatorEqual || operator == FilterOperatorNotEqual || operator == FilterOperatorIn
}

// 文字列型判定
//
//nolint:nlreturn // 単一の判定結果を返す。
func isTextColumn(dataType string) bool {
	value := strings.ToLower(dataType)
	return strings.Contains(value, "char") || strings.Contains(value, "text") || strings.Contains(value, "varchar") || strings.Contains(value, "enum")
}

// 順序比較可能型判定
func isOrderedColumn(dataType string) bool {
	value := strings.ToLower(dataType)

	return strings.Contains(value, "int") || strings.Contains(value, "decimal") || strings.Contains(value, "numeric") || strings.Contains(value, "real") || strings.Contains(value, "float") || strings.Contains(value, "double") || strings.Contains(value, "date") || strings.Contains(value, "time")
}

// バイナリー・JSON型判定
func isBinaryOrJSONColumn(dataType string) bool {
	value := strings.ToLower(dataType)

	return strings.Contains(value, "blob") || strings.Contains(value, "binary") || strings.Contains(value, "bytea") || strings.Contains(value, "json")
}

// テーブル構造検証
func (s TableStructure) Validate(ref TableRef) error {
	if s.Table.Namespace != ref.Namespace || s.Table.Name != ref.Name {
		return ErrInvalidSchema
	}
	if err := validateColumns(s.Table.Columns); err != nil {
		return err
	}
	for _, foreignKey := range s.ForeignKeys {
		if foreignKey.Name == "" || foreignKey.FromTable != ref.Name || foreignKey.ToTable == "" || len(foreignKey.FromColumns) == 0 || len(foreignKey.FromColumns) != len(foreignKey.ToColumns) {
			return ErrInvalidSchema
		}
	}
	for _, index := range s.Indexes {
		if index.Name == "" || index.Kind == "" || len(index.Columns) == 0 {
			return ErrInvalidSchema
		}
	}

	return nil
}

// カラム検証
func validateColumns(columns []Column) error {
	names := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		if column.Name == "" || column.DataType == "" {
			return ErrInvalidSchema
		}
		if _, found := names[column.Name]; found {
			return fmt.Errorf("%w: duplicate column", ErrInvalidSchema)
		}
		names[column.Name] = struct{}{}
	}

	return nil
}

// 行追加入力検証
func (r InsertRow) Validate(columns []Column) error {
	if _, err := NewTableRef(r.Table.Namespace, r.Table.Name); err != nil || len(columns) == 0 {
		return ErrInvalidRowInput
	}

	colMap := make(map[string]Column, len(columns))
	for _, col := range columns {
		if col.Name == "" {
			return ErrInvalidRowInput
		}
		colMap[col.Name] = col
	}

	provided := make(map[string]ColumnValueInput, len(r.Values))
	for _, val := range r.Values {
		col, found := colMap[val.Column]
		if !found {
			return ErrInvalidRowInput
		}
		if _, duplicate := provided[val.Column]; duplicate {
			return ErrInvalidRowInput
		}

		switch val.Kind {
		case CellKindNull:
			if !col.Nullable {
				return ErrInvalidRowInput
			}
		case CellKindValue:
			if val.Value == nil {
				return ErrInvalidRowInput
			}
		case CellKindDefault:
		default:
			return ErrInvalidRowInput
		}

		provided[val.Column] = val
	}

	for _, col := range columns {
		if !col.Nullable && col.DefaultValue == nil && !col.IsGenerated {
			input, found := provided[col.Name]
			if !found || input.Kind == CellKindDefault || input.Kind == CellKindNull {
				return ErrInvalidRowInput
			}
		}
	}

	return nil
}

// セル更新入力検証
func (c CellUpdate) Validate(columns []Column) error {
	if _, err := NewTableRef(c.Table.Namespace, c.Table.Name); err != nil || c.Column == "" || len(columns) == 0 {
		return ErrInvalidRowInput
	}

	columnMap := make(map[string]Column, len(columns))
	for _, column := range columns {
		if column.Name == "" {
			return ErrInvalidRowInput
		}
		columnMap[column.Name] = column
	}

	target, found := columnMap[c.Column]
	if !found || target.IsGenerated || !validUpdateValue(c.Value, target) {
		return ErrInvalidRowInput
	}

	if err := c.Locator.Validate(c.Table, columns); err != nil {
		return ErrInvalidRowInput
	}

	return nil
}

// 行位置指定検証
func (r RowLocator) Validate(table TableRef, columns []Column) error {
	if _, err := NewTableRef(table.Namespace, table.Name); err != nil || len(columns) == 0 || len(r.Values) == 0 {
		return ErrInvalidRowInput
	}

	columnMap := make(map[string]Column, len(columns))
	primaryKeys := make(map[string]struct{})
	for _, column := range columns {
		if column.Name == "" {
			return ErrInvalidRowInput
		}
		columnMap[column.Name] = column
		if column.IsPrimaryKey {
			primaryKeys[column.Name] = struct{}{}
		}
	}

	locator := make(map[string]struct{}, len(r.Values))
	for _, value := range r.Values {
		column, found := columnMap[value.Column]
		if !found || !validLocatorValue(value, column) {
			return ErrInvalidRowInput
		}
		if _, duplicate := locator[value.Column]; duplicate {
			return ErrInvalidRowInput
		}
		locator[value.Column] = struct{}{}
	}

	if len(primaryKeys) > 0 {
		if len(locator) != len(primaryKeys) {
			return ErrInvalidRowInput
		}
		for name := range primaryKeys {
			if _, found := locator[name]; !found {
				return ErrInvalidRowInput
			}
		}

		return nil
	}
	if len(locator) != len(columnMap) {
		return ErrInvalidRowInput
	}

	return nil
}

// 更新値妥当性判定
func validUpdateValue(value CellValue, column Column) bool {
	if value.Kind == CellKindNull {
		return column.Nullable
	}
	if value.Kind == CellKindDefault {
		return true
	}

	return value.Kind == CellKindValue
}

// 行位置値妥当性判定
func validLocatorValue(value ColumnValueInput, column Column) bool {
	if value.Kind == CellKindNull {
		return column.Nullable && value.Value == nil
	}

	return value.Kind == CellKindValue && value.Value != nil
}
