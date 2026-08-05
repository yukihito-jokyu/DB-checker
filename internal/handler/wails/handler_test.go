package wails

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
)

type connectionProfileRepositoryStub struct {
	profiles        []domain.Profile
	activeID        *string
	err             error
	calls           int
	saveErr         error
	saveCalls       int
	savedProfiles   []domain.Profile
	savedActiveID   *string
	credential      string
	credentialFound bool
	credentialErr   error
	credentialIDs   []string
	deleteErr       error
	deleteIDs       []string
	connectionErr   error
	connectionCalls int
	password        string
	flowState       domain.FlowState
	flowStateErr    error
	savedFlowState  domain.FlowState
	savedFlowID     string
	saveFlowErr     error
}

// スキーマ取得再現
func (s *connectionProfileRepositoryStub) InspectSchema(context.Context, domain.Profile, string) (domain.Schema, error) {
	return domain.Schema{}, nil
}

// テーブル構造取得再現
func (*connectionProfileRepositoryStub) InspectTableStructure(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStructure, error) {
	return domain.TableStructure{}, nil
}

// テーブル統計取得再現
func (*connectionProfileRepositoryStub) InspectTableStatistics(context.Context, domain.Profile, string, domain.TableRef) (domain.TableStatistics, error) {
	return domain.TableStatistics{}, nil
}

// テーブル行一覧再現
func (*connectionProfileRepositoryStub) ListRows(context.Context, domain.Profile, string, domain.TableQuery) (domain.TableRows, error) {
	return domain.TableRows{}, nil
}

// テーブル行追加再現
func (*connectionProfileRepositoryStub) InsertRow(context.Context, domain.Profile, string, domain.TableRef, domain.InsertRow) (domain.AffectedRows, error) {
	return domain.AffectedRows{}, nil
}

// テーブルセル更新再現
func (*connectionProfileRepositoryStub) UpdateCell(context.Context, domain.Profile, string, domain.TableRef, domain.CellUpdate) (domain.AffectedRows, error) {
	return domain.AffectedRows{}, nil
}

// フロー状態読込再現
func (s *connectionProfileRepositoryStub) LoadFlowState(string) (domain.FlowState, error) {
	return s.flowState, s.flowStateErr
}

// フロー状態保存再現
func (s *connectionProfileRepositoryStub) SaveFlowState(profileID string, state domain.FlowState) error {
	s.savedFlowID = profileID
	s.savedFlowState = state

	return s.saveFlowErr
}

// 接続プロファイル読込再現
func (s *connectionProfileRepositoryStub) LoadProfiles() ([]domain.Profile, *string, error) {
	s.calls++

	return s.profiles, s.activeID, s.err
}

// 接続プロファイル保存再現
func (s *connectionProfileRepositoryStub) SaveProfiles(profiles []domain.Profile, activeID *string) error {
	s.saveCalls++
	s.savedProfiles = profiles
	s.savedActiveID = activeID

	return s.saveErr
}

// 資格情報取得再現
func (s *connectionProfileRepositoryStub) GetCredential(profileID string) (string, bool, error) {
	s.credentialIDs = append(s.credentialIDs, profileID)

	return s.credential, s.credentialFound, s.credentialErr
}

// 資格情報設定再現
func (s *connectionProfileRepositoryStub) SetCredential(string, string) error {
	return nil
}

// 資格情報削除再現
func (s *connectionProfileRepositoryStub) DeleteCredential(profileID string) error {
	s.deleteIDs = append(s.deleteIDs, profileID)

	return s.deleteErr
}

// データベース接続確認再現
func (s *connectionProfileRepositoryStub) CheckConnection(_ context.Context, _ domain.Profile, password string) error {
	s.connectionCalls++
	s.password = password

	return s.connectionErr
}

// テスト用ハンドラー生成
func newTestAppHandler(t *testing.T, store *config.Store, repository *connectionProfileRepositoryStub) *AppHandler {
	t.Helper()

	logger := applogger.NewWithWriter(io.Discard, slog.LevelDebug)
	appUseCase := usecase.NewAppUseCase(repository)

	return NewAppHandler(logger, store, appUseCase)
}

// テスト用プロファイル生成
func newTestProfile(t *testing.T, id string, dbType domain.DBType) domain.Profile {
	t.Helper()

	schema := ""
	if dbType == domain.DBTypePostgres {
		schema = "public"
	}

	profile, err := domain.NewProfile(id, "Local "+id, dbType, "localhost", 5432, "app", schema, "user")
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	return profile
}

// 文字列ポインタ生成
func stringPointer(value string) *string {
	return &value
}
