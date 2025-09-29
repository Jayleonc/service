package server

import (
	"context"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/rbac"
	"github.com/Jayleonc/service/pkg/auth"
	"github.com/Jayleonc/service/pkg/cache"
	"github.com/Jayleonc/service/pkg/config"
	"github.com/Jayleonc/service/pkg/database"
	"github.com/Jayleonc/service/pkg/observe/logger"
	"github.com/Jayleonc/service/pkg/observe/metrics"
	"github.com/Jayleonc/service/pkg/observe/telemetry"
	"github.com/Jayleonc/service/pkg/validation"
)

// Bootstrap 负责初始化通用基础设施、注册全部业务模块并返回可运行的应用实例。
func Bootstrap(features []feature.Entry) (*App, error) {
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

	// ======= 初始化指标采集 =======
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

	// ======= 初始化链路追踪 =======
	if _, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: cfg.Telemetry.ServiceName,
		Endpoint:    cfg.Telemetry.Endpoint,
		Enabled:     cfg.Telemetry.Enabled,
	}); err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	// ======= 路由注册 =======
	guards := &feature.RouteGuards{}
	router := NewRouter(RouterConfig{
		Logger:           log,
		Registry:         registry,
		TelemetryEnabled: cfg.Telemetry.Enabled,
		TelemetryName:    cfg.Telemetry.ServiceName,
		Guards:           guards,
	})

	deps := &feature.Dependencies{
		Logger:    log,
		DB:        db,
		Router:    router,
		Auth:      authManager,
		Registry:  registry,
		Config:    cfg,
		Cache:     cacheClient,
		Validator: validation.Default(),
		Engine:    router.Engine(),
		Guards:    guards,
	}

	for _, entry := range features {
		if err := entry.Registrar(ctx, deps); err != nil {
			return nil, fmt.Errorf("register feature %s: %w", entry.Name, err)
		}
		log.Info("feature registered", "feature", entry.Name)
	}

	if deps.Router != nil && deps.PermissionEnforcer != nil {
		deps.Router.SetPermissionEnforcerFactory(deps.PermissionEnforcer)
	}

	if deps.Guards != nil && deps.PermissionEnforcer != nil {
		enforcer := deps.PermissionEnforcer
		adminGuard := append([]gin.HandlerFunc{}, deps.Guards.Authenticated...)
		adminGuard = append(adminGuard, enforcer(rbac.PermissionKey(rbac.ResourceSystem, rbac.ActionAdmin)))
		deps.Guards.Admin = adminGuard
	}

	return NewApp(router.Engine(), cfg, log), nil
}
