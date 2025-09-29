package validation

import (
	"reflect"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	current *validator.Validate
	mu      sync.RWMutex
)

// Init 创建新的校验器实例并设置为全局单例。
func Init() *validator.Validate {
	v := validator.New()
	SetDefault(v)
	return v
}

// SetDefault 将校验器设为全局默认实例。
func SetDefault(v *validator.Validate) {
	mu.Lock()
	defer mu.Unlock()
	current = v
}

// Default 返回当前配置的全局校验器。
func Default() *validator.Validate {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// dualTagValidator implements gin/binding.StructValidator and validates
// using BOTH tag names: "binding" and "validate". Either failing will
// cause validation to fail. This lets users choose whichever tag they prefer.
type dualTagValidator struct {
	bindingV  *validator.Validate
	validateV *validator.Validate
}

// NewDualTagValidator constructs a StructValidator that supports both
// `binding:""` and `validate:""` tags.
func NewDualTagValidator() binding.StructValidator {
	v1 := validator.New()
	v1.SetTagName("binding")

	v2 := validator.New()
	v2.SetTagName("validate")

	return &dualTagValidator{
		bindingV:  v1,
		validateV: v2,
	}
}

func (d *dualTagValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	val := reflect.ValueOf(obj)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	if err := d.bindingV.Struct(obj); err != nil {
		return err
	}
	if err := d.validateV.Struct(obj); err != nil {
		return err
	}
	return nil
}

// Engine returns an underlying *validator.Validate for compatibility with code
// that expects binding.Validator.Engine().(*validator.Validate).
// We return the "binding"-tag instance by default.
func (d *dualTagValidator) Engine() any { return d.bindingV }
