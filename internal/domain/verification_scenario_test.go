package domain

import (
	"reflect"
	"testing"
	"time"
)

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
