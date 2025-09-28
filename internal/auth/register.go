package auth

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/module"
)

// Register wires the auth module using the structured/DI development path.
func Register(ctx context.Context, deps module.Dependencies) error {
	if deps.DB == nil {
		return fmt.Errorf("auth module requires a database instance")
	}

	repo := NewRepository(deps.DB)
	if err := repo.Migrate(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	svc := NewService(repo)
	handler := NewHandler(svc)
	handler.RegisterRoutes(deps.API)

	if deps.Logger != nil {
		deps.Logger.Info("auth module initialised", "pattern", "structured")
	}

	return nil
}
