package repository

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	_ "modernc.org/sqlite"
)

const verificationScenarioDatabaseName = "verification.sqlite3"

var openVerificationScenarioDatabase = sql.Open

// SQLite検証シナリオリポジトリ
type SQLiteVerificationScenarioRepository struct {
	databasePath string
}

// SQLite検証シナリオリポジトリ生成
func NewSQLiteVerificationScenarioRepository(configDir string) *SQLiteVerificationScenarioRepository {
	return &SQLiteVerificationScenarioRepository{databasePath: filepath.Join(configDir, verificationScenarioDatabaseName)}
}

// シナリオDB初期化
func (r *SQLiteVerificationScenarioRepository) Initialize(ctx context.Context) error {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	if err := migrateVerificationScenarioDatabase(ctx, database); err != nil {
		return err
	}

	return nil
}

// 検証シナリオ一覧取得
func (r *SQLiteVerificationScenarioRepository) ListVerificationScenarios(ctx context.Context, profileID string) ([]domain.VerificationScenarioSummary, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return nil, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	rows, err := database.QueryContext(ctx, `SELECT id, name, primary_table, updated_at FROM scenarios WHERE profile_id = ? ORDER BY updated_at DESC, id ASC`, profileID)
	if err != nil {
		return nil, fmt.Errorf("query verification scenarios: %w", err)
	}
	defer rows.Close()

	scenarios := make([]domain.VerificationScenarioSummary, 0)
	for rows.Next() {
		var scenario domain.VerificationScenarioSummary
		var updatedAt string
		if err := rows.Scan(&scenario.ID, &scenario.Name, &scenario.PrimaryTable, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan verification scenario: %w", err)
		}

		scenario.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse verification scenario updated time: %w", err)
		}
		scenarios = append(scenarios, scenario)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate verification scenarios: %w", err)
	}

	return scenarios, nil
}

// シナリオDBマイグレーション
func migrateVerificationScenarioDatabase(ctx context.Context, database *sql.DB) error {
	var version int
	if err := database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read verification scenario schema version: %w", err)
	}
	if version > 1 {
		return fmt.Errorf("unsupported verification scenario schema version")
	}
	if version == 1 {
		return nil
	}

	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin verification scenario migration: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	if _, err := transaction.ExecContext(ctx, `CREATE TABLE scenarios (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		name TEXT NOT NULL,
		primary_table TEXT NOT NULL,
		definition_json TEXT NOT NULL,
		workspace_name TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create verification scenarios table: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `CREATE INDEX idx_scenarios_profile_updated_at ON scenarios (profile_id, updated_at DESC, id ASC)`); err != nil {
		return fmt.Errorf("create verification scenario index: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
		return fmt.Errorf("write verification scenario schema version: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit verification scenario migration: %w", err)
	}

	return nil
}
