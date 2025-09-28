package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/pkg/validation"
)

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
