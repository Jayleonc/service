package validation

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	current *validator.Validate
	mu      sync.RWMutex
)

// Init 创建新的校验器实例并设置为全局单例。
func Init() *validator.Validate {
	v := validator.New()
	SetDefault(v)
	return v
}

// SetDefault 将校验器设为全局默认实例。
func SetDefault(v *validator.Validate) {
	mu.Lock()
	defer mu.Unlock()
	current = v
}

// Default 返回当前配置的全局校验器。
func Default() *validator.Validate {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
