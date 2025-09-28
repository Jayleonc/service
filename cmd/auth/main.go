package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ardanlabs/service/internal/config"
	"github.com/ardanlabs/service/pkg/auth"
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

	manager, err := auth.NewManager(auth.Config{
		Issuer:   cfg.Auth.Issuer,
		Audience: cfg.Auth.Audience,
		Secret:   cfg.Auth.Secret,
		Duration: cfg.Auth.TTL,
	})
	if err != nil {
		log.Error("failed to configure auth manager", "error", err)
		os.Exit(1)
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/token", func(c *gin.Context) {
		var req struct {
			Subject string   `json:"sub"`
			Roles   []string `json:"roles"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, err := manager.GenerateToken(req.Subject, req.Roles)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           r,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("starting auth server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("auth server stopped", "error", err)
		}
	}()

	<-ctx.Done()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Error("failed to shutdown auth server", "error", err)
	}
}
