package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidSchema = errors.New("invalid schema")

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
