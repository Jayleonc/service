package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Config holds logging configuration.
type Config struct {
	Level  string
	Pretty bool
}

// New constructs a slog.Logger based on the provided config.
func New(cfg Config) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if cfg.Pretty {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// WithContext returns a context that stores the logger for retrieval in handlers.
func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// FromContext returns a logger stored in the context, or the default logger if missing.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return log
	}

	return slog.Default()
}

type ctxKey struct{}
