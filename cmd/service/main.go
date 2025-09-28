package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jayleonc/service/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	app, err := server.Bootstrap(ctx, os.Args[1:])
	if err != nil {
		slog.Error("failed to bootstrap application", "error", err)
		os.Exit(1)
	}

	errCh := app.Start()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
		}

		if err := <-errCh; err != nil && err != http.ErrServerClosed {
			slog.Error("server terminated unexpectedly", "error", err)
		}
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server terminated unexpectedly", "error", err)
		}
	}
}
