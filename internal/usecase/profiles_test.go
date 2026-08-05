package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type appRepositoryStub struct {
	profiles       []domain.Profile
	activeID       *string
	loadErr        error
	saveErr        error
	saveErrs       []error
	savedProfiles  []domain.Profile
	savedActiveID  *string
	savedHistory   []savedProfileState
	saveCalls      int
	credentials    *credentialState
	connection     *connectionState
	flowState      domain.FlowState
	flowStateErr   error
	flowStateID    string
	savedFlowState domain.FlowState
	savedFlowID    string
	saveFlowErr    error
}

// 接続プロファイル読込再現
func (s *appRepositoryStub) LoadProfiles() ([]domain.Profile, *string, error) {
	return s.profiles, s.activeID, s.loadErr
}

// フロー状態読込再現
func (s *appRepositoryStub) LoadFlowState(profileID string) (domain.FlowState, error) {
	s.flowStateID = profileID

	return s.flowState, s.flowStateErr
}

// フロー状態保存再現
func (s *appRepositoryStub) SaveFlowState(profileID string, state domain.FlowState) error {
	s.savedFlowID = profileID
	s.savedFlowState = state

	return s.saveFlowErr
}

// 接続プロファイル保存再現
func (s *appRepositoryStub) SaveProfiles(profiles []domain.Profile, activeID *string) error {
	s.saveCalls++
	s.savedProfiles = profiles
	s.savedActiveID = activeID
	s.savedHistory = append(s.savedHistory, savedProfileState{
		profiles: profiles,
		activeID: activeID,
	})
	if len(s.saveErrs) >= s.saveCalls && s.saveErrs[s.saveCalls-1] != nil {
		return s.saveErrs[s.saveCalls-1]
	}
	if s.saveErr != nil {
		return s.saveErr
	}

	s.profiles = profiles
	s.activeID = activeID

	return nil
}

// 資格情報取得再現
func (s *appRepositoryStub) GetCredential(profileID string) (string, bool, error) {
	s.credentials.getIDs = append(s.credentials.getIDs, profileID)

	return s.credentials.credential, s.credentials.found, s.credentials.getErr
}

// 資格情報設定再現
func (s *appRepositoryStub) SetCredential(_ string, credential string) error {
	s.credentials.setValues = append(s.credentials.setValues, credential)

	return s.credentials.setErr
}

// 資格情報削除再現
func (s *appRepositoryStub) DeleteCredential(profileID string) error {
	s.credentials.deleteIDs = append(s.credentials.deleteIDs, profileID)

	return s.credentials.deleteErr
}

// データベース接続確認再現
func (s *appRepositoryStub) CheckConnection(_ context.Context, profile domain.Profile, password string) error {
	s.connection.calls++
	s.connection.profile = profile
	s.connection.password = password

	return s.connection.err
}

// スキーマ取得再現
func (*appRepositoryStub) InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error) {
	return domain.Schema{}, nil
}

// テーブル構造取得再現
func (*appRepositoryStub) InspectTableStructure(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStructure, error) {
	return domain.TableStructure{}, nil
}

// テーブル統計取得再現
func (*appRepositoryStub) InspectTableStatistics(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStatistics, error) {
	return domain.TableStatistics{}, nil
}

// テーブル行一覧再現
func (*appRepositoryStub) ListRows(context.Context, domain.Profile, string, domain.TableQuery) (domain.TableRows, error) {
	return domain.TableRows{}, nil
}

// テーブル行追加再現
func (*appRepositoryStub) InsertRow(context.Context, domain.Profile, string, domain.TableRef, domain.InsertRow) (domain.AffectedRows, error) {
	return domain.AffectedRows{}, nil
}

// テーブルセル更新再現
func (*appRepositoryStub) UpdateCell(context.Context, domain.Profile, string, domain.TableRef, domain.CellUpdate) (domain.AffectedRows, error) {
	return domain.AffectedRows{}, nil
}

// 接続プロファイル読込
func TestAppUseCaseLoadProfiles(t *testing.T) {
	profile, err := domain.NewProfile("profile-1", "Local DB", domain.DBTypePostgres, "localhost", 5432, "app", "public", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	repositoryErr := errors.New("repository error")
	tests := []struct {
		name          string
		repository    appRepositoryStub
		wantProfiles  []domain.Profile
		wantActiveID  *string
		wantFound     bool
		wantCause     error
		wantErrorCode apperr.Code
	}{
		{
			name: "プロファイルを返す",
			repository: appRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer("profile-1"),
			},
			wantProfiles: []domain.Profile{profile},
			wantActiveID: stringPointer("profile-1"),
		},
		{
			name: "アクティブプロファイル未選択を返す",
			repository: appRepositoryStub{
				profiles: []domain.Profile{profile},
			},
			wantProfiles: []domain.Profile{profile},
		},
		{
			name: "リポジトリエラーを返す",
			repository: appRepositoryStub{
				loadErr: repositoryErr,
			},
			wantFound: true,
			wantCause: repositoryErr,
		},
		{
			name: "存在しないアクティブIDを設定エラーとして返す",
			repository: appRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer("missing"),
			},
			wantFound:     true,
			wantErrorCode: apperr.CodeConfigBroken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			useCase := NewAppUseCase(&repository)

			gotProfiles, gotActiveID, err := useCase.LoadProfiles()

			if !reflect.DeepEqual(gotProfiles, tt.wantProfiles) {
				t.Errorf("LoadProfiles() profiles = %#v, want %#v", gotProfiles, tt.wantProfiles)
			}
			if !reflect.DeepEqual(gotActiveID, tt.wantActiveID) {
				t.Errorf("LoadProfiles() active ID = %#v, want %#v", gotActiveID, tt.wantActiveID)
			}
			if repository.saveCalls != 0 {
				t.Errorf("LoadProfiles() SaveProfiles() calls = %d, want 0", repository.saveCalls)
			}
			if gotFound := err != nil; gotFound != tt.wantFound {
				t.Fatalf("LoadProfiles() error found = %v, want %v", gotFound, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if tt.wantErrorCode != "" && !apperr.IsCode(err, tt.wantErrorCode) {
				t.Errorf("LoadProfiles() error = %v, want code %v", err, tt.wantErrorCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("LoadProfiles() error = %v, want cause %v", err, tt.wantCause)
			}
		})
	}
}

// フロー状態取得検証
func TestAppUseCaseLoadFlowState(t *testing.T) {
	profile := newSaveTestProfile(t, "profile-1")
	wantState := domain.FlowState{
		Version: domain.FlowStateVersion,
		TableStates: map[string]domain.TableFlowState{
			"users": {
				X:        100,
				Y:        200,
				Expanded: true,
			},
		},
	}
	tests := []struct {
		name          string
		repository    appRepositoryStub
		wantState     domain.FlowState
		wantCode      apperr.Code
		wantProfileID string
	}{
		{
			name: "アクティブプロファイルの状態を返す",
			repository: appRepositoryStub{
				profiles:  []domain.Profile{profile},
				activeID:  stringPointer(profile.ID),
				flowState: wantState,
			},
			wantState:     wantState,
			wantProfileID: profile.ID,
		},
		{
			name: "アクティブプロファイル未選択を返す",
			repository: appRepositoryStub{
				profiles: []domain.Profile{profile},
			},
			wantCode: apperr.CodeProfileNotFound,
		},
		{
			name: "削除済みアクティブプロファイルを設定エラーとして返す",
			repository: appRepositoryStub{
				activeID: stringPointer(profile.ID),
			},
			wantCode: apperr.CodeConfigBroken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			got, err := NewAppUseCase(&repository).LoadFlowState()
			if gotCode := saveErrorCode(err); gotCode != tt.wantCode {
				t.Errorf("LoadFlowState() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if tt.wantCode != "" {
				return
			}
			if !reflect.DeepEqual(got, tt.wantState) {
				t.Errorf("LoadFlowState() = %#v, want %#v", got, tt.wantState)
			}
			if gotProfileID := repository.flowStateID; gotProfileID != tt.wantProfileID {
				t.Errorf("LoadFlowState() profile ID = %q, want %q", gotProfileID, tt.wantProfileID)
			}
		})
	}
}

// フロー状態保存検証
func TestAppUseCaseSaveFlowState(t *testing.T) {
	profile := newSaveTestProfile(t, "profile-1")
	repositoryErr := errors.New("repository error")
	state := domain.FlowState{
		Version: domain.FlowStateVersion,
		TableStates: map[string]domain.TableFlowState{
			"users": {
				X:        100,
				Y:        200,
				Expanded: true,
			},
		},
	}
	tests := []struct {
		name          string
		repository    appRepositoryStub
		state         domain.FlowState
		wantError     bool
		wantCode      apperr.Code
		wantCause     error
		wantProfileID string
		wantSave      bool
	}{
		{
			name: "アクティブプロファイルに状態を保存する",
			repository: appRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			state:         state,
			wantProfileID: profile.ID,
			wantSave:      true,
		},
		{
			name: "プロファイル読込失敗を返す",
			repository: appRepositoryStub{
				loadErr: repositoryErr,
			},
			state:     state,
			wantError: true,
			wantCause: repositoryErr,
		},
		{
			name: "不正な状態を入力エラーとして返す",
			repository: appRepositoryStub{
				profiles: []domain.Profile{profile},
				activeID: stringPointer(profile.ID),
			},
			state: domain.FlowState{
				Version:     domain.FlowStateVersion + 1,
				TableStates: map[string]domain.TableFlowState{},
			},
			wantError: true,
			wantCode:  apperr.CodeValidationFailed,
		},
		{
			name: "アクティブプロファイル未選択を返す",
			repository: appRepositoryStub{
				profiles: []domain.Profile{profile},
			},
			state:     state,
			wantError: true,
			wantCode:  apperr.CodeProfileNotFound,
		},
		{
			name: "設定保存失敗を返す",
			repository: appRepositoryStub{
				profiles:    []domain.Profile{profile},
				activeID:    stringPointer(profile.ID),
				saveFlowErr: apperr.New(apperr.CodeConfigWriteFailed),
			},
			state:     state,
			wantError: true,
			wantCode:  apperr.CodeConfigWriteFailed,
			wantSave:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			got, err := NewAppUseCase(&repository).SaveFlowState(tt.state)

			if gotFound := err != nil; gotFound != tt.wantError {
				t.Fatalf("SaveFlowState() error found = %v, want %v", gotFound, tt.wantError)
			}
			if gotCode := saveErrorCode(err); gotCode != tt.wantCode {
				t.Errorf("SaveFlowState() error code = %q, want %q", gotCode, tt.wantCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("SaveFlowState() error = %v, want cause %v", err, tt.wantCause)
			}
			if tt.wantError {
				if !tt.wantSave && repository.savedFlowID != "" {
					t.Errorf("SaveFlowState() profile ID = %q, want empty", repository.savedFlowID)
				}

				return
			}
			if !reflect.DeepEqual(got, tt.state) {
				t.Errorf("SaveFlowState() = %#v, want %#v", got, tt.state)
			}
			if gotProfileID := repository.savedFlowID; gotProfileID != tt.wantProfileID {
				t.Errorf("SaveFlowState() profile ID = %q, want %q", gotProfileID, tt.wantProfileID)
			}
			if gotSaved := !reflect.DeepEqual(repository.savedFlowState, domain.FlowState{}); gotSaved != tt.wantSave {
				t.Errorf("SaveFlowState() saved = %v, want %v", gotSaved, tt.wantSave)
			}
		})
	}
}

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}

type credentialState struct {
	credential string
	found      bool
	getErr     error
	setErr     error
	deleteErr  error
	getIDs     []string
	setValues  []string
	deleteIDs  []string
}

type savedProfileState struct {
	profiles []domain.Profile
	activeID *string
}

type connectionState struct {
	err      error
	calls    int
	profile  domain.Profile
	password string
}

// 接続プロファイル保存
func TestAppUseCaseSaveConnectionProfile(t *testing.T) {
	existing := newSaveTestProfile(t, "profile-1")
	repositoryErr := errors.New("repository error")
	connectionErr := errors.New("connection error")
	tests := []struct {
		name              string
		draft             domain.ProfileDraft
		profiles          []domain.Profile
		credentials       credentialState
		repositoryLoadErr error
		repositorySaveErr error
		connectionErr     error
		wantErrorCode     apperr.Code
		wantCause         error
		wantSaveCalls     int
		wantConnection    int
		wantPassword      string
		wantSetValues     []string
		wantDeleteCount   int
		wantProfileCount  int
	}{
		{
			name:          "不正な下書きを入力エラーとして返す",
			draft:         domain.ProfileDraft{},
			wantErrorCode: apperr.CodeValidationFailed,
		},
		{
			name:              "プロファイル読み込み失敗をそのまま返す",
			draft:             newSaveTestDraft(t, "", "new-password"),
			repositoryLoadErr: repositoryErr,
			wantCause:         repositoryErr,
		},
		{
			name:             "新規プロファイルを資格情報とともに保存する",
			draft:            newSaveTestDraft(t, "", "new-password"),
			wantSaveCalls:    1,
			wantConnection:   1,
			wantPassword:     "new-password",
			wantSetValues:    []string{"new-password"},
			wantProfileCount: 1,
		},
		{
			name:             "新規プロファイルをパスワードなしで保存する",
			draft:            newSaveTestDraft(t, "", ""),
			wantSaveCalls:    1,
			wantConnection:   1,
			wantProfileCount: 1,
		},
		{
			name:              "パスワードなしの設定保存失敗を返す",
			draft:             newSaveTestDraft(t, "", ""),
			repositorySaveErr: repositoryErr,
			wantErrorCode:     apperr.CodeConfigSaveFailed,
			wantSaveCalls:     1,
			wantConnection:    1,
		},
		{
			name:             "既存資格情報で編集する",
			draft:            newSaveTestDraft(t, "profile-1", ""),
			profiles:         []domain.Profile{existing},
			credentials:      credentialState{credential: "old-password", found: true},
			wantSaveCalls:    1,
			wantConnection:   1,
			wantPassword:     "old-password",
			wantProfileCount: 1,
		},
		{
			name:          "存在しないプロファイルの編集を拒否する",
			draft:         newSaveTestDraft(t, "missing", "new-password"),
			profiles:      []domain.Profile{existing},
			wantErrorCode: apperr.CodeProfileNotFound,
		},
		{
			name:          "既存資格情報がない編集を拒否する",
			draft:         newSaveTestDraft(t, "profile-1", ""),
			profiles:      []domain.Profile{existing},
			wantErrorCode: apperr.CodeCredentialUnavailable,
		},
		{
			name:           "接続失敗時は保存しない",
			draft:          newSaveTestDraft(t, "", "new-password"),
			connectionErr:  connectionErr,
			wantErrorCode:  apperr.CodeConnectionFailed,
			wantConnection: 1,
			wantPassword:   "new-password",
		},
		{
			name:           "保存前資格情報の取得失敗を返す",
			draft:          newSaveTestDraft(t, "profile-1", "new-password"),
			profiles:       []domain.Profile{existing},
			credentials:    credentialState{getErr: repositoryErr},
			wantErrorCode:  apperr.CodeCredentialUnavailable,
			wantConnection: 1,
			wantPassword:   "new-password",
		},
		{
			name:             "資格情報保存失敗時は設定を変更せず復旧しない",
			draft:            newSaveTestDraft(t, "", "new-password"),
			credentials:      credentialState{setErr: repositoryErr},
			wantErrorCode:    apperr.CodeCredentialSaveFailed,
			wantConnection:   1,
			wantPassword:     "new-password",
			wantSetValues:    []string{"new-password"},
			wantDeleteCount:  0,
			wantProfileCount: 0,
		},
		{
			name:              "設定保存失敗時は新規資格情報を削除する",
			draft:             newSaveTestDraft(t, "", "new-password"),
			repositorySaveErr: repositoryErr,
			wantErrorCode:     apperr.CodeConfigSaveFailed,
			wantSaveCalls:     1,
			wantConnection:    1,
			wantPassword:      "new-password",
			wantSetValues:     []string{"new-password"},
			wantDeleteCount:   1,
		},
		{
			name:              "設定保存失敗時は既存資格情報を復旧する",
			draft:             newSaveTestDraft(t, "profile-1", "new-password"),
			profiles:          []domain.Profile{existing},
			credentials:       credentialState{credential: "old-password", found: true},
			repositorySaveErr: repositoryErr,
			wantErrorCode:     apperr.CodeConfigSaveFailed,
			wantSaveCalls:     1,
			wantConnection:    1,
			wantPassword:      "new-password",
			wantSetValues:     []string{"new-password", "old-password"},
		},
		{
			name:              "資格情報復旧失敗を返す",
			draft:             newSaveTestDraft(t, "", "new-password"),
			repositorySaveErr: repositoryErr,
			credentials:       credentialState{deleteErr: repositoryErr},
			wantErrorCode:     apperr.CodeConsistencyRecoveryFailed,
			wantSaveCalls:     1,
			wantConnection:    1,
			wantPassword:      "new-password",
			wantSetValues:     []string{"new-password"},
			wantDeleteCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials := tt.credentials
			connection := &connectionState{err: tt.connectionErr}
			repository := &appRepositoryStub{
				profiles:    tt.profiles,
				loadErr:     tt.repositoryLoadErr,
				saveErr:     tt.repositorySaveErr,
				credentials: &credentials,
				connection:  connection,
			}
			useCase := NewAppUseCase(repository)

			profiles, _, err := useCase.SaveConnectionProfile(context.Background(), tt.draft)

			if gotCode := saveErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("SaveConnectionProfile() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("SaveConnectionProfile() error = %v, want cause %v", err, tt.wantCause)
			}
			if got := repository.saveCalls; got != tt.wantSaveCalls {
				t.Errorf("SaveProfiles() calls = %d, want %d", got, tt.wantSaveCalls)
			}
			if got := connection.calls; got != tt.wantConnection {
				t.Errorf("CheckConnection() calls = %d, want %d", got, tt.wantConnection)
			}
			if got := connection.password; got != tt.wantPassword {
				t.Errorf("CheckConnection() password = %q, want %q", got, tt.wantPassword)
			}
			if !equalStrings(credentials.setValues, tt.wantSetValues) {
				t.Errorf("SetCredential() values = %#v, want %#v", credentials.setValues, tt.wantSetValues)
			}
			if got := len(credentials.deleteIDs); got != tt.wantDeleteCount {
				t.Errorf("DeleteCredential() calls = %d, want %d", got, tt.wantDeleteCount)
			}
			if tt.wantErrorCode == "" && len(profiles) != tt.wantProfileCount {
				t.Errorf("SaveConnectionProfile() profiles = %d, want %d", len(profiles), tt.wantProfileCount)
			}
		})
	}
}

// アクティブ接続プロファイル切替
func TestAppUseCaseActivateConnectionProfile(t *testing.T) {
	first := newSaveTestProfile(t, "profile-1")
	second := newSaveTestProfile(t, "profile-2")
	repositoryErr := errors.New("repository error")
	connectionErr := errors.New("connection error")
	tests := []struct {
		name              string
		profileID         string
		profiles          []domain.Profile
		activeID          *string
		credentials       credentialState
		repositoryLoadErr error
		repositorySaveErr error
		connectionErr     error
		wantErrorCode     apperr.Code
		wantCause         error
		wantGetIDs        []string
		wantSaveCalls     int
		wantConnection    int
		wantPassword      string
		wantActiveID      *string
		wantStoredActive  *string
	}{
		{
			name:          "空のプロファイルIDを未存在として返す",
			profiles:      []domain.Profile{first},
			wantErrorCode: apperr.CodeProfileNotFound,
		},
		{
			name:          "存在しないプロファイルを未存在として返す",
			profileID:     "missing",
			profiles:      []domain.Profile{first},
			wantErrorCode: apperr.CodeProfileNotFound,
		},
		{
			name:              "プロファイル読み込み失敗をそのまま返す",
			profileID:         "profile-1",
			repositoryLoadErr: repositoryErr,
			wantCause:         repositoryErr,
		},
		{
			name:             "同じアクティブプロファイルは接続確認せずに返す",
			profileID:        "profile-1",
			profiles:         []domain.Profile{first, second},
			activeID:         stringPointer("profile-1"),
			wantActiveID:     stringPointer("profile-1"),
			wantStoredActive: stringPointer("profile-1"),
			wantSaveCalls:    0,
			wantConnection:   0,
		},
		{
			name:             "保存済み資格情報で接続確認して切り替える",
			profileID:        "profile-2",
			profiles:         []domain.Profile{first, second},
			activeID:         stringPointer("profile-1"),
			credentials:      credentialState{credential: "secret", found: true},
			wantGetIDs:       []string{"profile-2"},
			wantSaveCalls:    1,
			wantConnection:   1,
			wantPassword:     "secret",
			wantActiveID:     stringPointer("profile-2"),
			wantStoredActive: stringPointer("profile-2"),
		},
		{
			name:             "未登録資格情報では空パスワードで切り替える",
			profileID:        "profile-2",
			profiles:         []domain.Profile{first, second},
			activeID:         stringPointer("profile-1"),
			wantGetIDs:       []string{"profile-2"},
			wantSaveCalls:    1,
			wantConnection:   1,
			wantPassword:     "",
			wantActiveID:     stringPointer("profile-2"),
			wantStoredActive: stringPointer("profile-2"),
		},
		{
			name:             "資格情報取得失敗時はアクティブIDを変更しない",
			profileID:        "profile-2",
			profiles:         []domain.Profile{first, second},
			activeID:         stringPointer("profile-1"),
			credentials:      credentialState{getErr: repositoryErr},
			wantErrorCode:    apperr.CodeCredentialUnavailable,
			wantCause:        repositoryErr,
			wantGetIDs:       []string{"profile-2"},
			wantStoredActive: stringPointer("profile-1"),
		},
		{
			name:             "接続確認失敗時はアクティブIDを変更しない",
			profileID:        "profile-2",
			profiles:         []domain.Profile{first, second},
			activeID:         stringPointer("profile-1"),
			credentials:      credentialState{credential: "secret", found: true},
			connectionErr:    connectionErr,
			wantErrorCode:    apperr.CodeConnectionFailed,
			wantCause:        connectionErr,
			wantGetIDs:       []string{"profile-2"},
			wantConnection:   1,
			wantPassword:     "secret",
			wantStoredActive: stringPointer("profile-1"),
		},
		{
			name:              "設定保存失敗時はアクティブIDを変更しない",
			profileID:         "profile-2",
			profiles:          []domain.Profile{first, second},
			activeID:          stringPointer("profile-1"),
			credentials:       credentialState{credential: "secret", found: true},
			repositorySaveErr: repositoryErr,
			wantErrorCode:     apperr.CodeConfigSaveFailed,
			wantCause:         repositoryErr,
			wantGetIDs:        []string{"profile-2"},
			wantSaveCalls:     1,
			wantConnection:    1,
			wantPassword:      "secret",
			wantStoredActive:  stringPointer("profile-1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials := tt.credentials
			connection := &connectionState{err: tt.connectionErr}
			repository := &appRepositoryStub{
				profiles:    tt.profiles,
				activeID:    tt.activeID,
				loadErr:     tt.repositoryLoadErr,
				saveErr:     tt.repositorySaveErr,
				credentials: &credentials,
				connection:  connection,
			}

			profiles, activeID, err := NewAppUseCase(repository).ActivateConnectionProfile(context.Background(), tt.profileID)

			if gotCode := saveErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("ActivateConnectionProfile() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("ActivateConnectionProfile() error = %v, want cause %v", err, tt.wantCause)
			}
			if !reflect.DeepEqual(credentials.getIDs, tt.wantGetIDs) {
				t.Errorf("GetCredential() profile IDs = %#v, want %#v", credentials.getIDs, tt.wantGetIDs)
			}
			if got := repository.saveCalls; got != tt.wantSaveCalls {
				t.Errorf("SaveProfiles() calls = %d, want %d", got, tt.wantSaveCalls)
			}
			if got := connection.calls; got != tt.wantConnection {
				t.Errorf("CheckConnection() calls = %d, want %d", got, tt.wantConnection)
			}
			if got := connection.password; got != tt.wantPassword {
				t.Errorf("CheckConnection() password = %q, want %q", got, tt.wantPassword)
			}
			if !reflect.DeepEqual(repository.activeID, tt.wantStoredActive) {
				t.Errorf("repository active ID = %#v, want %#v", repository.activeID, tt.wantStoredActive)
			}
			if tt.wantErrorCode != "" || tt.wantCause != nil {
				if profiles != nil {
					t.Errorf("ActivateConnectionProfile() profiles = %#v, want nil", profiles)
				}
				if activeID != nil {
					t.Errorf("ActivateConnectionProfile() active ID = %#v, want nil", activeID)
				}

				return
			}
			if !reflect.DeepEqual(profiles, tt.profiles) {
				t.Errorf("ActivateConnectionProfile() profiles = %#v, want %#v", profiles, tt.profiles)
			}
			if !reflect.DeepEqual(activeID, tt.wantActiveID) {
				t.Errorf("ActivateConnectionProfile() active ID = %#v, want %#v", activeID, tt.wantActiveID)
			}
			if tt.wantSaveCalls == 1 && !reflect.DeepEqual(repository.savedActiveID, tt.wantActiveID) {
				t.Errorf("SaveProfiles() active ID = %#v, want %#v", repository.savedActiveID, tt.wantActiveID)
			}
		})
	}
}

// 接続プロファイル削除
func TestAppUseCaseDeleteConnectionProfile(t *testing.T) {
	first := newSaveTestProfile(t, "profile-1")
	second := newSaveTestProfile(t, "profile-2")
	repositoryErr := errors.New("repository error")
	credentialErr := errors.New("credential store failure: password=secret-password")
	tests := []struct {
		name                 string
		profileID            string
		profiles             []domain.Profile
		activeID             *string
		repositoryLoadErr    error
		repositorySaveErrs   []error
		credentialDeleteErr  error
		wantErrorCode        apperr.Code
		wantCause            error
		wantProfiles         []domain.Profile
		wantActiveID         *string
		wantStoredProfiles   []domain.Profile
		wantStoredActiveID   *string
		wantSaveCalls        int
		wantCredentialDelete []string
		wantConnectionCalls  int
		wantSavedHistory     []savedProfileState
	}{
		{
			name:          "存在しないプロファイルを未存在として返す",
			profileID:     "missing",
			profiles:      []domain.Profile{first},
			wantErrorCode: apperr.CodeProfileNotFound,
			wantStoredProfiles: []domain.Profile{
				first,
			},
		},
		{
			name:              "プロファイル読み込み失敗をそのまま返す",
			profileID:         "profile-1",
			repositoryLoadErr: repositoryErr,
			wantCause:         repositoryErr,
		},
		{
			name:      "非アクティブプロファイルと資格情報を削除する",
			profileID: "profile-1",
			profiles:  []domain.Profile{first, second},
			activeID:  stringPointer("profile-2"),
			wantProfiles: []domain.Profile{
				second,
			},
			wantActiveID: stringPointer("profile-2"),
			wantStoredProfiles: []domain.Profile{
				second,
			},
			wantStoredActiveID:   stringPointer("profile-2"),
			wantSaveCalls:        1,
			wantCredentialDelete: []string{"profile-1"},
			wantSavedHistory: []savedProfileState{
				{
					profiles: []domain.Profile{second},
					activeID: stringPointer("profile-2"),
				},
			},
		},
		{
			name:      "アクティブプロファイルを削除して未選択にする",
			profileID: "profile-1",
			profiles: []domain.Profile{
				first,
				second,
			},
			activeID: stringPointer("profile-1"),
			wantProfiles: []domain.Profile{
				second,
			},
			wantStoredProfiles: []domain.Profile{
				second,
			},
			wantSaveCalls:        1,
			wantCredentialDelete: []string{"profile-1"},
			wantSavedHistory: []savedProfileState{
				{
					profiles: []domain.Profile{second},
					activeID: nil,
				},
			},
		},
		{
			name:      "設定保存失敗時は資格情報を削除しない",
			profileID: "profile-1",
			profiles:  []domain.Profile{first},
			repositorySaveErrs: []error{
				repositoryErr,
			},
			wantErrorCode:      apperr.CodeConfigSaveFailed,
			wantCause:          repositoryErr,
			wantStoredProfiles: []domain.Profile{first},
			wantSaveCalls:      1,
			wantSavedHistory: []savedProfileState{
				{
					profiles: []domain.Profile{},
					activeID: nil,
				},
			},
		},
		{
			name:                "資格情報削除失敗時は設定を復旧する",
			profileID:           "profile-1",
			profiles:            []domain.Profile{first, second},
			activeID:            stringPointer("profile-1"),
			credentialDeleteErr: credentialErr,
			wantErrorCode:       apperr.CodeCredentialDeleteFailed,
			wantCause:           credentialErr,
			wantStoredProfiles: []domain.Profile{
				first,
				second,
			},
			wantStoredActiveID:   stringPointer("profile-1"),
			wantSaveCalls:        2,
			wantCredentialDelete: []string{"profile-1"},
			wantSavedHistory: []savedProfileState{
				{
					profiles: []domain.Profile{second},
					activeID: nil,
				},
				{
					profiles: []domain.Profile{first, second},
					activeID: stringPointer("profile-1"),
				},
			},
		},
		{
			name:      "設定復旧失敗時は整合性復旧エラーを返す",
			profileID: "profile-1",
			profiles:  []domain.Profile{first},
			activeID:  stringPointer("profile-1"),
			repositorySaveErrs: []error{
				nil,
				repositoryErr,
			},
			credentialDeleteErr: credentialErr,
			wantErrorCode:       apperr.CodeConsistencyRecoveryFailed,
			wantCause:           repositoryErr,
			wantStoredProfiles:  []domain.Profile{},
			wantSaveCalls:       2,
			wantCredentialDelete: []string{
				"profile-1",
			},
			wantSavedHistory: []savedProfileState{
				{
					profiles: []domain.Profile{},
					activeID: nil,
				},
				{
					profiles: []domain.Profile{first},
					activeID: stringPointer("profile-1"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials := credentialState{deleteErr: tt.credentialDeleteErr}
			connection := &connectionState{}
			repository := &appRepositoryStub{
				profiles:    tt.profiles,
				activeID:    tt.activeID,
				loadErr:     tt.repositoryLoadErr,
				saveErrs:    tt.repositorySaveErrs,
				credentials: &credentials,
				connection:  connection,
			}

			profiles, activeID, err := NewAppUseCase(repository).DeleteConnectionProfile(tt.profileID)

			if gotCode := saveErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("DeleteConnectionProfile() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("DeleteConnectionProfile() error = %v, want cause %v", err, tt.wantCause)
			}
			if got := repository.saveCalls; got != tt.wantSaveCalls {
				t.Errorf("SaveProfiles() calls = %d, want %d", got, tt.wantSaveCalls)
			}
			if !reflect.DeepEqual(credentials.deleteIDs, tt.wantCredentialDelete) {
				t.Errorf("DeleteCredential() profile IDs = %#v, want %#v", credentials.deleteIDs, tt.wantCredentialDelete)
			}
			if got := connection.calls; got != tt.wantConnectionCalls {
				t.Errorf("CheckConnection() calls = %d, want %d", got, tt.wantConnectionCalls)
			}
			if !reflect.DeepEqual(repository.profiles, tt.wantStoredProfiles) {
				t.Errorf("repository profiles = %#v, want %#v", repository.profiles, tt.wantStoredProfiles)
			}
			if !reflect.DeepEqual(repository.activeID, tt.wantStoredActiveID) {
				t.Errorf("repository active ID = %#v, want %#v", repository.activeID, tt.wantStoredActiveID)
			}
			if !reflect.DeepEqual(repository.savedHistory, tt.wantSavedHistory) {
				t.Errorf("SaveProfiles() history = %#v, want %#v", repository.savedHistory, tt.wantSavedHistory)
			}
			if tt.wantErrorCode != "" || tt.wantCause != nil {
				if profiles != nil {
					t.Errorf("DeleteConnectionProfile() profiles = %#v, want nil", profiles)
				}
				if activeID != nil {
					t.Errorf("DeleteConnectionProfile() active ID = %#v, want nil", activeID)
				}

				return
			}
			if !reflect.DeepEqual(profiles, tt.wantProfiles) {
				t.Errorf("DeleteConnectionProfile() profiles = %#v, want %#v", profiles, tt.wantProfiles)
			}
			if !reflect.DeepEqual(activeID, tt.wantActiveID) {
				t.Errorf("DeleteConnectionProfile() active ID = %#v, want %#v", activeID, tt.wantActiveID)
			}
		})
	}
}

// 保存前資格情報取得検証
func TestAppUseCasePreviousCredential(t *testing.T) {
	repositoryErr := errors.New("repository error")
	tests := []struct {
		name           string
		profileID      string
		editing        bool
		credentials    credentialState
		wantCredential string
		wantFound      bool
		wantErrorCode  apperr.Code
		wantGetIDs     []string
	}{
		{
			name:      "新規作成では資格情報を取得しない",
			profileID: "profile-1",
		},
		{
			name:      "既存の資格情報を返す",
			profileID: "profile-1",
			editing:   true,
			credentials: credentialState{
				credential: "secret",
				found:      true,
			},
			wantCredential: "secret",
			wantFound:      true,
			wantGetIDs:     []string{"profile-1"},
		},
		{
			name:      "未登録の資格情報を返す",
			profileID: "profile-1",
			editing:   true,
			wantGetIDs: []string{
				"profile-1",
			},
		},
		{
			name:      "資格情報取得失敗を利用不可エラーとして返す",
			profileID: "profile-1",
			editing:   true,
			credentials: credentialState{
				getErr: repositoryErr,
			},
			wantErrorCode: apperr.CodeCredentialUnavailable,
			wantGetIDs: []string{
				"profile-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials := tt.credentials
			repository := &appRepositoryStub{
				credentials: &credentials,
			}

			credential, found, err := NewAppUseCase(repository).previousCredential(tt.profileID, tt.editing)

			if got := credential; got != tt.wantCredential {
				t.Errorf("previousCredential() credential = %q, want %q", got, tt.wantCredential)
			}
			if got := found; got != tt.wantFound {
				t.Errorf("previousCredential() found = %v, want %v", got, tt.wantFound)
			}
			if gotCode := saveErrorCode(err); gotCode != tt.wantErrorCode {
				t.Errorf("previousCredential() error code = %q, want %q", gotCode, tt.wantErrorCode)
			}
			if !reflect.DeepEqual(credentials.getIDs, tt.wantGetIDs) {
				t.Errorf("GetCredential() profile IDs = %#v, want %#v", credentials.getIDs, tt.wantGetIDs)
			}
		})
	}
}

// プロファイル置換検証
func TestReplaceProfile(t *testing.T) {
	first := newSaveTestProfile(t, "profile-1")
	second := newSaveTestProfile(t, "profile-2")
	replacement := domain.Profile{
		ID:       "profile-1",
		Name:     "Updated DB",
		DBType:   domain.DBTypePostgres,
		Host:     "db.example.com",
		Port:     5433,
		Database: "updated",
		Schema:   "public",
		User:     "admin",
	}
	tests := []struct {
		name     string
		profiles []domain.Profile
		profile  domain.Profile
		editing  bool
		want     []domain.Profile
	}{
		{
			name:     "新規プロファイルを末尾に追加する",
			profiles: []domain.Profile{first},
			profile:  second,
			want: []domain.Profile{
				first,
				second,
			},
		},
		{
			name:     "既存プロファイルを置き換える",
			profiles: []domain.Profile{first, second},
			profile:  replacement,
			editing:  true,
			want: []domain.Profile{
				replacement,
				second,
			},
		},
		{
			name:     "存在しないプロファイルを編集しても変更しない",
			profiles: []domain.Profile{first},
			profile:  second,
			editing:  true,
			want: []domain.Profile{
				first,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := append([]domain.Profile(nil), tt.profiles...)
			got := replaceProfile(tt.profiles, tt.profile, tt.editing)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("replaceProfile() = %#v, want %#v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.profiles, original) {
				t.Errorf("replaceProfile() input = %#v, want %#v", tt.profiles, original)
			}
		})
	}
}

// 保存用テストプロファイル生成
func newSaveTestProfile(t *testing.T, id string) domain.Profile {
	t.Helper()

	profile, err := domain.NewProfile(id, "Local DB", domain.DBTypePostgres, "localhost", 5432, "app", "public", "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	return profile
}

// 保存用テスト下書き生成
func newSaveTestDraft(t *testing.T, id, password string) domain.ProfileDraft {
	t.Helper()

	draft, err := domain.NewProfileDraft(id, "New DB", domain.DBTypePostgres, "localhost", 5432, "app", "public", "user", password)
	if err != nil {
		t.Fatalf("NewProfileDraft() error = %v", err)
	}

	return draft
}

// 保存エラーコード取得
func saveErrorCode(err error) apperr.Code {
	if err == nil {
		return ""
	}

	if appErr := apperr.As(err); appErr != nil {
		return appErr.Code
	}

	return ""
}

// 文字列スライス比較
func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}

	return true
}
