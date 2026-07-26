//go:build integration

package usecase

import (
	"context"
	"net"
	"net/url"
	"path"
	"strconv"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
	"github.com/yukihito-jokyu/DB-checker/internal/repository"
	"github.com/yukihito-jokyu/DB-checker/test/integration/db"
)

type integrationCredentialRepository struct {
	credentials map[string]string
}

type integrationAppRepository struct {
	*repository.AppRepository
	credentials *integrationCredentialRepository
}

// 結合テスト用資格情報取得
func (r *integrationAppRepository) GetCredential(profileID string) (string, bool, error) {
	return r.credentials.GetCredential(profileID)
}

// 結合テスト用資格情報保存
func (r *integrationAppRepository) SetCredential(profileID, credential string) error {
	return r.credentials.SetCredential(profileID, credential)
}

// 結合テスト用資格情報削除
func (r *integrationAppRepository) DeleteCredential(profileID string) error {
	return r.credentials.DeleteCredential(profileID)
}

// 結合テスト用資格情報取得
func (r *integrationCredentialRepository) GetCredential(profileID string) (string, bool, error) {
	credential, found := r.credentials[profileID]

	return credential, found, nil
}

// 結合テスト用資格情報保存
func (r *integrationCredentialRepository) SetCredential(profileID, credential string) error {
	r.credentials[profileID] = credential

	return nil
}

// 結合テスト用資格情報削除
func (r *integrationCredentialRepository) DeleteCredential(profileID string) error {
	delete(r.credentials, profileID)

	return nil
}

// 接続プロファイル保存結合検証
func TestAppUseCaseSaveConnectionProfileIntegration(t *testing.T) {
	targets, err := db.TargetsFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range targets {
		t.Run(string(target.Kind), func(t *testing.T) {
			validDraft := integrationProfileDraft(t, target)
			invalidDraft := validDraft
			invalidDraft.Password = "invalid-integration-password"
			tests := []struct {
				name                string
				draft               domain.ProfileDraft
				wantErrorCode       apperr.Code
				wantProfileCount    int
				wantCredentialCount int
			}{
				{
					name:                "接続成功時にプロファイルと資格情報を保存する",
					draft:               validDraft,
					wantProfileCount:    1,
					wantCredentialCount: 1,
				},
				{
					name:                "接続失敗時に保存状態を変更しない",
					draft:               invalidDraft,
					wantErrorCode:       apperr.CodeConnectionFailed,
					wantProfileCount:    0,
					wantCredentialCount: 0,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					store := config.NewStore(t.TempDir())
					if err := store.Initialize(); err != nil {
						t.Fatalf("Store.Initialize() error = %v", err)
					}
					credentialRepository := &integrationCredentialRepository{credentials: map[string]string{}}
					appRepository := &integrationAppRepository{
						AppRepository: repository.NewAppRepository(store),
						credentials:   credentialRepository,
					}
					useCase := NewAppUseCase(appRepository)

					savedProfiles, savedActiveID, err := useCase.SaveConnectionProfile(context.Background(), tt.draft)
					if gotCode := saveErrorCode(err); gotCode != tt.wantErrorCode {
						t.Errorf("SaveConnectionProfile() error code = %q, want %q", gotCode, tt.wantErrorCode)
					}
					if tt.wantErrorCode == "" && err != nil {
						t.Fatalf("SaveConnectionProfile() error = %v", err)
					}
					if tt.wantErrorCode != "" && err == nil {
						t.Fatal("SaveConnectionProfile() error = nil, want non-nil")
					}
					if got := len(savedProfiles); got != tt.wantProfileCount {
						t.Errorf("SaveConnectionProfile() profiles = %d, want %d", got, tt.wantProfileCount)
					}
					if savedActiveID != nil {
						t.Errorf("SaveConnectionProfile() active ID = %q, want nil", *savedActiveID)
					}

					loadedProfiles, loadedActiveID, err := useCase.LoadProfiles()
					if err != nil {
						t.Fatalf("LoadProfiles() error = %v", err)
					}
					if got := len(loadedProfiles); got != tt.wantProfileCount {
						t.Fatalf("LoadProfiles() profiles = %d, want %d", got, tt.wantProfileCount)
					}
					if tt.wantProfileCount == 1 {
						got := loadedProfiles[0]
						want := domain.Profile{
							ID:       got.ID,
							Name:     tt.draft.Name,
							DBType:   tt.draft.DBType,
							Host:     tt.draft.Host,
							Port:     tt.draft.Port,
							Database: tt.draft.Database,
							Schema:   tt.draft.Schema,
							User:     tt.draft.User,
						}
						if got != want {
							t.Errorf("LoadProfiles() profile = %#v, want %#v", got, want)
						}
					}
					if loadedActiveID != nil {
						t.Errorf("LoadProfiles() active ID = %q, want nil", *loadedActiveID)
					}
					if got := len(credentialRepository.credentials); got != tt.wantCredentialCount {
						t.Errorf("credentials = %d, want %d", got, tt.wantCredentialCount)
					}
				})
			}
		})
	}
}

// 結合テスト用プロファイル下書き生成
func integrationProfileDraft(t *testing.T, target db.Target) domain.ProfileDraft {
	t.Helper()

	switch target.Kind {
	case db.MySQL:
		return integrationMySQLProfileDraft(t, target.DSN)
	case db.Postgres:
		return integrationPostgresProfileDraft(t, target.DSN)
	default:
		t.Fatalf("unsupported database kind: %s", target.Kind)

		return domain.ProfileDraft{}
	}
}

// MySQL結合テスト用プロファイル下書き生成
func integrationMySQLProfileDraft(t *testing.T, dsn string) domain.ProfileDraft {
	t.Helper()

	connectionConfig, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN() error = %v", err)
	}
	host, port := integrationHostPort(t, connectionConfig.Addr)
	draft, err := domain.NewProfileDraft("", "Integration MySQL", domain.DBTypeMySQL, host, port, connectionConfig.DBName, "", connectionConfig.User, connectionConfig.Passwd)
	if err != nil {
		t.Fatalf("NewProfileDraft() error = %v", err)
	}

	return draft
}

// PostgreSQL結合テスト用プロファイル下書き生成
func integrationPostgresProfileDraft(t *testing.T, dsn string) domain.ProfileDraft {
	t.Helper()

	connectionURL, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	password, passwordSet := connectionURL.User.Password()
	if !passwordSet {
		t.Fatal("PostgreSQL DSN password is required")
	}
	host, port := integrationHostPort(t, connectionURL.Host)
	draft, err := domain.NewProfileDraft("", "Integration PostgreSQL", domain.DBTypePostgres, host, port, path.Base(connectionURL.Path), "public", connectionURL.User.Username(), password)
	if err != nil {
		t.Fatalf("NewProfileDraft() error = %v", err)
	}

	return draft
}

// 結合テスト用ホストポート取得
func integrationHostPort(t *testing.T, address string) (string, int) {
	t.Helper()

	host, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("net.SplitHostPort(%q) error = %v", address, err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatalf("strconv.Atoi(%q) error = %v", rawPort, err)
	}

	return host, port
}
