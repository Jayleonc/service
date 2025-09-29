package rbac

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/pkg/constant"
)

// Service orchestrates RBAC operations.
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

var (
	serviceInstance *Service
)

// SetDefaultService stores the global service instance for other modules.
func SetDefaultService(svc *Service) {
	serviceInstance = svc
}

// DefaultService returns the globally registered service instance.
func DefaultService() *Service {
	return serviceInstance
}

// NewService creates a new Service.
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{repo: repo, validator: validator}
}

// CreateRoleInput defines the payload required to create a new role.
type CreateRoleInput struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"omitempty"`
}

// UpdateRoleInput defines the payload required to update an existing role.
type UpdateRoleInput struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	Name        *string   `json:"name" validate:"omitempty"`
	Description *string   `json:"description" validate:"omitempty"`
}

// DeleteRoleInput defines the payload required to delete a role.
type DeleteRoleInput struct {
	ID uuid.UUID `json:"id" validate:"required"`
}

// CreatePermissionInput defines the payload for creating a new permission.
type CreatePermissionInput struct {
	Resource    string `json:"resource" validate:"required"`
	Action      string `json:"action" validate:"required"`
	Description string `json:"description" validate:"omitempty"`
}

// UpdatePermissionInput defines the payload for updating an existing permission.
type UpdatePermissionInput struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	Resource    *string   `json:"resource" validate:"omitempty"`
	Action      *string   `json:"action" validate:"omitempty"`
	Description *string   `json:"description" validate:"omitempty"`
}

// DeletePermissionInput defines the payload for deleting a permission.
type DeletePermissionInput struct {
	ID uuid.UUID `json:"id" validate:"required"`
}

// AssignRolePermissionsInput defines the payload for assigning permissions to a role.
type AssignRolePermissionsInput struct {
	RoleID      uuid.UUID `json:"roleId" validate:"required"`
	Permissions []string  `json:"permissions" validate:"required,min=1,dive,required"`
}

// CreateRole creates a new role record.
func (s *Service) CreateRole(ctx context.Context, input CreateRoleInput) (*Role, error) {
	if err := s.validator.Struct(input); err != nil {
		return nil, err
	}

	name := NormalizeRoleName(input.Name)
	if name == "" {
		return nil, fmt.Errorf("role name is required")
	}

	role := &Role{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Description: strings.TrimSpace(input.Description),
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole updates an existing role record.
func (s *Service) UpdateRole(ctx context.Context, input UpdateRoleInput) (*Role, error) {
	if err := s.validator.Struct(input); err != nil {
		return nil, err
	}

	role, err := s.repo.FindRoleByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		normalized := NormalizeRoleName(*input.Name)
		if normalized == "" {
			return nil, fmt.Errorf("role name cannot be empty")
		}
		role.Name = normalized
	}
	if input.Description != nil {
		role.Description = strings.TrimSpace(*input.Description)
	}

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// DeleteRole removes a role record.
func (s *Service) DeleteRole(ctx context.Context, input DeleteRoleInput) error {
	if err := s.validator.Struct(input); err != nil {
		return err
	}
	return s.repo.DeleteRole(ctx, input.ID)
}

// ListRoles lists all roles.
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

// CreatePermission creates a new permission.
func (s *Service) CreatePermission(ctx context.Context, input CreatePermissionInput) (*Permission, error) {
	if err := s.validator.Struct(input); err != nil {
		return nil, err
	}

	resource := strings.ToLower(strings.TrimSpace(input.Resource))
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if resource == "" || action == "" {
		return nil, fmt.Errorf("resource and action are required")
	}

	permission := &Permission{
		ID:          uuid.Must(uuid.NewV7()),
		Resource:    resource,
		Action:      action,
		Description: strings.TrimSpace(input.Description),
	}

	if err := s.repo.CreatePermission(ctx, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

// UpdatePermission updates an existing permission record.
func (s *Service) UpdatePermission(ctx context.Context, input UpdatePermissionInput) (*Permission, error) {
	if err := s.validator.Struct(input); err != nil {
		return nil, err
	}

	permission, err := s.repo.FindPermissionByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Resource != nil {
		resource := strings.ToLower(strings.TrimSpace(*input.Resource))
		if resource == "" {
			return nil, fmt.Errorf("resource cannot be empty")
		}
		permission.Resource = resource
	}
	if input.Action != nil {
		action := strings.ToLower(strings.TrimSpace(*input.Action))
		if action == "" {
			return nil, fmt.Errorf("action cannot be empty")
		}
		permission.Action = action
	}
	if input.Description != nil {
		permission.Description = strings.TrimSpace(*input.Description)
	}

	if err := s.repo.UpdatePermission(ctx, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

// DeletePermission deletes an existing permission.
func (s *Service) DeletePermission(ctx context.Context, input DeletePermissionInput) error {
	if err := s.validator.Struct(input); err != nil {
		return err
	}
	return s.repo.DeletePermission(ctx, input.ID)
}

// ListPermissions returns all permissions.
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

// AssignPermissions assigns permissions to a role based on permission keys.
func (s *Service) AssignPermissions(ctx context.Context, input AssignRolePermissionsInput) (*Role, error) {
	if err := s.validator.Struct(input); err != nil {
		return nil, err
	}

	role, err := s.repo.FindRoleByID(ctx, input.RoleID)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(input.Permissions))
	for _, key := range input.Permissions {
		resource, action, ok := ParsePermissionKey(key)
		if !ok {
			continue
		}
		keys = append(keys, PermissionKey(resource, action))
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no valid permissions provided")
	}

	permissions, err := s.repo.FindPermissionsByKeys(ctx, keys)
	if err != nil {
		return nil, err
	}
	if len(permissions) == 0 {
		return nil, fmt.Errorf("permissions not found")
	}

	if err := s.repo.ReplaceRolePermissions(ctx, role, permissions); err != nil {
		return nil, err
	}
	role.Permissions = permissions
	return role, nil
}

// GetRolePermissions returns the permissions assigned to a role as permission keys.
func (s *Service) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	role, err := s.repo.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	if len(role.Permissions) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(role.Permissions))
	for _, permission := range role.Permissions {
		keys = append(keys, PermissionKey(permission.Resource, permission.Action))
	}
	return keys, nil
}

// GetRolesByNames returns roles for the provided names.
func (s *Service) GetRolesByNames(ctx context.Context, names []string) ([]*Role, error) {
	normalized := UniqueNormalized(names)
	if len(normalized) == 0 {
		return nil, nil
	}

	roles, err := s.repo.FindRolesByNames(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if len(roles) != len(normalized) {
		return nil, ErrResourceNotFound
	}
	return roles, nil
}

// Seed ensures that baseline roles and permissions exist.
func (s *Service) Seed(ctx context.Context) error {
	defaults := map[string]string{
		constant.RoleAdmin: "System administrator",
		constant.RoleUser:  "Standard user",
	}

	for name, desc := range defaults {
		normalized := NormalizeRoleName(name)
		if normalized == "" {
			continue
		}

		if _, err := s.repo.FindRoleByName(ctx, normalized); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if _, err := s.CreateRole(ctx, CreateRoleInput{Name: normalized, Description: desc}); err != nil {
					return err
				}
				continue
			}
			return err
		}
	}

	baselinePermissions := []CreatePermissionInput{
		{Resource: "user", Action: "create", Description: "Create users"},
		{Resource: "user", Action: "update", Description: "Update users"},
		{Resource: "user", Action: "delete", Description: "Delete users"},
		{Resource: "user", Action: "list", Description: "List users"},
		{Resource: "user", Action: "assign_roles", Description: "Assign roles to users"},
		{Resource: "rbac.role", Action: "create", Description: "Create roles"},
		{Resource: "rbac.role", Action: "update", Description: "Update roles"},
		{Resource: "rbac.role", Action: "delete", Description: "Delete roles"},
		{Resource: "rbac.role", Action: "list", Description: "List roles"},
		{Resource: "rbac.role", Action: "assign_permissions", Description: "Assign permissions to roles"},
		{Resource: "rbac.role", Action: "view_permissions", Description: "View role permissions"},
		{Resource: "rbac.permission", Action: "create", Description: "Create permissions"},
		{Resource: "rbac.permission", Action: "update", Description: "Update permissions"},
		{Resource: "rbac.permission", Action: "delete", Description: "Delete permissions"},
		{Resource: "rbac.permission", Action: "list", Description: "List permissions"},
	}

	for _, item := range baselinePermissions {
		resource := strings.ToLower(strings.TrimSpace(item.Resource))
		action := strings.ToLower(strings.TrimSpace(item.Action))
		key := PermissionKey(resource, action)
		existing, err := s.repo.FindPermissionsByKeys(ctx, []string{key})
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			continue
		}
		if _, err := s.CreatePermission(ctx, item); err != nil {
			return err
		}
	}

	adminRole, err := s.repo.FindRoleByName(ctx, NormalizeRoleName(constant.RoleAdmin))
	if err != nil {
		return err
	}

	allPermissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return err
	}

	permissions := make([]*Permission, 0, len(allPermissions))
	for i := range allPermissions {
		permissions = append(permissions, &allPermissions[i])
	}

	if err := s.repo.ReplaceRolePermissions(ctx, adminRole, permissions); err != nil {
		return err
	}

	return nil
}
