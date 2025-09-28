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
	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/validation"
)

// RouterConfig defines the common HTTP middleware configuration shared by all modules.
type RouterConfig struct {
	Logger           *slog.Logger
	Registry         *prometheus.Registry
	TelemetryEnabled bool
	TelemetryName    string
}

// Router centralises HTTP route registration and applies the correct middleware stack for
// public, authenticated, and admin endpoints.
type Router struct {
	Engine           *gin.Engine
	api              *gin.RouterGroup
	authService      *auth.Service
	authMiddlewares  []gin.HandlerFunc
	adminMiddlewares []gin.HandlerFunc
}

// NewRouter constructs a Router with the provided configuration and authentication service.
func NewRouter(cfg RouterConfig, authService *auth.Service) *Router {
	gin.SetMode(gin.DebugMode)

	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validation.SetDefault(engine)
	} else {
		validation.Init()
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(middleware.InjectLogger(cfg.Logger))
	engine.Use(middleware.Recovery())

	if cfg.Registry != nil {
		engine.Use(middleware.Metrics(cfg.Registry))
		engine.GET("/metrics", gin.WrapH(promhttp.HandlerFor(cfg.Registry, promhttp.HandlerOpts{})))
	}

	if cfg.TelemetryEnabled {
		engine.Use(otelgin.Middleware(cfg.TelemetryName))
	}

	engine.POST("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := engine.Group("/v1")

	router := &Router{
		Engine: engine,
		api:    api,
	}

	router.setAuthService(authService)
	return router
}

// RegisterModule registers the provided module routes under the specified prefix.
func (r *Router) RegisterModule(pathPrefix string, modRoutes feature.ModuleRoutes) {
	if r == nil || r.api == nil {
		return
	}

	r.refreshAuthChains()

	prefix := strings.TrimSpace(pathPrefix)
	if prefix == "/" {
		prefix = ""
	}
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	register := func(defs []feature.RouteDefinition, middlewares []gin.HandlerFunc) {
		if len(defs) == 0 {
			return
		}

		group := r.api.Group(prefix)
		if len(middlewares) > 0 {
			group.Use(middlewares...)
		}

		for _, def := range defs {
			if def.Handler == nil {
				continue
			}

			path := strings.TrimSpace(def.Path)
			if path == "" {
				path = "/"
			}
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			group.POST(path, def.Handler)
		}
	}

	register(modRoutes.PublicRoutes, nil)
	register(modRoutes.AuthenticatedRoutes, r.authMiddlewares)
	adminChain := r.adminMiddlewares
	if len(adminChain) == 0 {
		adminChain = r.authMiddlewares
	}
	register(modRoutes.AdminRoutes, adminChain)
}

func (r *Router) refreshAuthChains() {
	if r == nil {
		return
	}

	svc := r.authService
	if svc == nil {
		svc = auth.DefaultService()
	}
	if svc == nil {
		r.setAuthService(nil)
		return
	}

	if r.authService == svc && len(r.authMiddlewares) > 0 {
		return
	}

	r.setAuthService(svc)
}

func (r *Router) setAuthService(svc *auth.Service) {
	if r == nil {
		return
	}

	r.authService = svc
	if svc == nil {
		r.authMiddlewares = nil
		r.adminMiddlewares = []gin.HandlerFunc{middleware.RBAC(constant.RoleAdmin)}
		return
	}

	authenticated := middleware.Authenticated(svc)
	r.authMiddlewares = []gin.HandlerFunc{authenticated}
	r.adminMiddlewares = []gin.HandlerFunc{authenticated, middleware.RBAC(constant.RoleAdmin)}
}
