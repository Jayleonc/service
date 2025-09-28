package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/Jayleonc/service/internal/handler/v1"
	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/internal/service"
	"github.com/Jayleonc/service/pkg/auth"
)

// RouterConfig contains dependencies for building the HTTP router.
type RouterConfig struct {
	Logger           *slog.Logger
	Auth             *auth.Manager
	UserService      *service.UserService
	Registry         *prometheus.Registry
	TelemetryEnabled bool
	TelemetryName    string
}

// NewRouter constructs a fully configured gin.Engine.
func NewRouter(cfg RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.InjectLogger(cfg.Logger))
	r.Use(middleware.Recovery())
	r.Use(middleware.Logging())

	if cfg.Registry != nil {
		r.Use(middleware.Metrics(cfg.Registry))
		r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(cfg.Registry, promhttp.HandlerOpts{})))
	}

	if cfg.TelemetryEnabled {
		r.Use(otelgin.Middleware(cfg.TelemetryName))
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/v1")
	if cfg.Auth != nil {
		api.Use(middleware.Authenticated(cfg.Auth))
	}

	userHandler := v1.NewUserHandler(cfg.UserService)
	userHandler.RegisterRoutes(api)

	return r
}
