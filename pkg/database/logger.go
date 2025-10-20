package database

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	applogger "github.com/Jayleonc/service/pkg/observe/logger"
)

const (
	slowQueryThreshold = 200 * time.Millisecond
)

// Logger 实现 GORM 的日志接口，基于应用的 slog 日志系统。
type Logger struct {
	level                     slog.Level
	ignoreRecordNotFoundError bool
	slowThreshold             time.Duration
	printSQL                  bool
}

// NewLogger 创建一个 GORM 日志器。
func NewLogger(level slog.Level) gormlogger.Interface {
	return &Logger{
		level:                     level,
		ignoreRecordNotFoundError: true,
		slowThreshold:             slowQueryThreshold,
		printSQL:                  false,
	}
}

// LogMode 调整日志级别。
func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	switch level {
	case gormlogger.Silent:
		clone.level = slog.LevelError + 1
	case gormlogger.Error:
		clone.level = slog.LevelError
	case gormlogger.Warn:
		clone.level = slog.LevelWarn
	case gormlogger.Info:
		clone.level = slog.LevelInfo
	default:
		clone.level = l.level
	}
	if clone.level < l.level {
		clone.level = l.level
	}
	// 当 GORM 收到 Info 级别（例如调用 db.Debug()）时，仅在该会话内开启 SQL 正常执行日志
	clone.printSQL = level >= gormlogger.Info
	return &clone
}

// Info 记录信息日志。
func (l *Logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level > slog.LevelInfo {
		return
	}
	applogger.Info(ctx, msg, data...)
}

// Warn 记录警告日志。
func (l *Logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level > slog.LevelWarn {
		return
	}
	applogger.Warn(ctx, msg, data...)
}

// Error 记录错误日志。
func (l *Logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level > slog.LevelError {
		return
	}
	applogger.Error(ctx, msg, data...)
}

// Trace 记录 SQL 执行详情。
func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if fc == nil {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	args := []any{
		"sql", sql,
		"rows", rows,
		"duration", elapsed,
		"gorm_sql", true,
	}

	switch {
	case err != nil && !(errors.Is(err, gorm.ErrRecordNotFound) && l.ignoreRecordNotFoundError):
		args = append(args, "error", err)
		applogger.Error(ctx, "sql execution failed", args...)
	case l.slowThreshold > 0 && elapsed > l.slowThreshold:
		args = append(args, "threshold", l.slowThreshold)
		if l.level <= slog.LevelWarn {
			applogger.Warn(ctx, "slow query", args...)
		}
	default:
		if l.printSQL {
			applogger.Info(ctx, "sql executed", args...)
		}
	}
}

var _ gormlogger.Interface = (*Logger)(nil)
