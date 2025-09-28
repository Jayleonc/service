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

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/pkg/validation"
)

// RouterConfig defines the common HTTP middleware configuration shared by all features.
type RouterConfig struct {
	Logger           *slog.Logger
	Registry         *prometheus.Registry
	TelemetryEnabled bool
	TelemetryName    string
	Guards           *feature.RouteGuards
}

// Router encapsulates the Gin engine and exposes feature-oriented registration helpers.
type Router struct {
	engine *gin.Engine
	api    *gin.RouterGroup
	guards *feature.RouteGuards
}

// NewRouter constructs the base Gin engine and returns a feature-aware router.
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
		engine: r,
		api:    r.Group("/v1"),
		guards: cfg.Guards,
	}
}

// Engine exposes the underlying Gin engine for server start-up.
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// RegisterModule mounts the provided feature routes under the shared "/v1" API group.
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

			group.POST(path, def.Handler)
		}
	}

	register(routes.PublicRoutes, guards.Public)
	register(routes.AuthenticatedRoutes, guards.Authenticated)
	register(routes.AdminRoutes, guards.Admin)
}

func sanitizePath(prefix, path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}

	if strings.HasPrefix(p, "/") {
		return collapsePath(p)
	}

	pre := strings.TrimSpace(prefix)
	if pre == "" {
		return "/" + collapsePath(p)
	}

	if !strings.HasPrefix(pre, "/") {
		pre = "/" + pre
	}

	return collapsePath(strings.TrimRight(pre, "/") + "/" + strings.TrimLeft(p, "/"))
}

func collapsePath(path string) string {
	if path == "" {
		return ""
	}

	cleaned := "/" + strings.TrimPrefix(strings.ReplaceAll(path, "//", "/"), "/")
	if cleaned == "" {
		return ""
	}

	return cleaned
}
