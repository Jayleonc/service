package server

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/validation"
)

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

// RouterConfig defines the common HTTP middleware configuration shared by all modules.
type RouterConfig struct {
	Logger           *slog.Logger
	Registry         *prometheus.Registry
	TelemetryEnabled bool
	TelemetryName    string
}

// NewRouter constructs the base Gin engine and returns both the engine and the
// authenticated "/v1" API group used by modules.
func NewRouter(cfg RouterConfig) (*gin.Engine, *gin.RouterGroup) {
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

	api := r.Group("/v1")
	return r, api
}

// RegisterModuleRoutes 将模块路由注册到统一路由中
func RegisterModuleRoutes(api *gin.RouterGroup, authService *auth.Service, routes ModuleRoutes) {
	if api == nil {
		return
	}

	register := func(defs []RouteDefinition, middlewares ...gin.HandlerFunc) {
		if len(defs) == 0 {
			return
		}

		group := api.Group("")
		if len(middlewares) > 0 {
			group.Use(middlewares...)
		}

		for _, def := range defs {
			if def.Handler == nil {
				continue
			}

			path := strings.TrimSpace(def.Path)
			if path == "" {
				continue
			}

			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			group.POST(path, def.Handler)
		}
	}

	register(routes.PublicRoutes)

	if authService != nil {
		register(routes.AuthenticatedRoutes, middleware.Authenticated(authService))
		register(routes.AdminRoutes, middleware.Authenticated(authService), middleware.RBAC(constant.RoleAdmin))
	} else {
		register(routes.AuthenticatedRoutes)
		register(routes.AdminRoutes)
	}
}
