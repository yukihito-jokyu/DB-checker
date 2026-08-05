package wails

import (
	"context"
	"log/slog"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// データベーススキーマ取得
func (h *AppHandler) GetDatabaseSchema() Response[DatabaseSchemaResponse] {
	h.logger.Info(context.Background(), "database schema requested", slog.String("operation", "database_schema_get"))

	if h.appUseCase == nil {
		err := apperr.New(apperr.CodeSchemaLoadFailed)
		h.logFailureWithCode("database schema get failed", "database_schema_get", err)

		return Fail[DatabaseSchemaResponse](err)
	}

	profile, schema, err := h.appUseCase.GetDatabaseSchema(context.Background())
	if err != nil {
		h.logFailureWithCode("database schema get failed", "database_schema_get", err)

		return Fail[DatabaseSchemaResponse](err)
	}

	return OK(toDatabaseSchemaResponse(profile, schema))
}

// テーブル構造取得
func (h *AppHandler) GetTableStructure(table string) Response[TableStructureResponse] {
	h.logger.Info(context.Background(), "table structure requested", slog.String("operation", "table_structure_get"))

	if h.appUseCase == nil {
		err := apperr.New(apperr.CodeSchemaLoadFailed)
		h.logFailureWithCode("table structure get failed", "table_structure_get", err)

		return Fail[TableStructureResponse](err)
	}

	structure, err := h.appUseCase.GetTableStructure(context.Background(), table)
	if err != nil {
		h.logFailureWithCode("table structure get failed", "table_structure_get", err)

		return Fail[TableStructureResponse](err)
	}

	return OK(toTableStructureResponse(structure))
}

// データベーススキーマレスポンス変換
func toDatabaseSchemaResponse(profile domain.Profile, schema domain.Schema) DatabaseSchemaResponse {
	tables := make([]DatabaseTableResponse, 0, len(schema.Tables))
	for _, table := range schema.Tables {
		columns := make([]DatabaseColumnResponse, 0, len(table.Columns))
		for _, column := range table.Columns {
			columns = append(columns, DatabaseColumnResponse{
				Name:         column.Name,
				DataType:     column.DataType,
				Nullable:     column.Nullable,
				IsPrimaryKey: column.IsPrimaryKey,
				IsForeignKey: column.IsForeignKey,
				IsUnique:     column.IsUnique,
			})
		}
		tables = append(tables, DatabaseTableResponse{Namespace: table.Namespace, Name: table.Name, Columns: columns})
	}
	foreignKeys := make([]DatabaseForeignKeyResponse, 0, len(schema.ForeignKeys))
	for _, foreignKey := range schema.ForeignKeys {
		foreignKeys = append(foreignKeys, DatabaseForeignKeyResponse{Name: foreignKey.Name, FromTable: foreignKey.FromTable, FromColumns: foreignKey.FromColumns, ToTable: foreignKey.ToTable, ToColumns: foreignKey.ToColumns})
	}

	return DatabaseSchemaResponse{
		ActiveProfile: ActiveProfileResponse{ID: profile.ID, Name: profile.Name, DBType: string(profile.DBType), Database: profile.Database, Schema: profile.Schema},
		Tables:        tables,
		ForeignKeys:   foreignKeys,
	}
}

// テーブル構造レスポンス変換
func toTableStructureResponse(structure domain.TableStructure) TableStructureResponse {
	columns := make([]TableStructureColumnResponse, 0, len(structure.Table.Columns))
	for _, column := range structure.Table.Columns {
		columns = append(columns, TableStructureColumnResponse{
			Name:         column.Name,
			DataType:     column.DataType,
			Nullable:     column.Nullable,
			DefaultValue: column.DefaultValue,
			IsPrimaryKey: column.IsPrimaryKey,
			IsForeignKey: column.IsForeignKey,
			IsUnique:     column.IsUnique,
			IsGenerated:  column.IsGenerated,
		})
	}
	foreignKeys := make([]DatabaseForeignKeyResponse, 0, len(structure.ForeignKeys))
	for _, foreignKey := range structure.ForeignKeys {
		foreignKeys = append(foreignKeys, DatabaseForeignKeyResponse{Name: foreignKey.Name, FromTable: foreignKey.FromTable, FromColumns: foreignKey.FromColumns, ToTable: foreignKey.ToTable, ToColumns: foreignKey.ToColumns})
	}
	indexes := make([]TableStructureIndexResponse, 0, len(structure.Indexes))
	for _, index := range structure.Indexes {
		indexes = append(indexes, TableStructureIndexResponse{Name: index.Name, Columns: index.Columns, Unique: index.Unique, Kind: index.Kind})
	}

	return TableStructureResponse{
		Table:       TableStructureTableResponse{Namespace: structure.Table.Namespace, Name: structure.Table.Name, Columns: columns},
		ForeignKeys: foreignKeys,
		Indexes:     indexes,
	}
}
