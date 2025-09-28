package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Jayleonc/service/internal/module"
	"github.com/Jayleonc/service/pkg/auth"
	"github.com/Jayleonc/service/pkg/config"
	"github.com/Jayleonc/service/pkg/database"
	"github.com/Jayleonc/service/pkg/logger"
	"github.com/Jayleonc/service/pkg/metrics"
	"github.com/Jayleonc/service/pkg/telemetry"
)

// Bootstrap assembles the shared infrastructure, registers every module and returns
// a ready-to-run application instance.
func Bootstrap(ctx context.Context, args []string) (*App, error) {
	cfg, err := config.Init(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	log := logger.Init(logger.Config{Level: cfg.Log.Level, Pretty: cfg.Log.Pretty})

	db, err := database.Init(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	registry := metrics.InitRegistry()

	authManager, err := auth.Init(auth.Config{
		Issuer:   cfg.Auth.Issuer,
		Audience: cfg.Auth.Audience,
		Secret:   cfg.Auth.Secret,
		Duration: cfg.Auth.TTL,
	})
	if err != nil {
		return nil, fmt.Errorf("configure auth manager: %w", err)
	}

	telemetryProvider, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: cfg.Telemetry.ServiceName,
		Endpoint:    cfg.Telemetry.Endpoint,
		Enabled:     cfg.Telemetry.Enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	router, api := NewRouter(RouterConfig{
		Logger:           log,
		Auth:             authManager,
		Registry:         registry,
		TelemetryEnabled: cfg.Telemetry.Enabled,
		TelemetryName:    cfg.Telemetry.ServiceName,
	})

	deps := module.Dependencies{
		Logger:   log,
		DB:       db,
		Router:   router,
		API:      api,
		Auth:     authManager,
		Registry: registry,
		Config:   cfg,
	}

	for _, entry := range Modules {
		if err := entry.Registrar(ctx, deps); err != nil {
			return nil, fmt.Errorf("register module %s: %w", entry.Name, err)
		}
		log.Info("module registered", "module", entry.Name)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	var shutdowns []func(context.Context) error
	if telemetryProvider != nil {
		shutdowns = append(shutdowns, telemetryProvider.Shutdown)
	}

	return NewApp(log, srv, shutdowns...), nil
}
