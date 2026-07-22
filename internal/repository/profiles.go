package repository

import (
	"errors"

	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 接続プロファイル読込
func (r *AppRepository) LoadProfiles() ([]domain.Profile, *string, error) {
	result, err := r.store.Load()
	if err != nil {
		return nil, nil, err
	}

	profiles := make([]domain.Profile, 0, len(result.Config.ConnectionProfiles))
	for _, stored := range result.Config.ConnectionProfiles {
		profile, err := domain.NewProfile(stored.ID, stored.Name, domain.DBType(stored.DBType), stored.Host, stored.Port, stored.Database, stored.Schema, stored.User)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidProfile) {
				return nil, nil, apperr.Wrap(apperr.CodeConfigBroken, err)
			}

			// 単体テスト到達不可: domain.NewProfile は ErrInvalidProfile 以外を返さないため。
			return nil, nil, err
		}
		profiles = append(profiles, profile)
	}

	return profiles, result.Config.ActiveConnectionProfileID, nil
}
