package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Config 表示日志记录相关的配置。
type Config struct {
	Level  string
	Pretty bool
}

var (
	mu      sync.RWMutex
	current *slog.Logger
)

// New 根据配置构造一个 slog.Logger 实例。
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

// Init 使用给定配置创建日志记录器并设置为全局实例。
func Init(cfg Config) *slog.Logger {
	log := New(cfg)
	SetDefault(log)
	return log
}

// SetDefault 将日志记录器设为全局默认值，并同步更新 slog 的默认日志器。
func SetDefault(log *slog.Logger) {
	if log == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	current = log
	slog.SetDefault(log)
}

// FromContext 返回上下文中保存的日志记录器，没有时回退到全局默认值。
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Default()
	}

	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return log
	}

	return Default()
}

// WithContext 将日志记录器写入上下文，便于在处理链中提取。
func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// Default 返回全局日志记录器；若未设置，则返回 slog.Default()。
func Default() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if current != nil {
		return current
	}
	return slog.Default()
}

type ctxKey struct{}
