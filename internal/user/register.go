package user

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/role"
)

// Register wires the user module using the structured/DI development path.
func Register(ctx context.Context, deps feature.Dependencies) error {
	if deps.DB == nil {
		return fmt.Errorf("user module requires a database instance")
	}
	if deps.Router == nil {
		return fmt.Errorf("user module requires a route registrar")
	}
	if deps.Validator == nil {
		return fmt.Errorf("user module requires a validator instance")
	}

	authService := auth.DefaultService()
	if authService == nil {
		return fmt.Errorf("user module requires the auth service to be initialised")
	}

	repo := NewRepository(deps.DB)
	if err := repo.Migrate(ctx); err != nil {
		return fmt.Errorf("run user migrations: %w", err)
	}

	roleRepo := role.NewRepository(deps.DB)
	svc := NewService(repo, roleRepo, deps.Validator, authService)
	handler := NewHandler(svc)
	deps.Router.RegisterModule("/users", handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("user module initialised", "pattern", "structured")
	}

	return nil
}
