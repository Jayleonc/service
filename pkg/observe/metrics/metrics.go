package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	mu       sync.RWMutex
	registry *prometheus.Registry
)

// NewRegistry 返回一个新的 Prometheus Registry。
func NewRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

// InitRegistry 构造指标注册表并设置为全局实例。
func InitRegistry() *prometheus.Registry {
	reg := NewRegistry()
	SetDefault(reg)
	return reg
}

// SetDefault 将指标注册表设置为全局默认值。
func SetDefault(reg *prometheus.Registry) {
	mu.Lock()
	defer mu.Unlock()
	registry = reg
}

// Default 返回当前的全局指标注册表实例。
func Default() *prometheus.Registry {
	mu.RLock()
	defer mu.RUnlock()
	return registry
}
