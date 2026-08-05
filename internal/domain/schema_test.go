package domain

import "testing"

// テーブル参照生成検証
func TestNewTableRef(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		table     string
		wantErr   bool
	}{
		{name: "有効な参照を返す", namespace: "public", table: "items"},
		{name: "空白テーブル名を拒否する", namespace: "public", table: " ", wantErr: true},
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
			if got.Namespace != tt.namespace || got.Name != tt.table {
				t.Errorf("NewTableRef() = %#v, want namespace=%q name=%q", got, tt.namespace, tt.table)
			}
		})
	}
}

// テーブル構造検証
func TestTableStructureValidate(t *testing.T) {
	ref, err := NewTableRef("public", "items")
	if err != nil {
		t.Fatalf("NewTableRef() error = %v", err)
	}
	tests := []struct {
		name    string
		value   TableStructure
		wantErr bool
	}{
		{
			name: "詳細属性を含む構造を受け入れる",
			value: TableStructure{
				Table:   Table{Namespace: "public", Name: "items", Columns: []Column{{Name: "id", DataType: "int4"}}},
				Indexes: []Index{{Name: "items_pkey", Columns: []string{"id"}, Unique: true, Kind: "btree"}},
			},
		},
		{
			name:    "異なるテーブルを拒否する",
			value:   TableStructure{Table: Table{Namespace: "public", Name: "other", Columns: []Column{{Name: "id", DataType: "int4"}}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotErr := tt.value.Validate(ref) != nil; gotErr != tt.wantErr {
				t.Errorf("Validate() error = %v, want error %v", gotErr, tt.wantErr)
			}
		})
	}
}
