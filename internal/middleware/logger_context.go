package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/ardanlabs/service/pkg/logger"
)

// InjectLogger attaches the slog logger to the request context.
func InjectLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := logger.WithContext(c.Request.Context(), log)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
