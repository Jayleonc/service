package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jayleonc/service/internal/handler"
	"github.com/Jayleonc/service/internal/repository"
	"github.com/Jayleonc/service/internal/service"
	"github.com/Jayleonc/service/pkg/auth"
	"github.com/Jayleonc/service/pkg/config"
	"github.com/Jayleonc/service/pkg/database"
	"github.com/Jayleonc/service/pkg/logger"
	"github.com/Jayleonc/service/pkg/metrics"
	"github.com/Jayleonc/service/pkg/server"
	"github.com/Jayleonc/service/pkg/telemetry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg, err := config.Init(ctx, os.Args[1:])
	if err != nil {
		panic(err)
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
		log.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	repo := repository.NewUserRepository(db)
	if err := repo.Migrate(ctx); err != nil {
		log.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	authManager, err := auth.Init(auth.Config{
		Issuer:   cfg.Auth.Issuer,
		Audience: cfg.Auth.Audience,
		Secret:   cfg.Auth.Secret,
		Duration: cfg.Auth.TTL,
	})
	if err != nil {
		log.Error("failed to configure auth manager", "error", err)
		os.Exit(1)
	}

	telemetryProvider, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName: cfg.Telemetry.ServiceName,
		Endpoint:    cfg.Telemetry.Endpoint,
		Enabled:     cfg.Telemetry.Enabled,
	})
	if err != nil {
		log.Error("failed to setup telemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryProvider.Shutdown(shutdownCtx); err != nil {
			log.Error("failed to shutdown telemetry", "error", err)
		}
	}()

	registry := metrics.InitRegistry()
	userService := service.NewUserService(repo)

	router := handler.NewRouter(handler.RouterConfig{
		Logger:           log,
		Auth:             authManager,
		UserService:      userService,
		Registry:         registry,
		TelemetryEnabled: cfg.Telemetry.Enabled,
		TelemetryName:    cfg.Telemetry.ServiceName,
	})

	srv := server.NewHTTP(server.HTTPConfig{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}, router)

	errCh := make(chan error, 1)
	go func() {
		log.Info("starting http server", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownServer(log, srv)
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
		}
	}
}

func shutdownServer(log *slog.Logger, srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("shutting down server")
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to shutdown server", "error", err)
	}
}
