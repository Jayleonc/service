package server

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/internal/database"
	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/logger"
	"github.com/Jayleonc/service/internal/rbac"
	"github.com/Jayleonc/service/pkg/auth"
	"github.com/Jayleonc/service/pkg/cache"
	"github.com/Jayleonc/service/pkg/config"
	databasepkg "github.com/Jayleonc/service/pkg/database"
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

	// ======= 配置运行模式 =======
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "prod", "production", "pro":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	// ======= 初始化日志 =======
	log, err := logger.Init(logger.Config{
		Mode:      cfg.Mode,
		Level:     cfg.Logger.Level,
		Pretty:    cfg.Logger.Pretty,
		Directory: cfg.Logger.Directory,
	})
	if err != nil {
		return nil, fmt.Errorf("initialise logger: %w", err)
	}

	// ======= 初始化数据库 =======
	gormLogger := database.NewLogger(logger.Level())
	db, err := databasepkg.Init(databasepkg.Config{
		Driver:   cfg.Database.Driver,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
		Params:   cfg.Database.Params,
		Logger:   gormLogger,
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
