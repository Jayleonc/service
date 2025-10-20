package middleware

import (
	"log/slog"

	applogger "github.com/Jayleonc/service/pkg/observe/logger"
	"github.com/gin-gonic/gin"
)

// RequestID 为每个请求生成唯一 ID 并注入带有该 ID 的日志器。
func RequestID(base *slog.Logger) gin.HandlerFunc {
	if base == nil {
		base = applogger.Default()
	}
	return func(c *gin.Context) {
		id := applogger.GenerateRequestID()

		requestLogger := base.With(slog.String("request_id", id))
		ctx := applogger.WithContext(c.Request.Context(), requestLogger)
		ctx = applogger.ContextWithRequestID(ctx, id)

		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-ID", id)

		c.Next()
	}
}

// LoggerFromContext 返回上下文中的日志器（包含 request_id）。
func LoggerFromContext(c *gin.Context) *slog.Logger {
	if c == nil {
		return applogger.Default()
	}
	log := applogger.FromContext(c.Request.Context())
	if id := applogger.RequestIDFromContext(c.Request.Context()); id != "" {
		return log.With(slog.String("request_id", id))
	}
	return log
}

// RequestIDFromContext 返回请求上下文中的 request_id。
func RequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return applogger.RequestIDFromContext(c.Request.Context())
}

// RequestIDHeader 返回用于响应的请求 ID 头名称。
func RequestIDHeader() string {
	return "X-Request-ID"
}

// AbortWithRequestID 生成包含 request_id 的 JSON 错误响应。
func AbortWithRequestID(c *gin.Context, status int, payload gin.H) {
	if c == nil {
		return
	}
	if payload == nil {
		payload = gin.H{}
	}
	if _, exists := payload["request_id"]; !exists {
		payload["request_id"] = RequestIDFromContext(c)
	}
	c.AbortWithStatusJSON(status, payload)
}

// ContextRequestID 返回请求 ID。
func ContextRequestID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	return RequestIDFromContext(ctx)
}
