package feature

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	authpkg "github.com/Jayleonc/service/pkg/auth"
	"github.com/Jayleonc/service/pkg/config"
)

// RouteRegistrar exposes the capability needed by features to register their HTTP endpoints.
type RouteRegistrar interface {
	RegisterModule(pathPrefix string, routes ModuleRoutes)
}

// Dependencies captures the shared infrastructure that modules can leverage during registration.
type Dependencies struct {
	Logger    *slog.Logger
	DB        *gorm.DB
	Engine    *gin.Engine
	Router    RouteRegistrar
	Auth      *authpkg.Manager
	Registry  *prometheus.Registry
	Config    config.App
	Cache     *redis.Client
	Validator *validator.Validate
}

// Registrar declares the signature that modules must implement to self-register.
type Registrar func(context.Context, Dependencies) error

// Entry associates a human-friendly name with a registrar implementation.
type Entry struct {
	Name      string
	Registrar Registrar
}
