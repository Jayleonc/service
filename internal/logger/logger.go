package logger

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Config 描述日志初始化所需的选项。
type Config struct {
	Mode      string
	Level     *string
	Pretty    *bool
	Directory string
}

const (
	requestIDLength   = 12
	requestIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	logFilePrefix     = "service"
	retentionDays     = 7
)

var (
	globalMu    sync.RWMutex
	globalLog   *slog.Logger
	globalLevel slog.Level = slog.LevelInfo
)

// Init 根据配置初始化日志记录器，并将其设置为全局默认值。
func Init(cfg Config) (*slog.Logger, error) {
	level, pretty := resolveModeDefaults(cfg.Mode)

	if cfg.Level != nil {
		if parsed, ok := parseLevel(*cfg.Level); ok {
			level = parsed
		}
	}
	if cfg.Pretty != nil {
		pretty = *cfg.Pretty
	}

	handlerOpts := &slog.HandlerOptions{Level: level}

	writer := io.Writer(os.Stdout)
	if dir := strings.TrimSpace(cfg.Directory); dir != "" {
		rotating, err := newRotatingWriter(dir)
		if err != nil {
			return nil, fmt.Errorf("logger: create rotating writer: %w", err)
		}
		writer = io.MultiWriter(os.Stdout, rotating)
	}

	var handler slog.Handler
	if pretty {
		handler = slog.NewTextHandler(writer, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(writer, handlerOpts)
	}

	log := slog.New(handler)
	SetDefault(log, level)
	return log, nil
}

// SetDefault 替换全局日志记录器和日志级别。
func SetDefault(log *slog.Logger, level slog.Level) {
	if log == nil {
		return
	}
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLog = log
	globalLevel = level
	slog.SetDefault(log)
}

// Default 返回当前的全局日志记录器。
func Default() *slog.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalLog != nil {
		return globalLog
	}
	return slog.Default()
}

// Level 返回当前全局日志级别。
func Level() slog.Level {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLevel
}

type contextKey struct{}

type requestIDKey struct{}

// WithContext 将日志记录器注入到上下文中。
func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey{}, log)
}

// FromContext 从上下文中提取日志记录器，若不存在则返回全局默认值。
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Default()
	}
	if log, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	return Default()
}

// ContextWithRequestID 在上下文中写入 request_id。
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromContext 读取上下文中的 request_id。
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// Debug 记录调试级别日志。
func Debug(ctx context.Context, msg string, args ...any) {
	ensureContext(ctx)
	FromContext(ctx).Debug(msg, args...)
}

// Info 记录信息级别日志。
func Info(ctx context.Context, msg string, args ...any) {
	ensureContext(ctx)
	FromContext(ctx).Info(msg, args...)
}

// Warn 记录警告级别日志。
func Warn(ctx context.Context, msg string, args ...any) {
	ensureContext(ctx)
	FromContext(ctx).Warn(msg, args...)
}

// Error 记录错误级别日志。
func Error(ctx context.Context, msg string, args ...any) {
	ensureContext(ctx)
	FromContext(ctx).Error(msg, args...)
}

func ensureContext(ctx context.Context) {
	if ctx == nil {
		return
	}
	if _, ok := ctx.(*gin.Context); ok {
		panic("logger: gin.Context provided; use c.Request.Context() instead")
	}
}

func resolveModeDefaults(mode string) (slog.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "prod", "production", "pro":
		return slog.LevelInfo, false
	default:
		return slog.LevelDebug, true
	}
}

func parseLevel(level string) (slog.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}

type rotatingWriter struct {
	dir  string
	mu   sync.Mutex
	day  string
	file *os.File
}

func newRotatingWriter(dir string) (*rotatingWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &rotatingWriter{dir: dir}, nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().UTC().Format("2006-01-02")
	if w.file == nil || w.day != today {
		if err := w.rotateLocked(today); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	if err != nil {
		return n, err
	}
	if err := w.file.Sync(); err != nil {
		return n, err
	}
	return n, nil
}

func (w *rotatingWriter) rotateLocked(day string) error {
	if w.file != nil {
		_ = w.file.Close()
	}

	name := fmt.Sprintf("%s-%s.log", logFilePrefix, day)
	file, err := os.OpenFile(filepath.Join(w.dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	w.file = file
	w.day = day
	w.cleanupLocked()
	return nil
}

func (w *rotatingWriter) cleanupLocked() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, logFilePrefix+"-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, logFilePrefix+"-"), ".log")
		t, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			_ = os.Remove(filepath.Join(w.dir, name))
		}
	}
}

// GenerateRequestID 返回一个长度为 12 的随机请求 ID。
func GenerateRequestID() string {
	buf := make([]byte, requestIDLength)
	if _, err := rand.Read(buf); err != nil {
		fillDeterministic(buf)
	}
	for i := range buf {
		buf[i] = requestIDAlphabet[int(buf[i])%len(requestIDAlphabet)]
	}
	return string(buf)
}

func fillDeterministic(buf []byte) {
	seed := uint64(time.Now().UnixNano())
	for i := range buf {
		seed = seed*6364136223846793005 + 1
		buf[i] = byte((seed >> 32) % uint64(len(requestIDAlphabet)))
	}
}
