package feature

import "github.com/gin-gonic/gin"

// RouteDefinition 定义了最基础的路由信息
type RouteDefinition struct {
	Path    string
	Handler gin.HandlerFunc
}

// ModuleRoutes 是一个模块对外暴露的、按权限划分的路由清单
type ModuleRoutes struct {
	PublicRoutes        []RouteDefinition
	AuthenticatedRoutes []RouteDefinition
	AdminRoutes         []RouteDefinition
}
