package usecase

import (
	"context"
	"errors"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 接続プロファイルリポジトリ
type ConnectionProfileRepository interface {
	LoadProfiles() ([]domain.Profile, *string, error)
}

// 資格情報リポジトリ
type CredentialRepository interface {
	GetCredential(string) (credential string, found bool, err error)
}

// データベース接続リポジトリ
type DatabaseConnectionRepository interface {
	CheckConnection(context.Context, domain.Profile, string) error
}

// 接続プロファイル読込
func (u *AppUseCase) LoadProfiles() ([]domain.Profile, error) {
	profiles, activeID, err := u.profiles.LoadProfiles()
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateActiveProfile(profiles, activeID); err != nil {
		if errors.Is(err, domain.ErrInvalidActiveProfile) {
			return nil, apperr.Wrap(apperr.CodeConfigBroken, err)
		}

		// 単体テスト到達不可: domain.ValidateActiveProfile は ErrInvalidActiveProfile 以外を返さないため。
		return nil, err
	}

	return profiles, nil
}
