package apperr

import (
	stderrors "errors"
	"fmt"
	"testing"
)

// アプリケーションエラー生成検証
func TestErrorCreation(t *testing.T) {
	driverFailed := stderrors.New("driver failed")
	rawError := stderrors.New("raw error")

	tests := []struct {
		name      string
		newError  func() *Error
		wantCode  Code
		wantCause error
		wantText  string
	}{
		{
			name: "原因なし",
			newError: func() *Error {
				return New(CodeOperationTimeout)
			},
			wantCode: CodeOperationTimeout,
			wantText: "処理がタイムアウトしました",
		},
		{
			name: "原因付き",
			newError: func() *Error {
				return Wrap(CodeDBConnectFailed, driverFailed)
			},
			wantCode:  CodeDBConnectFailed,
			wantCause: driverFailed,
			wantText:  "DB 接続に失敗しました",
		},
		{
			name: "想定外エラー",
			newError: func() *Error {
				return NewUnexpected(rawError)
			},
			wantCode:  CodeUnexpected,
			wantCause: rawError,
			wantText:  "予期しないエラーが発生しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.newError()
			if err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", err.Code, tt.wantCode)
			}
			if got := string(err.Message); got != tt.wantText {
				t.Errorf("Message = %q, want %q", got, tt.wantText)
			}
			if tt.wantCause == nil {
				if err.Err != nil {
					t.Errorf("Err = %v, want nil", err.Err)
				}

				return
			}
			if !stderrors.Is(err.Err, tt.wantCause) {
				t.Errorf("Err = %v, want %v", err.Err, tt.wantCause)
			}
		})
	}
}

// ユーザー向けメッセージ検証
func TestError_Error(t *testing.T) {
	cause := stderrors.New("driver failed")

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "原因なし",
			err:  New(CodeConfigBroken),
			want: "設定ファイルが壊れています",
		},
		{
			name: "原因付き",
			err:  Wrap(CodeDBConnectFailed, cause),
			want: "DB 接続に失敗しました",
		},
		{
			name: "レシーバーなし",
			err:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// 原因エラー返却検証
func TestError_Unwrap(t *testing.T) {
	cause := stderrors.New("driver failed")

	tests := []struct {
		name string
		err  *Error
		want error
	}{
		{
			name: "原因付き",
			err:  Wrap(CodeDBConnectFailed, cause),
			want: cause,
		},
		{
			name: "原因なし",
			err:  New(CodeConfigBroken),
			want: nil,
		},
		{
			name: "レシーバーなし",
			err:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Unwrap()
			if tt.want == nil {
				if got != nil {
					t.Errorf("Unwrap() = %v, want nil", got)
				}

				return
			}
			if !stderrors.Is(got, tt.want) {
				t.Errorf("Unwrap() = %v, want %v", got, tt.want)
			}
			if !stderrors.Is(tt.err, tt.want) {
				t.Error("wrapped cause should be discoverable with errors.Is")
			}
		})
	}
}

// アプリケーションエラー抽出検証
func TestAs(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantFound bool
		wantCode  Code
	}{
		{
			name:      "アプリケーションエラーを取得する",
			err:       New(CodeConfigBroken),
			wantFound: true,
			wantCode:  CodeConfigBroken,
		},
		{
			name:      "ラップされたエラーから取得する",
			err:       fmt.Errorf("context: %w", New(CodeOperationTimeout)),
			wantFound: true,
			wantCode:  CodeOperationTimeout,
		},
		{
			name:      "結合されたエラーから取得する",
			err:       stderrors.Join(stderrors.New("context"), New(CodeDataLoadFailed)),
			wantFound: true,
			wantCode:  CodeDataLoadFailed,
		},
		{
			name:      "通常のエラーではnilを返す",
			err:       stderrors.New("raw error"),
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := As(tt.err)
			if gotFound := appErr != nil; gotFound != tt.wantFound {
				t.Fatalf("found = %v, want %v", gotFound, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if appErr.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", appErr.Code, tt.wantCode)
			}
		})
	}
}

// エラーコード一致検証
func TestIsCode(t *testing.T) {
	err := Wrap(CodeSchemaLoadFailed, stderrors.New("query failed"))

	tests := []struct {
		name string
		err  error
		code Code
		want bool
	}{
		{
			name: "同じコードに一致する",
			err:  err,
			code: CodeSchemaLoadFailed,
			want: true,
		},
		{
			name: "異なるコードに一致しない",
			err:  err,
			code: CodeDataLoadFailed,
			want: false,
		},
		{
			name: "通常のエラーに一致しない",
			err:  stderrors.New("raw error"),
			code: CodeUnexpected,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCode(tt.err, tt.code); got != tt.want {
				t.Errorf("IsCode() = %v, want %v", got, tt.want)
			}
		})
	}
}
