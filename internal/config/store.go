package config

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type Store struct {
	baseDir string
	now     func() time.Time
}

type LoadResult struct {
	Config     Config
	Recovered  bool
	BackupPath string
}

// NewDefaultStore は OS 標準のユーザー設定ディレクトリ配下を使う Store を作成する。
func NewDefaultStore() (*Store, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, apperr.NewUnexpected(err)
	}

	return NewStore(filepath.Join(userConfigDir, AppDirName)), nil
}

// NewStore は指定されたディレクトリを設定保存先にする Store を作成する。
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		now:     time.Now,
	}
}

// Path は設定ファイルの保存先パスを返す。
func (s *Store) Path() string {
	return filepath.Join(s.baseDir, FileName)
}

// Load は設定を読み込み、未作成または破損時は既定値を保存して返す。
func (s *Store) Load() (LoadResult, error) {
	if err := os.MkdirAll(s.baseDir, 0o750); err != nil {
		return LoadResult{}, apperr.Wrap(apperr.CodeConfigWriteFailed, err)
	}

	path := s.Path()
	bytes, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return LoadResult{}, apperr.Wrap(apperr.CodeConfigReadFailed, err)
		}

		// 初回起動時は空の既定設定を作成し、以降の読み込み形式を固定する。
		cfg := Default()
		if err := s.write(cfg); err != nil {
			return LoadResult{}, err
		}
		return LoadResult{Config: cfg}, nil
	}

	cfg, decodeErr := decodeAndValidate(bytes)
	if decodeErr == nil {
		return LoadResult{Config: cfg}, nil
	}

	// 壊れた設定は上書きせず、退避してから既定設定で復旧する。
	backupPath, err := s.recoverBrokenConfig(path)
	if err != nil {
		return LoadResult{}, apperr.Wrap(apperr.CodeConfigBroken, stderrors.Join(decodeErr, err))
	}

	cfg = Default()
	if err := s.write(cfg); err != nil {
		return LoadResult{}, apperr.Wrap(apperr.CodeConfigWriteFailed, stderrors.Join(decodeErr, err))
	}

	return LoadResult{
		Config:     cfg,
		Recovered:  true,
		BackupPath: backupPath,
	}, nil
}

// write は設定を JSON として一時ファイルへ書き込み、設定ファイルへ置き換える。
func (s *Store) write(cfg Config) error {
	if err := os.MkdirAll(s.baseDir, 0o750); err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err)
	}

	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return apperr.NewUnexpected(err)
	}
	bytes = append(bytes, '\n')

	// 書き込み途中の破損を避けるため、一時ファイルを作成してから置き換える。
	tmpPath := s.Path() + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0o600); err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err)
	}
	if err := os.Rename(tmpPath, s.Path()); err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err)
	}

	return nil
}

// recoverBrokenConfig は壊れた設定ファイルをタイムスタンプ付きのバックアップへ退避する。
func (s *Store) recoverBrokenConfig(path string) (string, error) {
	backupPath := fmt.Sprintf("%s.%s.%s", path, backupSuffix, s.now().Format("20060102T150405"))
	// 同一秒内に復旧が複数回走っても、既存バックアップを残す。
	for i := 1; fileExists(backupPath); i++ {
		backupPath = fmt.Sprintf("%s.%s.%s.%d", path, backupSuffix, s.now().Format("20060102T150405"), i)
	}
	if err := os.Rename(path, backupPath); err != nil {
		return "", err
	}
	return backupPath, nil
}

// decodeAndValidate は設定 JSON を読み取り、現在対応する最小スキーマを満たすか検証する。
func decodeAndValidate(bytes []byte) (Config, error) {
	var raw struct {
		Version                   *int             `json:"version"`
		ConnectionProfiles        *json.RawMessage `json:"connectionProfiles"`
		ActiveConnectionProfileID *json.RawMessage `json:"activeConnectionProfileId"`
		FlowStates                *json.RawMessage `json:"flowStates"`
	}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return Config{}, err
	}
	if raw.Version == nil || *raw.Version != FileVersion {
		return Config{}, fmt.Errorf("unsupported config version")
	}
	if raw.ConnectionProfiles == nil {
		return Config{}, fmt.Errorf("connectionProfiles is required")
	}
	if raw.FlowStates == nil {
		return Config{}, fmt.Errorf("flowStates is required")
	}
	if raw.ActiveConnectionProfileID == nil {
		return Config{}, fmt.Errorf("activeConnectionProfileId is required")
	}

	// 型の正しさは JSON の個別 decode に任せ、最低限のスキーマだけ検証する。
	var profiles []ConnectionProfile
	if err := json.Unmarshal(*raw.ConnectionProfiles, &profiles); err != nil {
		return Config{}, err
	}
	if profiles == nil {
		return Config{}, fmt.Errorf("connectionProfiles must be array")
	}

	activeProfileID, err := decodeActiveConnectionProfileID(raw.ActiveConnectionProfileID)
	if err != nil {
		return Config{}, err
	}

	var flowStates map[string]json.RawMessage
	if err := json.Unmarshal(*raw.FlowStates, &flowStates); err != nil {
		return Config{}, err
	}
	if flowStates == nil {
		return Config{}, fmt.Errorf("flowStates must be object")
	}

	return Config{
		Version:                   *raw.Version,
		ConnectionProfiles:        profiles,
		ActiveConnectionProfileID: activeProfileID,
		FlowStates:                flowStates,
	}, nil
}

// decodeActiveConnectionProfileID は null または文字列の activeConnectionProfileId を読み取る。
func decodeActiveConnectionProfileID(raw *json.RawMessage) (*string, error) {
	if raw == nil || string(*raw) == "null" {
		return nil, nil
	}

	var id string
	if err := json.Unmarshal(*raw, &id); err != nil {
		return nil, err
	}
	return &id, nil
}

// fileExists は指定パスにファイルシステムエントリが存在するか判定する。
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
