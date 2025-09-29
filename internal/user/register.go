package user

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/rbac"
)

// Register 以结构化/依赖注入方式初始化用户功能。
func Register(ctx context.Context, deps feature.Dependencies) error {
	if err := deps.Require("DB", "Router", "Validator"); err != nil {
		return fmt.Errorf("user feature dependencies: %w", err)
	}

	authService := auth.DefaultService()
	if authService == nil {
		return fmt.Errorf("user feature requires the auth service to be initialised")
	}

	repo := NewRepository(deps.DB)
	if err := repo.Migrate(ctx); err != nil {
		return fmt.Errorf("run user migrations: %w", err)
	}

	rbacService := rbac.DefaultService()
	if rbacService == nil {
		return fmt.Errorf("user feature requires the rbac service to be initialised")
	}

	rbac.SetPermissionChecker(repo)

	svc := NewService(repo, deps.Validator, authService, rbacService)
	handler := NewHandler(svc)
	deps.Router.RegisterModule("user", handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("user feature initialised", "pattern", "structured")
	}

	return nil
}
