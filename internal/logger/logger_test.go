package logger

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// 構造化ログ出力検証
func TestSlogLoggerOutput(t *testing.T) {
	driverFailed := stderrors.New("driver failed")
	requestFailed := stderrors.New("request failed")

	tests := []struct {
		name          string
		write         func(Logger)
		wantContained []string
		wantAbsent    []string
	}{
		{
			name: "情報ログ",
			write: func(log Logger) {
				log.Info(context.Background(), "status requested", slog.String("operation", "status"))
			},
			wantContained: []string{
				"level=INFO",
				`msg="status requested"`,
				"operation=status",
			},
		},
		{
			name: "情報レベルでデバッグログを除外する",
			write: func(log Logger) {
				log.Debug(context.Background(), "debug details")
			},
		},
		{
			name: "警告ログ",
			write: func(log Logger) {
				log.Warn(context.Background(), "retrying", slog.String("operation", "connect"))
			},
			wantContained: []string{
				"level=WARN",
				"msg=retrying",
				"operation=connect",
			},
		},
		{
			name: "エラーログ",
			write: func(log Logger) {
				log.Error(context.Background(), "db connect failed", stderrors.Join(driverFailed), slog.String("operation", "connect"))
			},
			wantContained: []string{
				"level=ERROR",
				`msg="db connect failed"`,
				"operation=connect",
				`error="driver failed"`,
			},
		},
		{
			name: "原因付きエラーログ",
			write: func(log Logger) {
				log.Error(context.Background(), "db connect failed", fmt.Errorf("connect: %w", requestFailed))
			},
			wantContained: []string{
				`error="connect: request failed"`,
				`cause="request failed"`,
			},
		},
		{
			name: "エラーなしのエラーログ",
			write: func(log Logger) {
				log.Error(context.Background(), "unexpected state", nil, slog.String("operation", "status"))
			},
			wantContained: []string{
				"level=ERROR",
				`msg="unexpected state"`,
				"operation=status",
			},
			wantAbsent: []string{
				"error=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			tt.write(NewWithWriter(&buffer, slog.LevelInfo))

			output := buffer.String()
			for _, want := range tt.wantContained {
				assertContains(t, output, want)
			}
			for _, unwanted := range tt.wantAbsent {
				if strings.Contains(output, unwanted) {
					t.Errorf("output = %q, want not to contain %q", output, unwanted)
				}
			}
		})
	}
}

// 標準エラーロガー生成検証
func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		level     slog.Level
		wantFound bool
	}{
		{
			name:      "情報レベルのロガー",
			level:     slog.LevelInfo,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFound := New(tt.level) != nil; gotFound != tt.wantFound {
				t.Errorf("New() logger found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

// 文字列包含検証
func assertContains(t *testing.T, output string, want string) {
	t.Helper()

	if !strings.Contains(output, want) {
		t.Errorf("output = %q, want to contain %q", output, want)
	}
}
