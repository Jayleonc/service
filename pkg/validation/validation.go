package validation

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	current *validator.Validate
	mu      sync.RWMutex
)

// Init creates a new validator instance and stores it as the default singleton.
func Init() *validator.Validate {
	v := validator.New()
	SetDefault(v)
	return v
}

// SetDefault stores v as the global validator instance.
func SetDefault(v *validator.Validate) {
	mu.Lock()
	defer mu.Unlock()
	current = v
}

// Default returns the globally configured validator instance.
func Default() *validator.Validate {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
