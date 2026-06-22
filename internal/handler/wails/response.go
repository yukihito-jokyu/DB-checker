package wails

import apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"

type Response[T any] struct {
	Data  *T             `json:"data"`
	Error *ErrorResponse `json:"error"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OK は成功時の Wails 共通レスポンスを作成する。
func OK[T any](data T) Response[T] {
	return Response[T]{
		Data:  &data,
		Error: nil,
	}
}

// Fail は失敗時の Wails 共通レスポンスを作成する。
func Fail[T any](err error) Response[T] {
	return Response[T]{
		Data:  nil,
		Error: ToErrorResponse(err),
	}
}

// ToErrorResponse は内部エラーを frontend 向けのエラー DTO に変換する。
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
