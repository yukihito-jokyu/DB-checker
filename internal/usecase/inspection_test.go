package usecase

import (
	"context"
	"errors"
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
					Columns:   []domain.Column{{Name: "id", DataType: "bigint"}},
				}}},
			},
			wantProfile: mysqlProfile,
			wantSchema: domain.Schema{Tables: []domain.Table{{
				Namespace: "app",
				Name:      "users",
				Columns:   []domain.Column{{Name: "id", DataType: "bigint"}},
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
					Columns:   []domain.Column{{Name: "id", DataType: "int"}},
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
		Table:   domain.Table{Namespace: "public", Name: "items", Columns: []domain.Column{{Name: "id", DataType: "int4", DefaultValue: &defaultValue, IsPrimaryKey: true, IsUnique: true}}},
		Indexes: []domain.Index{{Name: "items_pkey", Columns: []string{"id"}, Unique: true, Kind: "btree"}},
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
