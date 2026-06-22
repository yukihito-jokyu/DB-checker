package apperr

import (
	stderrors "errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeOperationTimeout)

	if err.Code != CodeOperationTimeout {
		t.Errorf("Code = %q, want %q", err.Code, CodeOperationTimeout)
	}
	if got := string(err.Message); got != "処理がタイムアウトしました" {
		t.Errorf("Message = %q, want %q", got, "処理がタイムアウトしました")
	}
	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
	}
}

func TestWrap(t *testing.T) {
	cause := stderrors.New("driver failed")
	err := Wrap(CodeDBConnectFailed, cause)

	if err.Code != CodeDBConnectFailed {
		t.Errorf("Code = %q, want %q", err.Code, CodeDBConnectFailed)
	}
	if got := string(err.Message); got != "DB 接続に失敗しました" {
		t.Errorf("Message = %q, want %q", got, "DB 接続に失敗しました")
	}
	if err.Err != cause {
		t.Errorf("Err = %v, want %v", err.Err, cause)
	}
}

func TestNewUnexpected(t *testing.T) {
	cause := stderrors.New("raw error")
	err := NewUnexpected(cause)

	if err.Code != CodeUnexpected {
		t.Errorf("Code = %q, want %q", err.Code, CodeUnexpected)
	}
	if err.Err != cause {
		t.Errorf("Err = %v, want %v", err.Err, cause)
	}
	if got := string(err.Message); got != "予期しないエラーが発生しました" {
		t.Errorf("Message = %q, want %q", got, "予期しないエラーが発生しました")
	}
}

func TestError_Error(t *testing.T) {
	cause := stderrors.New("driver failed")

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "without cause",
			err:  New(CodeConfigBroken),
			want: "設定ファイルが壊れています",
		},
		{
			name: "with cause",
			err:  Wrap(CodeDBConnectFailed, cause),
			want: "DB 接続に失敗しました: driver failed",
		},
		{
			name: "nil receiver",
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

func TestError_Unwrap(t *testing.T) {
	cause := stderrors.New("driver failed")

	tests := []struct {
		name string
		err  *Error
		want error
	}{
		{
			name: "with cause",
			err:  Wrap(CodeDBConnectFailed, cause),
			want: cause,
		},
		{
			name: "without cause",
			err:  New(CodeConfigBroken),
			want: nil,
		},
		{
			name: "nil receiver",
			err:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Unwrap(); got != tt.want {
				t.Errorf("Unwrap() = %v, want %v", got, tt.want)
			}
			if tt.want != nil && !stderrors.Is(tt.err, tt.want) {
				t.Error("wrapped cause should be discoverable with errors.Is")
			}
		})
	}
}

func TestAs(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantFound bool
		wantCode  Code
	}{
		{
			name:      "finds app error",
			err:       New(CodeConfigBroken),
			wantFound: true,
			wantCode:  CodeConfigBroken,
		},
		{
			name:      "finds app error in wrapped chain",
			err:       fmt.Errorf("context: %w", New(CodeOperationTimeout)),
			wantFound: true,
			wantCode:  CodeOperationTimeout,
		},
		{
			name:      "finds app error in joined chain",
			err:       stderrors.Join(stderrors.New("context"), New(CodeDataLoadFailed)),
			wantFound: true,
			wantCode:  CodeDataLoadFailed,
		},
		{
			name:      "returns nil for raw error",
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

func TestIsCode(t *testing.T) {
	err := Wrap(CodeSchemaLoadFailed, stderrors.New("query failed"))

	tests := []struct {
		name string
		err  error
		code Code
		want bool
	}{
		{
			name: "matches same code",
			err:  err,
			code: CodeSchemaLoadFailed,
			want: true,
		},
		{
			name: "does not match different code",
			err:  err,
			code: CodeDataLoadFailed,
			want: false,
		},
		{
			name: "does not match raw error",
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
