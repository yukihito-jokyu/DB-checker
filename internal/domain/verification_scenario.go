package domain

import (
	"encoding/json"
	"fmt"
	"time"
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
