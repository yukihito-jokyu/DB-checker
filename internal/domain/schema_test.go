package domain

import (
	"errors"
	"testing"
)

// 有効カラム
func validColumns() []Column {
	return []Column{
		{
			Name:     "id",
			DataType: "int4",
		},
	}
}

// 有効なテーブル問い合わせ
func validTableQuery() TableQuery {
	return TableQuery{
		Table: TableRef{
			Namespace: "public",
			Name:      "users",
		},
		Page: 1,
		Columns: []Column{
			{
				Name:     "name",
				DataType: "varchar",
			},
			{
				Name:     "age",
				DataType: "int",
			},
			{
				Name:     "payload",
				DataType: "json",
			},
			{
				Name:     "enabled",
				DataType: "bool",
			},
		},
	}
}

// テーブル参照生成検証
func TestNewTableRef(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		table     string
		wantErr   bool
	}{
		{
			name:      "有効な参照を返す",
			namespace: "public",
			table:     "items",
		},
		{
			name:      "空白namespaceを拒否する",
			namespace: " ",
			table:     "items",
			wantErr:   true,
		},
		{
			name:      "空白テーブル名を拒否する",
			namespace: "public",
			table:     " ",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTableRef(tt.namespace, tt.table)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("NewTableRef() error = %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Namespace != tt.namespace {
				t.Errorf("Namespace = %q, want %q", got.Namespace, tt.namespace)
			}
			if got.Name != tt.table {
				t.Errorf("Name = %q, want %q", got.Name, tt.table)
			}
		})
	}
}

// スキーマ検証
func TestSchemaValidate(t *testing.T) {
	tests := []struct {
		name    string
		value   Schema
		wantErr error
	}{
		{
			name: "外部キーを含むスキーマを受け入れる",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "public",
						Name:      "users",
						Columns:   validColumns(),
					},
					{
						Namespace: "public",
						Name:      "orders",
						Columns:   validColumns(),
					},
				},
				ForeignKeys: []ForeignKey{
					{
						Name:        "orders_user_id_fkey",
						FromTable:   "orders",
						FromColumns: []string{"user_id"},
						ToTable:     "users",
						ToColumns:   []string{"id"},
					},
				},
			},
		},
		{
			name: "異なるnamespaceを拒否する",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "other",
						Name:      "users",
						Columns:   validColumns(),
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "空テーブル名を拒否する",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "public",
						Columns:   validColumns(),
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "重複テーブル名を拒否する",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "public",
						Name:      "users",
						Columns:   validColumns(),
					},
					{
						Namespace: "public",
						Name:      "users",
						Columns:   validColumns(),
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "不正カラムを拒否する",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "public",
						Name:      "users",
						Columns: []Column{
							{
								DataType: "int4",
							},
						},
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "不正な外部キー定義を拒否する",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "public",
						Name:      "users",
						Columns:   validColumns(),
					},
				},
				ForeignKeys: []ForeignKey{
					{
						FromTable:   "users",
						FromColumns: []string{"user_id"},
						ToTable:     "users",
						ToColumns:   []string{"id"},
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "存在しない参照元テーブルを拒否する",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "public",
						Name:      "users",
						Columns:   validColumns(),
					},
				},
				ForeignKeys: []ForeignKey{
					{
						Name:        "orders_user_id_fkey",
						FromTable:   "orders",
						FromColumns: []string{"user_id"},
						ToTable:     "users",
						ToColumns:   []string{"id"},
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "存在しない参照先テーブルを拒否する",
			value: Schema{
				Tables: []Table{
					{
						Namespace: "public",
						Name:      "orders",
						Columns:   validColumns(),
					},
					{
						Namespace: "public",
						Name:      "users",
						Columns:   validColumns(),
					},
				},
				ForeignKeys: []ForeignKey{
					{
						Name:        "orders_user_id_fkey",
						FromTable:   "orders",
						FromColumns: []string{"user_id"},
						ToTable:     "accounts",
						ToColumns:   []string{"id"},
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.value.Validate("public")
			if gotErr := err != nil; gotErr != (tt.wantErr != nil) {
				t.Fatalf("Validate() error = %v, want error %v", err, tt.wantErr != nil)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// テーブル問い合わせ検証
func TestTableQueryValidate(t *testing.T) {
	tests := []struct {
		name    string
		query   TableQuery
		wantErr error
	}{
		{
			name:  "基本問い合わせを受け入れる",
			query: validTableQuery(),
		},
		{
			name: "有効なフィルターを受け入れる",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Page: 1,
				Columns: []Column{
					{
						Name:     "age",
						DataType: "int",
					},
				},
				Filter: &FilterGroup{
					Operator: FilterGroupOperatorAnd,
					Filters: []TableFilter{
						{
							Column:   "age",
							Operator: FilterOperatorGreater,
							Values:   []string{"20"},
						},
					},
				},
			},
		},
		{
			name: "不正なフィルターを拒否する",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Page: 1,
				Columns: []Column{
					{
						Name:     "age",
						DataType: "int",
					},
				},
				Filter: &FilterGroup{
					Operator: FilterGroupOperatorAnd,
					Filters: []TableFilter{
						{
							Column:   "age",
							Operator: FilterOperatorLike,
							Values:   []string{"2%"},
						},
					},
				},
			},
			wantErr: ErrInvalidTableFilter,
		},
		{
			name: "空白namespaceを拒否する",
			query: TableQuery{
				Table: TableRef{
					Namespace: " ",
					Name:      "users",
				},
				Page:    1,
				Columns: validColumns(),
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "不正ページを拒否する",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Columns: validColumns(),
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "空カラムを拒否する",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Page: 1,
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "不正カラム定義を拒否する",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Page: 1,
				Columns: []Column{
					{
						Name: "id",
					},
				},
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "有効な並び替えを受け入れる",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Page:    1,
				Columns: validColumns(),
				Sort: &TableSort{
					Column:    "id",
					Direction: SortDirectionAscending,
				},
			},
		},
		{
			name: "存在しない並び替え列を拒否する",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Page:    1,
				Columns: validColumns(),
				Sort: &TableSort{
					Column:    "unknown",
					Direction: SortDirectionAscending,
				},
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "不正な並び替え方向を拒否する",
			query: TableQuery{
				Table: TableRef{
					Namespace: "public",
					Name:      "users",
				},
				Page:    1,
				Columns: validColumns(),
				Sort: &TableSort{
					Column:    "id",
					Direction: "sideways",
				},
			},
			wantErr: ErrInvalidTableQuery,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if gotErr := err != nil; gotErr != (tt.wantErr != nil) {
				t.Fatalf("Validate() error = %v, want error %v", err, tt.wantErr != nil)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// フィルターグループ検証
func TestValidateFilterGroup(t *testing.T) {
	columns := map[string]Column{
		"id": {
			Name:     "id",
			DataType: "int",
		},
	}
	tooManyFilters := make([]TableFilter, 21)
	for i := range tooManyFilters {
		tooManyFilters[i] = TableFilter{
			Column:   "id",
			Operator: FilterOperatorEqual,
			Values:   []string{"1"},
		}
	}
	tests := []struct {
		name    string
		group   FilterGroup
		depth   int
		wantErr error
	}{
		{
			name: "フィルターを受け入れる",
			group: FilterGroup{
				Operator: FilterGroupOperatorAnd,
				Filters: []TableFilter{
					{
						Column:   "id",
						Operator: FilterOperatorEqual,
						Values:   []string{"1"},
					},
				},
			},
			depth: 1,
		},
		{
			name: "子グループを再帰して受け入れる",
			group: FilterGroup{
				Operator: FilterGroupOperatorOr,
				Groups: []FilterGroup{
					{
						Operator: FilterGroupOperatorAnd,
						Filters: []TableFilter{
							{
								Column:   "id",
								Operator: FilterOperatorGreater,
								Values:   []string{"1"},
							},
						},
					},
				},
			},
			depth: 1,
		},
		{
			name: "深さ超過を拒否する",
			group: FilterGroup{
				Operator: FilterGroupOperatorAnd,
				Filters: []TableFilter{
					{
						Column:   "id",
						Operator: FilterOperatorEqual,
						Values:   []string{"1"},
					},
				},
			},
			depth:   4,
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "不正グループ演算子を拒否する",
			group: FilterGroup{
				Operator: "xor",
				Filters: []TableFilter{
					{
						Column:   "id",
						Operator: FilterOperatorEqual,
						Values:   []string{"1"},
					},
				},
			},
			depth:   1,
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "空グループを拒否する",
			group: FilterGroup{
				Operator: FilterGroupOperatorAnd,
			},
			depth:   1,
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "21件のフィルターを拒否する",
			group: FilterGroup{
				Operator: FilterGroupOperatorAnd,
				Filters:  tooManyFilters,
			},
			depth:   1,
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "不正フィルターを拒否する",
			group: FilterGroup{
				Operator: FilterGroupOperatorAnd,
				Filters: []TableFilter{
					{
						Column:   "unknown",
						Operator: FilterOperatorEqual,
						Values:   []string{"1"},
					},
				},
			},
			depth:   1,
			wantErr: ErrInvalidTableFilter,
		},
		{
			name: "不正な子グループを拒否する",
			group: FilterGroup{
				Operator: FilterGroupOperatorAnd,
				Groups: []FilterGroup{
					{
						Operator: "xor",
						Filters: []TableFilter{
							{
								Column:   "id",
								Operator: FilterOperatorEqual,
								Values:   []string{"1"},
							},
						},
					},
				},
			},
			depth:   1,
			wantErr: ErrInvalidTableQuery,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			err := validateFilterGroup(tt.group, columns, tt.depth, &count)
			if gotErr := err != nil; gotErr != (tt.wantErr != nil) {
				t.Fatalf("validateFilterGroup() error = %v, want error %v", err, tt.wantErr != nil)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("validateFilterGroup() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// テーブルフィルター検証
func TestValidateTableFilter(t *testing.T) {
	columns := map[string]Column{
		"name": {
			Name:     "name",
			DataType: "varchar",
		},
		"age": {
			Name:     "age",
			DataType: "int",
		},
		"payload": {
			Name:     "payload",
			DataType: "json",
		},
		"enabled": {
			Name:     "enabled",
			DataType: "bool",
		},
	}
	tests := []struct {
		name    string
		filter  TableFilter
		wantErr error
	}{
		{
			name: "NULL判定を受け入れる",
			filter: TableFilter{
				Column:   "name",
				Operator: FilterOperatorIsNull,
			},
		},
		{
			name: "NOT NULL判定を受け入れる",
			filter: TableFilter{
				Column:   "name",
				Operator: FilterOperatorIsNotNull,
			},
		},
		{
			name: "NULL判定の値を拒否する",
			filter: TableFilter{
				Column:   "name",
				Operator: FilterOperatorIsNull,
				Values:   []string{"x"},
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "INを受け入れる",
			filter: TableFilter{
				Column:   "name",
				Operator: FilterOperatorIn,
				Values: []string{
					"a",
					"b",
				},
			},
		},
		{
			name: "空INを拒否する",
			filter: TableFilter{
				Column:   "name",
				Operator: FilterOperatorIn,
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "101件のINを拒否する",
			filter: TableFilter{
				Column:   "name",
				Operator: FilterOperatorIn,
				Values:   make([]string, 101),
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "BETWEENを受け入れる",
			filter: TableFilter{
				Column:   "age",
				Operator: FilterOperatorBetween,
				Values: []string{
					"1",
					"2",
				},
			},
		},
		{
			name: "値不足のBETWEENを拒否する",
			filter: TableFilter{
				Column:   "age",
				Operator: FilterOperatorBetween,
				Values:   []string{"1"},
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "比較演算子を受け入れる",
			filter: TableFilter{
				Column:   "age",
				Operator: FilterOperatorGreater,
				Values:   []string{"1"},
			},
		},
		{
			name: "比較演算子の値不足を拒否する",
			filter: TableFilter{
				Column:   "age",
				Operator: FilterOperatorGreater,
			},
			wantErr: ErrInvalidTableQuery,
		},
		{
			name: "不正演算子を拒否する",
			filter: TableFilter{
				Column:   "age",
				Operator: "REGEXP",
				Values:   []string{"1"},
			},
			wantErr: ErrInvalidTableFilter,
		},
		{
			name: "テキストのBETWEENを拒否する",
			filter: TableFilter{
				Column:   "name",
				Operator: FilterOperatorBetween,
				Values: []string{
					"a",
					"z",
				},
			},
			wantErr: ErrInvalidTableFilter,
		},
		{
			name: "JSONのLIKEを拒否する",
			filter: TableFilter{
				Column:   "payload",
				Operator: FilterOperatorLike,
				Values:   []string{"%a%"},
			},
			wantErr: ErrInvalidTableFilter,
		},
		{
			name: "真偽値の範囲比較を拒否する",
			filter: TableFilter{
				Column:   "enabled",
				Operator: FilterOperatorGreater,
				Values:   []string{"true"},
			},
			wantErr: ErrInvalidTableFilter,
		},
		{
			name: "存在しない列を拒否する",
			filter: TableFilter{
				Column:   "unknown",
				Operator: FilterOperatorEqual,
				Values:   []string{"x"},
			},
			wantErr: ErrInvalidTableFilter,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTableFilter(tt.filter, columns)
			if gotErr := err != nil; gotErr != (tt.wantErr != nil) {
				t.Fatalf("validateTableFilter() error = %v, want error %v", err, tt.wantErr != nil)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("validateTableFilter() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// フィルター演算子適用可否検証
func TestFilterOperatorAllowed(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		operator FilterOperator
		want     bool
	}{
		{
			name:     "バイナリーの一致を許可する",
			dataType: "bytea",
			operator: FilterOperatorEqual,
			want:     true,
		},
		{
			name:     "テキストのLIKEを許可する",
			dataType: "varchar",
			operator: FilterOperatorLike,
			want:     true,
		},
		{
			name:     "数値の範囲比較を許可する",
			dataType: "int",
			operator: FilterOperatorLessEq,
			want:     true,
		},
		{
			name:     "真偽値のINを許可する",
			dataType: "bool",
			operator: FilterOperatorIn,
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterOperatorAllowed(tt.dataType, tt.operator); got != tt.want {
				t.Errorf("filterOperatorAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// テーブル構造検証
func TestTableStructureValidate(t *testing.T) {
	ref := TableRef{
		Namespace: "public",
		Name:      "orders",
	}
	tests := []struct {
		name    string
		value   TableStructure
		wantErr error
	}{
		{
			name: "外部キーとインデックスを含む構造を受け入れる",
			value: TableStructure{
				Table: Table{
					Namespace: "public",
					Name:      "orders",
					Columns:   validColumns(),
				},
				ForeignKeys: []ForeignKey{
					{
						Name:        "orders_user_id_fkey",
						FromTable:   "orders",
						FromColumns: []string{"user_id"},
						ToTable:     "users",
						ToColumns:   []string{"id"},
					},
				},
				Indexes: []Index{
					{
						Name:    "orders_pkey",
						Columns: []string{"id"},
						Unique:  true,
						Kind:    "btree",
					},
				},
			},
		},
		{
			name: "異なるテーブルを拒否する",
			value: TableStructure{
				Table: Table{
					Namespace: "public",
					Name:      "users",
					Columns:   validColumns(),
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "不正カラムを拒否する",
			value: TableStructure{
				Table: Table{
					Namespace: "public",
					Name:      "orders",
					Columns: []Column{
						{
							Name: "id",
						},
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "不正外部キーを拒否する",
			value: TableStructure{
				Table: Table{
					Namespace: "public",
					Name:      "orders",
					Columns:   validColumns(),
				},
				ForeignKeys: []ForeignKey{
					{
						Name:        "orders_user_id_fkey",
						FromTable:   "users",
						FromColumns: []string{"user_id"},
						ToTable:     "users",
						ToColumns:   []string{"id"},
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "不正インデックスを拒否する",
			value: TableStructure{
				Table: Table{
					Namespace: "public",
					Name:      "orders",
					Columns:   validColumns(),
				},
				Indexes: []Index{
					{
						Columns: []string{"id"},
						Kind:    "btree",
					},
				},
			},
			wantErr: ErrInvalidSchema,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.value.Validate(ref)
			if gotErr := err != nil; gotErr != (tt.wantErr != nil) {
				t.Fatalf("Validate() error = %v, want error %v", err, tt.wantErr != nil)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// カラム検証
func TestValidateColumns(t *testing.T) {
	tests := []struct {
		name    string
		columns []Column
		wantErr error
	}{
		{
			name:    "空カラム一覧を受け入れる",
			columns: nil,
		},
		{
			name: "複数カラムを受け入れる",
			columns: []Column{
				{
					Name:     "id",
					DataType: "int",
				},
				{
					Name:     "name",
					DataType: "varchar",
				},
			},
		},
		{
			name: "空カラム名を拒否する",
			columns: []Column{
				{
					DataType: "int",
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "空データ型を拒否する",
			columns: []Column{
				{
					Name: "id",
				},
			},
			wantErr: ErrInvalidSchema,
		},
		{
			name: "重複カラム名を拒否する",
			columns: []Column{
				{
					Name:     "id",
					DataType: "int",
				},
				{
					Name:     "id",
					DataType: "int",
				},
			},
			wantErr: ErrInvalidSchema,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateColumns(tt.columns)
			if gotErr := err != nil; gotErr != (tt.wantErr != nil) {
				t.Fatalf("validateColumns() error = %v, want error %v", err, tt.wantErr != nil)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("validateColumns() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// 行追加入力検証テスト
func TestInsertRowValidate(t *testing.T) {
	t.Parallel()

	valStr := "test"
	cols := []Column{
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
		{
			Name:         "age",
			DataType:     "int",
			Nullable:     true,
			DefaultValue: nil,
		},
		{
			Name:         "created_at",
			DataType:     "datetime",
			Nullable:     false,
			DefaultValue: &valStr,
		},
	}

	tests := []struct {
		name    string
		row     InsertRow
		wantErr error
	}{
		{
			name: "正常な入力値を検証できる",
			row: InsertRow{
				Table: TableRef{Namespace: "main", Name: "users"},
				Values: []ColumnValueInput{
					{
						Column: "id",
						Kind:   CellKindDefault,
					},
					{
						Column: "name",
						Kind:   CellKindValue,
						Value:  &valStr,
					},
					{
						Column: "age",
						Kind:   CellKindNull,
					},
					{
						Column: "created_at",
						Kind:   CellKindDefault,
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "無効なテーブル参照を拒否する",
			row: InsertRow{
				Table: TableRef{Namespace: "", Name: "users"},
				Values: []ColumnValueInput{
					{
						Column: "name",
						Kind:   CellKindValue,
						Value:  &valStr,
					},
				},
			},
			wantErr: ErrInvalidRowInput,
		},
		{
			name: "存在しないカラム名を拒否する",
			row: InsertRow{
				Table: TableRef{Namespace: "main", Name: "users"},
				Values: []ColumnValueInput{
					{
						Column: "unknown",
						Kind:   CellKindValue,
						Value:  &valStr,
					},
				},
			},
			wantErr: ErrInvalidRowInput,
		},
		{
			name: "NOT NULLカラムへのNULL出力を拒否する",
			row: InsertRow{
				Table: TableRef{Namespace: "main", Name: "users"},
				Values: []ColumnValueInput{
					{
						Column: "id",
						Kind:   CellKindDefault,
					},
					{
						Column: "name",
						Kind:   CellKindNull,
					},
				},
			},
			wantErr: ErrInvalidRowInput,
		},
		{
			name: "必須カラム未指定を拒否する",
			row: InsertRow{
				Table: TableRef{Namespace: "main", Name: "users"},
				Values: []ColumnValueInput{
					{
						Column: "id",
						Kind:   CellKindDefault,
					},
					{
						Column: "age",
						Kind:   CellKindNull,
					},
				},
			},
			wantErr: ErrInvalidRowInput,
		},
		{
			name: "CellKindValueでのnil値を拒否する",
			row: InsertRow{
				Table: TableRef{Namespace: "main", Name: "users"},
				Values: []ColumnValueInput{
					{Column: "name", Kind: CellKindValue, Value: nil},
				},
			},
			wantErr: ErrInvalidRowInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.row.Validate(cols)
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// セル更新入力検証
func TestCellUpdateValidate(t *testing.T) {
	value := "1"
	name := "updated"
	columns := []Column{
		{Name: "id", DataType: "int4", IsPrimaryKey: true},
		{Name: "name", DataType: "text"},
		{Name: "note", DataType: "text", Nullable: true},
	}
	tests := []struct {
		name    string
		change  CellUpdate
		columns []Column
		wantErr bool
	}{
		{
			name: "主キーの編集前値でセルを更新できる",
			change: CellUpdate{
				Table:   TableRef{Namespace: "public", Name: "users"},
				Locator: RowLocator{Values: []ColumnValueInput{{Column: "id", Kind: CellKindValue, Value: &value}}},
				Column:  "name",
				Value:   CellValue{Kind: CellKindValue, Value: name},
			},
		},
		{
			name: "既定値へセルを更新できる",
			change: CellUpdate{
				Table:   TableRef{Namespace: "public", Name: "users"},
				Locator: RowLocator{Values: []ColumnValueInput{{Column: "id", Kind: CellKindValue, Value: &value}}},
				Column:  "name",
				Value:   CellValue{Kind: CellKindDefault},
			},
		},
		{
			name: "主キー以外の位置指定を拒否する",
			change: CellUpdate{
				Table:   TableRef{Namespace: "public", Name: "users"},
				Locator: RowLocator{Values: []ColumnValueInput{{Column: "name", Kind: CellKindValue, Value: &name}}},
				Column:  "note",
				Value:   CellValue{Kind: CellKindNull},
			},
			wantErr: true,
		},
		{
			name: "複合主キー位置指定の不足を拒否する",
			change: CellUpdate{
				Table:   TableRef{Namespace: "public", Name: "users"},
				Locator: RowLocator{Values: []ColumnValueInput{{Column: "id", Kind: CellKindValue, Value: &value}}},
				Column:  "name",
				Value:   CellValue{Kind: CellKindValue, Value: name},
			},
			columns: []Column{
				{Name: "id", DataType: "int4", IsPrimaryKey: true},
				{Name: "tenant_id", DataType: "int4", IsPrimaryKey: true},
				{Name: "name", DataType: "text"},
			},
			wantErr: true,
		},
		{
			name: "非NULL列へのNULL更新を拒否する",
			change: CellUpdate{
				Table:   TableRef{Namespace: "public", Name: "users"},
				Locator: RowLocator{Values: []ColumnValueInput{{Column: "id", Kind: CellKindValue, Value: &value}}},
				Column:  "name",
				Value:   CellValue{Kind: CellKindNull},
			},
			wantErr: true,
		},
		{
			name:    "空の入力を拒否する",
			change:  CellUpdate{},
			wantErr: true,
		},
		{
			name: "空名の構造列を拒否する",
			change: CellUpdate{
				Table:   TableRef{Namespace: "public", Name: "users"},
				Locator: RowLocator{Values: []ColumnValueInput{{Column: "id", Kind: CellKindValue, Value: &value}}},
				Column:  "name",
				Value:   CellValue{Kind: CellKindValue, Value: name},
			},
			columns: []Column{{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testColumns := columns
			if tt.columns != nil {
				testColumns = tt.columns
			}
			err := tt.change.Validate(testColumns)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 行位置NULL値検証
func TestValidLocatorValue(t *testing.T) {
	value := "1"
	tests := []struct {
		name   string
		value  ColumnValueInput
		column Column
		wantOK bool
	}{
		{
			name:   "NULL許可列のNULLを許可する",
			value:  ColumnValueInput{Kind: CellKindNull},
			column: Column{Nullable: true},
			wantOK: true,
		},
		{
			name:   "NULL値付きNULL位置指定を拒否する",
			value:  ColumnValueInput{Kind: CellKindNull, Value: &value},
			column: Column{Nullable: true},
			wantOK: false,
		},
		{
			name:   "NULL非許可列のNULLを拒否する",
			value:  ColumnValueInput{Kind: CellKindNull},
			column: Column{},
			wantOK: false,
		},
		{
			name:   "値なしの値位置指定を拒否する",
			value:  ColumnValueInput{Kind: CellKindValue},
			column: Column{},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validLocatorValue(tt.value, tt.column); got != tt.wantOK {
				t.Errorf("validLocatorValue() = %v, want %v", got, tt.wantOK)
			}
		})
	}
}

// 主キーなしセル更新入力検証
func TestCellUpdateValidateWithoutPrimaryKey(t *testing.T) {
	name := "before"
	note := "memo"
	columns := []Column{
		{Name: "name", DataType: "text"},
		{Name: "note", DataType: "text", Nullable: true},
	}
	tests := []struct {
		name    string
		change  CellUpdate
		wantErr bool
	}{
		{
			name: "全カラムの編集前値で更新できる",
			change: CellUpdate{
				Table: TableRef{Namespace: "public", Name: "logs"},
				Locator: RowLocator{Values: []ColumnValueInput{
					{Column: "name", Kind: CellKindValue, Value: &name},
					{Column: "note", Kind: CellKindValue, Value: &note},
				}},
				Column: "name",
				Value:  CellValue{Kind: CellKindValue, Value: "after"},
			},
		},
		{
			name: "カラム位置指定の欠落を拒否する",
			change: CellUpdate{
				Table:   TableRef{Namespace: "public", Name: "logs"},
				Locator: RowLocator{Values: []ColumnValueInput{{Column: "name", Kind: CellKindValue, Value: &name}}},
				Column:  "name",
				Value:   CellValue{Kind: CellKindValue, Value: "after"},
			},
			wantErr: true,
		},
		{
			name: "重複する位置指定を拒否する",
			change: CellUpdate{
				Table: TableRef{Namespace: "public", Name: "logs"},
				Locator: RowLocator{Values: []ColumnValueInput{
					{Column: "name", Kind: CellKindValue, Value: &name},
					{Column: "name", Kind: CellKindValue, Value: &name},
				}},
				Column: "name",
				Value:  CellValue{Kind: CellKindValue, Value: "after"},
			},
			wantErr: true,
		},
		{
			name: "未知カラムの位置指定を拒否する",
			change: CellUpdate{
				Table: TableRef{Namespace: "public", Name: "logs"},
				Locator: RowLocator{Values: []ColumnValueInput{
					{Column: "name", Kind: CellKindValue, Value: &name},
					{Column: "unknown", Kind: CellKindValue, Value: &note},
				}},
				Column: "name",
				Value:  CellValue{Kind: CellKindValue, Value: "after"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.change.Validate(columns)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 複合主キーセル更新入力検証
func TestCellUpdateValidateCompositePrimaryKey(t *testing.T) {
	partA := "1"
	partB := "2"
	columns := []Column{
		{Name: "part_a", DataType: "int4", IsPrimaryKey: true},
		{Name: "part_b", DataType: "int4", IsPrimaryKey: true},
		{Name: "name", DataType: "text"},
	}
	change := CellUpdate{
		Table: TableRef{Namespace: "public", Name: "pairs"},
		Locator: RowLocator{Values: []ColumnValueInput{
			{Column: "part_a", Kind: CellKindValue, Value: &partA},
			{Column: "part_b", Kind: CellKindValue, Value: &partB},
		}},
		Column: "part_a",
		Value:  CellValue{Kind: CellKindValue, Value: "3"},
	}

	if err := change.Validate(columns); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}
