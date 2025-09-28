package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	mu       sync.RWMutex
	registry *prometheus.Registry
)

// NewRegistry returns a new Prometheus registry instance.
func NewRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

// InitRegistry constructs a registry and stores it as the global instance.
func InitRegistry() *prometheus.Registry {
	reg := NewRegistry()
	SetDefault(reg)
	return reg
}

// SetDefault stores reg as the global metrics registry.
func SetDefault(reg *prometheus.Registry) {
	mu.Lock()
	defer mu.Unlock()
	registry = reg
}

// Default returns the global metrics registry.
func Default() *prometheus.Registry {
	mu.RLock()
	defer mu.RUnlock()
	return registry
}
