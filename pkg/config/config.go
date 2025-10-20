package config

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// App 聚合了服务运行所需的全部配置项。
type App struct {
	// Mode 控制应用运行模式（dev 或 prod）。
	Mode string `mapstructure:"mode"`
	// Server 控制 HTTP 服务监听与超时配置。
	Server ServerConfig `mapstructure:"server"`
	// Database 描述数据库驱动、连接信息与附加参数。
	Database DatabaseConfig `mapstructure:"database"`
	// Auth 定义认证系统的签发者、受众和令牌生命周期。
	Auth AuthConfig `mapstructure:"auth"`
	// Logger 控制日志级别、格式以及输出目标。
	Logger LoggerConfig `mapstructure:"logger"`
	// Telemetry 控制链路追踪导出行为。
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	// Redis 描述缓存服务的连接参数。
	Redis RedisConfig `mapstructure:"redis"`
}

// ServerConfig 控制 HTTP 服务器的基础行为。
type ServerConfig struct {
	// Host 指定服务监听的主机地址。
	Host string `mapstructure:"host"`
	// Port 指定 HTTP 服务监听端口。
	Port int `mapstructure:"port"`
	// ReadTimeout 配置请求读取阶段的超时时间。
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
	// WriteTimeout 配置响应写入阶段的超时时间。
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DatabaseConfig 描述使用 GORM 连接数据库所需的配置。
type DatabaseConfig struct {
	// Driver 指定数据库驱动类型（postgres、mysql 等）。
	Driver string `mapstructure:"driver"`
	// Host 指定数据库主机地址。
	Host string `mapstructure:"host"`
	// Port 指定数据库端口。
	Port int `mapstructure:"port"`
	// User 指定连接数据库使用的用户名。
	User string `mapstructure:"user"`
	// Password 指定连接数据库使用的密码。
	Password string `mapstructure:"password"`
	// Name 指定默认数据库名称。
	Name string `mapstructure:"name"`
	// SSLMode 定义 PostgreSQL 的 SSL 模式。
	SSLMode string `mapstructure:"sslmode"`
	// Params 用于附加自定义连接参数。
	Params map[string]string `mapstructure:"params"`
}

// AuthConfig 定义认证相关的配置。
type AuthConfig struct {
	// Issuer 为签发的令牌声明发行方。
	Issuer string `mapstructure:"issuer"`
	// Audience 指定令牌可接受的受众。
	Audience string `mapstructure:"audience"`
	// Secret 配置签名对称密钥。
	Secret string `mapstructure:"secret"`
	// AccessTTL 定义访问令牌的有效期。
	AccessTTL time.Duration `mapstructure:"access_ttl"`
	// RefreshTTL 定义刷新令牌的有效期。
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
}

// LoggerConfig 控制结构化日志的输出方式。
type LoggerConfig struct {
	// Level 指定日志级别，可选；若为空将根据 Mode 自动设置。
	Level *string `mapstructure:"level"`
	// Pretty 指定是否使用文本格式输出，可选；若为空将根据 Mode 自动设置。
	Pretty *bool `mapstructure:"pretty"`
	// Directory 指定启用文件日志的目录，可选。
	Directory string `mapstructure:"directory"`
}

// TelemetryConfig 描述链路追踪导出配置。
type TelemetryConfig struct {
	// ServiceName 指定上报时使用的服务名称。
	ServiceName string `mapstructure:"service_name"`
	// Enabled 控制是否启用链路追踪。
	Enabled bool `mapstructure:"enabled"`
	// Endpoint 指定 OTLP 采集端点地址。
	Endpoint string `mapstructure:"endpoint"`
}

// RedisConfig 描述缓存使用的 Redis 连接参数。
type RedisConfig struct {
	// Addr 指定 Redis 服务地址。
	Addr string `mapstructure:"addr"`
	// Username 指定连接使用的用户名。
	Username string `mapstructure:"username"`
	// Password 指定连接使用的密码。
	Password string `mapstructure:"password"`
	// DB 指定选择的数据库编号。
	DB int `mapstructure:"db"`
}

var (
	global App
	mu     sync.RWMutex
	set    bool
)

// Load 从环境变量与可选的 CLI 参数中读取配置。
func Load(_ context.Context, _ []string) (App, error) {
	v := viper.New()
	v.SetDefault("mode", "dev")
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

	v.SetDefault("logger.directory", "")

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

// Init 加载配置并注册全局配置实例。
func Init(ctx context.Context, args []string) (App, error) {
	cfg, err := Load(ctx, args)
	if err != nil {
		return App{}, err
	}
	Set(cfg)
	return cfg, nil
}

// Set 将 cfg 设为全局配置实例。
func Set(cfg App) {
	mu.Lock()
	defer mu.Unlock()
	global = cfg
	set = true
}

// Get 返回全局配置实例及其是否已设置。
func Get() (App, bool) {
	mu.RLock()
	defer mu.RUnlock()
	return global, set
}

// MustGet 返回全局配置，若未初始化则直接 panic。
func MustGet() App {
	if cfg, ok := Get(); ok {
		return cfg
	}
	panic("config: global configuration not initialised")
}
