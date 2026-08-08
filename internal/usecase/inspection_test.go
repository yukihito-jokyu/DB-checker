package usecase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type inspectionRepositoryStub struct {
	profiles        []domain.Profile
	activeID        *string
	loadErr         error
	credential      string
	credentialFound bool
	credentialErr   error
	schema          domain.Schema
	inspectErr      error
	structure       domain.TableStructure
	structureErr    error
	statistics      domain.TableStatistics
	statisticsErr   error
	rows            domain.TableRows
	rowsErr         error
	rowsQuery       domain.TableQuery
	affectedRows    domain.AffectedRows
	insertRowErr    error
	updateCellErr   error
	deleteRowErr    error
	deleteRowCalled bool
}

// プロファイル読込再現
func (s *inspectionRepositoryStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return s.profiles, s.activeID, s.loadErr
}

// フロー状態読込再現
func (*inspectionRepositoryStub) LoadFlowState(string) (domain.FlowState, error) {
	return domain.EmptyFlowState(), nil
}

// フロー状態保存再現
func (*inspectionRepositoryStub) SaveFlowState(string, domain.FlowState) error { return nil }

// プロファイル保存再現
func (*inspectionRepositoryStub) SaveProfiles([]domain.Profile, *string) error { return nil }

// 資格情報取得再現
func (s *inspectionRepositoryStub) GetCredential(string) (string, bool, error) {
	return s.credential, s.credentialFound, s.credentialErr
}

// 資格情報保存再現
func (*inspectionRepositoryStub) SetCredential(string, string) error {
	return nil
}

// 資格情報削除再現
func (*inspectionRepositoryStub) DeleteCredential(string) error {
	return nil
}

// 接続確認再現
func (*inspectionRepositoryStub) CheckConnection(context.Context, domain.Profile, string) error {
	return nil
}

// スキーマ取得再現
func (s *inspectionRepositoryStub) InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error) {
	return s.schema, s.inspectErr
}

// テーブル構造取得再現
func (s *inspectionRepositoryStub) InspectTableStructure(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStructure, error) {
	return s.structure, s.structureErr
}

// テーブル統計取得再現
func (s *inspectionRepositoryStub) InspectTableStatistics(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStatistics, error) {
	return s.statistics, s.statisticsErr
}

// テーブル行一覧再現
func (s *inspectionRepositoryStub) ListRows(_ context.Context, _ domain.Profile, _ string, query domain.TableQuery) (domain.TableRows, error) {
	s.rowsQuery = query

	return s.rows, s.rowsErr
}

// テーブル行追加再現
func (s *inspectionRepositoryStub) InsertRow(context.Context, domain.Profile, string, domain.TableRef, domain.InsertRow) (domain.AffectedRows, error) {
	return s.affectedRows, s.insertRowErr
}

// テーブルセル更新再現
func (s *inspectionRepositoryStub) UpdateCell(context.Context, domain.Profile, string, domain.TableRef, domain.CellUpdate) (domain.AffectedRows, error) {
	return s.affectedRows, s.updateCellErr
}

// テーブル行削除再現
func (s *inspectionRepositoryStub) DeleteRow(context.Context, domain.Profile, string, domain.TableRef, domain.RowLocator) (domain.AffectedRows, error) {
	s.deleteRowCalled = true

	return s.affectedRows, s.deleteRowErr
}

// データベーススキーマ取得検証
func TestAppUseCaseGetDatabaseSchema(t *testing.T) {
	profile := inspectionTestProfile(t)
	mysqlProfile := inspectionTestMySQLProfile(t)
	validSchema := domain.Schema{
		Tables: []domain.Table{
			{
				Namespace: "public",
				Name:      "parents",
				Columns: []domain.Column{
					{
						Name:         "id",
						DataType:     "int",
						IsPrimaryKey: true,
					},
				},
			},
			{
				Namespace: "public",
				Name:      "children",
				Columns: []domain.Column{
					{
						Name:         "parent_id",
						DataType:     "int",
						IsForeignKey: true,
					},
				},
			},
		},
		ForeignKeys: []domain.ForeignKey{
			{
				Name:        "children_parent_id_fkey",
				FromTable:   "children",
				FromColumns: []string{"parent_id"},
				ToTable:     "parents",
				ToColumns:   []string{"id"},
			},
		},
	}
	repositoryErr := errors.New("profiles failed")
	tests := []struct {
		name        string
		repository  inspectionRepositoryStub
		wantCode    apperr.Code
		wantCause   error
		wantProfile domain.Profile
		wantSchema  domain.Schema
		wantResult  bool
	}{
		{
			name: "有効プロファイルのスキーマを返す",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema:          validSchema,
			},
			wantProfile: profile,
			wantSchema:  validSchema,
			wantResult:  true,
		},
		{
			name: "MySQLのデータベース名でスキーマを検証する",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{mysqlProfile},
				activeID:        stringPointer(mysqlProfile.ID),
				credential:      "secret",
				credentialFound: true,
				schema: domain.Schema{Tables: []domain.Table{{
					Namespace: "app",
					Name:      "users",
					Columns: []domain.Column{
						{
							Name:     "id",
							DataType: "bigint",
						},
					},
				}}},
			},
			wantProfile: mysqlProfile,
			wantSchema: domain.Schema{Tables: []domain.Table{{
				Namespace: "app",
				Name:      "users",
				Columns: []domain.Column{
					{
						Name:     "id",
						DataType: "bigint",
					},
				},
			}}},
			wantResult: true,
		},
		{
			name: "プロファイル読込失敗を返す",
			repository: inspectionRepositoryStub{
				loadErr: repositoryErr,
			},
			wantCause: repositoryErr,
		},
		{
			name: "アクティブプロファイル未選択を返す",
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "削除済みアクティブプロファイルを返す",
			repository: inspectionRepositoryStub{
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "資格情報取得失敗を分類する",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credentialErr:   errors.New("credential store failed"),
				credentialFound: true,
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "資格情報未登録を返す",
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "スキーマ取得失敗を分類する",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				inspectErr:      errors.New("database error"),
			},
			wantCode: apperr.CodeSchemaLoadFailed,
		},
		{
			name: "不正なスキーマを分類する",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				schema: domain.Schema{Tables: []domain.Table{{
					Namespace: "unexpected",
					Name:      "users",
					Columns: []domain.Column{
						{
							Name:     "id",
							DataType: "int",
						},
					},
				}}},
			},
			wantCode: apperr.CodeSchemaLoadFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := NewAppUseCase(&tt.repository)

			gotProfile, gotSchema, err := useCase.GetDatabaseSchema(context.Background())
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Errorf("GetDatabaseSchema() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("GetDatabaseSchema() error = %v, want cause %v", err, tt.wantCause)
			}
			if tt.wantResult {
				if err != nil {
					t.Fatalf("GetDatabaseSchema() error = %v", err)
				}
				if gotProfile != tt.wantProfile {
					t.Errorf("GetDatabaseSchema() profile = %#v, want %#v", gotProfile, tt.wantProfile)
				}
				if !reflect.DeepEqual(gotSchema, tt.wantSchema) {
					t.Errorf("GetDatabaseSchema() schema = %#v, want %#v", gotSchema, tt.wantSchema)
				}
			}
		})
	}
}

// テーブル構造取得検証
func TestAppUseCaseGetTableStructure(t *testing.T) {
	profile := inspectionTestProfile(t)
	defaultValue := "nextval('items_id_seq')"
	profilesErr := errors.New("profiles failed")
	credentialErr := errors.New("credential failed")
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
				},
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
		repository inspectionRepositoryStub
		want       domain.TableStructure
		wantCode   apperr.Code
		wantCause  error
	}{
		{
			name:  "有効なテーブル構造を返す",
			table: "items",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
			},
			want: structure,
		},
		{
			name: "プロファイル読込失敗を返す",
			repository: inspectionRepositoryStub{
				loadErr: profilesErr,
			},
			wantCause: profilesErr,
		},
		{
			name: "有効プロファイルが未選択ならプロファイル未検出を返す",
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "選択済みプロファイルが存在しないならプロファイル未検出を返す",
			repository: inspectionRepositoryStub{
				activeID: stringPointer("missing"),
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name:  "空白テーブル名を入力検証する",
			table: " ",
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name:  "資格情報取得失敗を資格情報利用不可へ分類する",
			table: "items",
			repository: inspectionRepositoryStub{
				profiles:      []domain.Profile{profile},
				activeID:      stringPointer(profile.ID),
				credentialErr: credentialErr,
			},
			wantCode:  apperr.CodeCredentialUnavailable,
			wantCause: credentialErr,
		},
		{
			name:  "資格情報未登録を資格情報利用不可へ分類する",
			table: "items",
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name:  "存在しないテーブルをスキーマ取得失敗へ分類する",
			table: "missing",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				structureErr:    errors.New("table not found"),
			},
			wantCode: apperr.CodeSchemaLoadFailed,
		},
		{
			name:  "不正なテーブル構造をスキーマ取得失敗へ分類する",
			table: "items",
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				structure: domain.TableStructure{
					Table: domain.Table{
						Namespace: "public",
						Name:      "other",
					},
				},
			},
			wantCode: apperr.CodeSchemaLoadFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAppUseCase(&tt.repository).GetTableStructure(context.Background(), tt.table)
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Errorf("GetTableStructure() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("GetTableStructure() error = %v, want cause %v", err, tt.wantCause)
			}
			if tt.wantCode != "" || tt.wantCause != nil {
				return
			}
			if err != nil {
				t.Fatalf("GetTableStructure() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTableStructure() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// テーブル統計取得検証
func TestAppUseCaseGetTableStatistics(t *testing.T) {
	profile := inspectionTestProfile(t)
	rowCount := int64(2)
	partial := domain.TableStatistics{
		Table: domain.TableRef{
			Namespace: "public",
			Name:      "items",
		},
		RowCount: domain.StatisticCount{
			Value:  &rowCount,
			Status: domain.StatisticsStatusComplete,
		},
	}
	repositoryErr := errors.New("statistics failed")
	tests := []struct {
		name       string
		emptyTable bool
		ctx        context.Context
		repository inspectionRepositoryStub
		wantCode   apperr.Code
		wantStatus domain.StatisticsStatus
		wantCause  error
	}{
		{
			name: "完全統計を返す",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				statistics:      domain.TableStatistics{Status: domain.StatisticsStatusComplete},
			},
			wantStatus: domain.StatisticsStatusComplete,
		},
		{
			name: "期限超過は部分結果を返す",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				statistics:      partial,
				statisticsErr:   context.DeadlineExceeded,
			},
			wantStatus: domain.StatisticsStatusTimeout,
		},
		{
			name: "キャンセルは制御フローのエラーを返す",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				statisticsErr:   context.Canceled,
			},
			wantCode: apperr.CodeOperationCanceled,
		},
		{
			name: "プロファイル読込失敗を返す",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				loadErr: repositoryErr,
			},
			wantCause: repositoryErr,
		},
		{
			name: "アクティブプロファイル未選択を返す",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "削除済みアクティブプロファイルを返す",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name:       "空テーブル名を検証する",
			emptyTable: true,
			ctx:        context.Background(),
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
			},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "資格情報取得失敗を分類する",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				profiles:      []domain.Profile{profile},
				activeID:      stringPointer(profile.ID),
				credentialErr: repositoryErr,
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "資格情報未登録を返す",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "統計取得失敗を分類する",
			ctx:  context.Background(),
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				statisticsErr:   repositoryErr,
			},
			wantCode: apperr.CodeStatsLoadFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := "items"
			if tt.emptyTable {
				table = ""
			}
			got, err := NewAppUseCase(&tt.repository).GetTableStatistics(tt.ctx, table)
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Errorf("GetTableStatistics() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("GetTableStatistics() error = %v, want cause %v", err, tt.wantCause)
			}
			if tt.wantCode != "" || tt.wantCause != nil {
				return
			}
			if err != nil {
				t.Fatalf("GetTableStatistics() error = %v", err)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
		})
	}
}

// テーブル行一覧取得検証
func TestAppUseCaseListTableRows(t *testing.T) {
	profile := inspectionTestProfile(t)
	structure := domain.TableStructure{
		Table: domain.Table{
			Namespace: "public",
			Name:      "items",
			Columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int",
					IsPrimaryKey: true,
				},
				{
					Name:     "name",
					DataType: "varchar",
				},
				{
					Name:     "payload",
					DataType: "json",
				},
			},
		},
	}
	validRepository := inspectionRepositoryStub{
		profiles:        []domain.Profile{profile},
		activeID:        stringPointer(profile.ID),
		credential:      "secret",
		credentialFound: true,
		structure:       structure,
		rows: domain.TableRows{
			Rows: []domain.TableRow{},
		},
	}
	validQuery := domain.TableQuery{
		Table: domain.TableRef{
			Name: "items",
		},
		Page: 1,
	}
	tests := []struct {
		name       string
		query      domain.TableQuery
		repository inspectionRepositoryStub
		want       domain.TableRows
		wantCode   apperr.Code
		wantCause  string
	}{
		{
			name:       "行一覧を返す",
			query:      validQuery,
			repository: validRepository,
			want:       validRepository.rows,
		},
		{
			name:  "プロファイル読込失敗を返す",
			query: validQuery,
			repository: inspectionRepositoryStub{
				loadErr: errors.New("profiles failed"),
			},
			wantCause: "profiles failed",
		},
		{
			name:  "アクティブプロファイル未選択を返す",
			query: validQuery,
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name:  "選択済みプロファイル不在を返す",
			query: validQuery,
			repository: inspectionRepositoryStub{
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "空白テーブル名を入力不正として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: " "},
				Page:  1,
			},
			repository: validRepository,
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name: "不正ページを入力不正として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: "items"},
			},
			repository: validRepository,
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:  "資格情報取得失敗を分類する",
			query: validQuery,
			repository: inspectionRepositoryStub{
				profiles:      []domain.Profile{profile},
				activeID:      stringPointer(profile.ID),
				credentialErr: errors.New("credential failed"),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name:  "資格情報未登録を返す",
			query: validQuery,
			repository: inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name:  "構造取得失敗をデータ取得失敗へ分類する",
			query: validQuery,
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				structureErr:    errors.New("structure failed"),
			},
			wantCode: apperr.CodeDataLoadFailed,
		},
		{
			name: "空グループを入力不正として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: "items"},
				Page:  1,
				Filter: &domain.FilterGroup{
					Operator: domain.FilterGroupOperatorAnd,
				},
			},
			repository: validRepository,
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name: "値不足を入力不正として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: "items"},
				Page:  1,
				Filter: &domain.FilterGroup{
					Operator: domain.FilterGroupOperatorAnd,
					Filters: []domain.TableFilter{
						{
							Column:   "id",
							Operator: domain.FilterOperatorBetween,
							Values:   []string{"1"},
						},
					},
				},
			},
			repository: validRepository,
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name: "存在しない列をフィルター適用失敗として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: "items"},
				Page:  1,
				Filter: &domain.FilterGroup{
					Operator: domain.FilterGroupOperatorAnd,
					Filters: []domain.TableFilter{
						{
							Column:   "missing",
							Operator: domain.FilterOperatorEqual,
							Values:   []string{"1"},
						},
					},
				},
			},
			repository: validRepository,
			wantCode:   apperr.CodeFilterApplyFailed,
		},
		{
			name: "JSONのLIKEをフィルター適用失敗として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: "items"},
				Page:  1,
				Filter: &domain.FilterGroup{
					Operator: domain.FilterGroupOperatorAnd,
					Filters: []domain.TableFilter{
						{
							Column:   "payload",
							Operator: domain.FilterOperatorLike,
							Values:   []string{"%x%"},
						},
					},
				},
			},
			repository: validRepository,
			wantCode:   apperr.CodeFilterApplyFailed,
		},
		{
			name: "不正並び替え列を並び替え適用失敗として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: "items"},
				Page:  1,
				Sort: &domain.TableSort{
					Column:    "missing",
					Direction: domain.SortDirectionAscending,
				},
			},
			repository: validRepository,
			wantCode:   apperr.CodeSortApplyFailed,
		},
		{
			name: "不正並び替え方向を並び替え適用失敗として返す",
			query: domain.TableQuery{
				Table: domain.TableRef{Name: "items"},
				Page:  1,
				Sort: &domain.TableSort{
					Column:    "id",
					Direction: "up",
				},
			},
			repository: validRepository,
			wantCode:   apperr.CodeSortApplyFailed,
		},
		{
			name:  "行取得失敗をデータ取得失敗へ分類する",
			query: validQuery,
			repository: inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        stringPointer(profile.ID),
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				rowsErr:         errors.New("rows failed"),
			},
			wantCode: apperr.CodeDataLoadFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			got, err := NewAppUseCase(&repository).ListTableRows(context.Background(), tt.query)
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Errorf("ListTableRows() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if tt.wantCause != "" && (err == nil || err.Error() != tt.wantCause) {
				t.Errorf("ListTableRows() error = %v, want %q", err, tt.wantCause)
			}
			if tt.wantCode != "" || tt.wantCause != "" {
				return
			}
			if err != nil {
				t.Fatalf("ListTableRows() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListTableRows() = %#v, want %#v", got, tt.want)
			}
			if repository.rowsQuery.Table.Namespace != profile.Schema {
				t.Errorf("ListRows() query namespace = %q, want %q", repository.rowsQuery.Table.Namespace, profile.Schema)
			}
		})
	}
}

// MySQL検証用プロファイル生成
func inspectionTestMySQLProfile(t *testing.T) domain.Profile {
	t.Helper()

	profile, err := domain.NewProfile("profile-2", "MySQL", domain.DBTypeMySQL, "localhost", 3306, "app", "", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	return profile
}

// 検証用プロファイル生成
func inspectionTestProfile(t *testing.T) domain.Profile {
	t.Helper()
	profile, err := domain.NewProfile("profile-1", "PostgreSQL", domain.DBTypePostgres, "localhost", 5432, "app", "public", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	return profile
}

// 検証用エラーコード取得
func inspectionErrorCode(err error) apperr.Code {
	if appErr := apperr.As(err); appErr != nil {
		return appErr.Code
	}

	return ""
}

// テーブル行追加検証
func TestAppUseCaseInsertTableRow(t *testing.T) {
	profile := inspectionTestProfile(t)
	valStr := "Alice"
	structure := domain.TableStructure{
		Table: domain.Table{
			Namespace: profile.Schema,
			Name:      "users",
			Columns: []domain.Column{
				{
					Name:         "id",
					DataType:     "int",
					Nullable:     false,
					DefaultValue: nil,
					IsPrimaryKey: true,
					IsGenerated:  true,
				},
				{
					Name:         "name",
					DataType:     "varchar(255)",
					Nullable:     false,
					DefaultValue: nil,
				},
			},
		},
	}

	tests := []struct {
		name       string
		repository *inspectionRepositoryStub
		row        domain.InsertRow
		want       domain.AffectedRows
		wantCode   apperr.Code
	}{
		{
			name: "正常に行を追加できる",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				affectedRows:    domain.AffectedRows{AffectedRows: 1},
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindDefault,
					},
					{
						Column: "name",
						Kind:   domain.CellKindValue,
						Value:  &valStr,
					},
				},
			},
			want:     domain.AffectedRows{AffectedRows: 1},
			wantCode: "",
		},
		{
			name: "アクティブプロファイルなしを拒否する",
			repository: &inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: nil,
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "入力検証エラーを拒否する",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
				Values: []domain.ColumnValueInput{
					{Column: "name", Kind: domain.CellKindNull},
				},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "プロファイル読み込みエラーをそのまま返す",
			repository: &inspectionRepositoryStub{
				loadErr: errors.New("load profiles error"),
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
			},
			want:     domain.AffectedRows{},
			wantCode: "",
		},
		{
			name: "アクティブプロファイルIDが不一致のときPROFILE_NOT_FOUNDを返す",
			repository: &inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: testStringPtr("missing-id"),
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "空テーブル名時にVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: &profile.ID,
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: ""},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "資格情報取得エラー時にCREDENTIAL_UNAVAILABLEを返す",
			repository: &inspectionRepositoryStub{
				profiles:      []domain.Profile{profile},
				activeID:      &profile.ID,
				credentialErr: errors.New("credential store error"),
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "資格情報未存在時にCREDENTIAL_UNAVAILABLEを返す",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credentialFound: false,
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "テーブル構造取得エラー時にROW_ADD_FAILEDを返す",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structureErr:    errors.New("structure inspect error"),
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeRowAddFailed,
		},
		{
			name: "リポジトリエラーをROW_ADD_FAILEDへラップする",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				insertRowErr:    errors.New("db execution error"),
			},
			row: domain.InsertRow{
				Table: domain.TableRef{Name: "users"},
				Values: []domain.ColumnValueInput{
					{
						Column: "id",
						Kind:   domain.CellKindDefault,
					},
					{
						Column: "name",
						Kind:   domain.CellKindValue,
						Value:  &valStr,
					},
				},
			},
			want:     domain.AffectedRows{},
			wantCode: apperr.CodeRowAddFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := NewAppUseCase(tt.repository)
			got, err := useCase.InsertTableRow(context.Background(), tt.row)
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Fatalf("InsertTableRow() error code = %q, wantCode %q", gotCode, tt.wantCode)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InsertTableRow() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// テスト用文字列ポインター取得
func testStringPtr(value string) *string {
	return &value
}

// テーブルセル更新検証
func TestAppUseCaseUpdateTableCell(t *testing.T) {
	profile := inspectionTestProfile(t)
	unknownProfileID := "unknown"
	id := "1"
	name := "updated"
	structure := func(columns []domain.Column) domain.TableStructure {
		return domain.TableStructure{Table: domain.Table{
			Namespace: profile.Schema,
			Name:      "users",
			Columns:   columns,
		}}
	}
	change := func(column, value, locatorColumn, locatorValue string) domain.CellUpdate {
		return domain.CellUpdate{
			Table: domain.TableRef{Name: "users"},
			Locator: domain.RowLocator{Values: []domain.ColumnValueInput{{
				Column: locatorColumn,
				Kind:   domain.CellKindValue,
				Value:  testStringPtr(locatorValue),
			}}},
			Column: column,
			Value: domain.CellValue{
				Kind:  domain.CellKindValue,
				Value: value,
			},
		}
	}
	standardStructure := structure([]domain.Column{
		{Name: "id", DataType: "int4", IsPrimaryKey: true},
		{Name: "name", DataType: "text"},
	})
	tests := []struct {
		name       string
		repository *inspectionRepositoryStub
		change     domain.CellUpdate
		wantCode   apperr.Code
		wantErr    bool
	}{
		{
			name:       "正常にセルを更新できる",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: standardStructure, affectedRows: domain.AffectedRows{AffectedRows: 1}},
			change:     change("name", name, "id", id),
		},
		{
			name:       "更新失敗時はCELL_UPDATE_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: standardStructure, updateCellErr: errors.New("update failed")},
			change:     change("name", name, "id", id),
			wantCode:   apperr.CodeCellUpdateFailed,
		},
		{
			name:       "プロファイル読込失敗を返す",
			repository: &inspectionRepositoryStub{loadErr: errors.New("load failed")},
			change:     change("name", name, "id", id),
			wantErr:    true,
		},
		{
			name:       "有効プロファイル未設定時はPROFILE_NOT_FOUNDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}},
			change:     change("name", name, "id", id),
			wantCode:   apperr.CodeProfileNotFound,
		},
		{
			name:       "有効プロファイル不在時はPROFILE_NOT_FOUNDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &unknownProfileID},
			change:     change("name", name, "id", id),
			wantCode:   apperr.CodeProfileNotFound,
		},
		{
			name:       "テーブル名不正時はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID},
			change:     domain.CellUpdate{},
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:       "資格情報取得失敗時はCREDENTIAL_UNAVAILABLEを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credentialErr: errors.New("credential failed")},
			change:     change("name", name, "id", id),
			wantCode:   apperr.CodeCredentialUnavailable,
		},
		{
			name:       "資格情報不在時はCREDENTIAL_UNAVAILABLEを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID},
			change:     change("name", name, "id", id),
			wantCode:   apperr.CodeCredentialUnavailable,
		},
		{
			name:       "構造取得失敗時はCELL_UPDATE_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structureErr: errors.New("structure failed")},
			change:     change("name", name, "id", id),
			wantCode:   apperr.CodeCellUpdateFailed,
		},
		{
			name:       "入力検証失敗時はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: standardStructure},
			change:     change("unknown", name, "id", id),
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:       "更新値の不正base64はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure([]domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "binary_data", DataType: "bytea"}}), updateCellErr: domain.ErrInvalidRowInput},
			change:     change("binary_data", "not-base64", "id", id),
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:       "更新値の不正JSONはVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure([]domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "metadata", DataType: "json"}}), updateCellErr: domain.ErrInvalidRowInput},
			change:     change("metadata", "{invalid", "id", id),
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:       "更新値の不正RFC3339はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure([]domain.Column{{Name: "id", DataType: "int4", IsPrimaryKey: true}, {Name: "occurred_at", DataType: "timestamp"}}), updateCellErr: domain.ErrInvalidRowInput},
			change:     change("occurred_at", "not-rfc3339", "id", id),
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:       "位置指定値の不正base64はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure([]domain.Column{{Name: "binary_data", DataType: "bytea", IsPrimaryKey: true}, {Name: "name", DataType: "text"}}), updateCellErr: domain.ErrInvalidRowInput},
			change:     change("name", name, "binary_data", "not-base64"),
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:       "位置指定値の不正JSONはVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure([]domain.Column{{Name: "metadata", DataType: "json", IsPrimaryKey: true}, {Name: "name", DataType: "text"}}), updateCellErr: domain.ErrInvalidRowInput},
			change:     change("name", name, "metadata", "{invalid"),
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name:       "位置指定値の不正RFC3339はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{profiles: []domain.Profile{profile}, activeID: &profile.ID, credential: "secret", credentialFound: true, structure: structure([]domain.Column{{Name: "occurred_at", DataType: "timestamp", IsPrimaryKey: true}, {Name: "name", DataType: "text"}}), updateCellErr: domain.ErrInvalidRowInput},
			change:     change("name", name, "occurred_at", "not-rfc3339"),
			wantCode:   apperr.CodeValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAppUseCase(tt.repository).UpdateTableCell(context.Background(), tt.change)
			if gotErr := err != nil; gotErr != tt.wantErr && tt.wantCode == "" {
				t.Fatalf("UpdateTableCell() error = %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Fatalf("UpdateTableCell() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if tt.wantCode == "" && got.AffectedRows != 1 {
				t.Errorf("UpdateTableCell() = %#v, want affected rows 1", got)
			}
		})
	}
}

// テーブル行削除検証
func TestAppUseCaseDeleteTableRow(t *testing.T) {
	profile := inspectionTestProfile(t)
	id := "1"
	locator := domain.RowLocator{Values: []domain.ColumnValueInput{
		{
			Column: "id",
			Kind:   domain.CellKindValue,
			Value:  &id,
		},
	}}
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
	tests := []struct {
		name       string
		repository *inspectionRepositoryStub
		table      string
		locator    domain.RowLocator
		wantCode   apperr.Code
		wantRows   int64
		wantDelete bool
	}{
		{
			name: "行を削除できる",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				affectedRows:    domain.AffectedRows{AffectedRows: 2},
			},
			table:      "users",
			locator:    locator,
			wantRows:   2,
			wantDelete: true,
		},
		{
			name: "不正位置指定はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
			},
			table:    "users",
			locator:  domain.RowLocator{},
			wantCode: apperr.CodeValidationFailed,
		},
		{
			name: "repository起因の入力変換失敗はVALIDATION_FAILEDを返す",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				deleteRowErr:    fmt.Errorf("invalid binary locator: %w", domain.ErrInvalidRowInput),
			},
			table:      "users",
			locator:    locator,
			wantCode:   apperr.CodeValidationFailed,
			wantDelete: true,
		},
		{
			name: "削除失敗はROW_DELETE_FAILEDを返す",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structure:       structure,
				deleteRowErr:    errors.New("delete failed"),
			},
			table:      "users",
			locator:    locator,
			wantCode:   apperr.CodeRowDeleteFailed,
			wantDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAppUseCase(tt.repository).DeleteTableRow(context.Background(), tt.table, tt.locator)
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Fatalf("DeleteTableRow() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if got.AffectedRows != tt.wantRows {
				t.Errorf("DeleteTableRow() = %#v, want rows %d", got, tt.wantRows)
			}
			if tt.repository.deleteRowCalled != tt.wantDelete {
				t.Errorf("DeleteRow() called = %t, want %t", tt.repository.deleteRowCalled, tt.wantDelete)
			}
		})
	}
}

// テーブル行削除失敗検証
func TestAppUseCaseDeleteTableRowFailures(t *testing.T) {
	profile := inspectionTestProfile(t)
	id := "1"
	locator := domain.RowLocator{Values: []domain.ColumnValueInput{
		{
			Column: "id",
			Kind:   domain.CellKindValue,
			Value:  &id,
		},
	}}
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
	validRepository := func() *inspectionRepositoryStub {
		return &inspectionRepositoryStub{
			profiles:        []domain.Profile{profile},
			activeID:        &profile.ID,
			credential:      "secret",
			credentialFound: true,
			structure:       structure,
		}
	}
	tests := []struct {
		name       string
		repository *inspectionRepositoryStub
		table      string
		wantCode   apperr.Code
	}{
		{
			name: "プロファイル読込失敗を返す",
			repository: &inspectionRepositoryStub{
				loadErr: errors.New("load failed"),
			},
			table:    "users",
			wantCode: "",
		},
		{
			name:       "アクティブプロファイルなしを返す",
			repository: &inspectionRepositoryStub{},
			table:      "users",
			wantCode:   apperr.CodeProfileNotFound,
		},
		{
			name: "アクティブプロファイル不在を返す",
			repository: &inspectionRepositoryStub{
				activeID: &profile.ID,
			},
			table:    "users",
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name:       "不正テーブル名を検証する",
			repository: validRepository(),
			table:      "",
			wantCode:   apperr.CodeValidationFailed,
		},
		{
			name: "資格情報取得失敗を返す",
			repository: &inspectionRepositoryStub{
				profiles:      []domain.Profile{profile},
				activeID:      &profile.ID,
				credentialErr: errors.New("credential failed"),
			},
			table:    "users",
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "資格情報不在を返す",
			repository: &inspectionRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: &profile.ID,
			},
			table:    "users",
			wantCode: apperr.CodeCredentialUnavailable,
		},
		{
			name: "構造取得失敗を返す",
			repository: &inspectionRepositoryStub{
				profiles:        []domain.Profile{profile},
				activeID:        &profile.ID,
				credential:      "secret",
				credentialFound: true,
				structureErr:    errors.New("structure failed"),
			},
			table:    "users",
			wantCode: apperr.CodeRowDeleteFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAppUseCase(tt.repository).DeleteTableRow(context.Background(), tt.table, locator)
			if tt.wantCode == "" {
				if err == nil || err.Error() != "load failed" {
					t.Errorf("DeleteTableRow() error = %v, want %q", err, "load failed")
				}

				return
			}
			if gotCode := inspectionErrorCode(err); gotCode != tt.wantCode {
				t.Errorf("DeleteTableRow() error code = %q, want %q", gotCode, tt.wantCode)
			}
		})
	}
}
