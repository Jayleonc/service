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

// RouteRegistrar exposes the capability needed by features to register their HTTP endpoints.
type RouteRegistrar interface {
	RegisterModule(pathPrefix string, routes ModuleRoutes)
}

// Dependencies captures the shared infrastructure that features can leverage during registration.
type Dependencies struct {
	Logger    *slog.Logger
	DB        *gorm.DB
	Engine    *gin.Engine
	Router    RouteRegistrar
	Auth      *authpkg.Manager
	Registry  *prometheus.Registry
	Config    config.App
	Cache     *redis.Client
	Validator *validator.Validate
	Guards    *RouteGuards
}

// Require ensures that the provided dependency names are non-nil pointer fields.
func (d Dependencies) Require(names ...string) error {
	if len(names) == 0 {
		return nil
	}

	value := reflect.ValueOf(d)

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
			// Non-nilable types are ignored as they cannot be nil.
		}
	}

	return nil
}

// Registrar declares the signature that features must implement to self-register.
type Registrar func(context.Context, Dependencies) error

// Entry associates a human-friendly name with a registrar implementation.
type Entry struct {
	Name      string
	Registrar Registrar
}
