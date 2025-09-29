package rbac

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository provides database access for RBAC entities.
type Repository struct {
	db *gorm.DB
}

// NewRepository builds a new Repository instance.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Migrate ensures the required RBAC tables exist.
func (r *Repository) Migrate(ctx context.Context) error {
	if err := r.db.WithContext(ctx).AutoMigrate(&Permission{}, &Role{}); err != nil {
		return err
	}
	return nil
}

// CreateRole persists a new role.
func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// UpdateRole updates an existing role record.
func (r *Repository) UpdateRole(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// DeleteRole deletes a role by ID.
func (r *Repository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role := Role{ID: id}
		if err := tx.WithContext(ctx).Model(&role).Association("Permissions").Clear(); err != nil {
			return err
		}
		return tx.WithContext(ctx).Delete(&Role{}, "id = ?", id).Error
	})
}

// ListRoles returns all roles ordered by creation time.
func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	var roles []Role
	if err := r.db.WithContext(ctx).Preload("Permissions").Order("created_at ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// FindRoleByID returns a role by its identifier.
func (r *Repository) FindRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	var role Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// FindRoleByName retrieves a role by its name.
func (r *Repository) FindRoleByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// FindRolesByNames retrieves a collection of roles by their names.
func (r *Repository) FindRolesByNames(ctx context.Context, names []string) ([]*Role, error) {
	normalized := normalizeStrings(names)
	if len(normalized) == 0 {
		return nil, nil
	}

	var roles []*Role
	if err := r.db.WithContext(ctx).Preload("Permissions").Where("name IN ?", normalized).Order("created_at ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// CreatePermission persists a new permission.
func (r *Repository) CreatePermission(ctx context.Context, permission *Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

// UpdatePermission updates an existing permission record.
func (r *Repository) UpdatePermission(ctx context.Context, permission *Permission) error {
	return r.db.WithContext(ctx).Save(permission).Error
}

// DeletePermission deletes a permission by ID.
func (r *Repository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Permission{}, "id = ?", id).Error
}

// ListPermissions returns all permissions ordered by creation time.
func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
	var permissions []Permission
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// FindPermissionByID retrieves a permission by its identifier.
func (r *Repository) FindPermissionByID(ctx context.Context, id uuid.UUID) (*Permission, error) {
	var permission Permission
	if err := r.db.WithContext(ctx).First(&permission, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// FindPermissionsByKeys retrieves permissions by their computed keys.
func (r *Repository) FindPermissionsByKeys(ctx context.Context, keys []string) ([]*Permission, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	conditions := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys)*2)
	for _, key := range keys {
		resource, action, ok := ParsePermissionKey(key)
		if !ok {
			continue
		}
		conditions = append(conditions, "(resource = ? AND action = ?)")
		args = append(args, resource, action)
	}

	if len(conditions) == 0 {
		return nil, nil
	}

	query := r.db.WithContext(ctx).Model(&Permission{})
	query = query.Where(strings.Join(conditions, " OR "), args...)

	var permissions []*Permission
	if err := query.Order("created_at ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// ReplaceRolePermissions replaces the permission set for a role.
func (r *Repository) ReplaceRolePermissions(ctx context.Context, role *Role, permissions []*Permission) error {
	return r.db.WithContext(ctx).Model(role).Association("Permissions").Replace(permissions)
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToUpper(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
