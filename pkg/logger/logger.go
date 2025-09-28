package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Config holds logging configuration.
type Config struct {
	Level  string
	Pretty bool
}

var (
	mu      sync.RWMutex
	current *slog.Logger
)

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

// Init constructs a logger from cfg and stores it as the global instance.
func Init(cfg Config) *slog.Logger {
	log := New(cfg)
	SetDefault(log)
	return log
}

// SetDefault records log as the global logger and updates slog's default logger.
func SetDefault(log *slog.Logger) {
	if log == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	current = log
	slog.SetDefault(log)
}

// FromContext returns a logger stored in the context, or the default logger if missing.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Default()
	}

	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return log
	}

	return Default()
}

// WithContext returns a context that stores the logger for retrieval in handlers.
func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// Default returns the global logger if configured, otherwise slog.Default().
func Default() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if current != nil {
		return current
	}
	return slog.Default()
}

type ctxKey struct{}
