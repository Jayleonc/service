package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// RBAC 确保认证用户至少具备一个目标角色。
func RBAC(requiredRoles ...string) gin.HandlerFunc {
	normalized := make([]string, 0, len(requiredRoles))
	seen := make(map[string]struct{}, len(requiredRoles))
	for _, role := range requiredRoles {
		trimmed := strings.ToUpper(strings.TrimSpace(role))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return func(c *gin.Context) {
		session, ok := feature.GetAuthContext(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, ErrMissingSession)
			c.Abort()
			return
		}

		if len(normalized) == 0 {
			c.Next()
			return
		}

		if !hasAnyRole(session.Roles, normalized) {
			response.Error(c, http.StatusForbidden, ErrInsufficientPrivilege)
			c.Abort()
			return
		}

		c.Next()
	}
}

func hasAnyRole(userRoles, requiredRoles []string) bool {
	if len(userRoles) == 0 || len(requiredRoles) == 0 {
		return false
	}

	set := make(map[string]struct{}, len(userRoles))
	for _, role := range userRoles {
		trimmed := strings.ToUpper(strings.TrimSpace(role))
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}

	for _, required := range requiredRoles {
		if _, ok := set[required]; ok {
			return true
		}
	}

	return false
}
