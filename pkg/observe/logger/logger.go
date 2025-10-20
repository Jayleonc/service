package logger

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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
	requestIDLength   = 16
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
		if isDevMode(cfg.Mode) {
			handler = newDevTextHandler(writer, handlerOpts)
		} else {
			handler = slog.NewTextHandler(writer, handlerOpts)
		}
	} else {
		handler = slog.NewJSONHandler(writer, handlerOpts)
	}

	// In dev mode with pretty enabled, wrap with prettySQLHandler for multi-line, colorized SQL output
	if isDevMode(cfg.Mode) && pretty {
		handler = newPrettySQLHandler(handler)
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

	// 自动兼容 gin.Context
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return FromContext(ginCtx.Request.Context())
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
	// 自动兼容 gin.Context
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return RequestIDFromContext(ginCtx.Request.Context())
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

// ensureContext 确保上下文是 context.Context 类型。
// 如果你不希望使用这个，可以直接注释掉里面的逻辑
// 下游函数已经兼容了 *gin.Context 的情况
// 建议保留（模式一：严格引导），以保持代码的高度一致性
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

// GenerateRequestID 返回一个长度为 16 的随机请求 ID。
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

// ===== Dev-only pretty SQL handler and dev text handler (inlined for simplicity) =====

type prettySQLHandler struct {
	next   slog.Handler
	writer io.Writer
	colors bool
}

func newPrettySQLHandler(next slog.Handler) slog.Handler {
	return &prettySQLHandler{next: next, writer: os.Stdout, colors: true}
}

func (h *prettySQLHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *prettySQLHandler) Handle(ctx context.Context, r slog.Record) error {
	var (
		isSQL     bool
		sqlText   string
		rows      any
		duration  any
		threshold any
		errVal    any
	)
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "gorm_sql":
			if a.Value.Kind() == slog.KindBool && a.Value.Bool() {
				isSQL = true
			}
		case "sql":
			sqlText = fmt.Sprint(a.Value.Any())
		case "rows":
			rows = a.Value.Any()
		case "duration":
			duration = a.Value.Any()
		case "threshold":
			threshold = a.Value.Any()
		case "error":
			errVal = a.Value.Any()
		}
		return true
	})
	if !isSQL {
		return h.next.Handle(ctx, r)
	}
	b := &strings.Builder{}
	b.WriteString(colorize("[GORM]", 35, h.colors))
	b.WriteString(" ")
	b.WriteString(r.Time.Format(time.RFC3339))
	b.WriteString("\n")
	if errVal != nil {
		b.WriteString(colorize("ERROR:", 31, h.colors))
		b.WriteString(" ")
		b.WriteString(fmt.Sprint(errVal))
		b.WriteString("\n")
	}
	if duration != nil {
		b.WriteString(colorize("[time]", 36, h.colors))
		b.WriteString(" ")
		b.WriteString(fmt.Sprint(duration))
		if threshold != nil {
			b.WriteString(" (>")
			b.WriteString(fmt.Sprint(threshold))
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	if rows != nil {
		b.WriteString(colorize("[rows]", 36, h.colors))
		b.WriteString(" ")
		b.WriteString(fmt.Sprint(rows))
		b.WriteString("\n")
	}
	if sqlText != "" {
		b.WriteString(colorize("[sql]", 32, h.colors))
		b.WriteString("\n    ")
		b.WriteString(sqlText)
		b.WriteString("\n")
	}
	_, _ = fmt.Fprint(h.writer, b.String())
	return nil
}

func (h *prettySQLHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &prettySQLHandler{next: h.next.WithAttrs(attrs), writer: h.writer, colors: h.colors}
}

func (h *prettySQLHandler) WithGroup(name string) slog.Handler {
	return &prettySQLHandler{next: h.next.WithGroup(name), writer: h.writer, colors: h.colors}
}

func colorize(s string, color int, enable bool) string {
	if !enable {
		return s
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, s)
}

type devTextHandler struct {
	w      io.Writer
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func newDevTextHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if w == nil {
		w = os.Stdout
	}
	lvl := slog.LevelInfo
	if opts != nil && opts.Level != nil {
		lvl = opts.Level.Level()
	}
	return &devTextHandler{w: w, level: lvl}
}

func (h *devTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *devTextHandler) Handle(_ context.Context, r slog.Record) error {
	b := &strings.Builder{}
	b.WriteString("time=")
	b.WriteString(quote(r.Time.Format(time.RFC3339Nano)))
	b.WriteString(" ")
	b.WriteString("level=")
	b.WriteString(quote(strings.ToUpper(r.Level.String())))
	if r.Message != "" {
		b.WriteString(" ")
		b.WriteString("msg=")
		b.WriteString(quote(r.Message))
	}
	all := make([]slog.Attr, 0, len(h.attrs)+8)
	all = append(all, h.flattenedAttrs(h.attrs)...)
	r.Attrs(func(a slog.Attr) bool { all = append(all, h.flattenAttr(a)); return true })
	sort.SliceStable(all, func(i, j int) bool { return all[i].Key < all[j].Key })
	for _, a := range all {
		if a.Equal(slog.Attr{}) {
			continue
		}
		b.WriteString(" ")
		b.WriteString(formatAttr(a))
	}
	b.WriteString("\n")
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *devTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *devTextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func (h *devTextHandler) flattenedAttrs(attrs []slog.Attr) []slog.Attr {
	res := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		res = append(res, h.flattenAttr(a))
	}
	return res
}

func (h *devTextHandler) flattenAttr(a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	if len(h.groups) > 0 {
		pref := strings.Join(h.groups, ".")
		if a.Key != "" {
			a.Key = pref + "." + a.Key
		} else {
			a.Key = pref
		}
	}
	return a
}

func formatAttr(a slog.Attr) string {
	v := a.Value
	switch v.Kind() {
	case slog.KindString:
		return fmt.Sprintf("%s=%s", a.Key, quote(v.String()))
	case slog.KindBool:
		return fmt.Sprintf("%s=%t", a.Key, v.Bool())
	case slog.KindInt64:
		return fmt.Sprintf("%s=%d", a.Key, v.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%s=%d", a.Key, v.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%s=%g", a.Key, v.Float64())
	case slog.KindTime:
		return fmt.Sprintf("%s=%s", a.Key, quote(v.Time().Format(time.RFC3339Nano)))
	case slog.KindDuration:
		return fmt.Sprintf("%s=%s", a.Key, quote(v.Duration().String()))
	default:
		return fmt.Sprintf("%s=%s", a.Key, quote(fmt.Sprint(v.Any())))
	}
}

func quote(s string) string {
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	return "\"" + escaped + "\""
}

func isDevMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "prod", "production", "pro":
		return false
	default:
		return true
	}
}
