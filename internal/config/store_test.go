package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

func TestStoreLoadReturnsConfigReadFailed(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := os.MkdirAll(store.Path(), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	_, err := store.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !apperr.IsCode(err, apperr.CodeConfigReadFailed) {
		t.Errorf("Load() error code = %v, want %s", err, apperr.CodeConfigReadFailed)
	}
}

func TestStoreLoadReturnsConfigWriteFailed(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "config-parent")
	if err := os.WriteFile(baseDir, []byte("not directory"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
	store := NewStore(baseDir)

	_, err := store.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !apperr.IsCode(err, apperr.CodeConfigWriteFailed) {
		t.Errorf("Load() error code = %v, want %s", err, apperr.CodeConfigWriteFailed)
	}
}

func TestStoreLoadCreatesDefaultConfig(t *testing.T) {
	store := NewStore(t.TempDir())

	result, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if result.Recovered {
		t.Error("Recovered = true, want false")
	}
	assertDefaultConfig(t, result.Config)

	bytes, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}

	var fileConfig Config
	if err := json.Unmarshal(bytes, &fileConfig); err != nil {
		t.Fatalf("Unmarshal() error = %v, want nil", err)
	}
	assertDefaultConfig(t, fileConfig)
}

func TestStoreLoadReadsValidConfig(t *testing.T) {
	store := NewStore(t.TempDir())
	activeID := "local-postgres"
	config := Config{
		Version: FileVersion,
		ConnectionProfiles: []ConnectionProfile{
			{
				ID:       activeID,
				Name:     "Local PostgreSQL",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "developer",
				Password: "password",
			},
		},
		ActiveConnectionProfileID: &activeID,
		FlowStates: map[string]json.RawMessage{
			activeID: json.RawMessage(`{"nodes":[]}`),
		},
	}
	writeConfigFile(t, store.Path(), config)

	result, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if result.Recovered {
		t.Error("Recovered = true, want false")
	}
	if result.Config.Version != FileVersion {
		t.Errorf("Version = %d, want %d", result.Config.Version, FileVersion)
	}
	if got := len(result.Config.ConnectionProfiles); got != 1 {
		t.Fatalf("len(ConnectionProfiles) = %d, want 1", got)
	}
	if result.Config.ConnectionProfiles[0].ID != activeID {
		t.Errorf("ConnectionProfiles[0].ID = %q, want %q", result.Config.ConnectionProfiles[0].ID, activeID)
	}
	if result.Config.ActiveConnectionProfileID == nil {
		t.Fatal("ActiveConnectionProfileID = nil, want active id")
	}
	if *result.Config.ActiveConnectionProfileID != activeID {
		t.Errorf("ActiveConnectionProfileID = %q, want %q", *result.Config.ActiveConnectionProfileID, activeID)
	}
	if got := string(result.Config.FlowStates[activeID]); got != `{"nodes":[]}` {
		t.Errorf("FlowStates[%q] = %s, want %s", activeID, got, `{"nodes":[]}`)
	}
}

func TestStoreLoadRecoversBrokenJSON(t *testing.T) {
	store := NewStore(t.TempDir())
	store.now = func() time.Time {
		return time.Date(2026, 6, 29, 15, 30, 12, 0, time.UTC)
	}
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}
	if err := os.WriteFile(store.Path(), []byte(`{"version":`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	result, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if !result.Recovered {
		t.Error("Recovered = false, want true")
	}
	wantBackupPath := store.Path() + ".broken.20260629T153012"
	if result.BackupPath != wantBackupPath {
		t.Errorf("BackupPath = %q, want %q", result.BackupPath, wantBackupPath)
	}
	backupBytes, err := os.ReadFile(wantBackupPath)
	if err != nil {
		t.Fatalf("ReadFile(backup) error = %v, want nil", err)
	}
	if got := string(backupBytes); got != `{"version":` {
		t.Errorf("backup content = %q, want %q", got, `{"version":`)
	}
	assertDefaultConfig(t, result.Config)
	assertConfigFileIsDefault(t, store.Path())
}

func TestStoreLoadRecoversInvalidSchema(t *testing.T) {
	store := NewStore(t.TempDir())
	store.now = func() time.Time {
		return time.Date(2026, 6, 29, 15, 30, 12, 0, time.UTC)
	}
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}
	if err := os.WriteFile(store.Path(), []byte(`{"version":999,"connectionProfiles":[],"activeConnectionProfileId":null,"flowStates":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	result, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if !result.Recovered {
		t.Error("Recovered = false, want true")
	}
	assertDefaultConfig(t, result.Config)
	assertConfigFileIsDefault(t, store.Path())
}

func writeConfigFile(t *testing.T, path string, cfg Config) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}

	bytes, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal() error = %v, want nil", err)
	}
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
}

func assertDefaultConfig(t *testing.T, cfg Config) {
	t.Helper()

	if cfg.Version != FileVersion {
		t.Errorf("Version = %d, want %d", cfg.Version, FileVersion)
	}
	if cfg.ConnectionProfiles == nil {
		t.Fatal("ConnectionProfiles = nil, want empty slice")
	}
	if got := len(cfg.ConnectionProfiles); got != 0 {
		t.Errorf("len(ConnectionProfiles) = %d, want 0", got)
	}
	if cfg.ActiveConnectionProfileID != nil {
		t.Errorf("ActiveConnectionProfileID = %q, want nil", *cfg.ActiveConnectionProfileID)
	}
	if cfg.FlowStates == nil {
		t.Fatal("FlowStates = nil, want empty map")
	}
	if got := len(cfg.FlowStates); got != 0 {
		t.Errorf("len(FlowStates) = %d, want 0", got)
	}
}

func assertConfigFileIsDefault(t *testing.T, path string) {
	t.Helper()

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}

	var cfg Config
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		t.Fatalf("Unmarshal() error = %v, want nil", err)
	}
	assertDefaultConfig(t, cfg)
}
