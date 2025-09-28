package auth

import "github.com/gin-gonic/gin"

const contextSessionKey = "auth.session"

// ContextSessionKey exposes the key used to store session data in Gin's context.
func ContextSessionKey() string {
	return contextSessionKey
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
