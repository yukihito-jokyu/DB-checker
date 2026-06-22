package wails

import (
	stderrors "errors"
	"testing"

	apperr "github.com/yukihito-jokyu/DB-checker/internal/errors"
)

func TestOK(t *testing.T) {
	tests := []struct {
		name string
		data StatusData
	}{
		{
			name: "wraps status data",
			data: StatusData{
				Name:    "DB-checker",
				Ready:   true,
				Version: "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := OK(tt.data)

			if response.Data == nil {
				t.Fatal("Data = nil, want status data")
			}
			if response.Error != nil {
				t.Fatalf("Error = %#v, want nil", response.Error)
			}
			if response.Data.Name != tt.data.Name {
				t.Errorf("Data.Name = %q, want %q", response.Data.Name, tt.data.Name)
			}
			if response.Data.Ready != tt.data.Ready {
				t.Errorf("Data.Ready = %v, want %v", response.Data.Ready, tt.data.Ready)
			}
			if response.Data.Version != tt.data.Version {
				t.Errorf("Data.Version = %q, want %q", response.Data.Version, tt.data.Version)
			}
		})
	}
}

func TestFail(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    string
		wantMessage string
	}{
		{
			name: "converts app error",
			err: apperr.Wrap(
				apperr.CodeConfigBroken,
				stderrors.New("invalid json"),
			),
			wantCode:    string(apperr.CodeConfigBroken),
			wantMessage: "設定ファイルが壊れています",
		},
		{
			name:        "converts raw error to unexpected",
			err:         stderrors.New("raw internal failure"),
			wantCode:    string(apperr.CodeUnexpected),
			wantMessage: "予期しないエラーが発生しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := Fail[StatusData](tt.err)

			if response.Data != nil {
				t.Fatalf("Data = %#v, want nil", response.Data)
			}
			if response.Error == nil {
				t.Fatal("Error = nil, want error response")
			}
			if response.Error.Code != tt.wantCode {
				t.Errorf("Error.Code = %q, want %q", response.Error.Code, tt.wantCode)
			}
			if response.Error.Message != tt.wantMessage {
				t.Errorf("Error.Message = %q, want %q", response.Error.Message, tt.wantMessage)
			}
		})
	}
}
