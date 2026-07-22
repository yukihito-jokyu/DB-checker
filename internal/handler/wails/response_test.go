package wails

import (
	stderrors "errors"
	"testing"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

// 成功レスポンス検証
func TestOK(t *testing.T) {
	tests := []struct {
		name string
		data StatusResponse
	}{
		{
			name: "ステータスデータをラップする",
			data: StatusResponse{
				Name:    "DB-checker",
				Ready:   true,
				Version: "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OK(tt.data)

			if got.Data == nil {
				t.Fatal("OK() Data = nil, want non-nil")
			}

			if got.Error != nil {
				t.Errorf("OK() Error = %#v, want nil", got.Error)
			}

			if *got.Data != tt.data {
				t.Errorf("OK() Data = %#v, want %#v", *got.Data, tt.data)
			}
		})
	}
}

// 失敗レスポンス検証
func TestFail(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorResponse
	}{
		{
			name: "アプリケーションエラーを変換する",
			err: apperr.Wrap(
				apperr.CodeConfigBroken,
				stderrors.New("invalid json"),
			),
			want: ErrorResponse{
				Code:    string(apperr.CodeConfigBroken),
				Message: "設定ファイルが壊れています",
			},
		},
		{
			name: "通常のエラーを想定外エラーへ変換する",
			err:  stderrors.New("raw internal failure"),
			want: ErrorResponse{
				Code:    string(apperr.CodeUnexpected),
				Message: "予期しないエラーが発生しました",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Fail[StatusResponse](tt.err)

			if got.Data != nil {
				t.Errorf("Fail() Data = %#v, want nil", got.Data)
			}

			if got.Error == nil {
				t.Fatal("Fail() Error = nil, want non-nil")
			}

			if *got.Error != tt.want {
				t.Errorf("Fail() Error = %#v, want %#v", *got.Error, tt.want)
			}
		})
	}
}
