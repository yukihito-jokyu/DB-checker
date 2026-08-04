package repository

import (
	"encoding/json"
	"errors"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type storedFlowState struct {
	Version     int                              `json:"version"`
	TableStates *map[string]storedTableFlowState `json:"tableStates"`
}

type storedTableFlowState struct {
	X        *float64 `json:"x"`
	Y        *float64 `json:"y"`
	Expanded *bool    `json:"expanded"`
}

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

// プロファイル別フロー状態読込
func (r *AppRepository) LoadFlowState(profileID string) (domain.FlowState, error) {
	result, err := r.store.Load()
	if err != nil {
		return domain.FlowState{}, err
	}

	raw, found := result.Config.FlowStates[profileID]
	if !found {
		return domain.EmptyFlowState(), nil
	}

	return decodeFlowState(raw), nil
}

// フロー状態復号
func decodeFlowState(raw json.RawMessage) domain.FlowState {
	var stored storedFlowState
	if err := json.Unmarshal(raw, &stored); err != nil {
		return domain.EmptyFlowState()
	}
	if stored.TableStates == nil {
		return domain.EmptyFlowState()
	}

	tableStates := make(map[string]domain.TableFlowState, len(*stored.TableStates))
	for tableName, tableState := range *stored.TableStates {
		if tableState.X == nil || tableState.Y == nil || tableState.Expanded == nil {
			return domain.EmptyFlowState()
		}

		tableStates[tableName] = domain.TableFlowState{
			X:        *tableState.X,
			Y:        *tableState.Y,
			Expanded: *tableState.Expanded,
		}
	}

	state := domain.FlowState{
		Version:     stored.Version,
		TableStates: tableStates,
	}
	if err := state.Validate(); err != nil {
		return domain.EmptyFlowState()
	}

	return state
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
