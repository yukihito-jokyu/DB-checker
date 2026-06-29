package logger

import (
	"bytes"
	"context"
	stderrors "errors"
	"log/slog"
	"strings"
	"testing"
)

func TestSlogLoggerInfo(t *testing.T) {
	var buffer bytes.Buffer
	log := NewWithWriter(&buffer, slog.LevelInfo)

	log.Info(context.Background(), "status requested", slog.String("operation", "status"))

	output := buffer.String()
	assertContains(t, output, "level=INFO")
	assertContains(t, output, `msg="status requested"`)
	assertContains(t, output, "operation=status")
}

func TestSlogLoggerFiltersDebugByLevel(t *testing.T) {
	var buffer bytes.Buffer
	log := NewWithWriter(&buffer, slog.LevelInfo)

	log.Debug(context.Background(), "debug details")

	if got := buffer.String(); got != "" {
		t.Errorf("output = %q, want empty", got)
	}
}

func TestSlogLoggerError(t *testing.T) {
	var buffer bytes.Buffer
	log := NewWithWriter(&buffer, slog.LevelInfo)
	cause := stderrors.New("driver failed")
	err := stderrors.Join(cause)

	log.Error(context.Background(), "db connect failed", err, slog.String("operation", "connect"))

	output := buffer.String()
	assertContains(t, output, "level=ERROR")
	assertContains(t, output, `msg="db connect failed"`)
	assertContains(t, output, "operation=connect")
	assertContains(t, output, `error="driver failed"`)
}

func TestSlogLoggerErrorWithNilError(t *testing.T) {
	var buffer bytes.Buffer
	log := NewWithWriter(&buffer, slog.LevelInfo)

	log.Error(context.Background(), "unexpected state", nil, slog.String("operation", "status"))

	output := buffer.String()
	assertContains(t, output, "level=ERROR")
	assertContains(t, output, `msg="unexpected state"`)
	assertContains(t, output, "operation=status")
	if strings.Contains(output, "error=") {
		t.Errorf("output = %q, want no error attribute", output)
	}
}

func assertContains(t *testing.T, output string, want string) {
	t.Helper()

	if !strings.Contains(output, want) {
		t.Errorf("output = %q, want to contain %q", output, want)
	}
}
