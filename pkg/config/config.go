package config

import (
	"context"
	"errors"
	"strings"
	"sync"
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
	Redis     RedisConfig     `mapstructure:"redis"`
}

// ServerConfig controls HTTP server behaviour.
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DatabaseConfig represents Postgres configuration for GORM.
type DatabaseConfig struct {
	Driver   string            `mapstructure:"driver"`
	Host     string            `mapstructure:"host"`
	Port     int               `mapstructure:"port"`
	User     string            `mapstructure:"user"`
	Password string            `mapstructure:"password"`
	Name     string            `mapstructure:"name"`
	SSLMode  string            `mapstructure:"sslmode"`
	Params   map[string]string `mapstructure:"params"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Issuer     string        `mapstructure:"issuer"`
	Audience   string        `mapstructure:"audience"`
	Secret     string        `mapstructure:"secret"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
}

// LogConfig configures structured logging.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Pretty bool   `mapstructure:"pretty"`
}

// TelemetryConfig configures tracing exporters.
type TelemetryConfig struct {
	ServiceName string `mapstructure:"service_name"`
	Enabled     bool   `mapstructure:"enabled"`
	Endpoint    string `mapstructure:"endpoint"`
}

// RedisConfig describes the cache connection options.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

var (
	global App
	mu     sync.RWMutex
	set    bool
)

// Load reads configuration from the environment and optional CLI args.
func Load(_ context.Context, _ []string) (App, error) {
	v := viper.New()
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 3000)
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.write_timeout", "5s")

	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.name", "auth")
	v.SetDefault("database.sslmode", "disable")

	v.SetDefault("auth.issuer", "toolbox")
	v.SetDefault("auth.audience", "auth-service")
	v.SetDefault("auth.secret", "supersecret")
	v.SetDefault("auth.access_ttl", "15m")
	v.SetDefault("auth.refresh_ttl", "720h")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.pretty", false)

	v.SetDefault("telemetry.service_name", "auth-service")
	v.SetDefault("telemetry.enabled", false)
	v.SetDefault("telemetry.endpoint", "localhost:4317")

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.username", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetEnvPrefix("AUTH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFound) {
			return App{}, err
		}
	}

	var cfg App
	if err := v.Unmarshal(&cfg); err != nil {
		return App{}, err
	}
	return cfg, nil
}

// Init loads the configuration and stores it as the global instance.
func Init(ctx context.Context, args []string) (App, error) {
	cfg, err := Load(ctx, args)
	if err != nil {
		return App{}, err
	}
	Set(cfg)
	return cfg, nil
}

// Set stores cfg as the global configuration instance.
func Set(cfg App) {
	mu.Lock()
	defer mu.Unlock()
	global = cfg
	set = true
}

// Get retrieves the global configuration instance and whether it is set.
func Get() (App, bool) {
	mu.RLock()
	defer mu.RUnlock()
	return global, set
}

// MustGet returns the global configuration or panics if it has not been initialised.
func MustGet() App {
	if cfg, ok := Get(); ok {
		return cfg
	}
	panic("config: global configuration not initialised")
}
