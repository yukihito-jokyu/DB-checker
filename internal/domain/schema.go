package domain

import (
	"errors"
	"fmt"
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
	IsPrimaryKey bool
	IsForeignKey bool
	IsUnique     bool
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
