package app

import (
	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/internal/server"
	"github.com/Jayleonc/service/internal/user"
)

// Features 列举了启动时需要初始化的全部业务模块。
var Features = []feature.Entry{
	{Name: "auth", Registrar: auth.Register},
	{Name: "user", Registrar: user.Register},
	// {Name: "rbac", Registrar: rbac.Register}, // 取消注释以启用高级RBAC插件（建议保持在列表末尾）
}

// Bootstrap 负责组装共享基础设施并注册每个业务模块。
func Bootstrap() (*server.App, error) {
	return server.Bootstrap(Features)
}
