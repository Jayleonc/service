package rbac

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/ginx/response"
	"github.com/Jayleonc/service/pkg/xerr"
)

// PermissionChecker describes the ability to validate whether a user possesses a permission.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

// NewPermissionMiddleware returns a factory that produces permission enforcement middlewares.
func NewPermissionMiddleware(checker PermissionChecker) func(string) gin.HandlerFunc {
	if checker == nil {
		return nil
	}

	return func(permission string) gin.HandlerFunc {
		return func(c *gin.Context) {
			if permission == "" {
				c.Next()
				return
			}

			session, ok := feature.GetAuthContext(c)
			if !ok {
				response.Error(c, http.StatusUnauthorized, xerr.ErrUnauthorized.WithMessage("missing authorization context"))
				c.Abort()
				return
			}

			for _, role := range session.Roles {
				if NormalizeRoleName(role) == NormalizeRoleName(constant.RoleAdmin) {
					c.Next()
					return
				}
			}

			allowed, err := checker.HasPermission(c.Request.Context(), session.UserID, permission)
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
}
