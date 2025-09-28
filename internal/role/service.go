package role

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service 负责角色相关业务逻辑
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService 创建角色服务
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{repo: repo, validator: validator}
}

// CreateRoleRequest 创建角色请求体
type CreateRoleRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"omitempty"`
}

// UpdateRoleRequest 更新角色请求体
type UpdateRoleRequest struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	Name        *string   `json:"name" validate:"omitempty"`
	Description *string   `json:"description" validate:"omitempty"`
}

// DeleteRoleRequest 删除角色请求体
type DeleteRoleRequest struct {
	ID uuid.UUID `json:"id" validate:"required"`
}

// Create 创建角色
func (s *Service) Create(ctx context.Context, req CreateRoleRequest) (*Role, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, err
	}

	role := &Role{ID: uuid.New(), Name: req.Name, Description: req.Description}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// Update 更新角色
func (s *Service) Update(ctx context.Context, req UpdateRoleRequest) (*Role, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, err
	}

	role, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}

	if err := s.repo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

// Delete 删除角色
func (s *Service) Delete(ctx context.Context, req DeleteRoleRequest) error {
	if err := s.validator.Struct(req); err != nil {
		return err
	}
	return s.repo.Delete(ctx, req.ID)
}

// List 返回所有角色
func (s *Service) List(ctx context.Context) ([]Role, error) {
	return s.repo.List(ctx)
}

// FindByNames 根据角色名集合获取角色
func (s *Service) FindByNames(ctx context.Context, names []string) ([]Role, error) {
	return s.repo.FindByNames(ctx, names)
}

// EnsureDefaultRoles 确保默认角色存在
func (s *Service) EnsureDefaultRoles(ctx context.Context, defaults map[string]string) error {
	for name, desc := range defaults {
		normalized := normalizeRoleName(name)
		if normalized == "" {
			continue
		}

		_, err := s.repo.GetByName(ctx, normalized)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := s.repo.Create(ctx, &Role{ID: uuid.New(), Name: normalized, Description: desc}); err != nil {
					return err
				}
				continue
			}
			return err
		}
	}
	return nil
}
