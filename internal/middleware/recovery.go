package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery 将 panic 转换成 JSON 错误响应。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			}
		}()

		c.Next()
	}
}
