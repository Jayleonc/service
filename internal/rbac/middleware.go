package rbac

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// PermissionChecker describes the ability to validate whether a user possesses a permission.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

var permissionChecker PermissionChecker

// SetPermissionChecker wires the global permission checker used by the middleware.
func SetPermissionChecker(checker PermissionChecker) {
	permissionChecker = checker
}

// PermissionMiddleware ensures the current user has the required permission.
func PermissionMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if permission == "" {
			c.Next()
			return
		}

		session, ok := feature.GetAuthContext(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, ErrPermissionDenied)
			c.Abort()
			return
		}

		for _, role := range session.Roles {
			if NormalizeRoleName(role) == NormalizeRoleName(constant.RoleAdmin) {
				c.Next()
				return
			}
		}

		if permissionChecker == nil {
			response.Error(c, http.StatusInternalServerError, ErrPermissionServiceUnavailable)
			c.Abort()
			return
		}

		allowed, err := permissionChecker.HasPermission(c.Request.Context(), session.UserID, permission)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err)
			c.Abort()
			return
		}
		if !allowed {
			response.Error(c, http.StatusForbidden, ErrPermissionDenied)
			c.Abort()
			return
		}

		c.Next()
	}
}
