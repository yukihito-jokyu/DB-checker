package seed

import (
	"testing"

	"github.com/yukihito-jokyu/DB-checker/test/integration/db"
)

// DB種別SQL取得検証
func TestStatementSQLFor(t *testing.T) {
	statement := Statement{
		MySQL:    "select 1",
		Postgres: "select 2",
	}

	tests := []struct {
		name    string
		kind    db.Kind
		wantSQL string
		wantErr bool
	}{
		{
			name:    "mysql",
			kind:    db.MySQL,
			wantSQL: "select 1",
		},
		{
			name:    "postgres",
			kind:    db.Postgres,
			wantSQL: "select 2",
		},
		{
			name:    "unsupported",
			kind:    db.Kind("sqlite"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, err := statement.SQLFor(tt.kind)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotSQL != tt.wantSQL {
				t.Errorf("SQLFor() = %q, want %q", gotSQL, tt.wantSQL)
			}
		})
	}
}
