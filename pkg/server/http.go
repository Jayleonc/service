package server

import (
	"fmt"
	"net/http"
	"time"
)

// HTTPConfig configures an HTTP server.
type HTTPConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// NewHTTP constructs an *http.Server using cfg and handler.
func NewHTTP(cfg HTTPConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		ReadHeaderTimeout: 10 * time.Second,
	}
}
