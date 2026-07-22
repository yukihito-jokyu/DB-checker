package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type Store struct{ baseDir string }
type LoadResult struct{ Config Config }

// 既定設定ストア生成
func NewDefaultStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, apperr.NewUnexpected(err)
	}

	return NewStore(filepath.Join(dir, AppDirName)), nil
}

// 設定ストア生成
func NewStore(baseDir string) *Store { return &Store{baseDir: baseDir} }

// 設定ファイルパス取得
func (s *Store) Path() string { return filepath.Join(s.baseDir, FileName) }

// 設定ファイル読込
func (s *Store) Load() (LoadResult, error) {
	bytes, err := os.ReadFile(s.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return LoadResult{}, apperr.Wrap(apperr.CodeConfigNotFound, err)
		}

		return LoadResult{}, apperr.Wrap(apperr.CodeConfigReadFailed, err)
	}
	cfg, err := decodeAndValidate(bytes)
	if err != nil {
		return LoadResult{}, apperr.Wrap(apperr.CodeConfigBroken, err)
	}

	return LoadResult{Config: cfg}, nil
}

// 未作成設定初期化
func (s *Store) Initialize() error {
	_, err := s.Load()
	if err == nil {
		return nil
	}
	if !apperr.IsCode(err, apperr.CodeConfigNotFound) {
		return err
	}

	return s.Save(Default())
}

// 設定ファイル保存
func (s *Store) Save(cfg Config) error {
	if err := os.MkdirAll(s.baseDir, 0o750); err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err)
	}
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		// 単体テスト到達不可: Config は JSON 変換に失敗し得る値を持たないため。
		return apperr.NewUnexpected(err)
	}
	bytes = append(bytes, '\n')
	tmp := s.Path() + ".tmp"
	if err := os.WriteFile(tmp, bytes, 0o600); err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err)
	}
	if err := os.Rename(tmp, s.Path()); err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err)
	}

	return nil
}

// 設定内容復号検証
func decodeAndValidate(content []byte) (Config, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(content, &raw); err != nil {
		return Config{}, err
	}
	versionRaw, hasVersion := raw["version"]
	profilesRaw, hasProfiles := raw["connectionProfiles"]
	activeIDRaw, hasActiveID := raw["activeConnectionProfileId"]
	flowStatesRaw, hasFlowStates := raw["flowStates"]
	if !hasVersion || !hasProfiles || !hasActiveID || !hasFlowStates {
		return Config{}, fmt.Errorf("invalid config schema")
	}
	var version int
	if err := json.Unmarshal(versionRaw, &version); err != nil || version != FileVersion {
		if err != nil {
			return Config{}, err
		}

		return Config{}, fmt.Errorf("unsupported config version")
	}

	profiles, err := decodeConnectionProfiles(profilesRaw)
	if err != nil {
		return Config{}, err
	}
	activeID, err := decodeActiveConnectionProfileID(activeIDRaw)
	if err != nil {
		return Config{}, err
	}
	if activeID != nil && !containsConnectionProfileID(profiles, *activeID) {
		return Config{}, fmt.Errorf("active connection profile ID does not exist")
	}
	var flowStates map[string]json.RawMessage
	if err := json.Unmarshal(flowStatesRaw, &flowStates); err != nil || flowStates == nil {
		if err != nil {
			return Config{}, err
		}

		return Config{}, fmt.Errorf("flowStates must be an object")
	}

	return Config{
		Version:                   version,
		ConnectionProfiles:        profiles,
		ActiveConnectionProfileID: activeID,
		FlowStates:                flowStates,
	}, nil
}

// 接続プロファイル復号
func decodeConnectionProfiles(raw json.RawMessage) ([]ConnectionProfile, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("connectionProfiles must be an array")
	}

	profiles := make([]ConnectionProfile, 0, len(items))
	ids := make(map[string]struct{}, len(items))
	for _, item := range items {
		decoder := json.NewDecoder(bytes.NewReader(item))
		decoder.DisallowUnknownFields()
		var profile ConnectionProfile
		if err := decoder.Decode(&profile); err != nil {
			return nil, err
		}
		// 単体テスト到達不可: json.RawMessage は JSON 配列の各要素として単一の JSON 値だけを保持するため。
		if decoder.More() {
			return nil, fmt.Errorf("invalid connection profile")
		}
		if _, exists := ids[profile.ID]; exists {
			return nil, fmt.Errorf("connection profile ID is duplicated")
		}
		ids[profile.ID] = struct{}{}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// アクティブID復号
func decodeActiveConnectionProfileID(raw json.RawMessage) (*string, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	var activeID string
	if err := json.Unmarshal(raw, &activeID); err != nil {
		return nil, err
	}

	return &activeID, nil
}

// 接続プロファイルID存在確認
func containsConnectionProfileID(profiles []ConnectionProfile, id string) bool {
	for _, profile := range profiles {
		if profile.ID == id {
			return true
		}
	}

	return false
}
