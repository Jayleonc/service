package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/pkg/observe/logger"
)

// InjectLogger 将 slog 日志器注入到请求上下文中。
func InjectLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := logger.WithContext(c.Request.Context(), log)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
