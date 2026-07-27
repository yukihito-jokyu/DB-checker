package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 接続プロファイル保存
func (u *AppUseCase) SaveConnectionProfile(ctx context.Context, draft domain.ProfileDraft) ([]domain.Profile, *string, error) {
	if err := draft.Validate(); err != nil {
		return nil, nil, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	profiles, activeID, err := u.LoadProfiles()
	if err != nil {
		return nil, nil, err
	}

	profileID := draft.ID
	editing := profileID != ""
	if !editing {
		profileID = uuid.NewString()
	} else if !containsProfile(profiles, profileID) {
		return nil, nil, apperr.New(apperr.CodeProfileNotFound)
	}

	profile, err := draft.ToProfile(profileID)
	if err != nil {
		// 単体テスト到達不可: draft.Validate 成功後は、非空IDと同じ検証条件を使うため。
		return nil, nil, apperr.Wrap(apperr.CodeValidationFailed, err)
	}

	password, hasPassword, err := u.passwordForSave(draft, editing)
	if err != nil {
		return nil, nil, err
	}

	if err := u.repository.CheckConnection(ctx, profile, password); err != nil {
		return nil, nil, apperr.Wrap(apperr.CodeConnectionFailed, err)
	}

	nextProfiles := replaceProfile(profiles, profile, editing)
	if !hasPassword {
		if err := u.repository.SaveProfiles(nextProfiles, activeID); err != nil {
			return nil, nil, apperr.Wrap(apperr.CodeConfigSaveFailed, err)
		}

		return nextProfiles, activeID, nil
	}

	previousCredential, previousCredentialFound, err := u.previousCredential(profileID, editing)
	if err != nil {
		return nil, nil, err
	}
	if err := u.repository.SetCredential(profileID, password); err != nil {
		return nil, nil, apperr.Wrap(apperr.CodeCredentialSaveFailed, err)
	}
	if err := u.repository.SaveProfiles(nextProfiles, activeID); err != nil {
		if recoveryErr := u.restoreCredential(profileID, previousCredential, previousCredentialFound); recoveryErr != nil {
			return nil, nil, apperr.Wrap(apperr.CodeConsistencyRecoveryFailed, recoveryErr)
		}

		return nil, nil, apperr.Wrap(apperr.CodeConfigSaveFailed, err)
	}

	return nextProfiles, activeID, nil
}

// アクティブ接続プロファイル切替
func (u *AppUseCase) ActivateConnectionProfile(ctx context.Context, profileID string) ([]domain.Profile, *string, error) {
	profiles, activeID, err := u.LoadProfiles()
	if err != nil {
		return nil, nil, err
	}

	profile, found := findProfile(profiles, profileID)
	if !found {
		return nil, nil, apperr.New(apperr.CodeProfileNotFound)
	}
	if activeID != nil && *activeID == profileID {
		return profiles, activeID, nil
	}

	password, found, err := u.repository.GetCredential(profileID)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}
	if !found {
		password = ""
	}

	if err := u.repository.CheckConnection(ctx, profile, password); err != nil {
		return nil, nil, apperr.Wrap(apperr.CodeConnectionFailed, err)
	}

	nextActiveID := profileID
	if err := u.repository.SaveProfiles(profiles, &nextActiveID); err != nil {
		return nil, nil, apperr.Wrap(apperr.CodeConfigSaveFailed, err)
	}

	return profiles, &nextActiveID, nil
}

// 接続プロファイル削除
func (u *AppUseCase) DeleteConnectionProfile(profileID string) ([]domain.Profile, *string, error) {
	profiles, activeID, err := u.LoadProfiles()
	if err != nil {
		return nil, nil, err
	}
	if !containsProfile(profiles, profileID) {
		return nil, nil, apperr.New(apperr.CodeProfileNotFound)
	}

	nextProfiles := removeProfile(profiles, profileID)
	nextActiveID := activeID
	if activeID != nil && *activeID == profileID {
		nextActiveID = nil
	}
	if err := u.repository.SaveProfiles(nextProfiles, nextActiveID); err != nil {
		return nil, nil, apperr.Wrap(apperr.CodeConfigSaveFailed, err)
	}

	if err := u.repository.DeleteCredential(profileID); err != nil {
		if recoveryErr := u.repository.SaveProfiles(profiles, activeID); recoveryErr != nil {
			return nil, nil, apperr.Wrap(apperr.CodeConsistencyRecoveryFailed, recoveryErr)
		}

		return nil, nil, apperr.Wrap(apperr.CodeCredentialDeleteFailed, err)
	}

	return nextProfiles, nextActiveID, nil
}

// 保存用パスワード取得
func (u *AppUseCase) passwordForSave(draft domain.ProfileDraft, editing bool) (string, bool, error) {
	if draft.Password != "" {
		return draft.Password, true, nil
	}
	if !editing {
		return "", false, nil
	}

	credential, found, err := u.repository.GetCredential(draft.ID)
	if err != nil || !found {
		if err == nil {
			err = errors.New("credential not found")
		}

		return "", false, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}

	return credential, false, nil
}

// 保存前資格情報取得
func (u *AppUseCase) previousCredential(profileID string, editing bool) (string, bool, error) {
	if !editing {
		return "", false, nil
	}

	credential, found, err := u.repository.GetCredential(profileID)
	if err != nil {
		return "", false, apperr.Wrap(apperr.CodeCredentialUnavailable, err)
	}

	return credential, found, nil
}

// 資格情報復旧
func (u *AppUseCase) restoreCredential(profileID, credential string, found bool) error {
	if found {
		return u.repository.SetCredential(profileID, credential)
	}

	return u.repository.DeleteCredential(profileID)
}

// プロファイル存在判定
func containsProfile(profiles []domain.Profile, profileID string) bool {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return true
		}
	}

	return false
}

// プロファイル検索
func findProfile(profiles []domain.Profile, profileID string) (domain.Profile, bool) {
	for _, profile := range profiles {
		if profile.ID == profileID {
			return profile, true
		}
	}

	return domain.Profile{}, false
}

// プロファイル置換
func replaceProfile(profiles []domain.Profile, profile domain.Profile, editing bool) []domain.Profile {
	next := append([]domain.Profile(nil), profiles...)
	if !editing {
		return append(next, profile)
	}

	for index, current := range next {
		if current.ID == profile.ID {
			next[index] = profile

			return next
		}
	}

	return next
}

// プロファイル除外
func removeProfile(profiles []domain.Profile, profileID string) []domain.Profile {
	next := make([]domain.Profile, 0, len(profiles)-1)
	for _, profile := range profiles {
		if profile.ID != profileID {
			next = append(next, profile)
		}
	}

	return next
}

// 接続プロファイル読込
func (u *AppUseCase) LoadProfiles() ([]domain.Profile, *string, error) {
	profiles, activeID, err := u.repository.LoadProfiles()
	if err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateActiveProfile(profiles, activeID); err != nil {
		if errors.Is(err, domain.ErrInvalidActiveProfile) {
			return nil, nil, apperr.Wrap(apperr.CodeConfigBroken, err)
		}

		// 単体テスト到達不可: domain.ValidateActiveProfile は ErrInvalidActiveProfile 以外を返さないため。
		return nil, nil, err
	}

	return profiles, activeID, nil
}
