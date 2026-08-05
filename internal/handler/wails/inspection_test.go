package wails

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

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
	flowState       domain.FlowState
	flowStateErr    error
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
