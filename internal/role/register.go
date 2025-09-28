package role

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/constant"
)

// Register initialises the role module following the singleton-friendly path.
func Register(ctx context.Context, deps feature.Dependencies) error {
	if deps.DB == nil {
		return fmt.Errorf("role module requires a database instance")
	}
	if deps.Router == nil {
		return fmt.Errorf("role module requires a route registrar")
	}
	if deps.Validator == nil {
		return fmt.Errorf("role module requires a validator instance")
	}

	authService := auth.DefaultService()
	if authService == nil {
		return fmt.Errorf("role module requires the auth service to be initialised")
	}

	repo := NewRepository(deps.DB)
	if err := repo.Migrate(ctx); err != nil {
		return fmt.Errorf("run role migrations: %w", err)
	}

	svc := NewService(repo, deps.Validator)
	if err := svc.EnsureDefaultRoles(ctx, map[string]string{
		constant.RoleAdmin: "Administrator",
		constant.RoleUser:  "Standard user",
	}); err != nil {
		return fmt.Errorf("ensure default roles: %w", err)
	}

	handler := NewHandler(svc)
	deps.Router.RegisterModule("/roles", handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("role module initialised", "pattern", "singleton")
	}

	return nil
}
