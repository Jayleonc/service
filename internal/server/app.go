package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

// App coordinates the lifecycle of the HTTP server and shared infrastructure.
type App struct {
	server    *http.Server
	logger    *slog.Logger
	shutdowns []func(context.Context) error
}

// NewApp constructs an application instance ready to be started.
func NewApp(logger *slog.Logger, server *http.Server, shutdowns ...func(context.Context) error) *App {
	return &App{
		server:    server,
		logger:    logger,
		shutdowns: shutdowns,
	}
}

// Start begins serving HTTP traffic asynchronously and returns a channel that
// emits the outcome of the listener.
func (a *App) Start() <-chan error {
	errCh := make(chan error, 1)

	go func() {
		a.logger.Info("starting http server", "addr", a.server.Addr)

		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("http server terminated", "error", err)
			errCh <- err
			return
		}

		errCh <- nil
	}()

	return errCh
}

// Shutdown gracefully stops the HTTP server and runs any registered shutdown hooks.
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down http server")

	var joined error

	if err := a.server.Shutdown(ctx); err != nil {
		joined = errors.Join(joined, err)
	}

	for _, fn := range a.shutdowns {
		if fn == nil {
			continue
		}

		if err := fn(ctx); err != nil {
			a.logger.Error("shutdown hook failed", "error", err)
			joined = errors.Join(joined, err)
		}
	}

	return joined
}
