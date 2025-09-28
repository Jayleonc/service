package config

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// App contains all configuration for the service.
type App struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Log       LogConfig       `mapstructure:"log"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type AuthConfig struct {
	Issuer   string        `mapstructure:"issuer"`
	Audience string        `mapstructure:"audience"`
	Secret   string        `mapstructure:"secret"`
	TTL      time.Duration `mapstructure:"ttl"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Pretty bool   `mapstructure:"pretty"`
}

type TelemetryConfig struct {
	ServiceName string `mapstructure:"service_name"`
	Enabled     bool   `mapstructure:"enabled"`
	Endpoint    string `mapstructure:"endpoint"`
}

// Parse loads configuration from the environment and command line arguments.
func Parse(_ context.Context, _ []string) (App, error) {
	v := viper.New()
	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 3000)
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.write_timeout", "5s")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.name", "service")
	v.SetDefault("database.sslmode", "disable")

	v.SetDefault("auth.issuer", "ardanlabs")
	v.SetDefault("auth.audience", "service")
	v.SetDefault("auth.secret", "supersecret")
	v.SetDefault("auth.ttl", "15m")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.pretty", false)

	v.SetDefault("telemetry.service_name", "sales-service")
	v.SetDefault("telemetry.enabled", false)
	v.SetDefault("telemetry.endpoint", "localhost:4317")

	// Environment variables
	v.SetEnvPrefix("SALES")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind legacy env var names used in compose/k8s
	_ = v.BindEnv("database.host", "SALES_DB_HOST")
	_ = v.BindEnv("database.port", "SALES_DB_PORT")
	_ = v.BindEnv("database.user", "SALES_DB_USER")
	_ = v.BindEnv("database.password", "SALES_DB_PASSWORD")
	_ = v.BindEnv("database.name", "SALES_DB_NAME")
	_ = v.BindEnv("database.sslmode", "SALES_DB_SSLMODE")

	var cfg App
	if err := v.Unmarshal(&cfg); err != nil {
		return App{}, err
	}
	return cfg, nil
}
