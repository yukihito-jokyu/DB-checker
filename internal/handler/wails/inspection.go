package wails

import (
	"context"
	"log/slog"
	"time"

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

// テーブル統計取得
func (h *AppHandler) GetTableStatistics(table string) Response[TableStatisticsResponse] {
	h.logger.Info(context.Background(), "table statistics requested", slog.String("operation", "table_statistics_get"))

	if h.appUseCase == nil {
		err := apperr.New(apperr.CodeStatsLoadFailed)
		h.logFailureWithCode("table statistics get failed", "table_statistics_get", err)

		return Fail[TableStatisticsResponse](err)
	}

	ctx, cancel, requestID := h.beginTableStatistics()
	defer h.finishTableStatistics(requestID, cancel)

	statistics, err := h.appUseCase.GetTableStatistics(ctx, table)
	if err != nil {
		h.logFailureWithCode("table statistics get failed", "table_statistics_get", err)

		return Fail[TableStatisticsResponse](err)
	}

	return OK(toTableStatisticsResponse(statistics))
}

// テーブル行一覧取得
func (h *AppHandler) ListTableRows(request ListTableRowsRequest) Response[TableRowsResponse] {
	h.logger.Debug(context.Background(), "table rows requested", slog.String("operation", "table_rows_list"))
	if h.appUseCase == nil {
		err := apperr.New(apperr.CodeDataLoadFailed)
		h.logFailureWithCode("table rows list failed", "table_rows_list", err)

		return Fail[TableRowsResponse](err)
	}
	query := domain.TableQuery{
		Table: domain.TableRef{
			Name: request.Table,
		},
		Page:   request.Page,
		Sort:   toTableSort(request.Sort),
		Filter: toFilterGroup(request.Filter),
	}
	rows, err := h.appUseCase.ListTableRows(context.Background(), query)
	if err != nil {
		h.logFailureWithCode("table rows list failed", "table_rows_list", err)

		return Fail[TableRowsResponse](err)
	}

	return OK(toTableRowsResponse(rows))
}

// テーブル行追加
func (h *AppHandler) InsertTableRow(request InsertTableRowRequest) Response[AffectedRowsResponse] {
	h.logger.Info(context.Background(), "table row insert requested", slog.String("operation", "table_row_insert"))
	if h.appUseCase == nil {
		err := apperr.New(apperr.CodeRowAddFailed)
		h.logFailureWithCode("table row insert failed", "table_row_insert", err)

		return Fail[AffectedRowsResponse](err)
	}

	values := make([]domain.ColumnValueInput, 0, len(request.Values))
	for _, val := range request.Values {
		values = append(values, domain.ColumnValueInput{
			Column: val.Column,
			Kind:   domain.CellKind(val.Kind),
			Value:  val.Value,
		})
	}

	row := domain.InsertRow{
		Table: domain.TableRef{
			Name: request.Table,
		},
		Values: values,
	}

	affected, err := h.appUseCase.InsertTableRow(context.Background(), row)
	if err != nil {
		h.logFailureWithCode("table row insert failed", "table_row_insert", err)

		return Fail[AffectedRowsResponse](err)
	}

	return OK(AffectedRowsResponse{
		AffectedRows: affected.AffectedRows,
	})
}

// 統計要求開始
func (h *AppHandler) beginTableStatistics() (context.Context, context.CancelFunc, uint64) {
	h.statisticsMu.Lock()
	defer h.statisticsMu.Unlock()

	if h.statisticsCancel != nil {
		h.statisticsCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	h.statisticsCancel = cancel
	h.statisticsRequestID++

	return ctx, cancel, h.statisticsRequestID
}

// 統計要求終了
func (h *AppHandler) finishTableStatistics(requestID uint64, cancel context.CancelFunc) {
	h.statisticsMu.Lock()
	defer h.statisticsMu.Unlock()

	if h.statisticsRequestID == requestID {
		h.statisticsCancel = nil
	}
	cancel()
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
		tables = append(tables, DatabaseTableResponse{
			Namespace: table.Namespace,
			Name:      table.Name,
			Columns:   columns,
		})
	}
	foreignKeys := make([]DatabaseForeignKeyResponse, 0, len(schema.ForeignKeys))
	for _, foreignKey := range schema.ForeignKeys {
		foreignKeys = append(foreignKeys, DatabaseForeignKeyResponse{
			Name:        foreignKey.Name,
			FromTable:   foreignKey.FromTable,
			FromColumns: foreignKey.FromColumns,
			ToTable:     foreignKey.ToTable,
			ToColumns:   foreignKey.ToColumns,
		})
	}

	return DatabaseSchemaResponse{
		ActiveProfile: ActiveProfileResponse{
			ID:       profile.ID,
			Name:     profile.Name,
			DBType:   string(profile.DBType),
			Database: profile.Database,
			Schema:   profile.Schema,
		},
		Tables:      tables,
		ForeignKeys: foreignKeys,
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
		foreignKeys = append(foreignKeys, DatabaseForeignKeyResponse{
			Name:        foreignKey.Name,
			FromTable:   foreignKey.FromTable,
			FromColumns: foreignKey.FromColumns,
			ToTable:     foreignKey.ToTable,
			ToColumns:   foreignKey.ToColumns,
		})
	}
	indexes := make([]TableStructureIndexResponse, 0, len(structure.Indexes))
	for _, index := range structure.Indexes {
		indexes = append(indexes, TableStructureIndexResponse{
			Name:    index.Name,
			Columns: index.Columns,
			Unique:  index.Unique,
			Kind:    index.Kind,
		})
	}

	return TableStructureResponse{
		Table: TableStructureTableResponse{
			Namespace: structure.Table.Namespace,
			Name:      structure.Table.Name,
			Columns:   columns,
		},
		ForeignKeys: foreignKeys,
		Indexes:     indexes,
	}
}

// テーブル統計レスポンス変換
func toTableStatisticsResponse(statistics domain.TableStatistics) TableStatisticsResponse {
	columns := make([]ColumnStatisticsResponse, 0, len(statistics.Columns))
	for _, column := range statistics.Columns {
		columns = append(columns, ColumnStatisticsResponse{
			Name:           column.Name,
			NullCount:      toStatisticCountResponse(column.NullCount),
			DistinctCount:  toStatisticCountResponse(column.DistinctCount),
			DuplicateCount: toStatisticCountResponse(column.DuplicateCount),
			Min:            toStatisticValueResponse(column.Min),
			Max:            toStatisticValueResponse(column.Max),
		})
	}
	foreignKeys := make([]ForeignKeyStatisticsResponse, 0, len(statistics.ForeignKeys))
	for _, foreignKey := range statistics.ForeignKeys {
		foreignKeys = append(foreignKeys, ForeignKeyStatisticsResponse{
			Name:                  foreignKey.Name,
			FromColumns:           foreignKey.FromColumns,
			ToTable:               foreignKey.ToTable,
			ToColumns:             foreignKey.ToColumns,
			SourceRowCount:        toStatisticCountResponse(foreignKey.SourceRowCount),
			NullCount:             toStatisticCountResponse(foreignKey.NullCount),
			ReferencedRowCount:    toStatisticCountResponse(foreignKey.ReferencedRowCount),
			MissingReferenceCount: toStatisticCountResponse(foreignKey.MissingReferenceCount),
		})
	}

	var collectedAt *string
	if !statistics.CollectedAt.IsZero() {
		formatted := statistics.CollectedAt.Format(time.RFC3339Nano)
		collectedAt = &formatted
	}

	return TableStatisticsResponse{
		Table:       statistics.Table.Name,
		RowCount:    toStatisticCountResponse(statistics.RowCount),
		ColumnCount: statistics.ColumnCount,
		CollectedAt: collectedAt,
		Status:      string(statistics.Status),
		Columns:     columns,
		ForeignKeys: foreignKeys,
	}
}

// 件数統計レスポンス変換
func toStatisticCountResponse(statistic domain.StatisticCount) StatisticCountResponse {
	return StatisticCountResponse{
		Value:  statistic.Value,
		Status: string(statistic.Status),
		Reason: statistic.Reason,
	}
}

// 値統計レスポンス変換
func toStatisticValueResponse(statistic domain.StatisticValue) StatisticValueResponse {
	return StatisticValueResponse{
		Value:  statistic.Value,
		Status: string(statistic.Status),
		Reason: statistic.Reason,
	}
}

// 並び替え変換
//
//nolint:nlreturn // nilと変換結果を返すだけの変換関数。
func toTableSort(request *TableSortRequest) *domain.TableSort {
	if request == nil {
		return nil
	}
	return &domain.TableSort{
		Column:    request.Column,
		Direction: domain.SortDirection(request.Direction),
	}
}

// フィルター変換
//
//nolint:nlreturn // 再帰的な変換を連続して扱う。
func toFilterGroup(request *FilterGroupRequest) *domain.FilterGroup {
	if request == nil {
		return nil
	}
	filters := make([]domain.TableFilter, 0, len(request.Filters))
	for _, filter := range request.Filters {
		filters = append(filters, domain.TableFilter{
			Column:   filter.Column,
			Operator: domain.FilterOperator(filter.Operator),
			Values:   filter.Values,
		})
	}
	groups := make([]domain.FilterGroup, 0, len(request.Groups))
	for _, child := range request.Groups {
		if group := toFilterGroup(&child); group != nil {
			groups = append(groups, *group)
		}
	}
	return &domain.FilterGroup{
		Operator: domain.FilterGroupOperator(request.Operator),
		Filters:  filters,
		Groups:   groups,
	}
}

// テーブル行レスポンス変換
//
//nolint:nlreturn // 行とセルを連続して変換する。
func toTableRowsResponse(rows domain.TableRows) TableRowsResponse {
	result := TableRowsResponse{
		Rows:       make([]TableRowResponse, 0, len(rows.Rows)),
		TotalCount: rows.TotalCount,
		Page:       rows.Page,
		PageSize:   rows.PageSize,
		Sort:       toTableSortResponse(rows.Sort),
		Filter:     toFilterGroupResponse(rows.Filter),
	}
	for _, row := range rows.Rows {
		cells := make([]TableCellResponse, 0, len(row.Cells))
		for _, cell := range row.Cells {
			cells = append(cells, TableCellResponse{
				Kind:  string(cell.Kind),
				Value: cell.Value,
			})
		}
		result.Rows = append(result.Rows, TableRowResponse{
			Cells: cells,
		})
	}
	return result
}

// 並び替えレスポンス変換
//
//nolint:nlreturn // nilと変換結果を返すだけの変換関数。
func toTableSortResponse(sort *domain.TableSort) *TableSortRequest {
	if sort == nil {
		return nil
	}
	return &TableSortRequest{
		Column:    sort.Column,
		Direction: string(sort.Direction),
	}
}

// フィルターレスポンス変換
//
//nolint:nlreturn // 再帰的な変換を連続して扱う。
func toFilterGroupResponse(group *domain.FilterGroup) *FilterGroupRequest {
	if group == nil {
		return nil
	}
	filters := make([]TableFilterRequest, 0, len(group.Filters))
	for _, filter := range group.Filters {
		filters = append(filters, TableFilterRequest{
			Column:   filter.Column,
			Operator: string(filter.Operator),
			Values:   filter.Values,
		})
	}
	groups := make([]FilterGroupRequest, 0, len(group.Groups))
	for index := range group.Groups {
		groups = append(groups, *toFilterGroupResponse(&group.Groups[index]))
	}
	return &FilterGroupRequest{Operator: string(group.Operator), Filters: filters, Groups: groups}
}
