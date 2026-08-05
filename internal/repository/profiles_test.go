package repository

import (
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 接続プロファイル読込検証
func TestAppRepositoryLoadProfiles(t *testing.T) {
	tests := []struct {
		name          string
		profiles      []config.ConnectionProfile
		activeID      *string
		wantProfiles  []domain.Profile
		wantActiveID  *string
		wantErrorCode apperr.Code
		withoutConfig bool
	}{
		{
			name: "パスワードを含めずに読み込む",
			profiles: []config.ConnectionProfile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			wantProfiles: []domain.Profile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   domain.DBTypePostgres,
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
		},
		{
			name: "アクティブIDを読み込む",
			profiles: []config.ConnectionProfile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			activeID: stringPointer("profile-1"),
			wantProfiles: []domain.Profile{{
				ID:       "profile-1",
				Name:     "Local DB",
				DBType:   domain.DBTypePostgres,
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			wantActiveID: stringPointer("profile-1"),
		},
		{
			name: "不正な設定を設定エラーとして返す",
			profiles: []config.ConnectionProfile{{
				ID:       "profile-1",
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Schema:   "public",
				User:     "user",
			}},
			wantErrorCode: apperr.CodeConfigBroken,
		},
		{
			name:          "設定ストアエラーを返す",
			wantErrorCode: apperr.CodeConfigNotFound,
			withoutConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := config.NewStore(t.TempDir())
			if !tt.withoutConfig {
				configuration := config.Default()
				configuration.ConnectionProfiles = tt.profiles
				configuration.ActiveConnectionProfileID = tt.activeID
				if err := store.Save(configuration); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			profiles, activeID, err := NewAppRepository(store).LoadProfiles()
			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("LoadProfiles() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode != "" {
				return
			}

			if !reflect.DeepEqual(profiles, tt.wantProfiles) {
				t.Errorf("LoadProfiles() profiles = %#v, want %#v", profiles, tt.wantProfiles)
			}
			if gotFound := activeID != nil; gotFound != (tt.wantActiveID != nil) {
				t.Fatalf("LoadProfiles() active ID found = %v, want %v", gotFound, tt.wantActiveID != nil)
			}
			if tt.wantActiveID != nil && *activeID != *tt.wantActiveID {
				t.Errorf("LoadProfiles() active ID = %q, want %q", *activeID, *tt.wantActiveID)
			}
		})
	}
}

// 接続プロファイル保存検証
func TestAppRepositorySaveProfiles(t *testing.T) {
	profile, err := domain.NewProfile("profile-2", "Remote DB", domain.DBTypeMySQL, "db.example.com", 3306, "app", "", "admin")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	tests := []struct {
		name          string
		profiles      []domain.Profile
		activeID      *string
		wantProfiles  []config.ConnectionProfile
		wantActiveID  *string
		wantFlowState string
		wantErrorCode apperr.Code
	}{
		{
			name:     "プロファイルとアクティブIDを保存する",
			profiles: []domain.Profile{profile},
			activeID: stringPointer("profile-2"),
			wantProfiles: []config.ConnectionProfile{
				{
					ID:       "profile-2",
					Name:     "Remote DB",
					DBType:   "mysql",
					Host:     "db.example.com",
					Port:     3306,
					Database: "app",
					Schema:   "",
					User:     "admin",
				},
			},
			wantActiveID:  stringPointer("profile-2"),
			wantFlowState: "complete",
		},
		{
			name:          "空のプロファイルと未選択アクティブIDを保存する",
			profiles:      []domain.Profile{},
			activeID:      nil,
			wantProfiles:  []config.ConnectionProfile{},
			wantActiveID:  nil,
			wantFlowState: "complete",
		},
		{
			name:          "設定ストアエラーを返す",
			wantErrorCode: apperr.CodeConfigNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := config.NewStore(t.TempDir())
			if tt.wantErrorCode == "" {
				configuration := config.Default()
				configuration.FlowStates["wizard"] = []byte(`{"step":"complete"}`)
				if err := store.Save(configuration); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			err := NewAppRepository(store).SaveProfiles(tt.profiles, tt.activeID)
			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("SaveProfiles() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode != "" {
				return
			}

			result, err := store.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !reflect.DeepEqual(result.Config.ConnectionProfiles, tt.wantProfiles) {
				t.Errorf("SaveProfiles() profiles = %#v, want %#v", result.Config.ConnectionProfiles, tt.wantProfiles)
			}
			if !reflect.DeepEqual(result.Config.ActiveConnectionProfileID, tt.wantActiveID) {
				t.Errorf("SaveProfiles() active ID = %#v, want %#v", result.Config.ActiveConnectionProfileID, tt.wantActiveID)
			}
			var flowState map[string]string
			if err := json.Unmarshal(result.Config.FlowStates["wizard"], &flowState); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got := flowState["step"]; got != tt.wantFlowState {
				t.Errorf("SaveProfiles() flow state = %q, want %q", got, tt.wantFlowState)
			}
		})
	}
}

// フロー状態保存検証
func TestAppRepositorySaveFlowState(t *testing.T) {
	state := domain.FlowState{
		Version: domain.FlowStateVersion,
		TableStates: map[string]domain.TableFlowState{
			"users": {X: 100.5, Y: -20, Expanded: true},
		},
	}
	tests := []struct {
		name          string
		withoutConfig bool
		wantErrorCode apperr.Code
	}{
		{
			name: "対象プロファイルの状態だけを保存する",
		},
		{
			name:          "設定未作成を保存失敗として返す",
			withoutConfig: true,
			wantErrorCode: apperr.CodeConfigNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := config.NewStore(t.TempDir())
			if !tt.withoutConfig {
				configuration := config.Default()
				configuration.FlowStates["profile-2"] = json.RawMessage(`{"extension":{"keep":true}}`)
				if err := store.Save(configuration); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			err := NewAppRepository(store).SaveFlowState("profile-1", state)
			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("SaveFlowState() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode != "" {
				return
			}

			result, err := store.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got := decodeFlowState(result.Config.FlowStates["profile-1"]); !reflect.DeepEqual(got, state) {
				t.Errorf("SaveFlowState() state = %#v, want %#v", got, state)
			}
			var gotExtension map[string]map[string]bool
			if err := json.Unmarshal(result.Config.FlowStates["profile-2"], &gotExtension); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got := gotExtension["extension"]["keep"]; !got {
				t.Errorf("SaveFlowState() other profile state = %#v, want preserved", gotExtension)
			}
		})
	}
}

// 設定更新直後の読込停止ストア
type blockingConfigStore struct {
	store       *config.Store
	loadStarted chan struct{}
	allowLoad   chan struct{}
}

// 設定読込
func (s *blockingConfigStore) Load() (config.LoadResult, error) {
	result, err := s.store.Load()
	if err != nil {
		return config.LoadResult{}, err
	}

	s.loadStarted <- struct{}{}
	<-s.allowLoad

	return result, nil
}

// 設定保存
func (s *blockingConfigStore) Save(configuration config.Config) error {
	return s.store.Save(configuration)
}

// 競合する設定更新検証
func TestAppRepositorySaveConfigUpdatesConcurrently(t *testing.T) {
	// 共有ロックの保持範囲と保存結果を一つのシナリオで検証するため、テーブル駆動にしない。
	baseStore := config.NewStore(t.TempDir())
	if err := baseStore.Save(config.Default()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store := &blockingConfigStore{
		store:       baseStore,
		loadStarted: make(chan struct{}, 2),
		allowLoad:   make(chan struct{}),
	}
	repository := newAppRepository(store, nil)
	state := domain.FlowState{
		Version: domain.FlowStateVersion,
		TableStates: map[string]domain.TableFlowState{
			"users": {X: 100, Y: 200, Expanded: true},
		},
	}
	profile, err := domain.NewProfile("profile-1", "Local DB", domain.DBTypePostgres, "localhost", 5432, "app", "public", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	activeID := "profile-1"

	errs := make(chan error, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		errs <- repository.SaveFlowState("profile-1", state)
	}()
	<-store.loadStarted
	if repository.configMu.TryLock() {
		repository.configMu.Unlock()
		t.Error("SaveFlowState() does not hold configMu while the configuration is loaded")
	}

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		errs <- repository.SaveProfiles([]domain.Profile{profile}, &activeID)
	}()

	store.allowLoad <- struct{}{}
	<-store.loadStarted
	if repository.configMu.TryLock() {
		repository.configMu.Unlock()
		t.Error("SaveProfiles() does not hold configMu while the configuration is loaded")
	}
	store.allowLoad <- struct{}{}

	waitGroup.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("configuration update error = %v", err)
		}
	}

	result, err := baseStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := decodeFlowState(result.Config.FlowStates["profile-1"]); !reflect.DeepEqual(got, state) {
		t.Errorf("flow state = %#v, want %#v", got, state)
	}
	if want := []config.ConnectionProfile{{
		ID:       profile.ID,
		Name:     profile.Name,
		DBType:   string(profile.DBType),
		Host:     profile.Host,
		Port:     profile.Port,
		Database: profile.Database,
		Schema:   profile.Schema,
		User:     profile.User,
	}}; !reflect.DeepEqual(result.Config.ConnectionProfiles, want) {
		t.Errorf("connection profiles = %#v, want %#v", result.Config.ConnectionProfiles, want)
	}
	if got := result.Config.ActiveConnectionProfileID; got == nil || *got != activeID {
		t.Errorf("active connection profile ID = %#v, want %q", got, activeID)
	}
}

// フロー状態読込検証
func TestAppRepositoryLoadFlowState(t *testing.T) {
	wantState := domain.FlowState{
		Version: domain.FlowStateVersion,
		TableStates: map[string]domain.TableFlowState{
			"users": {X: 100.5, Y: -20, Expanded: true},
		},
	}
	tests := []struct {
		name          string
		raw           string
		withoutConfig bool
		want          domain.FlowState
		wantErrorCode apperr.Code
	}{
		{
			name: "保存済み状態を返す",
			raw:  `{"version":1,"tableStates":{"users":{"x":100.5,"y":-20,"expanded":true}}}`,
			want: wantState,
		},
		{
			name: "未保存状態を空状態として返す",
			want: domain.EmptyFlowState(),
		},
		{
			name: "FlowStateとして不正なJSON値を空状態として返す",
			raw:  `"invalid"`,
			want: domain.EmptyFlowState(),
		},
		{
			name: "未知バージョンを空状態として返す",
			raw:  `{"version":2,"tableStates":{}}`,
			want: domain.EmptyFlowState(),
		},
		{
			name: "非有限数値を空状態として返す",
			raw:  `{"version":1,"tableStates":{"users":{"x":1e1000,"y":0,"expanded":true}}}`,
			want: domain.EmptyFlowState(),
		},
		{
			name: "nullのテーブル状態を空状態として返す",
			raw:  `{"version":1,"tableStates":null}`,
			want: domain.EmptyFlowState(),
		},
		{
			name:          "設定読込失敗を返す",
			withoutConfig: true,
			wantErrorCode: apperr.CodeConfigNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := config.NewStore(t.TempDir())
			if !tt.withoutConfig {
				configuration := config.Default()
				if tt.raw != "" {
					configuration.FlowStates["profile-1"] = json.RawMessage(tt.raw)
				}
				if err := store.Save(configuration); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			got, err := NewAppRepository(store).LoadFlowState("profile-1")
			if gotCode := errorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("LoadFlowState() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantErrorCode != "" {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadFlowState() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// プロファイル別フロー状態読込検証
func TestAppRepositoryLoadFlowStateByProfile(t *testing.T) {
	store := config.NewStore(t.TempDir())
	configuration := config.Default()
	configuration.FlowStates = map[string]json.RawMessage{
		"profile-1": json.RawMessage(`{"version":1,"tableStates":{"users":{"x":100,"y":200,"expanded":true}}}`),
		"profile-2": json.RawMessage(`{"version":1,"tableStates":{"orders":{"x":300,"y":400,"expanded":false}}}`),
	}
	if err := store.Save(configuration); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	tests := []struct {
		name      string
		profileID string
		want      domain.FlowState
	}{
		{
			name:      "profile-1の状態を返す",
			profileID: "profile-1",
			want: domain.FlowState{
				Version: domain.FlowStateVersion,
				TableStates: map[string]domain.TableFlowState{
					"users": {X: 100, Y: 200, Expanded: true},
				},
			},
		},
		{
			name:      "profile-2の状態を返す",
			profileID: "profile-2",
			want: domain.FlowState{
				Version: domain.FlowStateVersion,
				TableStates: map[string]domain.TableFlowState{
					"orders": {X: 300, Y: 400, Expanded: false},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAppRepository(store).LoadFlowState(tt.profileID)
			if err != nil {
				t.Fatalf("LoadFlowState() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadFlowState() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// 設定全体構文不正時フロー状態読込検証
func TestAppRepositoryLoadFlowStateWithBrokenConfig(t *testing.T) {
	store := config.NewStore(t.TempDir())
	content := `{"version":1,"connectionProfiles":[],"activeConnectionProfileId":null,"flowStates":{"profile-1":}}`
	if err := os.WriteFile(store.Path(), []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := NewAppRepository(store).LoadFlowState("profile-1")
	if gotCode := errorCode(err); gotCode != apperr.CodeConfigBroken {
		t.Errorf("LoadFlowState() error code = %q, want %q", gotCode, apperr.CodeConfigBroken)
	}
}

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}
