package wails

import (
	"testing"

	"github.com/yukihito-jokyu/DB-checker/internal/config"
)

// アプリ状態返却検証
func TestAppHandlerGetStatus(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testing.T) *AppHandler
		wantData StatusResponse
	}{
		{
			name: "アプリケーション状態を返す",
			setup: func(t *testing.T) *AppHandler {
				return newTestAppHandler(t, config.NewStore(t.TempDir()), &connectionProfileRepositoryStub{})
			},
			wantData: StatusResponse{
				Name:    "DB-checker",
				Ready:   true,
				Version: "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setup(t)

			got := handler.GetStatus()

			if got.Data == nil {
				t.Fatal("GetStatus() Data = nil, want non-nil")
			}

			if got.Error != nil {
				t.Errorf("GetStatus() Error = %#v, want nil", got.Error)
			}

			if *got.Data != tt.wantData {
				t.Errorf("GetStatus() Data = %#v, want %#v", *got.Data, tt.wantData)
			}
		})
	}
}
