package module

import (
        "context"
        "log/slog"

        "github.com/gin-gonic/gin"
        "github.com/prometheus/client_golang/prometheus"
        "github.com/redis/go-redis/v9"
        "gorm.io/gorm"

        authpkg "github.com/Jayleonc/service/pkg/auth"
        "github.com/Jayleonc/service/pkg/config"
)

// Dependencies captures the shared infrastructure that modules can leverage during registration.
type Dependencies struct {
        Logger   *slog.Logger
        DB       *gorm.DB
        Router   *gin.Engine
        API      *gin.RouterGroup
        Auth     *authpkg.Manager
        Registry *prometheus.Registry
        Config   config.App
        Cache    *redis.Client
}

// Registrar declares the signature that modules must implement to self-register.
type Registrar func(context.Context, Dependencies) error

// Entry associates a human-friendly name with a registrar implementation.
type Entry struct {
	Name      string
	Registrar Registrar
}
