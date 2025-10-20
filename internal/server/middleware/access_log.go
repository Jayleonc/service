package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	applogger "github.com/Jayleonc/service/internal/logger"
)

// AccessLogger 输出结构化访问日志。
func AccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := c.Writer.Status()
		args := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"duration", duration,
			"ip", c.ClientIP(),
			"bytes_in", c.Request.ContentLength,
			"bytes_out", c.Writer.Size(),
		}
		if err := c.Errors.Last(); err != nil {
			args = append(args, "error", err.Error())
		} else if len(c.Errors) > 0 {
			args = append(args, "error", c.Errors.String())
		}

		ctx := c.Request.Context()
		switch {
		case status >= http.StatusInternalServerError:
			applogger.Error(ctx, "request failed", args...)
		case status >= http.StatusBadRequest:
			applogger.Warn(ctx, "request completed with client error", args...)
		default:
			applogger.Info(ctx, "request completed", args...)
		}
	}
}
