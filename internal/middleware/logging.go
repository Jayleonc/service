package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/pkg/observe/logger"
)

// Logging 使用 slog 记录基础的请求日志信息。
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.FromContext(c.Request.Context())
		start := time.Now()

		c.Next()

		dur := time.Since(start)
		log.Info("request completed",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"duration", dur,
		)
	}
}
