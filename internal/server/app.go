package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/pkg/config"
)

// App holds the initialized Gin engine and shared configuration for the service.
type App struct {
	Engine *gin.Engine
	Config config.App
	Logger *slog.Logger
}

// NewApp constructs an application instance that wraps the engine and configuration.
func NewApp(router *gin.Engine, cfg config.App, logger *slog.Logger) *App {
	return &App{
		Engine: router,
		Config: cfg,
		Logger: logger,
	}
}
