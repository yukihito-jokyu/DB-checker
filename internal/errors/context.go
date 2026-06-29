package apperr

import (
	"context"
	stderrors "errors"
)

// FromContextError は context 由来のエラーをアプリケーションエラーへ変換する。
func FromContextError(err error) *Error {
	if stderrors.Is(err, context.Canceled) {
		return Wrap(CodeOperationCanceled, err)
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return Wrap(CodeOperationTimeout, err)
	}
	return nil
}
