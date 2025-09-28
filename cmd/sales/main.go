package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/ardanlabs/service/internal/config"
	"github.com/ardanlabs/service/internal/handler"
	"github.com/ardanlabs/service/internal/repository"
	"github.com/ardanlabs/service/internal/service"
	"github.com/ardanlabs/service/pkg/auth"
	"github.com/ardanlabs/service/pkg/database"
	"github.com/ardanlabs/service/pkg/logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg, err := config.Parse(ctx, os.Args[1:])
	if err != nil {
		panic(err)
	}

	log := logger.New(logger.Config{Level: cfg.Log.Level, Pretty: cfg.Log.Pretty})
	slog.SetDefault(log)

	db, err := database.New(database.Config{
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

	authManager, err := auth.NewManager(auth.Config{
		Issuer:   cfg.Auth.Issuer,
		Audience: cfg.Auth.Audience,
		Secret:   cfg.Auth.Secret,
		Duration: cfg.Auth.TTL,
	})
	if err != nil {
		log.Error("failed to init auth manager", "error", err)
		os.Exit(1)
	}

	registry := prometheus.NewRegistry()
	userService := service.NewUserService(repo)

	shutdownTracing, err := setupTelemetry(ctx, cfg.Telemetry.ServiceName, cfg.Telemetry.Endpoint, cfg.Telemetry.Enabled)
	if err != nil {
		log.Error("failed to setup telemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		if shutdownTracing != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdownTracing(ctx); err != nil {
				log.Error("failed to shutdown tracing", "error", err)
			}
		}
	}()

	router := handler.NewRouter(handler.RouterConfig{
		Logger:           log,
		Auth:             authManager,
		UserService:      userService,
		Registry:         registry,
		TelemetryEnabled: cfg.Telemetry.Enabled,
		TelemetryName:    cfg.Telemetry.ServiceName,
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

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

func setupTelemetry(ctx context.Context, serviceName, endpoint string, enabled bool) (func(context.Context) error, error) {
	if !enabled {
		return nil, nil
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	))
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
