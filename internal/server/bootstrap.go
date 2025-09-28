package server

import (
	"context"
	"fmt"
	"os"

	"github.com/Jayleonc/service/internal/module"
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
func Bootstrap() (*App, error) {
	ctx := context.Background()

	cfg, err := config.Init(ctx, os.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	log := logger.Init(logger.Config{Level: cfg.Log.Level, Pretty: cfg.Log.Pretty})

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

	cacheClient, err := cache.Init(cache.Config{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	registry := metrics.InitRegistry()

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

	if _, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: cfg.Telemetry.ServiceName,
		Endpoint:    cfg.Telemetry.Endpoint,
		Enabled:     cfg.Telemetry.Enabled,
	}); err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	router, api := NewRouter(RouterConfig{
		Logger:           log,
		Registry:         registry,
		TelemetryEnabled: cfg.Telemetry.Enabled,
		TelemetryName:    cfg.Telemetry.ServiceName,
	})

	deps := module.Dependencies{
		Logger:    log,
		DB:        db,
		Router:    router,
		API:       api,
		Auth:      authManager,
		Registry:  registry,
		Config:    cfg,
		Cache:     cacheClient,
		Validator: validation.Default(),
	}

	for _, entry := range Modules {
		if err := entry.Registrar(ctx, deps); err != nil {
			return nil, fmt.Errorf("register module %s: %w", entry.Name, err)
		}
		log.Info("module registered", "module", entry.Name)
	}

	return NewApp(router, cfg, log), nil
}
