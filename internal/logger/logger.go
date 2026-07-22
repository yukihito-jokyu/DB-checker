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

// 標準エラーロガー生成
func New(level slog.Leveler) Logger {
	return NewWithWriter(os.Stderr, level)
}

// 出力先指定ロガー生成
func NewWithWriter(w io.Writer, level slog.Leveler) Logger {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})

	return &slogLogger{
		logger: slog.New(handler),
	}
}

// Debugログ出力
func (l *slogLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelDebug, msg, attrs...)
}

// Infoログ出力
func (l *slogLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelInfo, msg, attrs...)
}

// Warnログ出力
func (l *slogLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelWarn, msg, attrs...)
}

// Errorログ出力
func (l *slogLogger) Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		if cause := errors.Unwrap(err); cause != nil {
			attrs = append(attrs, slog.String("cause", cause.Error()))
		}
	}
	l.log(ctx, slog.LevelError, msg, attrs...)
}

// 構造化ログ出力
func (l *slogLogger) log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	// 単体テスト到達不可: Context を nil で渡さない規約のため。
	if ctx == nil {
		ctx = context.Background()
	}
	l.logger.LogAttrs(ctx, level, msg, attrs...)
}
