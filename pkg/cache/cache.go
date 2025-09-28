package cache

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config 描述缓存层使用的 Redis 连接参数。
type Config struct {
	Addr     string
	Password string
	DB       int
}

var (
	mu      sync.RWMutex
	current *redis.Client
)

// Init 创建 Redis 客户端、校验连接并将其设置为默认实例。
func Init(cfg Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	SetDefault(client)
	return client, nil
}

// SetDefault 将客户端记录为全局默认的缓存实例。
func SetDefault(client *redis.Client) {
	mu.Lock()
	defer mu.Unlock()
	current = client
}

// Default 返回已经初始化的全局 Redis 客户端。
func Default() *redis.Client {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
