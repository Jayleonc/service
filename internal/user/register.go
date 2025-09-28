package user

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/module"
	"github.com/Jayleonc/service/internal/role"
	"github.com/Jayleonc/service/internal/server"
)

// Register wires the user module using the structured/DI development path.
func Register(ctx context.Context, deps module.Dependencies) error {
	if deps.DB == nil {
		return fmt.Errorf("user module requires a database instance")
	}
	if deps.API == nil {
		return fmt.Errorf("user module requires an API router group")
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
	server.RegisterModuleRoutes(deps.API, authService, handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("user module initialised", "pattern", "structured")
	}

	return nil
}
