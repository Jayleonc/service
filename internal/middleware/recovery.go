package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery converts panics into JSON errors.
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
