package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/pkg/logger"
)

// Logging records basic request metrics using slog.
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
