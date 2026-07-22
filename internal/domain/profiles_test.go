package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type connectionProfileArgs struct {
	id       string
	name     string
	dbType   DBType
	host     string
	port     int
	database string
	schema   string
	user     string
}

// プロファイル生成検証
func TestNewProfile(t *testing.T) {
	valid := connectionProfileArgs{
		id:       "profile-1",
		name:     "Local DB",
		dbType:   DBTypePostgres,
		host:     "localhost",
		port:     5432,
		database: "app",
		schema:   "public",
		user:     "user",
	}
	tests := []struct {
		name    string
		args    connectionProfileArgs
		wantErr bool
	}{
		{
			name: "PostgreSQLプロファイル",
			args: valid,
		},
		{
			name: "MySQLプロファイル",
			args: connectionProfileArgs{
				id:       "profile-1",
				name:     "Local DB",
				dbType:   DBTypeMySQL,
				host:     "localhost",
				port:     3306,
				database: "app",
				user:     "user",
			},
		},
		{
			name: "空のID",
			args: connectionProfileArgs{
				name:     valid.name,
				dbType:   valid.dbType,
				host:     valid.host,
				port:     valid.port,
				database: valid.database,
				schema:   valid.schema,
				user:     valid.user,
			},
			wantErr: true,
		},
		{
			name: "空白だけの名前",
			args: connectionProfileArgs{
				id:       valid.id,
				name:     " ",
				dbType:   valid.dbType,
				host:     valid.host,
				port:     valid.port,
				database: valid.database,
				schema:   valid.schema,
				user:     valid.user,
			},
			wantErr: true,
		},
		{
			name: "未対応のデータベース種別",
			args: connectionProfileArgs{
				id:       valid.id,
				name:     valid.name,
				dbType:   "sqlite",
				host:     valid.host,
				port:     valid.port,
				database: valid.database,
				schema:   valid.schema,
				user:     valid.user,
			},
			wantErr: true,
		},
		{
			name: "空白を含むホスト",
			args: connectionProfileArgs{
				id:       valid.id,
				name:     valid.name,
				dbType:   valid.dbType,
				host:     "local host",
				port:     valid.port,
				database: valid.database,
				schema:   valid.schema,
				user:     valid.user,
			},
			wantErr: true,
		},
		{
			name: "範囲外のポート",
			args: connectionProfileArgs{
				id:       valid.id,
				name:     valid.name,
				dbType:   valid.dbType,
				host:     valid.host,
				port:     0,
				database: valid.database,
				schema:   valid.schema,
				user:     valid.user,
			},
			wantErr: true,
		},
		{
			name: "MySQLのスキーマ",
			args: connectionProfileArgs{
				id:       valid.id,
				name:     valid.name,
				dbType:   DBTypeMySQL,
				host:     valid.host,
				port:     3306,
				database: valid.database,
				schema:   "public",
				user:     valid.user,
			},
			wantErr: true,
		},
		{
			name: "PostgreSQLの空スキーマ",
			args: connectionProfileArgs{
				id:       valid.id,
				name:     valid.name,
				dbType:   valid.dbType,
				host:     valid.host,
				port:     valid.port,
				database: valid.database,
				user:     valid.user,
			},
			wantErr: true,
		},
		{
			name: "長すぎる名前",
			args: connectionProfileArgs{
				id:       valid.id,
				name:     strings.Repeat("a", 101),
				dbType:   valid.dbType,
				host:     valid.host,
				port:     valid.port,
				database: valid.database,
				schema:   valid.schema,
				user:     valid.user,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProfile(tt.args.id, tt.args.name, tt.args.dbType, tt.args.host, tt.args.port, tt.args.database, tt.args.schema, tt.args.user)
			if gotErr := errors.Is(err, ErrInvalidProfile); gotErr != tt.wantErr {
				t.Errorf("NewProfile() invalid profile error = %v, want %v (error = %v)", gotErr, tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}

			want := Profile{
				ID:       tt.args.id,
				Name:     tt.args.name,
				DBType:   tt.args.dbType,
				Host:     tt.args.host,
				Port:     tt.args.port,
				Database: tt.args.database,
				Schema:   tt.args.schema,
				User:     tt.args.user,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("NewProfile() = %#v, want %#v", got, want)
			}
		})
	}
}

// アクティブプロファイル検証
func TestValidateActiveProfile(t *testing.T) {
	profile := Profile{
		ID: "profile-1",
	}

	tests := []struct {
		name     string
		activeID *string
		wantErr  bool
	}{
		{
			name:     "アクティブIDなし",
			activeID: nil,
		},
		{
			name:     "存在するアクティブID",
			activeID: stringPointer("profile-1"),
		},
		{
			name:     "存在しないアクティブID",
			activeID: stringPointer("missing"),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActiveProfile([]Profile{profile}, tt.activeID)
			if gotErr := errors.Is(err, ErrInvalidActiveProfile); gotErr != tt.wantErr {
				t.Errorf("ValidateActiveProfile() invalid active error = %v, want %v (error = %v)", gotErr, tt.wantErr, err)
			}
		})
	}
}

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}
