package user

import (
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/rbac"
	"github.com/Jayleonc/service/pkg/model"
)

// User 用户实体
type User struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey"`
	Name         string       `gorm:"size:255"`
	Email        string       `gorm:"size:255;uniqueIndex"`
	PasswordHash string       `gorm:"column:password_hash"`
	Phone        string       `gorm:"size:64"`
	Roles        []*rbac.Role `gorm:"many2many:user_roles;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	model.Base
}

func (u *User) TableName() string {
	return "user"
}
