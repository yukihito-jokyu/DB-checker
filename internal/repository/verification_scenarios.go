package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"time"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	_ "modernc.org/sqlite"
)

const verificationScenarioDatabaseName = "verification.sqlite3"

var openVerificationScenarioDatabase = sql.Open

var verificationWorkspaceIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

var openVerificationWorkspaceDatabase = sql.Open

// 検証先資格情報取得元
type verificationWorkspaceCredentialSource interface {
	GetCredential(string) (string, bool, error)
}

// 外部検証先リポジトリ
type VerificationWorkspaceRepository struct {
	credentials verificationWorkspaceCredentialSource
	open        func(string, string) (*sql.DB, error)
}

// 外部検証先リポジトリ生成
func NewVerificationWorkspaceRepository(credentials verificationWorkspaceCredentialSource) *VerificationWorkspaceRepository {
	return &VerificationWorkspaceRepository{credentials: credentials, open: openVerificationWorkspaceDatabase}
}

// 外部検証先作成
func (r *VerificationWorkspaceRepository) CreateWorkspace(ctx context.Context, profile domain.Profile, workspaceName string) error {
	if !verificationWorkspaceIdentifier.MatchString(workspaceName) {
		return fmt.Errorf("invalid verification workspace identifier")
	}
	if !isVerificationWorkspaceHost(profile.Host) {
		return errors.New("verification workspace host is not allowed")
	}

	password, found, err := r.credentials.GetCredential(profile.ID)
	if err != nil {
		return errors.New("load verification credential")
	}
	if !found {
		return fmt.Errorf("verification credential unavailable")
	}

	driverName, dsn := connectionDSN(profile, password)
	database, err := r.open(driverName, dsn)
	if err != nil {
		return errors.New("open verification connection")
	}
	defer database.Close()

	statement := "CREATE DATABASE IF NOT EXISTS " + quoteVerificationIdentifier(profile.DBType, workspaceName) //nolint:gosec // workspaceNameは内部生成した英数字とアンダースコアだけに検証済み。
	if profile.DBType == domain.DBTypePostgres {
		statement = "CREATE SCHEMA IF NOT EXISTS " + quoteVerificationIdentifier(profile.DBType, workspaceName)
	}
	if _, err := database.ExecContext(ctx, statement); err != nil {
		return errors.New("create verification workspace")
	}

	return nil
}

// 外部検証先削除
func (r *VerificationWorkspaceRepository) DeleteWorkspace(ctx context.Context, profile domain.Profile, workspaceName string) error {
	if !verificationWorkspaceIdentifier.MatchString(workspaceName) {
		return fmt.Errorf("invalid verification workspace identifier")
	}
	if !isVerificationWorkspaceHost(profile.Host) {
		return errors.New("verification workspace host is not allowed")
	}

	password, found, err := r.credentials.GetCredential(profile.ID)
	if err != nil {
		return errors.New("load verification credential")
	}
	if !found {
		return fmt.Errorf("verification credential unavailable")
	}

	driverName, dsn := connectionDSN(profile, password)
	database, err := r.open(driverName, dsn)
	if err != nil {
		return errors.New("open verification connection")
	}
	defer database.Close()

	statement := "DROP DATABASE IF EXISTS " + quoteVerificationIdentifier(profile.DBType, workspaceName) //nolint:gosec // workspaceNameは内部生成した英数字とアンダースコアだけに検証済み。
	if profile.DBType == domain.DBTypePostgres {
		statement = "DROP SCHEMA IF EXISTS " + quoteVerificationIdentifier(profile.DBType, workspaceName) + " CASCADE"
	}
	if _, err := database.ExecContext(ctx, statement); err != nil {
		return errors.New("delete verification workspace")
	}

	return nil
}

// 検証先接続先判定
func isVerificationWorkspaceHost(host string) bool {
	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

// 検証先識別子引用
func quoteVerificationIdentifier(dbType domain.DBType, identifier string) string {
	if dbType == domain.DBTypeMySQL {
		return "`" + identifier + "`"
	}

	return `"` + identifier + `"`
}

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

// 検証シナリオ保存
func (r *SQLiteVerificationScenarioRepository) CreateVerificationScenario(ctx context.Context, profileID string, scenario domain.VerificationScenario) error {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	definitionJSON, err := json.Marshal(scenario.Definition)
	if err != nil {
		return fmt.Errorf("encode verification scenario definition: %w", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO scenarios (id, profile_id, name, primary_table, definition_json, workspace_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, scenario.ID, profileID, scenario.Name, scenario.PrimaryTable, string(definitionJSON), nil, scenario.CreatedAt.UTC().Format(time.RFC3339Nano), scenario.UpdatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert verification scenario: %w", err)
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

// 検証シナリオ詳細取得
func (r *SQLiteVerificationScenarioRepository) GetVerificationScenario(ctx context.Context, profileID, scenarioID string) (domain.VerificationScenario, bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return domain.VerificationScenario{}, false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	var id, name, primaryTable, definitionJSON, createdAtValue, updatedAtValue string
	var workspaceName sql.NullString
	err = database.QueryRowContext(ctx, `SELECT id, name, primary_table, definition_json, workspace_name, created_at, updated_at FROM scenarios WHERE profile_id = ? AND id = ?`, profileID, scenarioID).Scan(&id, &name, &primaryTable, &definitionJSON, &workspaceName, &createdAtValue, &updatedAtValue)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.VerificationScenario{}, false, nil
	}
	if err != nil {
		return domain.VerificationScenario{}, false, fmt.Errorf("query verification scenario: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtValue)
	if err != nil {
		return domain.VerificationScenario{}, false, fmt.Errorf("parse verification scenario created time: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339, updatedAtValue)
	if err != nil {
		return domain.VerificationScenario{}, false, fmt.Errorf("parse verification scenario updated time: %w", err)
	}

	var workspaceNameValue *string
	if workspaceName.Valid {
		workspaceNameValue = &workspaceName.String
	}
	scenario, err := domain.NewVerificationScenario(id, name, primaryTable, []byte(definitionJSON), workspaceNameValue, createdAt, updatedAt)
	if err != nil {
		return domain.VerificationScenario{}, false, fmt.Errorf("decode verification scenario: %w", err)
	}

	return scenario, true, nil
}

// 検証シナリオ更新
func (r *SQLiteVerificationScenarioRepository) UpdateVerificationScenario(ctx context.Context, profileID string, scenario domain.VerificationScenario) (bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	definitionJSON, err := json.Marshal(scenario.Definition)
	if err != nil {
		return false, fmt.Errorf("encode verification scenario definition: %w", err)
	}

	result, err := database.ExecContext(ctx, `UPDATE scenarios SET name = ?, primary_table = ?, definition_json = ?, updated_at = ? WHERE profile_id = ? AND id = ?`, scenario.Name, scenario.PrimaryTable, string(definitionJSON), scenario.UpdatedAt.UTC().Format(time.RFC3339Nano), profileID, scenario.ID)
	if err != nil {
		return false, fmt.Errorf("update verification scenario: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count updated verification scenarios: %w", err)
	}

	return updated > 0, nil
}

// 検証シナリオ削除
func (r *SQLiteVerificationScenarioRepository) DeleteVerificationScenario(ctx context.Context, profileID, scenarioID string, removeWorkspace bool) (bool, bool, bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return false, false, false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return false, false, false, fmt.Errorf("begin verification scenario deletion: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	var workspaceActive, runActive bool
	if err := transaction.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM verification_workspaces WHERE profile_id = ? AND scenario_id = ? AND state IN ('creating', 'active', 'test'))`, profileID, scenarioID).Scan(&workspaceActive); err != nil {
		return false, false, false, fmt.Errorf("check verification workspace state: %w", err)
	}
	if err := transaction.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM verification_runs WHERE profile_id = ? AND scenario_id = ? AND state IN ('prepared', 'running', 'canceling'))`, profileID, scenarioID).Scan(&runActive); err != nil {
		return false, false, false, fmt.Errorf("check verification run state: %w", err)
	}
	if workspaceActive || runActive {
		return false, false, true, nil
	}
	if removeWorkspace {
		return false, false, false, errors.New("verification workspace removal must be coordinated by usecase")
	}

	result, err := transaction.ExecContext(ctx, `DELETE FROM scenarios WHERE profile_id = ? AND id = ?`, profileID, scenarioID)
	if err != nil {
		return false, false, false, fmt.Errorf("delete verification scenario: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return false, false, false, fmt.Errorf("count deleted verification scenarios: %w", err)
	}
	if deleted == 0 {
		return false, false, false, nil
	}

	if err := transaction.Commit(); err != nil {
		return false, false, false, fmt.Errorf("commit verification scenario deletion: %w", err)
	}

	return true, false, false, nil
}

// 検証ワークスペース取得
func (r *SQLiteVerificationScenarioRepository) GetVerificationWorkspace(ctx context.Context, profileID, scenarioID string) (string, string, bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return "", "", false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()
	var name, state string
	err = database.QueryRowContext(ctx, `SELECT workspace_name, state FROM verification_workspaces WHERE profile_id = ? AND scenario_id = ?`, profileID, scenarioID).Scan(&name, &state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("query verification workspace: %w", err)
	}

	return state, name, true, nil
}

// 検証ワークスペース保存
func (r *SQLiteVerificationScenarioRepository) SaveVerificationWorkspace(ctx context.Context, profileID, scenarioID, name, state string) error {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()
	_, err = database.ExecContext(ctx, `INSERT INTO verification_workspaces (profile_id, scenario_id, workspace_name, state) VALUES (?, ?, ?, ?) ON CONFLICT(profile_id, scenario_id) DO UPDATE SET workspace_name = excluded.workspace_name, state = excluded.state`, profileID, scenarioID, name, state)
	if err != nil {
		return fmt.Errorf("save verification workspace: %w", err)
	}

	return nil
}

// 検証ワークスペース削除
func (r *SQLiteVerificationScenarioRepository) DeleteVerificationWorkspace(ctx context.Context, profileID, scenarioID string) error {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()
	if _, err := database.ExecContext(ctx, `DELETE FROM verification_workspaces WHERE profile_id = ? AND scenario_id = ?`, profileID, scenarioID); err != nil {
		return fmt.Errorf("delete verification workspace: %w", err)
	}

	return nil
}

// 検証実行状態作成
//
//nolint:nlreturn // SQLiteの短命接続を直近で閉じるため。
func (r *SQLiteVerificationScenarioRepository) CreateVerificationRun(ctx context.Context, profileID, scenarioID, runID string) error {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()
	if _, err := database.ExecContext(ctx, `INSERT INTO verification_runs (id, profile_id, scenario_id, state) VALUES (?, ?, ?, 'prepared')`, runID, profileID, scenarioID); err != nil {
		return fmt.Errorf("insert verification run: %w", err)
	}
	return nil
}

// 検証実行状態取得
func (r *SQLiteVerificationScenarioRepository) GetVerificationRun(ctx context.Context, profileID, runID string) (string, string, bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return "", "", false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	var scenarioID, state string
	err = database.QueryRowContext(ctx, `SELECT scenario_id, state FROM verification_runs WHERE profile_id = ? AND id = ?`, profileID, runID).Scan(&scenarioID, &state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("query verification run: %w", err)
	}

	return scenarioID, state, true, nil
}

// 検証実行状態更新
//
//nolint:nlreturn // SQLiteの短命接続を直近で閉じるため。
func (r *SQLiteVerificationScenarioRepository) UpdateVerificationRunState(ctx context.Context, profileID, runID, state string) (bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()
	result, err := database.ExecContext(ctx, `UPDATE verification_runs SET state = ? WHERE profile_id = ? AND id = ?`, state, profileID, runID)
	if err != nil {
		return false, fmt.Errorf("update verification run: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count updated verification runs: %w", err)
	}
	return updated > 0, nil
}

// 検証シナリオ使用中判定
//
//nolint:nlreturn // SQLiteの短命接続を直近で閉じるため。
func (r *SQLiteVerificationScenarioRepository) IsVerificationScenarioBusy(ctx context.Context, profileID, scenarioID string) (bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()
	var busy bool
	err = database.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM verification_workspaces WHERE profile_id = ? AND scenario_id = ? AND state IN ('creating', 'active', 'test', 'deleting')) OR EXISTS(SELECT 1 FROM verification_runs WHERE profile_id = ? AND scenario_id = ? AND state IN ('prepared', 'running', 'canceling'))`, profileID, scenarioID, profileID, scenarioID).Scan(&busy)
	if err != nil {
		return false, fmt.Errorf("check verification scenario busy state: %w", err)
	}
	return busy, nil
}

// 検証実行使用中判定
func (r *SQLiteVerificationScenarioRepository) IsVerificationRunBusy(ctx context.Context, profileID, scenarioID string) (bool, error) {
	database, err := openVerificationScenarioDatabase("sqlite", r.databasePath)
	if err != nil {
		return false, fmt.Errorf("open verification scenario database: %w", err)
	}
	defer database.Close()

	var busy bool
	err = database.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM verification_runs WHERE profile_id = ? AND scenario_id = ? AND state IN ('prepared', 'running', 'canceling'))`, profileID, scenarioID).Scan(&busy)
	if err != nil {
		return false, fmt.Errorf("check verification run state: %w", err)
	}

	return busy, nil
}

// シナリオDBマイグレーション
func migrateVerificationScenarioDatabase(ctx context.Context, database *sql.DB) error {
	var version int
	if err := database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read verification scenario schema version: %w", err)
	}
	if version > 3 {
		return fmt.Errorf("unsupported verification scenario schema version")
	}
	if version == 3 {
		return nil
	}

	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin verification scenario migration: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	if version == 0 {
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
	}
	if version < 2 {
		if _, err := transaction.ExecContext(ctx, `CREATE TABLE verification_workspaces (
		profile_id TEXT NOT NULL,
		scenario_id TEXT NOT NULL,
		workspace_name TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL,
		PRIMARY KEY (profile_id, scenario_id)
	)`); err != nil {
			return fmt.Errorf("create verification workspaces table: %w", err)
		}
	}
	if version < 2 {
		if _, err := transaction.ExecContext(ctx, `CREATE TABLE verification_runs (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		scenario_id TEXT NOT NULL,
		state TEXT NOT NULL
	)`); err != nil {
			return fmt.Errorf("create verification runs table: %w", err)
		}
	}
	if version == 2 {
		if _, err := transaction.ExecContext(ctx, `ALTER TABLE verification_workspaces ADD COLUMN workspace_name TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add verification workspace name: %w", err)
		}
	}
	if _, err := transaction.ExecContext(ctx, "PRAGMA user_version = 3"); err != nil {
		return fmt.Errorf("write verification scenario schema version: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit verification scenario migration: %w", err)
	}

	return nil
}
