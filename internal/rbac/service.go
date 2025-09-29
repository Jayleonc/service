package rbac

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/pkg/constant"
)

// RepositoryContract 定义了 Service 赖以运作的仓储能力。
type RepositoryContract interface {
	Migrate(ctx context.Context) error
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context) ([]Role, error)
	FindRoleByID(ctx context.Context, id uuid.UUID) (*Role, error)
	FindRoleByName(ctx context.Context, name string) (*Role, error)
	FindRolesByNames(ctx context.Context, names []string) ([]*Role, error)
	CreatePermission(ctx context.Context, permission *Permission) error
	UpdatePermission(ctx context.Context, permission *Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]Permission, error)
	FindPermissionByID(ctx context.Context, id uuid.UUID) (*Permission, error)
	FindPermissionsByKeys(ctx context.Context, keys []string) ([]*Permission, error)
	ReplaceRolePermissions(ctx context.Context, role *Role, permissions []*Permission) error
	UserHasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

// Service orchestrates RBAC operations.
type Service struct {
	repo RepositoryContract
}

var (
	serviceInstance *Service
	serviceMu       sync.RWMutex
)

// SetDefaultService stores the global service instance for other modules.
func SetDefaultService(svc *Service) {
	serviceMu.Lock()
	defer serviceMu.Unlock()
	serviceInstance = svc
}

// DefaultService returns the globally registered service instance.
func DefaultService() *Service {
	serviceMu.RLock()
	defer serviceMu.RUnlock()
	return serviceInstance
}

// NewService creates a new Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

var _ RepositoryContract = (*Repository)(nil)

// EnsureService initialises the default service instance if it has not been created yet.
func EnsureService(ctx context.Context, repo *Repository) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("rbac repository is required")
	}

	serviceMu.Lock()
	defer serviceMu.Unlock()

	if serviceInstance != nil {
		return serviceInstance, nil
	}

	if err := repo.Migrate(ctx); err != nil {
		return nil, err
	}

	svc := NewService(repo)
	if err := svc.ensureBaselineRoles(ctx); err != nil {
		return nil, err
	}

	serviceInstance = svc
	return serviceInstance, nil
}

// CreateRoleInput defines the payload required to create a new role.
type CreateRoleInput struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" binding:"omitempty"`
}

// UpdateRoleInput defines the payload required to update an existing role.
type UpdateRoleInput struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	Name        string    `json:"name" validate:"omitempty"`
	Description string    `json:"description" validate:"omitempty"`
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
	Resource    string    `json:"resource" validate:"omitempty"`
	Action      string    `json:"action" validate:"omitempty"`
	Description string    `json:"description" validate:"omitempty"`
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
	role, err := s.repo.FindRoleByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		normalized := NormalizeRoleName(input.Name)
		if normalized == "" {
			return nil, fmt.Errorf("role name cannot be empty")
		}
		role.Name = normalized
	}
	if input.Description != "" {
		role.Description = strings.TrimSpace(input.Description)
	}

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// DeleteRole removes a role record.
func (s *Service) DeleteRole(ctx context.Context, input DeleteRoleInput) error {
	return s.repo.DeleteRole(ctx, input.ID)
}

// ListRoles lists all roles.
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

// CreatePermission creates a new permission.
func (s *Service) CreatePermission(ctx context.Context, input CreatePermissionInput) (*Permission, error) {
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
	permission, err := s.repo.FindPermissionByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Resource != "" {
		resource := strings.ToLower(strings.TrimSpace(input.Resource))
		if resource == "" {
			return nil, fmt.Errorf("resource cannot be empty")
		}
		permission.Resource = resource
	}
	if input.Action != "" {
		action := strings.ToLower(strings.TrimSpace(input.Action))
		if action == "" {
			return nil, fmt.Errorf("action cannot be empty")
		}
		permission.Action = action
	}
	if input.Description != "" {
		permission.Description = strings.TrimSpace(input.Description)
	}

	if err := s.repo.UpdatePermission(ctx, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

// DeletePermission deletes an existing permission.
func (s *Service) DeletePermission(ctx context.Context, input DeletePermissionInput) error {
	return s.repo.DeletePermission(ctx, input.ID)
}

// ListPermissions returns all permissions.
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

// AssignPermissions assigns permissions to a role based on permission keys.
func (s *Service) AssignPermissions(ctx context.Context, input AssignRolePermissionsInput) (*Role, error) {
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

// EnsurePermissionsExist creates permissions in batch when they are missing from the database.
func (s *Service) EnsurePermissionsExist(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(keys))
	normalized := make([]string, 0, len(keys))
	for _, key := range keys {
		resource, action, ok := ParsePermissionKey(key)
		if !ok {
			continue
		}
		normalizedKey := PermissionKey(resource, action)
		if _, exists := unique[normalizedKey]; exists {
			continue
		}
		unique[normalizedKey] = struct{}{}
		normalized = append(normalized, normalizedKey)
	}

	if len(normalized) == 0 {
		return nil
	}

	existing, err := s.repo.FindPermissionsByKeys(ctx, normalized)
	if err != nil {
		return err
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, permission := range existing {
		key := PermissionKey(permission.Resource, permission.Action)
		existingSet[key] = struct{}{}
	}

	for key := range unique {
		if _, ok := existingSet[key]; ok {
			continue
		}

		resource, action, ok := ParsePermissionKey(key)
		if !ok {
			continue
		}

		permission := &Permission{
			ID:       uuid.Must(uuid.NewV7()),
			Resource: resource,
			Action:   action,
		}

		if err := s.repo.CreatePermission(ctx, permission); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				continue
			}
			return err
		}
	}

	return nil
}

// EnsureAdminHasAllPermissions grants the ADMIN role every available permission.
func (s *Service) EnsureAdminHasAllPermissions(ctx context.Context) error {
	if err := s.ensureBaselineRoles(ctx); err != nil {
		return err
	}

	adminRole, err := s.repo.FindRoleByName(ctx, NormalizeRoleName(constant.RoleAdmin))
	if err != nil {
		return err
	}

	allPermissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return err
	}

	if len(allPermissions) == 0 {
		return nil
	}

	permissions := make([]*Permission, 0, len(allPermissions))
	for i := range allPermissions {
		permissions = append(permissions, &allPermissions[i])
	}

	return s.repo.ReplaceRolePermissions(ctx, adminRole, permissions)
}

// HasPermission checks whether the given user owns the permission key.
func (s *Service) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	return s.repo.UserHasPermission(ctx, userID, permission)
}

func (s *Service) ensureBaselineRoles(ctx context.Context) error {
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
				role := &Role{
					ID:          uuid.Must(uuid.NewV7()),
					Name:        normalized,
					Description: strings.TrimSpace(desc),
				}
				if err := s.repo.CreateRole(ctx, role); err != nil {
					if errors.Is(err, gorm.ErrDuplicatedKey) {
						continue
					}
					return err
				}
				continue
			}
			return err
		}
	}

	return nil
}
