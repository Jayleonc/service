package user

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/internal/role"
)

// Repository provides database access for users.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create persists a new user.
func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user record.
func (r *Repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete removes a user record by ID.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target := &User{ID: id}
		if err := tx.Model(target).Association("Roles").Clear(); err != nil {
			return err
		}
		return tx.Delete(&User{}, "id = ?", id).Error
	})
}

// Get retrieves a user by ID including roles.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email including roles.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Preload("Roles").First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Query returns a base query for listing users.
func (r *Repository) Query(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&User{}).Preload("Roles")
}

// ReplaceRoles 替换用户的角色集合
func (r *Repository) ReplaceRoles(ctx context.Context, user *User, roles []role.Role) error {
	return r.db.WithContext(ctx).Model(user).Association("Roles").Replace(roles)
}

// Migrate performs the schema migration for users.
func (r *Repository) Migrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&User{})
}
