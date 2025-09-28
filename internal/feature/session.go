package feature

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const contextSessionKey = "auth.session"

// SessionData captures the authenticated user context shared between features.
type SessionData struct {
	SessionID    string
	UserID       uuid.UUID
	Roles        []string
	RefreshToken string
}

// SetContextSession stores the authenticated session data into the Gin context.
func SetContextSession(c *gin.Context, session SessionData) {
	c.Set(contextSessionKey, session)
}

// SessionFromContext retrieves the authenticated session from the Gin context.
func SessionFromContext(c *gin.Context) (SessionData, bool) {
	value, ok := c.Get(contextSessionKey)
	if !ok {
		return SessionData{}, false
	}

	session, ok := value.(SessionData)
	if !ok {
		return SessionData{}, false
	}

	return session, true
}
