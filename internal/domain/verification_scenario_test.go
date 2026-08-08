package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// 検証シナリオ下書き検証
func TestNewVerificationScenarioDraft(t *testing.T) {
	tests := []struct {
		name          string
		nameValue     string
		primaryTable  string
		definition    map[string]any
		wantError     error
		wantName      string
		wantPrimary   string
		wantCreatedAt time.Time
	}{
		{
			name:         "必須項目を正規化して生成する",
			nameValue:    " 検証 ",
			primaryTable: " orders ",
			definition:   validVerificationScenarioDefinition(),
			wantName:     "検証",
			wantPrimary:  "orders",
		},
		{
			name:         "主対象の一意生成規則不足を拒否する",
			nameValue:    "検証",
			primaryTable: "orders",
			definition: map[string]any{
				"childTables": []string{},
				"rowCounts":   map[string]int{"orders": 1},
				"columnGenerators": map[string]map[string]map[string]string{
					"orders": {"id": {"kind": "fixed"}},
				},
				"sql":              []string{"SELECT 1"},
				"warmupRuns":       0,
				"iterations":       1,
				"timeLimitSeconds": 1,
			},
			wantError: ErrPrimaryKeyRequired,
		},
		{
			name:         "件数範囲違反を拒否する",
			nameValue:    "検証",
			primaryTable: "orders",
			definition: map[string]any{
				"childTables": []string{},
				"rowCounts":   map[string]int{"orders": 0},
				"columnGenerators": map[string]map[string]map[string]string{
					"orders": {"id": {"kind": "sequence"}},
				},
				"sql":              []string{"SELECT 1"},
				"warmupRuns":       0,
				"iterations":       1,
				"timeLimitSeconds": 1,
			},
			wantError: ErrInvalidVerificationScenarioDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewVerificationScenarioDraft(tt.nameValue, tt.primaryTable, tt.definition)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewVerificationScenarioDraft() error = %v, want %v", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("NewVerificationScenarioDraft() error = %v", err)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.PrimaryTable != tt.wantPrimary {
				t.Errorf("PrimaryTable = %q, want %q", got.PrimaryTable, tt.wantPrimary)
			}
		})
	}
}

// 検証シナリオ下書き全項目検証
func TestNewVerificationScenarioDraftValidation(t *testing.T) {
	tests := []struct {
		name       string
		nameValue  string
		primary    string
		definition map[string]any
		wantError  error
	}{
		{
			name:       "名前の100文字境界を許可する",
			nameValue:  strings.Repeat("あ", 100),
			primary:    "orders",
			definition: validVerificationScenarioDefinition(),
		},
		{
			name:       "名前の101文字を拒否する",
			nameValue:  strings.Repeat("あ", 101),
			primary:    "orders",
			definition: validVerificationScenarioDefinition(),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "制御文字を含む名前を拒否する",
			nameValue:  "検証\n名",
			primary:    "orders",
			definition: validVerificationScenarioDefinition(),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:      "nil定義を拒否する",
			nameValue: "検証",
			primary:   "orders",
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "JSON化できない定義を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: map[string]any{"unsupported": func() {}},
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "空の主対象を拒否する",
			nameValue:  "検証",
			primary:    " \t ",
			definition: validVerificationScenarioDefinition(),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "制御文字を含む主対象を拒否する",
			nameValue:  "検証",
			primary:    "orders\x00",
			definition: validVerificationScenarioDefinition(),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:      "子テーブルの空文字を拒否する",
			nameValue: "検証",
			primary:   "orders",
			definition: map[string]any{
				"childTables": []string{" "},
				"rowCounts": map[string]int{
					"orders": 1,
					"":       1,
				},
				"columnGenerators": map[string]any{
					"orders": map[string]any{"id": map[string]any{"kind": "sequence"}},
					"":       map[string]any{},
				},
				"sql":              []string{"SELECT 1"},
				"warmupRuns":       0,
				"iterations":       1,
				"timeLimitSeconds": 1,
			},
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name:      "文字列以外の子テーブルを拒否する",
			nameValue: "検証",
			primary:   "orders",
			definition: map[string]any{
				"childTables": []any{float64(1)},
				"rowCounts": map[string]any{
					"orders": float64(1),
				},
				"columnGenerators": map[string]any{
					"orders": map[string]any{"id": map[string]any{"kind": "sequence"}},
				},
				"sql":              []any{"SELECT 1"},
				"warmupRuns":       float64(0),
				"iterations":       float64(1),
				"timeLimitSeconds": float64(1),
			},
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name:      "重複する子テーブルを拒否する",
			nameValue: "検証",
			primary:   "orders",
			definition: scenarioDefinitionWithChildTables([]string{
				"items",
				"items",
			}),
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "主対象と同じ子テーブルを拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: scenarioDefinitionWithChildTables([]string{"orders"}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "主対象の件数不足を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("rowCounts", map[string]int{"orders": 0}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "件数オブジェクト以外を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("rowCounts", []int{1}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "件数の上限を許可する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("rowCounts", map[string]int{"orders": 100000}),
		},
		{
			name:       "件数の上限超過を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("rowCounts", map[string]int{"orders": 100001}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "子テーブルの件数不足を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWithoutChildRowCount(),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "主対象の生成規則不足を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("columnGenerators", map[string]any{}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "生成規則オブジェクト以外を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("columnGenerators", []string{"sequence"}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "生成規則のkind不足を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("columnGenerators", map[string]any{"orders": map[string]any{"id": map[string]any{}}}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "許可外の生成規則を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("columnGenerators", map[string]any{"orders": map[string]any{"id": map[string]any{"kind": "random"}}}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "SQLの空配列を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("sql", []string{}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "SQLの50件境界を許可する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("sql", repeatedStatements(50)),
		},
		{
			name:       "SQLの51件を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("sql", repeatedStatements(51)),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "空白のみのSQLを拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("sql", []string{" \t"}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "NULを含むSQLを拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWith("sql", []string{"SELECT\x00 1"}),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "数値の最小最大境界を許可する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWithNumbers(1000, 100000, 3600),
		},
		{
			name:       "warmupRunsの上限超過を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWithNumbers(1001, 1, 1),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "iterationsの下限未満を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWithNumbers(0, 0, 1),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
		{
			name:       "timeLimitSecondsの上限超過を拒否する",
			nameValue:  "検証",
			primary:    "orders",
			definition: definitionWithNumbers(0, 1, 3601),
			wantError:  ErrInvalidVerificationScenarioDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVerificationScenarioDraft(tt.nameValue, tt.primary, tt.definition)
			if !errors.Is(err, tt.wantError) {
				t.Errorf("NewVerificationScenarioDraft() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

// シナリオ下書きのJSON正規化検証
func TestNewVerificationScenarioDraftNormalizesJSON(t *testing.T) {
	draft, err := NewVerificationScenarioDraft(" 検証 ", " orders ", validVerificationScenarioDefinition())
	if err != nil {
		t.Fatalf("NewVerificationScenarioDraft() error = %v", err)
	}

	childTables, ok := draft.Definition["childTables"].([]any)
	if !ok {
		t.Fatalf("childTables type = %T, want []any", draft.Definition["childTables"])
	}
	if !reflect.DeepEqual(childTables, []any{}) {
		t.Errorf("childTables = %#v, want empty []any", childTables)
	}
	rowCounts, ok := draft.Definition["rowCounts"].(map[string]any)
	if !ok {
		t.Fatalf("rowCounts type = %T, want map[string]any", draft.Definition["rowCounts"])
	}
	if got := rowCounts["orders"]; got != float64(1) {
		t.Errorf("rowCounts[orders] = %#v, want float64(1)", got)
	}
}

// 保存用シナリオの下書き変換検証
func TestVerificationScenarioDraftNewVerificationScenario(t *testing.T) {
	draft, err := NewVerificationScenarioDraft("検証", "orders", validVerificationScenarioDefinition())
	if err != nil {
		t.Fatalf("NewVerificationScenarioDraft() error = %v", err)
	}
	createdAt := time.Date(2026, time.August, 8, 11, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	tests := []struct {
		name      string
		id        string
		draft     VerificationScenarioDraft
		wantError error
	}{
		{
			name:  "UTC時刻で生成する",
			id:    "scenario-1",
			draft: draft,
		},
		{
			name:      "空IDを拒否する",
			draft:     draft,
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name: "不正な下書きを拒否する",
			id:   "scenario-1",
			draft: VerificationScenarioDraft{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			},
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name: "JSON化できない定義を拒否する",
			id:   "scenario-1",
			draft: VerificationScenarioDraft{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition: map[string]any{
					"childTables": []any{},
					"rowCounts": map[string]any{
						"orders": float64(1),
					},
					"columnGenerators": map[string]any{
						"orders": map[string]any{"id": map[string]any{"kind": "sequence"}},
					},
					"sql":              []any{"SELECT 1"},
					"warmupRuns":       float64(0),
					"iterations":       float64(1),
					"timeLimitSeconds": float64(1),
					"extra":            func() {},
				},
			},
			wantError: ErrInvalidVerificationScenarioDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.draft.NewVerificationScenario(tt.id, createdAt)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewVerificationScenario() error = %v, want %v", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("NewVerificationScenario() error = %v", err)
			}
			if got.CreatedAt.Location() != time.UTC || !got.CreatedAt.Equal(createdAt) {
				t.Errorf("CreatedAt = %v, want UTC %v", got.CreatedAt, createdAt.UTC())
			}
			if !got.UpdatedAt.Equal(got.CreatedAt) {
				t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, got.CreatedAt)
			}
		})
	}
}

// 主対象一意生成規則判定検証
func TestHasPrimaryKeyGenerator(t *testing.T) {
	tests := []struct {
		name         string
		value        any
		primaryTable string
		want         bool
	}{
		{
			name:         "規則オブジェクト以外は不成立",
			value:        []any{},
			primaryTable: "orders",
		},
		{
			name:         "主対象の規則なしは不成立",
			value:        map[string]any{},
			primaryTable: "orders",
		},
		{
			name: "主対象の規則がオブジェクト以外は不成立",
			value: map[string]any{
				"orders": []any{},
			},
			primaryTable: "orders",
		},
		{
			name: "不正な規則は不成立",
			value: map[string]any{
				"orders": map[string]any{"id": "sequence"},
			},
			primaryTable: "orders",
		},
		{
			name: "sequence規則は成立",
			value: map[string]any{
				"orders": map[string]any{"id": map[string]any{"kind": "sequence"}},
			},
			primaryTable: "orders",
			want:         true,
		},
		{
			name: "uuid規則は成立",
			value: map[string]any{
				"orders": map[string]any{"id": map[string]any{"kind": "uuid"}},
			},
			primaryTable: "orders",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasPrimaryKeyGenerator(tt.value, tt.primaryTable); got != tt.want {
				t.Errorf("hasPrimaryKeyGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 有効なシナリオ定義生成
func validVerificationScenarioDefinition() map[string]any {
	return map[string]any{
		"childTables": []string{},
		"rowCounts":   map[string]int{"orders": 1},
		"columnGenerators": map[string]map[string]map[string]string{
			"orders": {"id": {"kind": "sequence"}},
		},
		"sql":              []string{"SELECT 1"},
		"warmupRuns":       0,
		"iterations":       1,
		"timeLimitSeconds": 1,
	}
}

// 子テーブル付き定義生成
func scenarioDefinitionWithChildTables(childTables []string) map[string]any {
	rowCounts := map[string]int{"orders": 1}
	columnGenerators := map[string]any{
		"orders": map[string]any{"id": map[string]any{"kind": "sequence"}},
	}
	for _, childTable := range childTables {
		rowCounts[childTable] = 1
		columnGenerators[childTable] = map[string]any{}
	}

	return map[string]any{
		"childTables":      childTables,
		"rowCounts":        rowCounts,
		"columnGenerators": columnGenerators,
		"sql":              []string{"SELECT 1"},
		"warmupRuns":       0,
		"iterations":       1,
		"timeLimitSeconds": 1,
	}
}

// 項目差し替え定義生成
func definitionWith(key string, value any) map[string]any {
	definition := validVerificationScenarioDefinition()
	definition[key] = value

	return definition
}

// 子テーブル件数不足定義生成
func definitionWithoutChildRowCount() map[string]any {
	definition := scenarioDefinitionWithChildTables([]string{"items"})
	definition["rowCounts"] = map[string]int{"orders": 1}

	return definition
}

// 数値項目差し替え定義生成
func definitionWithNumbers(warmupRuns, iterations, timeLimitSeconds int) map[string]any {
	definition := validVerificationScenarioDefinition()
	definition["warmupRuns"] = warmupRuns
	definition["iterations"] = iterations
	definition["timeLimitSeconds"] = timeLimitSeconds

	return definition
}

// SQL配列生成
func repeatedStatements(count int) []string {
	statements := make([]string, count)
	for index := range statements {
		statements[index] = "SELECT 1"
	}

	return statements
}

// 検証シナリオ複製検証
func TestVerificationScenarioDuplicateVerificationScenario(t *testing.T) {
	workspaceName := "verification_orders"
	scenario := VerificationScenario{
		ID:            "scenario-1",
		Name:          "検証",
		PrimaryTable:  "orders",
		Definition:    validVerificationScenarioDefinition(),
		WorkspaceName: &workspaceName,
		CreatedAt:     time.Date(2026, time.August, 8, 11, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC),
	}
	createdAt := time.Date(2026, time.August, 8, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	tests := []struct {
		name      string
		id        string
		scenario  VerificationScenario
		wantError error
	}{
		{
			name:     "新しいIDと独立した定義で複製する",
			id:       "scenario-2",
			scenario: scenario,
		},
		{
			name:      "空IDを拒否する",
			scenario:  scenario,
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name: "不正な定義を拒否する",
			id:   "scenario-2",
			scenario: VerificationScenario{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			},
			wantError: ErrInvalidVerificationScenarioDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.scenario.DuplicateVerificationScenario(tt.id, createdAt)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("DuplicateVerificationScenario() error = %v, want %v", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("DuplicateVerificationScenario() error = %v", err)
			}
			if got.ID != tt.id || got.Name != scenario.Name || got.PrimaryTable != scenario.PrimaryTable {
				t.Errorf("DuplicateVerificationScenario() = %#v, want copied identity fields", got)
			}
			if got.WorkspaceName != nil {
				t.Errorf("WorkspaceName = %v, want nil", got.WorkspaceName)
			}
			if !got.CreatedAt.Equal(createdAt.UTC()) || !got.UpdatedAt.Equal(createdAt.UTC()) {
				t.Errorf("timestamps = %v, %v, want %v", got.CreatedAt, got.UpdatedAt, createdAt.UTC())
			}
			got.Definition["rowCounts"].(map[string]any)["orders"] = float64(99)
			if scenario.Definition["rowCounts"].(map[string]int)["orders"] != 1 {
				t.Errorf("source Definition = %#v, want unchanged", scenario.Definition)
			}
		})
	}
}

// シナリオ下書きの更新用変換検証
func TestVerificationScenarioDraftUpdateVerificationScenario(t *testing.T) {
	workspaceName := "verification_orders"
	createdAt := time.Date(2026, time.August, 8, 11, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	updatedAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	existing := VerificationScenario{
		ID:            "scenario-1",
		Name:          "更新前",
		PrimaryTable:  "orders",
		Definition:    validVerificationScenarioDefinition(),
		WorkspaceName: &workspaceName,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	draft, err := NewVerificationScenarioDraft("更新後", "orders", validVerificationScenarioDefinition())
	if err != nil {
		t.Fatalf("NewVerificationScenarioDraft() error = %v", err)
	}
	tests := []struct {
		name      string
		scenario  VerificationScenario
		draft     VerificationScenarioDraft
		wantError error
	}{
		{
			name:     "識別情報とワークスペースを保持してUTC更新日時を設定する",
			scenario: existing,
			draft:    draft,
		},
		{
			name:      "IDなしを拒否する",
			scenario:  VerificationScenario{},
			draft:     draft,
			wantError: ErrInvalidVerificationScenarioDraft,
		},
		{
			name:     "形式違反の下書きを拒否する",
			scenario: existing,
			draft: VerificationScenarioDraft{
				Name:         "検証",
				PrimaryTable: "orders",
				Definition:   map[string]any{},
			},
			wantError: ErrInvalidVerificationScenarioDraft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.draft.UpdateVerificationScenario(tt.scenario, updatedAt)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("UpdateVerificationScenario() error = %v, want %v", err, tt.wantError)
				}

				return
			}
			if err != nil {
				t.Fatalf("UpdateVerificationScenario() error = %v", err)
			}
			if got.ID != existing.ID || !got.CreatedAt.Equal(existing.CreatedAt) || got.WorkspaceName != existing.WorkspaceName {
				t.Errorf("UpdateVerificationScenario() = %#v, want preserved identity and workspace", got)
			}
			if got.UpdatedAt.Location() != time.UTC || !got.UpdatedAt.Equal(updatedAt.UTC()) {
				t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, updatedAt.UTC())
			}
			if got.Name != draft.Name || got.PrimaryTable != draft.PrimaryTable || !reflect.DeepEqual(got.Definition, draft.Definition) {
				t.Errorf("UpdateVerificationScenario() = %#v, want draft values", got)
			}
		})
	}
}

// 検証シナリオ生成検証
func TestNewVerificationScenario(t *testing.T) {
	createdAt := time.Date(2026, time.August, 8, 11, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	workspaceName := "verification_orders"
	tests := []struct {
		name           string
		definitionJSON string
		workspaceName  *string
		wantDefinition map[string]any
		wantError      bool
	}{
		{
			name:           "定義オブジェクトをデコードする",
			definitionJSON: `{"rowCounts":{"orders":10}}`,
			workspaceName:  &workspaceName,
			wantDefinition: map[string]any{
				"rowCounts": map[string]any{
					"orders": float64(10),
				},
			},
		},
		{
			name:           "不正なJSONを拒否する",
			definitionJSON: `{`,
			wantError:      true,
		},
		{
			name:           "nullの定義を拒否する",
			definitionJSON: `null`,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewVerificationScenario("scenario-1", "検証", "orders", []byte(tt.definitionJSON), tt.workspaceName, createdAt, updatedAt)
			if tt.wantError {
				if err == nil {
					t.Fatal("NewVerificationScenario() error = nil, want error")
				}

				return
			}
			if err != nil {
				t.Fatalf("NewVerificationScenario() error = %v", err)
			}
			if got.ID != "scenario-1" {
				t.Errorf("ID = %q, want %q", got.ID, "scenario-1")
			}
			if !reflect.DeepEqual(got.Definition, tt.wantDefinition) {
				t.Errorf("Definition = %#v, want %#v", got.Definition, tt.wantDefinition)
			}
			if got.WorkspaceName != tt.workspaceName {
				t.Errorf("WorkspaceName = %p, want %p", got.WorkspaceName, tt.workspaceName)
			}
			if !got.CreatedAt.Equal(createdAt) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, createdAt)
			}
			if !got.UpdatedAt.Equal(updatedAt) {
				t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, updatedAt)
			}
		})
	}
}
