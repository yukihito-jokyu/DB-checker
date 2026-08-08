package wails

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
)

type schemaRepositoryStub struct {
	profile         domain.Profile
	activeID        *string
	credential      string
	credentialFound bool
	schema          domain.Schema
	inspectErr      error
	structure       domain.TableStructure
	structureErr    error
	statistics      domain.TableStatistics
	statisticsErr   error
	statisticsFunc  func(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStatistics, error)
	rows            domain.TableRows
	rowsErr         error
	flowState       domain.FlowState
	flowStateErr    error
	affectedRows    domain.AffectedRows
	insertRowErr    error
	updateCellErr   error
	deleteRowErr    error
}

// プロファイル読込再現
func (s *schemaRepositoryStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return []domain.Profile{s.profile}, s.activeID, nil
}

// フロー状態読込再現
func (s *schemaRepositoryStub) LoadFlowState(string) (domain.FlowState, error) {
	return s.flowState, s.flowStateErr
}

// フロー状態保存再現
func (*schemaRepositoryStub) SaveFlowState(string, domain.FlowState) error { return nil }

// プロファイル保存再現
func (*schemaRepositoryStub) SaveProfiles([]domain.Profile, *string) error { return nil }

// 資格情報取得再現
func (s *schemaRepositoryStub) GetCredential(string) (string, bool, error) {
	return s.credential, s.credentialFound, nil
}

// 資格情報保存再現
func (*schemaRepositoryStub) SetCredential(string, string) error { return nil }

// 資格情報削除再現
func (*schemaRepositoryStub) DeleteCredential(string) error { return nil }

// 接続確認再現
func (*schemaRepositoryStub) CheckConnection(context.Context, domain.Profile, string) error {
	return nil
}

// スキーマ取得再現
func (s *schemaRepositoryStub) InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error) {
	return s.schema, s.inspectErr
}

// テーブル構造取得再現
func (s *schemaRepositoryStub) InspectTableStructure(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStructure, error) {
	return s.structure, s.structureErr
}

// テーブル統計取得再現
func (s *schemaRepositoryStub) InspectTableStatistics(ctx context.Context, profile domain.Profile, credential string, ref domain.TableRef) (domain.TableStatistics, error) {
	if s.statisticsFunc != nil {
		return s.statisticsFunc(ctx, profile, credential, ref)
	}

	return s.statistics, s.statisticsErr
}

// テーブル行一覧再現
func (s *schemaRepositoryStub) ListRows(context.Context, domain.Profile, string, domain.TableQuery) (domain.TableRows, error) {
	return s.rows, s.rowsErr
}

// テーブル行追加再現
func (s *schemaRepositoryStub) InsertRow(context.Context, domain.Profile, string, domain.TableRef, domain.InsertRow) (domain.AffectedRows, error) {
	return s.affectedRows, s.insertRowErr
}

// テーブルセル更新再現
func (s *schemaRepositoryStub) UpdateCell(context.Context, domain.Profile, string, domain.TableRef, domain.CellUpdate) (domain.AffectedRows, error) {
	return s.affectedRows, s.updateCellErr
}

// テーブル行削除再現
func (s *schemaRepositoryStub) DeleteRow(context.Context, domain.Profile, string, domain.TableRef, domain.RowLocator) (domain.AffectedRows, error) {
	return s.affectedRows, s.deleteRowErr
}

// データベーススキーマ取得レスポンス検証
func TestAppHandlerGetDatabaseSchema(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	schema := domain.Schema{
		Tables: []domain.Table{
			{
				Namespace: "public",
				Name:      "items",
				Columns: []domain.Column{{
					Name:         "id",
					DataType:     "int4",
					Nullable:     false,
					IsPrimaryKey: true,
					IsForeignKey: false,
					IsUnique:     true,
				}},
			},
			{
				Namespace: "public",
				Name:      "parents",
				Columns: []domain.Column{{
					Name:         "id",
					DataType:     "int4",
					IsPrimaryKey: true,
					IsUnique:     true,
				}},
			},
		},
		ForeignKeys: []domain.ForeignKey{{
			Name:        "fk_items_parent",
			FromTable:   "items",
			FromColumns: []string{"parent_id"},
			ToTable:     "parents",
			ToColumns:   []string{"id"},
		}},
	}
	tests := []struct {
		name       string
		repository schemaRepositoryStub
		wantCode   apperr.Code
		wantData   bool
		wantLog    bool
	}{
		{
			name: "スキーマDTOの全フィールドを返す",
			repository: schemaRepositoryStub{
				profile:         profile,
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema:          schema,
			},
			wantData: true,
		},
		{
			name: "取得失敗を安全なエラーとエラーコードログで返す",
			repository: schemaRepositoryStub{
				profile:         profile,
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				inspectErr:      errors.New("dsn=postgres://user:password=secret@host/database query failed"),
			},
			wantCode: apperr.CodeSchemaLoadFailed,
			wantLog:  true,
		},
		{
			name:     "未注入ユースケースを安全な失敗で返す",
			wantCode: apperr.CodeSchemaLoadFailed,
			wantLog:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := applogger.NewWithWriter(&output, slog.LevelDebug)
			var appUseCase *usecase.AppUseCase
			if tt.repository.activeID != nil {
				appUseCase = usecase.NewAppUseCase(&tt.repository)
			}
			handler := NewAppHandler(logger, config.NewStore(t.TempDir()), appUseCase)
			got := handler.GetDatabaseSchema()
			if tt.wantData {
				if got.Data == nil {
					t.Fatal("GetDatabaseSchema() Data = nil, want non-nil")
				}
				if got.Error != nil {
					t.Errorf("GetDatabaseSchema() Error = %#v, want nil", got.Error)
				}
				assertDatabaseSchemaResponse(t, got.Data, profile)

				return
			}
			if got.Data != nil {
				t.Errorf("GetDatabaseSchema() Data = %#v, want nil", got.Data)
			}
			if got.Error == nil {
				t.Fatal("GetDatabaseSchema() Error = nil, want non-nil")
			}
			if got.Error.Code != string(tt.wantCode) {
				t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if strings.Contains(got.Error.Message, "secret") || strings.Contains(got.Error.Message, "dsn=") {
				t.Errorf("Error.Message = %q, want no secret", got.Error.Message)
			}
			if tt.wantLog {
				assertSafeSchemaFailureLog(t, output.String())
			}
		})
	}
}

// テーブル構造レスポンス検証
func TestAppHandlerGetTableStructure(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	defaultValue := "generated default"
	structure := domain.TableStructure{
		Table: domain.Table{
			Namespace: "public",
			Name:      "items",
			Columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					DefaultValue: &defaultValue,
					IsPrimaryKey: true,
					IsUnique:     true,
					IsGenerated:  true,
				},
			},
		},
		ForeignKeys: []domain.ForeignKey{
			{
				Name:        "fk_items_parent",
				FromTable:   "items",
				FromColumns: []string{"parent_id"},
				ToTable:     "parents",
				ToColumns:   []string{"id"},
			},
		},
		Indexes: []domain.Index{
			{
				Name:    "items_pkey",
				Columns: []string{"id"},
				Unique:  true,
				Kind:    "btree",
			},
		},
	}
	tests := []struct {
		name       string
		table      string
		repository schemaRepositoryStub
		nilUseCase bool
		wantCode   apperr.Code
		wantData   bool
		wantLog    bool
	}{
		{
			name:  "詳細DTOの全フィールドを返す",
			table: "items",
			repository: schemaRepositoryStub{
				profile:         profile,
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
			},
			wantData: true,
		},
		{
			name:  "失敗を安全なエラーとコードログで返す",
			table: "items",
			repository: schemaRepositoryStub{
				profile:         profile,
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				structureErr:    errors.New("dsn=postgres://password=secret query failed"),
			},
			wantCode: apperr.CodeSchemaLoadFailed,
			wantLog:  true,
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			table:      "items",
			nilUseCase: true,
			wantCode:   apperr.CodeSchemaLoadFailed,
			wantLog:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := applogger.NewWithWriter(&output, slog.LevelDebug)
			var appUseCase *usecase.AppUseCase
			if !tt.nilUseCase {
				appUseCase = usecase.NewAppUseCase(&tt.repository)
			}
			handler := NewAppHandler(logger, config.NewStore(t.TempDir()), appUseCase)
			got := handler.GetTableStructure(tt.table)
			if tt.wantData {
				if got.Data == nil {
					t.Fatal("GetTableStructure() Data = nil, want non-nil")
				}
				if got.Error != nil {
					t.Errorf("GetTableStructure() Error = %#v, want nil", got.Error)
				}
				if got.Data.Table.Namespace != "public" || got.Data.Table.Name != "items" || len(got.Data.Table.Columns) != 1 || got.Data.Table.Columns[0].DefaultValue == nil || *got.Data.Table.Columns[0].DefaultValue != defaultValue || !got.Data.Table.Columns[0].IsGenerated || len(got.Data.Indexes) != 1 || got.Data.Indexes[0].Kind != "btree" {
					t.Errorf("GetTableStructure() Data = %#v, want detailed table structure", got.Data)
				}

				return
			}
			if got.Error == nil {
				t.Fatal("GetTableStructure() Error = nil, want non-nil")
			}
			if got.Error.Code != string(tt.wantCode) {
				t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if strings.Contains(got.Error.Message, "secret") || strings.Contains(got.Error.Message, "dsn=") {
				t.Errorf("Error.Message = %q, want no secret", got.Error.Message)
			}
			if tt.wantLog {
				assertSafeSchemaFailureLog(t, output.String())
			}
		})
	}
}

// テーブル統計レスポンス検証
func TestAppHandlerGetTableStatistics(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	rowCount := int64(3)
	nullCount := int64(1)
	minimum := "a"
	statistics := domain.TableStatistics{
		Table:       domain.TableRef{Namespace: "public", Name: "items"},
		RowCount:    domain.StatisticCount{Value: &rowCount, Status: domain.StatisticsStatusComplete},
		ColumnCount: 1,
		CollectedAt: time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC),
		Status:      domain.StatisticsStatusComplete,
		Columns: []domain.ColumnStatistics{{
			Name:           "name",
			NullCount:      domain.StatisticCount{Value: &nullCount, Status: domain.StatisticsStatusComplete},
			DistinctCount:  domain.StatisticCount{Status: domain.StatisticsStatusUnavailable, Reason: stringPointer("not collected")},
			DuplicateCount: domain.StatisticCount{Status: domain.StatisticsStatusUnavailable, Reason: stringPointer("not collected")},
			Min:            domain.StatisticValue{Value: &minimum, Status: domain.StatisticsStatusComplete},
			Max:            domain.StatisticValue{Status: domain.StatisticsStatusUnavailable, Reason: stringPointer("unsupported data type")},
		}},
		ForeignKeys: []domain.ForeignKeyStatistics{{
			Name:                  "fk_items_parent",
			FromColumns:           []string{"parent_id"},
			ToTable:               "parents",
			ToColumns:             []string{"id"},
			SourceRowCount:        domain.StatisticCount{Value: &rowCount, Status: domain.StatisticsStatusComplete},
			NullCount:             domain.StatisticCount{Value: &nullCount, Status: domain.StatisticsStatusComplete},
			ReferencedRowCount:    domain.StatisticCount{Value: &rowCount, Status: domain.StatisticsStatusComplete},
			MissingReferenceCount: domain.StatisticCount{Value: &nullCount, Status: domain.StatisticsStatusComplete},
		}},
	}
	tests := []struct {
		name       string
		repository schemaRepositoryStub
		wantCode   apperr.Code
		wantData   bool
	}{
		{
			name: "統計DTOの項目状態を返す",
			repository: schemaRepositoryStub{
				profile:         profile,
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				statistics:      statistics,
			},
			wantData: true,
		},
		{
			name:     "未注入ユースケースを安全な失敗で返す",
			wantCode: apperr.CodeStatsLoadFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := applogger.NewWithWriter(io.Discard, slog.LevelDebug)
			var appUseCase *usecase.AppUseCase
			if tt.repository.activeID != nil {
				appUseCase = usecase.NewAppUseCase(&tt.repository)
			}
			handler := NewAppHandler(logger, config.NewStore(t.TempDir()), appUseCase)
			got := handler.GetTableStatistics("items")
			if tt.wantData {
				if got.Data == nil {
					t.Fatal("GetTableStatistics() Data = nil, want non-nil")
				}
				if got.Error != nil {
					t.Errorf("GetTableStatistics() Error = %#v, want nil", got.Error)
				}
				if got.Data.Table != "items" || got.Data.RowCount.Value == nil || *got.Data.RowCount.Value != rowCount || got.Data.Columns[0].DistinctCount.Status != string(domain.StatisticsStatusUnavailable) || got.Data.Columns[0].Min.Value == nil || *got.Data.Columns[0].Min.Value != minimum {
					t.Errorf("GetTableStatistics() Data = %#v, want converted statistics", got.Data)
				}

				return
			}
			if got.Error == nil {
				t.Fatal("GetTableStatistics() Error = nil, want non-nil")
			}
			if got.Error.Code != string(tt.wantCode) {
				t.Errorf("Error.Code = %q, want %q", got.Error.Code, tt.wantCode)
			}
		})
	}
}

// テーブル統計要求競合検証
func TestAppHandlerGetTableStatisticsCancelsPreviousRequest(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	var calls atomic.Int32
	repository := schemaRepositoryStub{
		profile:         profile,
		activeID:        stringPointer(profile.ID),
		credential:      "secret",
		credentialFound: true,
		statisticsFunc: func(ctx context.Context, _ domain.Profile, _ string, ref domain.TableRef) (domain.TableStatistics, error) {
			switch calls.Add(1) {
			case 1:
				close(firstStarted)
				<-ctx.Done()

				return domain.TableStatistics{Table: ref}, ctx.Err()
			case 2:
				close(secondStarted)

				return domain.TableStatistics{Table: ref, Status: domain.StatisticsStatusComplete}, nil
			default:
				return domain.TableStatistics{Table: ref, Status: domain.StatisticsStatusComplete}, nil
			}
		},
	}
	handler := NewAppHandler(applogger.NewWithWriter(io.Discard, slog.LevelDebug), config.NewStore(t.TempDir()), usecase.NewAppUseCase(&repository))
	firstResult := make(chan Response[TableStatisticsResponse], 1)
	go func() {
		firstResult <- handler.GetTableStatistics("first")
	}()
	<-firstStarted
	secondResult := handler.GetTableStatistics("second")
	<-secondStarted

	if secondResult.Data == nil || secondResult.Data.Table != "second" || secondResult.Error != nil {
		t.Errorf("second GetTableStatistics() = %#v, want successful second result", secondResult)
	}
	first := <-firstResult
	if first.Error == nil || first.Error.Code != string(apperr.CodeOperationCanceled) {
		t.Errorf("first GetTableStatistics() = %#v, want canceled error", first)
	}
	thirdResult := handler.GetTableStatistics("third")
	if thirdResult.Data == nil || thirdResult.Data.Table != "third" || thirdResult.Error != nil {
		t.Errorf("third GetTableStatistics() = %#v, want successful result after stale cleanup", thirdResult)
	}
}

// テーブル行一覧レスポンス契約検証
func TestAppHandlerListTableRows(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	structure := domain.TableStructure{
		Table: domain.Table{
			Namespace: "public",
			Name:      "items",
			Columns: []domain.Column{
				{Name: "id", DataType: "int", IsPrimaryKey: true},
				{Name: "payload", DataType: "json"},
			},
		},
	}
	baseRepository := schemaRepositoryStub{
		profile:         profile,
		activeID:        stringPointer(profile.ID),
		credential:      "secret",
		credentialFound: true,
		structure:       structure,
		rows: domain.TableRows{
			Rows: []domain.TableRow{
				{Cells: []domain.CellValue{
					{Kind: domain.CellKindValue, Value: "1"},
					{Kind: domain.CellKindValue, Value: `{"key":"value"}`},
				}},
			},
			TotalCount: 1,
			Page:       1,
			PageSize:   domain.TablePageSize,
			Sort: &domain.TableSort{
				Column:    "id",
				Direction: domain.SortDirectionAscending,
			},
			Filter: &domain.FilterGroup{
				Operator: domain.FilterGroupOperatorAnd,
				Filters: []domain.TableFilter{
					{
						Column:   "id",
						Operator: domain.FilterOperatorEqual,
						Values:   []string{"1"},
					},
				},
				Groups: []domain.FilterGroup{
					{
						Operator: domain.FilterGroupOperatorOr,
						Filters: []domain.TableFilter{
							{
								Column:   "id",
								Operator: domain.FilterOperatorEqual,
								Values:   []string{"2"},
							},
						},
					},
				},
			},
		},
	}
	tests := []struct {
		name       string
		request    ListTableRowsRequest
		customRows *domain.TableRows
		nilUseCase bool
		wantCode   apperr.Code
		wantData   bool
	}{
		{
			name:     "行DTOを返す",
			request:  ListTableRowsRequest{Table: "items", Page: 1},
			wantData: true,
		},
		{
			name: "SortおよびFilterがnilの行DTOを返す",
			request: ListTableRowsRequest{
				Table: "items",
				Page:  1,
			},
			customRows: &domain.TableRows{
				Rows: []domain.TableRow{
					{Cells: []domain.CellValue{
						{Kind: domain.CellKindValue, Value: "1"},
						{Kind: domain.CellKindValue, Value: `{"key":"value"}`},
					}},
				},
				TotalCount: 1,
				Page:       1,
				PageSize:   domain.TablePageSize,
			},
			wantData: true,
		},
		{
			name:       "未注入ユースケースを安全な失敗で返す",
			request:    ListTableRowsRequest{Table: "items", Page: 1},
			nilUseCase: true,
			wantCode:   apperr.CodeDataLoadFailed,
		},
		{
			name: "ネストされたフィルターグループと並び替えを入力として処理できる",
			request: ListTableRowsRequest{
				Table: "items",
				Page:  1,
				Sort:  &TableSortRequest{Column: "id", Direction: "asc"},
				Filter: &FilterGroupRequest{
					Operator: "and",
					Filters: []TableFilterRequest{{
						Column:   "id",
						Operator: "=",
						Values:   []string{"1"},
					}},
					Groups: []FilterGroupRequest{{
						Operator: "or",
						Filters: []TableFilterRequest{{
							Column:   "id",
							Operator: "=",
							Values:   []string{"2"},
						}},
					}},
				},
			},
			wantData: true,
		},
		{
			name: "空グループを入力不正として返す",
			request: ListTableRowsRequest{
				Table:  "items",
				Page:   1,
				Filter: &FilterGroupRequest{Operator: "and"},
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "JSONへのLIKEをフィルター適用失敗として返す",
			request: ListTableRowsRequest{
				Table: "items",
				Page:  1,
				Filter: &FilterGroupRequest{
					Operator: "and",
					Filters: []TableFilterRequest{{
						Column:   "payload",
						Operator: "LIKE",
						Values:   []string{"%x%"},
					}},
				},
			},
			wantCode: apperr.CodeFilterApplyFailed,
		},
		{
			name: "不正な並び替えを並び替え適用失敗として返す",
			request: ListTableRowsRequest{
				Table: "items",
				Page:  1,
				Sort:  &TableSortRequest{Column: "missing", Direction: "asc"},
			},
			wantCode: apperr.CodeSortApplyFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := baseRepository
			if tt.customRows != nil {
				repository.rows = *tt.customRows
			}
			var appUseCase *usecase.AppUseCase
			if !tt.nilUseCase {
				appUseCase = usecase.NewAppUseCase(&repository)
			}
			handler := NewAppHandler(applogger.NewWithWriter(io.Discard, slog.LevelDebug), config.NewStore(t.TempDir()), appUseCase)
			got := handler.ListTableRows(tt.request)
			if gotCode := tableRowsResponseErrorCode(got.Error); gotCode != tt.wantCode {
				t.Errorf("ListTableRows() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if gotData := got.Data != nil; gotData != tt.wantData {
				t.Fatalf("ListTableRows() data found = %v, want %v", gotData, tt.wantData)
			}
			if !tt.wantData {
				return
			}
			if got.Data.TotalCount != 1 || len(got.Data.Rows) != 1 || got.Data.Rows[0].Cells[1].Value != `{"key":"value"}` {
				t.Errorf("ListTableRows() data = %#v, want row DTO", got.Data)
			}
		})
	}
}

// テーブル行一覧エラーコード取得
func tableRowsResponseErrorCode(response *ErrorResponse) apperr.Code {
	if response == nil {
		return ""
	}

	return apperr.Code(response.Code)
}

// データベーススキーマレスポンス全項目検証
func assertDatabaseSchemaResponse(t *testing.T, got *DatabaseSchemaResponse, profile domain.Profile) {
	t.Helper()

	if got.ActiveProfile.ID != profile.ID || got.ActiveProfile.Name != profile.Name || got.ActiveProfile.DBType != string(profile.DBType) || got.ActiveProfile.Database != profile.Database || got.ActiveProfile.Schema != profile.Schema {
		t.Errorf("ActiveProfile = %#v, want profile fields %#v", got.ActiveProfile, profile)
	}
	if len(got.Tables) != 2 {
		t.Fatalf("Tables = %d, want 2", len(got.Tables))
	}
	table := got.Tables[0]
	if table.Namespace != "public" || table.Name != "items" {
		t.Errorf("Table = %#v, want public.items", table)
	}
	if len(table.Columns) != 1 {
		t.Fatalf("Columns = %d, want 1", len(table.Columns))
	}
	column := table.Columns[0]
	if column.Name != "id" || column.DataType != "int4" || column.Nullable || !column.IsPrimaryKey || column.IsForeignKey || !column.IsUnique {
		t.Errorf("Column = %#v, want id int4 non-null primary unique", column)
	}
	if len(got.ForeignKeys) != 1 {
		t.Fatalf("ForeignKeys = %d, want 1", len(got.ForeignKeys))
	}
	foreignKey := got.ForeignKeys[0]
	if foreignKey.Name != "fk_items_parent" || foreignKey.FromTable != "items" || foreignKey.ToTable != "parents" || !equalStringSlices(foreignKey.FromColumns, []string{"parent_id"}) || !equalStringSlices(foreignKey.ToColumns, []string{"id"}) {
		t.Errorf("ForeignKey = %#v, want fk_items_parent", foreignKey)
	}
}

// 安全な失敗ログ検証
func assertSafeSchemaFailureLog(t *testing.T, output string) {
	t.Helper()

	if !strings.Contains(output, string(apperr.CodeSchemaLoadFailed)) {
		t.Errorf("log = %q, want code %q", output, apperr.CodeSchemaLoadFailed)
	}
	if strings.Contains(output, "secret") || strings.Contains(output, "dsn=") || strings.Contains(output, "query failed") {
		t.Errorf("log = %q, want no sensitive failure detail", output)
	}
}

// 文字列スライス一致判定
func equalStringSlices(left, right []string) bool {
	return reflect.DeepEqual(left, right)
}

// テーブル行追加ハンドラーレスポンス検証
func TestAppHandlerInsertTableRow(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	valStr := "Alice"
	structure := domain.TableStructure{
		Table: domain.Table{
			Namespace: profile.Schema,
			Name:      "users",
			Columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int4",
					Nullable:     false,
					IsPrimaryKey: true,
					IsGenerated:  true,
				},
				{
					Name:     "name",
					DataType: "text",
					Nullable: false,
				},
			},
		},
	}

	tests := []struct {
		name       string
		repository *schemaRepositoryStub
		request    InsertTableRowRequest
		want       Response[AffectedRowsResponse]
	}{
		{
			name: "正常に行を追加できる",
			repository: &schemaRepositoryStub{
				profile:         profile,
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				affectedRows:    domain.AffectedRows{AffectedRows: 1},
			},
			request: InsertTableRowRequest{
				Table: "users",
				Values: []ColumnValueInputRequest{
					{
						Column: "id",
						Kind:   "default",
					},
					{
						Column: "name",
						Kind:   "value",
						Value:  &valStr,
					},
				},
			},
			want: OK(AffectedRowsResponse{AffectedRows: 1}),
		},
		{
			name: "ユースケース未初期化時はROW_ADD_FAILEDを返す",
			repository: &schemaRepositoryStub{
				profile:  profile,
				activeID: &profile.ID,
			},
			request: InsertTableRowRequest{
				Table: "users",
			},
			want: Fail[AffectedRowsResponse](apperr.New(apperr.CodeRowAddFailed)),
		},
		{
			name: "バリデーションエラー時はVALIDATION_FAILEDを返す",
			repository: &schemaRepositoryStub{
				profile:         profile,
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
			},
			request: InsertTableRowRequest{
				Table: "users",
				Values: []ColumnValueInputRequest{
					{Column: "name", Kind: "null"},
				},
			},
			want: Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := applogger.NewWithWriter(io.Discard, slog.LevelDebug)
			var appUseCase *usecase.AppUseCase
			if tt.name != "ユースケース未初期化時はROW_ADD_FAILEDを返す" {
				appUseCase = usecase.NewAppUseCase(tt.repository)
			}
			handler := NewAppHandler(logger, config.NewStore(t.TempDir()), appUseCase)
			got := handler.InsertTableRow(tt.request)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InsertTableRow() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// テーブルセル更新ハンドラーレスポンス検証
func TestAppHandlerUpdateTableCell(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	id := "1"
	name := "updated"
	structure := domain.TableStructure{Table: domain.Table{Namespace: profile.Schema, Name: "users", Columns: []domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "name", DataType: "text"}}}}
	request := UpdateTableCellRequest{Table: "users", Locator: []ColumnValueInputRequest{{Column: "id", Kind: "value", Value: &id}}, Column: "name", Value: TableCellResponse{Kind: "value", Value: name}}
	invalidInputRepository := func(columns []domain.Column) *schemaRepositoryStub {
		return &schemaRepositoryStub{
			profile:         profile,
			activeID:        &profile.ID,
			credential:      "secret",
			credentialFound: true,
			structure: domain.TableStructure{Table: domain.Table{
				Namespace: profile.Schema,
				Name:      "users",
				Columns:   columns,
			}},
			updateCellErr: domain.ErrInvalidRowInput,
		}
	}
	invalidInputRequest := func(column, value, locatorColumn, locatorValue string) UpdateTableCellRequest {
		return UpdateTableCellRequest{
			Table: "users",
			Locator: []ColumnValueInputRequest{{
				Column: locatorColumn,
				Kind:   "value",
				Value:  &locatorValue,
			}},
			Column: column,
			Value: TableCellResponse{
				Kind:  "value",
				Value: value,
			},
		}
	}
	tests := []struct {
		name       string
		repository *schemaRepositoryStub
		request    UpdateTableCellRequest
		want       Response[AffectedRowsResponse]
		wantLog    string
		avoidLog   string
	}{
		{
			name:       "正常にセルを更新できる",
			repository: &schemaRepositoryStub{profile: profile, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure, affectedRows: domain.AffectedRows{AffectedRows: 1}},
			request:    request,
			want:       OK(AffectedRowsResponse{AffectedRows: 1}),
		},
		{
			name:       "既定値へセルを更新できる",
			repository: &schemaRepositoryStub{profile: profile, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure, affectedRows: domain.AffectedRows{AffectedRows: 1}},
			request:    UpdateTableCellRequest{Table: "users", Locator: []ColumnValueInputRequest{{Column: "id", Kind: "value", Value: &id}}, Column: "name", Value: TableCellResponse{Kind: "default"}},
			want:       OK(AffectedRowsResponse{AffectedRows: 1}),
		},
		{
			name:       "バリデーションエラー時はVALIDATION_FAILEDを返す",
			repository: &schemaRepositoryStub{profile: profile, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure},
			request:    UpdateTableCellRequest{Table: "users", Locator: []ColumnValueInputRequest{{Column: "id", Kind: "value", Value: &id}}, Column: "name", Value: TableCellResponse{Kind: "null"}},
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
			wantLog:    string(apperr.CodeValidationFailed),
		},
		{
			name:       "更新値の不正base64はVALIDATION_FAILEDを返し入力値をログへ出さない",
			repository: invalidInputRepository([]domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "binary_data", DataType: "bytea"}}),
			request:    invalidInputRequest("binary_data", "not-base64", "id", id),
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
			wantLog:    string(apperr.CodeValidationFailed),
			avoidLog:   "not-base64",
		},
		{
			name:       "更新値の不正JSONはVALIDATION_FAILEDを返し入力値をログへ出さない",
			repository: invalidInputRepository([]domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "metadata", DataType: "json"}}),
			request:    invalidInputRequest("metadata", "{invalid", "id", id),
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
			wantLog:    string(apperr.CodeValidationFailed),
			avoidLog:   "{invalid",
		},
		{
			name:       "更新値の不正RFC3339はVALIDATION_FAILEDを返し入力値をログへ出さない",
			repository: invalidInputRepository([]domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "occurred_at", DataType: "timestamp"}}),
			request:    invalidInputRequest("occurred_at", "not-rfc3339", "id", id),
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
			wantLog:    string(apperr.CodeValidationFailed),
			avoidLog:   "not-rfc3339",
		},
		{
			name:       "位置指定値の不正base64はVALIDATION_FAILEDを返し入力値をログへ出さない",
			repository: invalidInputRepository([]domain.Column{{Name: "binary_data", DataType: "bytea", IsPrimaryKey: true}, {Name: "name", DataType: "text"}}),
			request:    invalidInputRequest("name", name, "binary_data", "not-base64"),
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
			wantLog:    string(apperr.CodeValidationFailed),
			avoidLog:   "not-base64",
		},
		{
			name:       "位置指定値の不正JSONはVALIDATION_FAILEDを返し入力値をログへ出さない",
			repository: invalidInputRepository([]domain.Column{{Name: "metadata", DataType: "json", IsPrimaryKey: true}, {Name: "name", DataType: "text"}}),
			request:    invalidInputRequest("name", name, "metadata", "{invalid"),
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
			wantLog:    string(apperr.CodeValidationFailed),
			avoidLog:   "{invalid",
		},
		{
			name:       "位置指定値の不正RFC3339はVALIDATION_FAILEDを返し入力値をログへ出さない",
			repository: invalidInputRepository([]domain.Column{{Name: "occurred_at", DataType: "timestamp", IsPrimaryKey: true}, {Name: "name", DataType: "text"}}),
			request:    invalidInputRequest("name", name, "occurred_at", "not-rfc3339"),
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeValidationFailed)),
			wantLog:    string(apperr.CodeValidationFailed),
			avoidLog:   "not-rfc3339",
		},
		{
			name:       "更新失敗時はCELL_UPDATE_FAILEDを返し機密情報をログへ出さない",
			repository: &schemaRepositoryStub{profile: profile, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure, updateCellErr: errors.New("password=secret")},
			request:    request,
			want:       Fail[AffectedRowsResponse](apperr.New(apperr.CodeCellUpdateFailed)),
			wantLog:    string(apperr.CodeCellUpdateFailed),
			avoidLog:   "password=secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), usecase.NewAppUseCase(tt.repository))
			got := handler.UpdateTableCell(tt.request)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateTableCell() = %#v, want %#v", got, tt.want)
			}
			if tt.wantLog != "" && !strings.Contains(output.String(), tt.wantLog) {
				t.Errorf("log = %q, want code %q", output.String(), tt.wantLog)
			}
			if tt.avoidLog != "" && strings.Contains(output.String(), tt.avoidLog) {
				t.Errorf("log = %q, must not contain %q", output.String(), tt.avoidLog)
			}
			if strings.Contains(output.String(), "secret") {
				t.Errorf("log = %q, must not contain credential", output.String())
			}
		})
	}
}

// テーブルセル更新ハンドラー未設定検証
func TestAppHandlerUpdateTableCellWithoutUseCase(t *testing.T) {
	var output bytes.Buffer
	handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil)

	got := handler.UpdateTableCell(UpdateTableCellRequest{})
	want := Fail[AffectedRowsResponse](apperr.New(apperr.CodeCellUpdateFailed))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UpdateTableCell() = %#v, want %#v", got, want)
	}
	if !strings.Contains(output.String(), string(apperr.CodeCellUpdateFailed)) {
		t.Errorf("log = %q, want code %q", output.String(), apperr.CodeCellUpdateFailed)
	}
}

// テーブル行削除ハンドラーレスポンス検証
func TestAppHandlerDeleteTableRow(t *testing.T) {
	profile := newTestProfile(t, "profile-1", domain.DBTypePostgres)
	id := "1"
	structure := domain.TableStructure{Table: domain.Table{
		Namespace: profile.Schema,
		Name:      "users",
		Columns: []domain.Column{
			{
				Name:         "id",
				DataType:     "int4",
				IsPrimaryKey: true,
			},
		},
	}}
	request := DeleteTableRowRequest{
		Table: "users",
		Locator: []ColumnValueInputRequest{
			{
				Column: "id",
				Kind:   "value",
				Value:  &id,
			},
		},
	}
	tests := []struct {
		name       string
		repository *schemaRepositoryStub
		request    DeleteTableRowRequest
		want       Response[AffectedRowsResponse]
		avoidLogs  []string
	}{
		{
			name: "行を削除できる",
			repository: &schemaRepositoryStub{
				profile:         profile,
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				affectedRows:    domain.AffectedRows{AffectedRows: 2},
			},
			request: request,
			want:    OK(AffectedRowsResponse{AffectedRows: 2}),
		},
		{
			name: "削除失敗時は安全なエラーを返す",
			repository: &schemaRepositoryStub{
				profile:         profile,
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				deleteRowErr:    errors.New("password=secret locator=1"),
			},
			request: request,
			want:    Fail[AffectedRowsResponse](apperr.New(apperr.CodeRowDeleteFailed)),
			avoidLogs: []string{
				"password=secret",
				"locator=1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), usecase.NewAppUseCase(tt.repository))
			got := handler.DeleteTableRow(tt.request)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeleteTableRow() = %#v, want %#v", got, tt.want)
			}
			for _, avoidLog := range tt.avoidLogs {
				if strings.Contains(output.String(), avoidLog) {
					t.Errorf("log = %q, must not contain %q", output.String(), avoidLog)
				}
			}
		})
	}
}

// テーブル行削除ハンドラー未設定検証
func TestAppHandlerDeleteTableRowWithoutUseCase(t *testing.T) {
	var output bytes.Buffer
	handler := NewAppHandler(applogger.NewWithWriter(&output, slog.LevelDebug), config.NewStore(t.TempDir()), nil)

	got := handler.DeleteTableRow(DeleteTableRowRequest{})
	want := Fail[AffectedRowsResponse](apperr.New(apperr.CodeRowDeleteFailed))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DeleteTableRow() = %#v, want %#v", got, want)
	}
	if !strings.Contains(output.String(), string(apperr.CodeRowDeleteFailed)) {
		t.Errorf("log = %q, want code %q", output.String(), apperr.CodeRowDeleteFailed)
	}
}
