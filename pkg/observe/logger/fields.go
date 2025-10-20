package logger

import "log/slog"

// 为了方便使用，我们将 slog 的标准字段构造器直接作为我们包的变量导出。
// 开发者只需要 import 我们的 logger 包，就能使用这些工具。

var (
	Any      = slog.Any
	Bool     = slog.Bool
	Duration = slog.Duration
	Float64  = slog.Float64
	Group    = slog.Group
	Int      = slog.Int
	Int64    = slog.Int64
	String   = slog.String
	Time     = slog.Time
	Uint64   = slog.Uint64
)
