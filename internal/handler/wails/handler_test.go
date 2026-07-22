package wails

import (
	"io"
	"log/slog"
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
	"github.com/yukihito-jokyu/DB-checker/internal/usecase"
)

type connectionProfileRepositoryStub struct {
	profiles []domain.Profile
	activeID *string
	err      error
	calls    int
}

// 接続プロファイル読込再現
func (s *connectionProfileRepositoryStub) LoadProfiles() ([]domain.Profile, *string, error) {
	s.calls++

	return s.profiles, s.activeID, s.err
}

// テスト用ハンドラー生成
func newTestAppHandler(t *testing.T, store *config.Store, repository *connectionProfileRepositoryStub) *AppHandler {
	t.Helper()

	logger := applogger.NewWithWriter(io.Discard, slog.LevelDebug)
	appUseCase := usecase.NewAppUseCase(repository, nil)

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
