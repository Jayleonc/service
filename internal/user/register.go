package user

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/feature"
)

// Register wires the user feature using the structured/DI development path.
func Register(ctx context.Context, deps feature.Dependencies) error {
	if deps.DB == nil {
		return fmt.Errorf("user feature requires a database instance")
	}
	if deps.Router == nil {
		return fmt.Errorf("user feature requires a route registrar")
	}
	if deps.Validator == nil {
		return fmt.Errorf("user feature requires a validator instance")
	}

	authService := auth.DefaultService()
	if authService == nil {
		return fmt.Errorf("user feature requires the auth service to be initialised")
	}

	repo := NewRepository(deps.DB)
	if err := repo.Migrate(ctx); err != nil {
		return fmt.Errorf("run user migrations: %w", err)
	}

	svc := NewService(repo, deps.Validator, authService)
	handler := NewHandler(svc)
	deps.Router.RegisterModule("", handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("user feature initialised", "pattern", "structured")
	}

	return nil
}
