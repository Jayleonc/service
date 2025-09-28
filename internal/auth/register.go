package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/pkg/constant"
)

var (
	serviceMu sync.RWMutex
	service   *Service
)

// Register wires the authentication feature using the structured/DI development path.
func Register(ctx context.Context, deps feature.Dependencies) error {
	if deps.Auth == nil {
		return fmt.Errorf("auth feature requires an auth manager")
	}
	if deps.Cache == nil {
		return fmt.Errorf("auth feature requires a cache client")
	}
	if deps.Router == nil {
		return fmt.Errorf("auth feature requires a route registrar")
	}
	if deps.Guards == nil {
		return fmt.Errorf("auth feature requires route guards")
	}

	store := NewSessionStore(deps.Cache)
	svc := NewService(deps.Auth, store)
	setDefaultService(svc)

	handler := NewHandler(svc)
	deps.Router.RegisterModule("", handler.GetRoutes())

	deps.Guards.Authenticated = []gin.HandlerFunc{AuthenticatedMiddleware(svc)}
	deps.Guards.Admin = []gin.HandlerFunc{AuthenticatedMiddleware(svc), middleware.RBAC(constant.RoleAdmin)}

	if deps.Logger != nil {
		deps.Logger.Info("auth feature initialised", "pattern", "structured")
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
