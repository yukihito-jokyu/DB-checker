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

	if h.inspectionUseCase == nil {
		err := apperr.New(apperr.CodeSchemaLoadFailed)
		h.logFailureWithCode("database schema get failed", "database_schema_get", err)

		return Fail[DatabaseSchemaResponse](err)
	}

	profile, schema, err := h.inspectionUseCase.GetDatabaseSchema(context.Background())
	if err != nil {
		h.logFailureWithCode("database schema get failed", "database_schema_get", err)

		return Fail[DatabaseSchemaResponse](err)
	}

	return OK(toDatabaseSchemaResponse(profile, schema))
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
