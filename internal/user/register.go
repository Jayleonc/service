package user

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/module"
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

	svc := NewService(repo, deps.Validator, authService)
	handler := NewHandler(svc, authService)
	handler.RegisterRoutes(deps.API)

	if deps.Logger != nil {
		deps.Logger.Info("user module initialised", "pattern", "structured")
	}

	return nil
}
