package repository

import (
	"errors"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
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

// 接続プロファイル保存
func (r *AppRepository) SaveProfiles(profiles []domain.Profile, activeID *string) error {
	result, err := r.store.Load()
	if err != nil {
		return err
	}

	storedProfiles := make([]config.ConnectionProfile, 0, len(profiles))
	for _, profile := range profiles {
		storedProfiles = append(storedProfiles, config.ConnectionProfile{
			ID:       profile.ID,
			Name:     profile.Name,
			DBType:   string(profile.DBType),
			Host:     profile.Host,
			Port:     profile.Port,
			Database: profile.Database,
			Schema:   profile.Schema,
			User:     profile.User,
		})
	}

	result.Config.ConnectionProfiles = storedProfiles
	result.Config.ActiveConnectionProfileID = activeID

	return r.store.Save(result.Config)
}
