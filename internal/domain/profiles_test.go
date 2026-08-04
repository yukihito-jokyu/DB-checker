package domain

import (
	"errors"
	"math"
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

type profileDraftArgs struct {
	id       string
	name     string
	dbType   DBType
	host     string
	port     int
	database string
	schema   string
	user     string
	password string
}

// プロファイル下書き生成検証
func TestNewProfileDraft(t *testing.T) {
	valid := profileDraftArgs{
		name:     "Local DB",
		dbType:   DBTypePostgres,
		host:     "localhost",
		port:     5432,
		database: "app",
		schema:   "public",
		user:     "user",
		password: "secret",
	}
	tests := []struct {
		name    string
		args    profileDraftArgs
		want    ProfileDraft
		wantErr bool
	}{
		{
			name: "PostgreSQL下書き",
			args: valid,
			want: ProfileDraft{
				Name:     "Local DB",
				DBType:   DBTypePostgres,
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
				Password: "secret",
			},
		},
		{
			name: "MySQL下書き",
			args: profileDraftArgs{
				id:       "profile-1",
				name:     "Local MySQL",
				dbType:   DBTypeMySQL,
				host:     "localhost",
				port:     3306,
				database: "app",
				user:     "user",
			},
			want: ProfileDraft{
				ID:       "profile-1",
				Name:     "Local MySQL",
				DBType:   DBTypeMySQL,
				Host:     "localhost",
				Port:     3306,
				Database: "app",
				User:     "user",
			},
		},
		{
			name: "不正な下書き",
			args: profileDraftArgs{
				name:     "Local DB",
				dbType:   DBTypePostgres,
				host:     "",
				port:     5432,
				database: "app",
				schema:   "public",
				user:     "user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProfileDraft(tt.args.id, tt.args.name, tt.args.dbType, tt.args.host, tt.args.port, tt.args.database, tt.args.schema, tt.args.user, tt.args.password)

			if gotErr := errors.Is(err, ErrInvalidProfile); gotErr != tt.wantErr {
				t.Errorf("NewProfileDraft() invalid profile error = %v, want %v (error = %v)", gotErr, tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProfileDraft() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// プロファイル下書き検証
func TestProfileDraftValidate(t *testing.T) {
	valid := ProfileDraft{
		Name:     "Local DB",
		DBType:   DBTypePostgres,
		Host:     "localhost",
		Port:     5432,
		Database: "app",
		Schema:   "public",
		User:     "user",
	}
	tests := []struct {
		name    string
		draft   ProfileDraft
		wantErr bool
	}{
		{
			name:  "PostgreSQL下書き",
			draft: valid,
		},
		{
			name: "MySQL下書き",
			draft: ProfileDraft{
				Name:     "Local MySQL",
				DBType:   DBTypeMySQL,
				Host:     "localhost",
				Port:     3306,
				Database: "app",
				User:     "user",
			},
		},
		{
			name: "空白だけの名前",
			draft: ProfileDraft{
				Name:     " ",
				DBType:   valid.DBType,
				Host:     valid.Host,
				Port:     valid.Port,
				Database: valid.Database,
				Schema:   valid.Schema,
				User:     valid.User,
			},
			wantErr: true,
		},
		{
			name: "未対応のデータベース種別",
			draft: ProfileDraft{
				Name:     valid.Name,
				DBType:   "sqlite",
				Host:     valid.Host,
				Port:     valid.Port,
				Database: valid.Database,
				Schema:   valid.Schema,
				User:     valid.User,
			},
			wantErr: true,
		},
		{
			name: "空のホスト",
			draft: ProfileDraft{
				Name:     valid.Name,
				DBType:   valid.DBType,
				Port:     valid.Port,
				Database: valid.Database,
				Schema:   valid.Schema,
				User:     valid.User,
			},
			wantErr: true,
		},
		{
			name: "範囲外のポート",
			draft: ProfileDraft{
				Name:     valid.Name,
				DBType:   valid.DBType,
				Host:     valid.Host,
				Port:     0,
				Database: valid.Database,
				Schema:   valid.Schema,
				User:     valid.User,
			},
			wantErr: true,
		},
		{
			name: "空のデータベース名",
			draft: ProfileDraft{
				Name:   valid.Name,
				DBType: valid.DBType,
				Host:   valid.Host,
				Port:   valid.Port,
				Schema: valid.Schema,
				User:   valid.User,
			},
			wantErr: true,
		},
		{
			name: "空のユーザー名",
			draft: ProfileDraft{
				Name:     valid.Name,
				DBType:   valid.DBType,
				Host:     valid.Host,
				Port:     valid.Port,
				Database: valid.Database,
				Schema:   valid.Schema,
			},
			wantErr: true,
		},
		{
			name: "PostgreSQLの空スキーマ",
			draft: ProfileDraft{
				Name:     valid.Name,
				DBType:   valid.DBType,
				Host:     valid.Host,
				Port:     valid.Port,
				Database: valid.Database,
				User:     valid.User,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.draft.Validate()
			if gotErr := errors.Is(err, ErrInvalidProfile); gotErr != tt.wantErr {
				t.Errorf("Validate() invalid profile error = %v, want %v (error = %v)", gotErr, tt.wantErr, err)
			}
		})
	}
}

// 接続プロファイル変換検証
func TestProfileDraftToProfile(t *testing.T) {
	draft := ProfileDraft{
		Name:     "Local DB",
		DBType:   DBTypePostgres,
		Host:     "localhost",
		Port:     5432,
		Database: "app",
		Schema:   "public",
		User:     "user",
		Password: "secret",
	}
	tests := []struct {
		name    string
		draft   ProfileDraft
		id      string
		want    Profile
		wantErr bool
	}{
		{
			name:  "下書きをプロファイルへ変換する",
			draft: draft,
			id:    "profile-1",
			want: Profile{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   DBTypePostgres,
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			},
		},
		{
			name:    "空のIDを拒否する",
			draft:   draft,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.draft.ToProfile(tt.id)

			if gotErr := errors.Is(err, ErrInvalidProfile); gotErr != tt.wantErr {
				t.Errorf("ToProfile() invalid profile error = %v, want %v (error = %v)", gotErr, tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToProfile() = %#v, want %#v", got, tt.want)
			}
		})
	}
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

// フロー状態検証
func TestFlowStateValidate(t *testing.T) {
	tests := []struct {
		name  string
		state FlowState
		valid bool
	}{
		{
			name: "テーブル状態を検証する",
			state: FlowState{
				Version: FlowStateVersion,
				TableStates: map[string]TableFlowState{
					"users": {X: 120.5, Y: -20, Expanded: true},
				},
			},
			valid: true,
		},
		{
			name:  "空状態を検証する",
			state: EmptyFlowState(),
			valid: true,
		},
		{
			name: "未知バージョンを拒否する",
			state: FlowState{
				Version:     FlowStateVersion + 1,
				TableStates: map[string]TableFlowState{},
			},
		},
		{
			name: "空テーブル名を拒否する",
			state: FlowState{
				Version: FlowStateVersion,
				TableStates: map[string]TableFlowState{
					" ": {X: 0, Y: 0},
				},
			},
		},
		{
			name: "非有限座標を拒否する",
			state: FlowState{
				Version: FlowStateVersion,
				TableStates: map[string]TableFlowState{
					"users": {X: math.Inf(1), Y: 0},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if gotValid := err == nil; gotValid != tt.valid {
				t.Errorf("Validate() valid = %v, want %v", gotValid, tt.valid)
			}
		})
	}
}

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}
