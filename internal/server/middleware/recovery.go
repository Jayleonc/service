package middleware

import (
	"net/http"
	"runtime/debug"

	applogger "github.com/Jayleonc/service/pkg/observe/logger"
	"github.com/gin-gonic/gin"
)

// Recovery 捕获 panic 并记录详细错误信息。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				ctx := c.Request.Context()
				args := []any{
					"panic", rec,
					"stack", string(stack),
					"request_id", applogger.RequestIDFromContext(ctx),
				}
				applogger.Error(ctx, "panic recovered", args...)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "internal server error",
					"request_id": applogger.RequestIDFromContext(ctx),
				})
			}
		}()

		c.Next()
	}
}
