package role

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository 封装 Role 数据库操作
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建新的角色仓储
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Migrate 执行角色表迁移
func (r *Repository) Migrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&Role{})
}

// Create 创建新的角色记录
func (r *Repository) Create(ctx context.Context, role *Role) error {
	role.Name = normalizeRoleName(role.Name)
	return r.db.WithContext(ctx).Create(role).Error
}

// Update 更新角色信息
func (r *Repository) Update(ctx context.Context, role *Role) error {
	role.Name = normalizeRoleName(role.Name)
	return r.db.WithContext(ctx).Save(role).Error
}

// Delete 删除指定角色
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Role{}, "id = ?", id).Error
}

// Get 获取指定角色
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	var record Role
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByName 根据名称获取角色
func (r *Repository) GetByName(ctx context.Context, name string) (*Role, error) {
	var record Role
	if err := r.db.WithContext(ctx).First(&record, "name = ?", normalizeRoleName(name)).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// List 返回所有角色
func (r *Repository) List(ctx context.Context) ([]Role, error) {
	var records []Role
	if err := r.db.WithContext(ctx).Order("date_created ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// FindByNames 批量查询角色
func (r *Repository) FindByNames(ctx context.Context, names []string) ([]Role, error) {
	if len(names) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		trimmed := normalizeRoleName(name)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil, nil
	}

	var records []Role
	if err := r.db.WithContext(ctx).
		Where("name IN ?", normalized).
		Order("date_created ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func normalizeRoleName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
