package usecase

// アプリケーションユースケース
type AppUseCase struct {
	profiles    ConnectionProfileRepository
	credentials CredentialRepository
}

// アプリケーションユースケース生成
func NewAppUseCase(profiles ConnectionProfileRepository, credentials CredentialRepository) *AppUseCase {
	return &AppUseCase{
		profiles:    profiles,
		credentials: credentials,
	}
}
