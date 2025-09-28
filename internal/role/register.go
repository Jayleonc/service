package role

import (
	"context"
	"fmt"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/database"
)

// Register 以单例模式初始化角色功能模块。
func Register(ctx context.Context, deps feature.Dependencies) error {
	if err := deps.Require("Router"); err != nil {
		return fmt.Errorf("role feature dependencies: %w", err)
	}

	if database.Default() == nil {
		if deps.DB == nil {
			return fmt.Errorf("role feature requires a database instance")
		}
		database.SetDefault(deps.DB)
	}

	if err := Migrate(ctx); err != nil {
		return fmt.Errorf("run role migrations: %w", err)
	}

	if err := EnsureDefaultRoles(ctx, map[string]string{
		constant.RoleAdmin: "Administrator",
		constant.RoleUser:  "Standard user",
	}); err != nil {
		return fmt.Errorf("ensure default roles: %w", err)
	}

	handler := NewHandler()
	deps.Router.RegisterModule("", handler.GetRoutes())

	if deps.Logger != nil {
		deps.Logger.Info("role feature initialised", "pattern", "singleton")
	}

	return nil
}
