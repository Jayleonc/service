package cache

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config describes Redis connection options for the cache layer.
type Config struct {
	Addr     string
	Password string
	DB       int
}

var (
	mu      sync.RWMutex
	current *redis.Client
)

// Init creates a Redis client, validates the connection and stores it as the default instance.
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

// SetDefault records client as the global cache instance.
func SetDefault(client *redis.Client) {
	mu.Lock()
	defer mu.Unlock()
	current = client
}

// Default returns the configured Redis client, if one has been initialised.
func Default() *redis.Client {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
