package user

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/internal/role"
)

// Repository 提供用户数据的数据库访问能力。
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建 Repository 实例。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 持久化新用户。
func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 更新已有的用户记录。
func (r *Repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 根据 ID 删除用户，并清理角色关联关系。
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target := &User{ID: id}
		if err := tx.Model(target).Association("Roles").Clear(); err != nil {
			return err
		}
		return tx.Delete(&User{}, "id = ?", id).Error
	})
}

// Get 根据 ID 查询用户并加载角色信息。
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱查询用户并加载角色信息。
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Preload("Roles").First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Query 返回用于列表查询的基础链式查询对象。
func (r *Repository) Query(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&User{}).Preload("Roles")
}

// ReplaceRoles 替换用户的角色集合
func (r *Repository) ReplaceRoles(ctx context.Context, user *User, roles []role.Role) error {
	return r.db.WithContext(ctx).Model(user).Association("Roles").Replace(roles)
}

// Migrate 执行用户表结构迁移。
func (r *Repository) Migrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&User{})
}
