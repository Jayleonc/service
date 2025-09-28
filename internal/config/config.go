package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ardanlabs/conf/v3"
)

// App contains all configuration for the service.
type App struct {
	Server struct {
		Host         string        `conf:"default:0.0.0.0"`
		Port         int           `conf:"default:3000"`
		ReadTimeout  time.Duration `conf:"default:5s"`
		WriteTimeout time.Duration `conf:"default:5s"`
	}
	Database struct {
		Host     string `conf:"default:localhost"`
		Port     int    `conf:"default:5432"`
		User     string `conf:"default:postgres"`
		Password string `conf:"default:postgres,mask"`
		Name     string `conf:"default:service"`
		SSLMode  string `conf:"default:disable"`
	}
	Auth struct {
		Issuer   string        `conf:"default:ardanlabs"`
		Audience string        `conf:"default:service"`
		Secret   string        `conf:"default:supersecret,mask"`
		TTL      time.Duration `conf:"default:15m"`
	}
	Log struct {
		Level  string `conf:"default:info"`
		Pretty bool   `conf:"default:false"`
	}
	Telemetry struct {
		ServiceName string `conf:"default:sales-service"`
		Enabled     bool   `conf:"default:false"`
		Endpoint    string `conf:"default:localhost:4317"`
	}
}

// Parse loads configuration from the environment and command line arguments.
func Parse(_ context.Context, args []string) (App, error) {
	var cfg App
	if len(args) > 0 {
		os.Args = append([]string{os.Args[0]}, args...)
	}

	usage, err := conf.Parse("SALES", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return App{}, fmt.Errorf("%w\n%s", err, usage)
		}
		return App{}, err
	}

	return cfg, nil
}
