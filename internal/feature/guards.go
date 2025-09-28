package feature

import "github.com/gin-gonic/gin"

// RouteGuards 描述不同权限路由所需的中间件链。
type RouteGuards struct {
	Public        []gin.HandlerFunc
	Authenticated []gin.HandlerFunc
	Admin         []gin.HandlerFunc
}
