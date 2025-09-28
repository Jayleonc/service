package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/pkg/config"
)

// App 保存已初始化的 Gin 引擎以及服务级共享配置。
type App struct {
	Engine *gin.Engine
	Config config.App
	Logger *slog.Logger
}

// NewApp 根据路由引擎与配置构建应用实例。
func NewApp(router *gin.Engine, cfg config.App, logger *slog.Logger) *App {
	return &App{
		Engine: router,
		Config: cfg,
		Logger: logger,
	}
}
