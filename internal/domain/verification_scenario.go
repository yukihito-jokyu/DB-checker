package domain

import "time"

type VerificationScenarioSummary struct {
	ID           string
	Name         string
	PrimaryTable string
	UpdatedAt    time.Time
}
