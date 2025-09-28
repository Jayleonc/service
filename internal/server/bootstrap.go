package server

import (
	"context"
	"fmt"
	"os"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/auth"
	"github.com/Jayleonc/service/pkg/cache"
	"github.com/Jayleonc/service/pkg/config"
	"github.com/Jayleonc/service/pkg/database"
	"github.com/Jayleonc/service/pkg/logger"
	"github.com/Jayleonc/service/pkg/metrics"
	"github.com/Jayleonc/service/pkg/telemetry"
	"github.com/Jayleonc/service/pkg/validation"
)

// Bootstrap assembles the shared infrastructure, registers every module and returns
// a ready-to-run application instance.
func Bootstrap(modules []feature.Entry) (*App, error) {
	ctx := context.Background()

	// ======= 初始化配置 =======
	cfg, err := config.Init(ctx, os.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// ======= 初始化日志 =======
	log := logger.Init(logger.Config{Level: cfg.Log.Level, Pretty: cfg.Log.Pretty})

	// ======= 初始化数据库 =======
	db, err := database.Init(database.Config{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
		Params:   cfg.Database.Params,
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	// ======= 初始化缓存 Redis =======
	cacheClient, err := cache.Init(cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	// ======= 初始化指标检测 =======
	registry := metrics.InitRegistry()

	// ======= 初始化认证服务 =======
	authManager, err := auth.Init(auth.Config{
		Issuer:     cfg.Auth.Issuer,
		Audience:   cfg.Auth.Audience,
		Secret:     cfg.Auth.Secret,
		AccessTTL:  cfg.Auth.AccessTTL,
		RefreshTTL: cfg.Auth.RefreshTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("configure auth manager: %w", err)
	}

	// ======= 初始化指标检测 =======
	if _, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: cfg.Telemetry.ServiceName,
		Endpoint:    cfg.Telemetry.Endpoint,
		Enabled:     cfg.Telemetry.Enabled,
	}); err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	// ======= 路由注册 =======
	router := NewRouter(RouterConfig{
		Logger:           log,
		Registry:         registry,
		TelemetryEnabled: cfg.Telemetry.Enabled,
		TelemetryName:    cfg.Telemetry.ServiceName,
	}, nil)

	deps := feature.Dependencies{
		Logger:    log,
		DB:        db,
		Engine:    router.Engine,
		Router:    router,
		Auth:      authManager,
		Registry:  registry,
		Config:    cfg,
		Cache:     cacheClient,
		Validator: validation.Default(),
	}

	for _, entry := range modules {
		if err := entry.Registrar(ctx, deps); err != nil {
			return nil, fmt.Errorf("register module %s: %w", entry.Name, err)
		}
		log.Info("module registered", "module", entry.Name)
	}

	return NewApp(router.Engine, cfg, log), nil
}
