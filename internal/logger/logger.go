package logger

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
)

// Logger はアプリケーション内で利用する構造化 logger の最小 interface。
type Logger interface {
	Debug(ctx context.Context, msg string, attrs ...slog.Attr)
	Info(ctx context.Context, msg string, attrs ...slog.Attr)
	Warn(ctx context.Context, msg string, attrs ...slog.Attr)
	Error(ctx context.Context, msg string, err error, attrs ...slog.Attr)
}

type slogLogger struct {
	logger *slog.Logger
}

// New は標準エラー出力へ text 形式で出力する logger を作成する。
func New(level slog.Leveler) Logger {
	return NewWithWriter(os.Stderr, level)
}

// NewWithWriter は指定した出力先へ text 形式で出力する logger を作成する。
func NewWithWriter(w io.Writer, level slog.Leveler) Logger {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return &slogLogger{
		logger: slog.New(handler),
	}
}

func (l *slogLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelDebug, msg, attrs...)
}

func (l *slogLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelInfo, msg, attrs...)
}

func (l *slogLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelWarn, msg, attrs...)
}

func (l *slogLogger) Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		if cause := errors.Unwrap(err); cause != nil {
			attrs = append(attrs, slog.String("cause", cause.Error()))
		}
	}
	l.log(ctx, slog.LevelError, msg, attrs...)
}

func (l *slogLogger) log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	l.logger.LogAttrs(ctx, level, msg, attrs...)
}
