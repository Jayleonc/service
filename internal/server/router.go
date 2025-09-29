package server

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/pkg/validation"
)

// RouterConfig 定义所有模块共享的 HTTP 中间件配置。
type RouterConfig struct {
	Logger           *slog.Logger
	Registry         *prometheus.Registry
	TelemetryEnabled bool
	TelemetryName    string
	Guards           *feature.RouteGuards
}

// Router 封装 Gin 引擎并提供面向功能模块的注册能力。
type Router struct {
	engine             *gin.Engine
	api                *gin.RouterGroup
	guards             *feature.RouteGuards
	permissionEnforcer func(string) gin.HandlerFunc
	collected          map[string]struct{}
}

// NewRouter 构建基础 Gin 引擎并返回具备模块注册能力的路由器。
func NewRouter(cfg RouterConfig) *Router {
	gin.SetMode(gin.DebugMode)

	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validation.SetDefault(engine)
	} else {
		validation.Init()
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.InjectLogger(cfg.Logger))
	r.Use(middleware.Recovery())

	if cfg.Registry != nil {
		r.Use(middleware.Metrics(cfg.Registry))
		r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(cfg.Registry, promhttp.HandlerOpts{})))
	}

	if cfg.TelemetryEnabled {
		r.Use(otelgin.Middleware(cfg.TelemetryName))
	}

	r.POST("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return &Router{
		engine:    r,
		api:       r.Group("/v1"),
		guards:    cfg.Guards,
		collected: make(map[string]struct{}),
	}
}

// Engine 返回底层 Gin 引擎供启动使用。
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// RegisterModule 将功能模块的路由挂载到共享的 /v1 API 分组。
func (r *Router) RegisterModule(pathPrefix string, routes feature.ModuleRoutes) {
	if r == nil || r.api == nil {
		return
	}

	guards := feature.RouteGuards{}
	if r.guards != nil {
		guards = *r.guards
	}

	register := func(defs []feature.RouteDefinition, middlewares []gin.HandlerFunc) {
		if len(defs) == 0 {
			return
		}

		group := r.api.Group("")
		if len(middlewares) > 0 {
			group.Use(middlewares...)
		}

		for _, def := range defs {
			if def.Handler == nil {
				continue
			}

			path := sanitizePath(pathPrefix, def.Path)
			if path == "" {
				continue
			}

			handlers := make([]gin.HandlerFunc, 0, 1)
			if def.RequiredPermission != "" {
				r.collected[def.RequiredPermission] = struct{}{}
				if r.permissionEnforcer != nil {
					handlers = append(handlers, r.permissionEnforcer(def.RequiredPermission))
				}
			}
			handlers = append(handlers, def.Handler)
			group.POST(path, handlers...)
		}
	}

	register(routes.PublicRoutes, guards.Public)
	register(routes.AuthenticatedRoutes, guards.Authenticated)
	register(routes.AdminRoutes, guards.Admin)
}

func sanitizePath(prefix, path string) string {
	// 直接拼接 prefix 和 path，然后交由 collapsePath 清理。
	// 这种方式确保 prefix 总是被应用，同时能优雅处理各种斜杠组合。
	return collapsePath(prefix + "/" + path)
}

func collapsePath(path string) string {
	if path == "" {
		return "/"
	}

	// 循环替换，确保 ///a//b -> /a/b
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// 移除首尾可能多余的斜杠，并确保最终结果以单个 / 开头
	return "/" + strings.Trim(path, "/")
}

// SetPermissionEnforcerFactory 注册权限中间件工厂函数。
func (r *Router) SetPermissionEnforcerFactory(factory func(string) gin.HandlerFunc) {
	if r == nil {
		return
	}
	r.permissionEnforcer = factory
}

// CollectedRoutePermissions 返回注册阶段收集的权限键集合（按字典序排序）。
func (r *Router) CollectedRoutePermissions() []string {
	if r == nil || len(r.collected) == 0 {
		return nil
	}
	keys := make([]string, 0, len(r.collected))
	for key := range r.collected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
