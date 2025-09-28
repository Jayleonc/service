package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// AuthenticatedMiddleware 校验请求是否携带合法 JWT，并将会话信息写入上下文。
func AuthenticatedMiddleware(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Error(c, http.StatusUnauthorized, ErrMissingAuthorizationHeader)
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Error(c, http.StatusUnauthorized, ErrInvalidAuthorizationHeader)
			c.Abort()
			return
		}

		session, err := service.Validate(c.Request.Context(), parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, ErrInvalidToken)
			c.Abort()
			return
		}

		feature.SetAuthContext(c, session)
		c.Next()
	}
}
