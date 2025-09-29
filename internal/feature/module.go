package feature

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	authpkg "github.com/Jayleonc/service/pkg/auth"
	"github.com/Jayleonc/service/pkg/config"
)

// RouteRegistrar 定义功能模块注册 HTTP 路由所需的能力。
type RouteRegistrar interface {
	RegisterModule(pathPrefix string, routes ModuleRoutes)
	SetPermissionEnforcerFactory(func(permission string) gin.HandlerFunc)
	CollectedRoutePermissions() []string
}

// Dependencies 描述模块注册阶段可复用的通用基础设施。
type Dependencies struct {
	Logger             *slog.Logger
	DB                 *gorm.DB
	Engine             *gin.Engine
	Router             RouteRegistrar
	Auth               *authpkg.Manager
	Registry           *prometheus.Registry
	Config             config.App
	Cache              *redis.Client
	Validator          *validator.Validate
	Guards             *RouteGuards
	PermissionEnforcer func(permission string) gin.HandlerFunc
}

// Require 校验给定的依赖字段是否已经注入。
func (d *Dependencies) Require(names ...string) error {
	if len(names) == 0 {
		return nil
	}

	if d == nil {
		return fmt.Errorf("dependencies pointer is nil")
	}

	value := reflect.ValueOf(*d)

	for _, name := range names {
		field := value.FieldByName(name)
		if !field.IsValid() {
			return fmt.Errorf("dependency %q is not defined on feature.Dependencies", name)
		}

		kind := field.Kind()
		switch kind {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if field.IsNil() {
				return fmt.Errorf("dependency %q is required", name)
			}
		default:
			// 非指针类型无法判空，直接跳过。
		}
	}

	return nil
}

// Registrar 定义功能模块对外暴露的注册函数签名。
type Registrar func(context.Context, *Dependencies) error

// Entry 关联模块名称与对应的注册函数。
type Entry struct {
	Name      string
	Registrar Registrar
}
