package wails

import (
	"github.com/yukihito-jokyu/DB-checker/internal/domain"
	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

type StatusResponse struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
}

type ConfigResponse struct {
	Version                   int               `json:"version"`
	ConnectionProfiles        []ProfileResponse `json:"connectionProfiles"`
	ActiveConnectionProfileID *string           `json:"activeConnectionProfileId"`
	FlowStates                map[string]any    `json:"flowStates"`
}

type ProfileResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DBType   string `json:"dbType"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
	User     string `json:"user"`
}

type ProfileCheckResponse struct {
	Valid        bool `json:"valid"`
	ProfileCount int  `json:"profileCount"`
}

type ConnectionProfilesResponse struct {
	Profiles                  []ProfileResponse `json:"profiles"`
	ActiveConnectionProfileID *string           `json:"activeConnectionProfileId"`
}

type Response[T any] struct {
	Data  *T             `json:"data"`
	Error *ErrorResponse `json:"error"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// プロファイルレスポンス変換
func toProfileResponses(profiles []domain.Profile) []ProfileResponse {
	responses := make([]ProfileResponse, 0, len(profiles))
	for _, profile := range profiles {
		responses = append(responses, ProfileResponse{
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

	return responses
}

// 成功レスポンス生成
func OK[T any](data T) Response[T] {
	return Response[T]{
		Data:  &data,
		Error: nil,
	}
}

// 失敗レスポンス生成
func Fail[T any](err error) Response[T] {
	return Response[T]{
		Data:  nil,
		Error: ToErrorResponse(err),
	}
}

// エラーレスポンス変換
func ToErrorResponse(err error) *ErrorResponse {
	if appErr := apperr.As(err); appErr != nil {
		return &ErrorResponse{
			Code:    string(appErr.Code),
			Message: string(appErr.Message),
		}
	}

	// 未分類エラーの詳細は frontend に漏らさず、共通の想定外エラーへ正規化する。
	appErr := apperr.NewUnexpected(err)

	return &ErrorResponse{
		Code:    string(appErr.Code),
		Message: string(appErr.Message),
	}
}
