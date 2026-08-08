package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidVerificationScenarioDraft = errors.New("invalid verification scenario draft")
	ErrPrimaryKeyRequired               = errors.New("primary key generator required")
)

type VerificationScenarioSummary struct {
	ID           string
	Name         string
	PrimaryTable string
	UpdatedAt    time.Time
}

type VerificationScenario struct {
	ID            string
	Name          string
	PrimaryTable  string
	Definition    map[string]any
	WorkspaceName *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type VerificationScenarioDraft struct {
	Name         string
	PrimaryTable string
	Definition   map[string]any
}

type VerificationRunPreviewTable struct {
	Name               string
	RowCount           int
	AutomaticallyAdded bool
	GeneratedColumns   []string
}

type VerificationRunPreview struct {
	Ready       bool
	InsertOrder []VerificationRunPreviewTable
	DeleteOrder []VerificationRunPreviewTable
	Warnings    []string
}

// 検証実行プレビュー生成
func PreviewVerificationRun(scenario VerificationScenarioDraft, schema Schema) VerificationRunPreview {
	requestedTables := append([]string{scenario.PrimaryTable}, scenarioChildTables(scenario.Definition)...)
	rowCounts := scenarioRowCounts(scenario.Definition)
	generatedColumns := scenarioGeneratedColumns(scenario.Definition)
	tables := make(map[string]Table, len(schema.Tables))
	for _, table := range schema.Tables {
		tables[table.Name] = table
	}

	warnings := make([]string, 0)
	for _, table := range requestedTables {
		if _, found := tables[table]; !found {
			warnings = append(warnings, "対象テーブルが見つかりません: "+table)
		}
	}
	if len(warnings) > 0 {
		return VerificationRunPreview{Ready: false, InsertOrder: []VerificationRunPreviewTable{}, DeleteOrder: []VerificationRunPreviewTable{}, Warnings: warnings}
	}

	requested := make(map[string]struct{}, len(requestedTables))
	for _, table := range requestedTables {
		requested[table] = struct{}{}
	}
	order, automaticallyAdded, cyclic := previewInsertionOrder(requestedTables, requested, schema.ForeignKeys)
	if cyclic {
		warnings = append(warnings, "外部キーの循環参照があるため投入順を確定できません")
	}
	for _, table := range order {
		if !hasPrimaryKey(tables[table]) {
			warnings = append(warnings, "主キーがありません: "+table)
		}
	}
	if len(warnings) > 0 {
		return VerificationRunPreview{Ready: false, InsertOrder: []VerificationRunPreviewTable{}, DeleteOrder: []VerificationRunPreviewTable{}, Warnings: warnings}
	}

	insertOrder := make([]VerificationRunPreviewTable, 0, len(order))
	for _, table := range order {
		count := rowCounts[table]
		if automaticallyAdded[table] {
			count = dependentRowCount(table, requestedTables, rowCounts, schema.ForeignKeys)
		}
		insertOrder = append(insertOrder, VerificationRunPreviewTable{Name: table, RowCount: count, AutomaticallyAdded: automaticallyAdded[table], GeneratedColumns: generatedColumns[table]})
	}
	deleteOrder := make([]VerificationRunPreviewTable, len(insertOrder))
	for index := range insertOrder {
		deleteOrder[len(insertOrder)-1-index] = insertOrder[index]
	}

	return VerificationRunPreview{Ready: true, InsertOrder: insertOrder, DeleteOrder: deleteOrder, Warnings: []string{}}
}

// シナリオ子対象取得
func scenarioChildTables(definition map[string]any) []string {
	childTables, _ := scenarioStringSlice(definition["childTables"])

	return childTables
}

// シナリオ行数取得
func scenarioRowCounts(definition map[string]any) map[string]int {
	values, _ := definition["rowCounts"].(map[string]any)
	rowCounts := make(map[string]int, len(values))
	for table, value := range values {
		if number, ok := value.(float64); ok {
			rowCounts[table] = int(number)
		}
	}

	return rowCounts
}

// シナリオ生成対象列取得
func scenarioGeneratedColumns(definition map[string]any) map[string][]string {
	values, _ := definition["columnGenerators"].(map[string]any)
	columns := make(map[string][]string, len(values))
	for table, value := range values {
		rules, ok := value.(map[string]any)
		if !ok {
			continue
		}
		columns[table] = make([]string, 0, len(rules))
		for column := range rules {
			columns[table] = append(columns[table], column)
		}
		sort.Strings(columns[table])
	}

	return columns
}

// 投入順計算
func previewInsertionOrder(requestedTables []string, requested map[string]struct{}, foreignKeys []ForeignKey) ([]string, map[string]bool, bool) {
	parents := make(map[string][]string)
	for _, foreignKey := range foreignKeys {
		parents[foreignKey.FromTable] = append(parents[foreignKey.FromTable], foreignKey.ToTable)
	}
	for table := range parents {
		sort.Strings(parents[table])
	}

	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	automaticallyAdded := make(map[string]bool)
	order := make([]string, 0, len(requestedTables))
	cyclic := false
	var visit func(string)
	visit = func(table string) {
		if visiting[table] {
			cyclic = true

			return
		}
		if visited[table] {
			return
		}
		visiting[table] = true
		for _, parent := range parents[table] {
			if _, selected := requested[parent]; !selected {
				automaticallyAdded[parent] = true
			}
			visit(parent)
		}
		delete(visiting, table)
		visited[table] = true
		order = append(order, table)
	}
	for _, table := range requestedTables {
		visit(table)
	}

	return order, automaticallyAdded, cyclic
}

// 依存先行数算出
func dependentRowCount(parent string, requestedTables []string, rowCounts map[string]int, foreignKeys []ForeignKey) int {
	count := 1
	for _, child := range requestedTables {
		for _, foreignKey := range foreignKeys {
			if foreignKey.FromTable == child && foreignKey.ToTable == parent && rowCounts[child] > count {
				count = rowCounts[child]
			}
		}
	}

	return count
}

// 主キー存在判定
func hasPrimaryKey(table Table) bool {
	for _, column := range table.Columns {
		if column.IsPrimaryKey {
			return true
		}
	}

	return false
}

// 検証シナリオ下書き生成
func NewVerificationScenarioDraft(name, primaryTable string, definition map[string]any) (VerificationScenarioDraft, error) {
	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return VerificationScenarioDraft{}, ErrInvalidVerificationScenarioDraft
	}

	var normalizedDefinition map[string]any
	if err := json.Unmarshal(definitionJSON, &normalizedDefinition); err != nil || normalizedDefinition == nil {
		return VerificationScenarioDraft{}, ErrInvalidVerificationScenarioDraft
	}

	if childTables, ok := scenarioStringSlice(normalizedDefinition["childTables"]); ok {
		normalizedChildTables := make([]any, 0, len(childTables))
		for _, childTable := range childTables {
			normalizedChildTables = append(normalizedChildTables, childTable)
		}
		normalizedDefinition["childTables"] = normalizedChildTables
	}

	draft := VerificationScenarioDraft{
		Name:         strings.TrimSpace(name),
		PrimaryTable: strings.TrimSpace(primaryTable),
		Definition:   normalizedDefinition,
	}
	if err := draft.Validate(); err != nil {
		return VerificationScenarioDraft{}, err
	}

	return draft, nil
}

// 検証シナリオ下書き検証
func (d VerificationScenarioDraft) Validate() error {
	if !validScenarioName(d.Name) || !validScenarioTableName(d.PrimaryTable) {
		return ErrInvalidVerificationScenarioDraft
	}

	childTables, ok := scenarioStringSlice(d.Definition["childTables"])
	if !ok || !validChildTables(childTables, d.PrimaryTable) {
		return ErrInvalidVerificationScenarioDraft
	}
	if !validScenarioRowCounts(d.Definition["rowCounts"], d.PrimaryTable, childTables) {
		return ErrInvalidVerificationScenarioDraft
	}
	if !validScenarioColumnGenerators(d.Definition["columnGenerators"], d.PrimaryTable, childTables) {
		return ErrInvalidVerificationScenarioDraft
	}
	if !validScenarioSQL(d.Definition["sql"]) || !validScenarioRange(d.Definition["warmupRuns"], 0, 1000) || !validScenarioRange(d.Definition["iterations"], 1, 100000) || !validScenarioRange(d.Definition["timeLimitSeconds"], 1, 3600) {
		return ErrInvalidVerificationScenarioDraft
	}
	if !hasPrimaryKeyGenerator(d.Definition["columnGenerators"], d.PrimaryTable) {
		return ErrPrimaryKeyRequired
	}

	return nil
}

// 保存用検証シナリオ生成
func (d VerificationScenarioDraft) NewVerificationScenario(id string, createdAt time.Time) (VerificationScenario, error) {
	if id == "" {
		return VerificationScenario{}, ErrInvalidVerificationScenarioDraft
	}
	if err := d.Validate(); err != nil {
		return VerificationScenario{}, err
	}

	definitionJSON, err := json.Marshal(d.Definition)
	if err != nil {
		return VerificationScenario{}, ErrInvalidVerificationScenarioDraft
	}

	createdAt = createdAt.UTC()

	return NewVerificationScenario(id, d.Name, d.PrimaryTable, definitionJSON, nil, createdAt, createdAt)
}

// 複製用検証シナリオ生成
func (s VerificationScenario) DuplicateVerificationScenario(id string, createdAt time.Time) (VerificationScenario, error) {
	draft, err := NewVerificationScenarioDraft(s.Name, s.PrimaryTable, s.Definition)
	if err != nil {
		return VerificationScenario{}, err
	}

	return draft.NewVerificationScenario(id, createdAt)
}

// 更新用検証シナリオ生成
func (d VerificationScenarioDraft) UpdateVerificationScenario(scenario VerificationScenario, updatedAt time.Time) (VerificationScenario, error) {
	if scenario.ID == "" {
		return VerificationScenario{}, ErrInvalidVerificationScenarioDraft
	}
	if err := d.Validate(); err != nil {
		return VerificationScenario{}, err
	}

	return VerificationScenario{
		ID:            scenario.ID,
		Name:          d.Name,
		PrimaryTable:  d.PrimaryTable,
		Definition:    d.Definition,
		WorkspaceName: scenario.WorkspaceName,
		CreatedAt:     scenario.CreatedAt,
		UpdatedAt:     updatedAt.UTC(),
	}, nil
}

// 検証シナリオ生成
func NewVerificationScenario(id, name, primaryTable string, definitionJSON []byte, workspaceName *string, createdAt, updatedAt time.Time) (VerificationScenario, error) {
	var definition map[string]any
	if err := json.Unmarshal(definitionJSON, &definition); err != nil {
		return VerificationScenario{}, fmt.Errorf("decode verification scenario definition: %w", err)
	}
	if definition == nil {
		return VerificationScenario{}, fmt.Errorf("verification scenario definition must be an object")
	}

	return VerificationScenario{
		ID:            id,
		Name:          name,
		PrimaryTable:  primaryTable,
		Definition:    definition,
		WorkspaceName: workspaceName,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}, nil
}

// シナリオ名判定
func validScenarioName(value string) bool {
	if value == "" || len([]rune(value)) > 100 {
		return false
	}

	return !containsControlCharacter(value)
}

// シナリオテーブル名判定
func validScenarioTableName(value string) bool {
	return value != "" && !containsControlCharacter(value)
}

// 制御文字包含判定
func containsControlCharacter(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}

// 子テーブル名配列変換
func scenarioStringSlice(value any) ([]string, bool) {
	values, ok := value.([]any)
	if !ok {
		return nil, false
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		item, ok := value.(string)
		if !ok {
			return nil, false
		}
		result = append(result, strings.TrimSpace(item))
	}

	return result, true
}

// 子テーブル名判定
func validChildTables(childTables []string, primaryTable string) bool {
	seen := make(map[string]struct{}, len(childTables))
	for _, childTable := range childTables {
		if !validScenarioTableName(childTable) || childTable == primaryTable {
			return false
		}
		if _, exists := seen[childTable]; exists {
			return false
		}
		seen[childTable] = struct{}{}
	}

	return true
}

// テーブル別件数判定
func validScenarioRowCounts(value any, primaryTable string, childTables []string) bool {
	rowCounts, ok := value.(map[string]any)
	if !ok {
		return false
	}

	for _, table := range append([]string{primaryTable}, childTables...) {
		if !validScenarioRange(rowCounts[table], 1, 100000) {
			return false
		}
	}

	return true
}

// カラム生成規則判定
func validScenarioColumnGenerators(value any, primaryTable string, childTables []string) bool {
	columnGenerators, ok := value.(map[string]any)
	if !ok {
		return false
	}

	for _, table := range append([]string{primaryTable}, childTables...) {
		rules, ok := columnGenerators[table].(map[string]any)
		if !ok {
			return false
		}
		for _, value := range rules {
			rule, ok := value.(map[string]any)
			if !ok || !validGeneratorKind(rule["kind"]) {
				return false
			}
		}
	}

	return true
}

// 生成規則種別判定
func validGeneratorKind(value any) bool {
	kind, ok := value.(string)
	if !ok {
		return false
	}

	return kind == "sequence" || kind == "uuid" || kind == "template" || kind == "fixed" || kind == "null"
}

// 主対象一意生成規則判定
func hasPrimaryKeyGenerator(value any, primaryTable string) bool {
	columnGenerators, ok := value.(map[string]any)
	if !ok {
		return false
	}
	rules, ok := columnGenerators[primaryTable].(map[string]any)
	if !ok {
		return false
	}
	for _, value := range rules {
		rule, ok := value.(map[string]any)
		if !ok {
			continue
		}
		kind, _ := rule["kind"].(string)
		if kind == "sequence" || kind == "uuid" {
			return true
		}
	}

	return false
}

// SQL配列判定
func validScenarioSQL(value any) bool {
	statements, ok := value.([]any)
	if !ok || len(statements) == 0 || len(statements) > 50 {
		return false
	}
	for _, value := range statements {
		statement, ok := value.(string)
		if !ok || strings.TrimSpace(statement) == "" || strings.ContainsRune(statement, '\x00') {
			return false
		}
	}

	return true
}

// 数値範囲判定
func validScenarioRange(value any, minimum, maximum int) bool {
	number, ok := value.(float64)
	if !ok || math.Trunc(number) != number {
		return false
	}

	return number >= float64(minimum) && number <= float64(maximum)
}
