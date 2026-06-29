package apperr

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
)

func TestFromContextError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantFound bool
		wantCode  Code
		wantCause error
	}{
		{
			name:      "canceled",
			err:       context.Canceled,
			wantFound: true,
			wantCode:  CodeOperationCanceled,
			wantCause: context.Canceled,
		},
		{
			name:      "deadline exceeded",
			err:       context.DeadlineExceeded,
			wantFound: true,
			wantCode:  CodeOperationTimeout,
			wantCause: context.DeadlineExceeded,
		},
		{
			name:      "wrapped canceled",
			err:       fmt.Errorf("query stopped: %w", context.Canceled),
			wantFound: true,
			wantCode:  CodeOperationCanceled,
			wantCause: context.Canceled,
		},
		{
			name:      "raw error",
			err:       stderrors.New("driver failed"),
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := FromContextError(tt.err)
			if gotFound := appErr != nil; gotFound != tt.wantFound {
				t.Fatalf("found = %v, want %v", gotFound, tt.wantFound)
			}
			if !tt.wantFound {
				return
			}
			if appErr.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", appErr.Code, tt.wantCode)
			}
			if !stderrors.Is(appErr.Err, tt.wantCause) {
				t.Errorf("Err = %v, want %v", appErr.Err, tt.wantCause)
			}
		})
	}
}
