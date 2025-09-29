package rbac

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/feature"
)

// Register initialises the RBAC feature in a structured/DI fashion.
func Register(ctx context.Context, deps *feature.Dependencies) error {
	if err := deps.Require("DB", "Router"); err != nil {
		return fmt.Errorf("rbac feature dependencies: %w", err)
	}

	repo := NewRepository(deps.DB)
	svc, err := EnsureService(ctx, repo)
	if err != nil {
		return fmt.Errorf("ensure rbac service: %w", err)
	}

	factory := NewPermissionMiddleware(svc)
	if factory != nil {
		deps.PermissionEnforcer = factory
		deps.Router.UsePermissionEnforcer(factory)
	}

	handler := NewHandler(svc)
	deps.Router.RegisterModule("rbac", handler.GetRoutes())

	permissions := deps.Router.CollectedRoutePermissions()
	permissions = append(permissions, PermissionKey(ResourceSystem, ActionAdmin))

	if err := svc.EnsurePermissionsExist(ctx, permissions); err != nil {
		return fmt.Errorf("ensure permissions: %w", err)
	}

	if err := svc.EnsureAdminHasAllPermissions(ctx); err != nil {
		return fmt.Errorf("sync admin permissions: %w", err)
	}

	if deps.Logger != nil {
		deps.Logger.Info("rbac feature initialised", "pattern", "structured")
	}

	return nil
}
