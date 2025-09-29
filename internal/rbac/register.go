package rbac

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/feature"
)

// Register initialises the RBAC feature in a structured/DI fashion.
func Register(ctx context.Context, deps feature.Dependencies) error {
	if err := deps.Require("DB", "Router", "Validator"); err != nil {
		return fmt.Errorf("rbac feature dependencies: %w", err)
	}

	repo := NewRepository(deps.DB)
	if err := repo.Migrate(ctx); err != nil {
		return fmt.Errorf("run rbac migrations: %w", err)
	}

	svc := NewService(repo, deps.Validator)
	SetDefaultService(svc)

	if err := svc.Seed(ctx); err != nil {
		return fmt.Errorf("seed rbac data: %w", err)
	}

	handler := NewHandler(svc)
	deps.Router.RegisterModule("rbac", handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("rbac feature initialised", "pattern", "structured")
	}

	return nil
}
