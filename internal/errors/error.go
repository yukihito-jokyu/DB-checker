package apperr

import stderrors "errors"

type Error struct {
	Code    Code
	Message message
	Err     error
}

// New は原因エラーを持たないアプリケーションエラーを作成する。
func New(code Code) *Error {
	return &Error{
		Code:    code,
		Message: defaultMessages[code],
	}
}

// Wrap は原因エラーを保持したアプリケーションエラーを作成する。
func Wrap(code Code, err error) *Error {
	return &Error{
		Code:    code,
		Message: defaultMessages[code],
		Err:     err,
	}
}

// NewUnexpected は未分類の内部エラーを想定外エラーとして包む。
func NewUnexpected(err error) *Error {
	return Wrap(CodeUnexpected, err)
}

// Error はユーザー向けメッセージだけを返す。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Message)
}

// Unwrap は errors.Is / errors.As で原因エラーを辿れるようにする。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// As は error chain からアプリケーションエラーを取り出す。
func As(err error) *Error {
	var appErr *Error
	if stderrors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// IsCode は error chain 内のアプリケーションエラーが指定コードか判定する。
func IsCode(err error, code Code) bool {
	appErr := As(err)
	return appErr != nil && appErr.Code == code
}
