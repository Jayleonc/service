package feature

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const contextAuthContextKey = "auth.context"

// AuthContext captures the authenticated user context shared between features.
type AuthContext struct {
	SessionID    string
	UserID       uuid.UUID
	Roles        []string
	RefreshToken string
}

// SetAuthContext stores the authenticated context data into the Gin context.
func SetAuthContext(c *gin.Context, ctx AuthContext) {
	c.Set(contextAuthContextKey, ctx)
}

// AuthContextFromContext retrieves the authenticated context from the Gin context.
func AuthContextFromContext(c *gin.Context) (AuthContext, bool) {
	value, ok := c.Get(contextAuthContextKey)
	if !ok {
		return AuthContext{}, false
	}

	session, ok := value.(AuthContext)
	if !ok {
		return AuthContext{}, false
	}

	return session, true
}
