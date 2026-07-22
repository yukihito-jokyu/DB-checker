package apperr

import stderrors "errors"

type Error struct {
	Code    Code
	Message message
	Err     error
}

// アプリケーションエラー生成
func New(code Code) *Error {
	return &Error{
		Code:    code,
		Message: defaultMessages[code],
	}
}

// 原因付きエラー生成
func Wrap(code Code, err error) *Error {
	return &Error{
		Code:    code,
		Message: defaultMessages[code],
		Err:     err,
	}
}

// 想定外エラー生成
func NewUnexpected(err error) *Error {
	return Wrap(CodeUnexpected, err)
}

// ユーザー向けメッセージ返却
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	return string(e.Message)
}

// 原因エラー返却
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// アプリケーションエラー抽出
func As(err error) *Error {
	var appErr *Error
	if stderrors.As(err, &appErr) {
		return appErr
	}

	return nil
}

// エラーコード一致判定
func IsCode(err error, code Code) bool {
	appErr := As(err)

	return appErr != nil && appErr.Code == code
}
