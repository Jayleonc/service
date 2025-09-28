package feature

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const contextAuthContextKey = "auth.context"

// AuthContext 描述功能模块之间共享的认证上下文。
type AuthContext struct {
	SessionID    string
	UserID       uuid.UUID
	Roles        []string
	RefreshToken string
}

// SetAuthContext 将认证上下文写入 Gin Context。
func SetAuthContext(c *gin.Context, ctx AuthContext) {
	c.Set(contextAuthContextKey, ctx)
}

// GetAuthContext 从 Gin Context 中读取认证上下文。
func GetAuthContext(c *gin.Context) (AuthContext, bool) {
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

// MustGetAuthContext 获取认证上下文，若不存在则直接 panic。
func MustGetAuthContext(c *gin.Context) AuthContext {
	session, ok := GetAuthContext(c)
	if !ok {
		panic("feature: auth context not available")
	}
	return session
}
