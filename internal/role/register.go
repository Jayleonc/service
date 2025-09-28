package role

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/module"
	"github.com/Jayleonc/service/pkg/database"
)

// Register initialises the role module following the singleton-friendly path.
func Register(ctx context.Context, deps module.Dependencies) error {
	db := database.Default()
	if db == nil {
		return fmt.Errorf("role module requires the global database to be initialised")
	}

	if err := db.WithContext(ctx).AutoMigrate(&assignment{}); err != nil {
		return fmt.Errorf("run role migrations: %w", err)
	}

	handler := NewHandler()
	handler.RegisterRoutes(deps.API)

	if deps.Logger != nil {
		deps.Logger.Info("role module initialised", "pattern", "singleton")
	}

	return nil
}
