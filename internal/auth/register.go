package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/Jayleonc/service/internal/feature"
)

var (
	serviceMu sync.RWMutex
	service   *Service
)

// Register wires the authentication module using the structured/DI development path.
func Register(ctx context.Context, deps feature.Dependencies) error {
	if deps.Auth == nil {
		return fmt.Errorf("auth module requires an auth manager")
	}
	if deps.Cache == nil {
		return fmt.Errorf("auth module requires a cache client")
	}
	if deps.Router == nil {
		return fmt.Errorf("auth module requires a route registrar")
	}

	store := NewSessionStore(deps.Cache)
	svc := NewService(deps.Auth, store)
	setDefaultService(svc)

	handler := NewHandler(svc)
	deps.Router.RegisterModule("/auth", handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("auth module initialised", "pattern", "structured")
	}

	return nil
}

// DefaultService returns the globally registered auth service instance.
func DefaultService() *Service {
	serviceMu.RLock()
	defer serviceMu.RUnlock()
	return service
}

func setDefaultService(svc *Service) {
	serviceMu.Lock()
	defer serviceMu.Unlock()
	service = svc
}
