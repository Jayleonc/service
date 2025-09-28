package feature

import "github.com/gin-gonic/gin"

// RouteGuards captures the middleware stacks applied to feature routes.
type RouteGuards struct {
        Public        []gin.HandlerFunc
        Authenticated []gin.HandlerFunc
        Admin         []gin.HandlerFunc
}
