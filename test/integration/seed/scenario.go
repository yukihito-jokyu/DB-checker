package seed

import (
	"fmt"

	"github.com/yukihito-jokyu/DB-checker/test/integration/db"
)

type Scenario struct {
	Name       string
	Schema     []Statement
	Data       []Statement
	Cleanup    []Statement
	Assertions []string
}

type Statement struct {
	Name     string
	MySQL    string
	Postgres string
}

// DB種別SQL取得
func (s Statement) SQLFor(kind db.Kind) (string, error) {
	switch kind {
	case db.MySQL:
		return s.MySQL, nil
	case db.Postgres:
		return s.Postgres, nil
	default:
		return "", fmt.Errorf("unsupported db kind: %s", kind)
	}
}
