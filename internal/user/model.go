package user

import (
	"time"

	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/role"
)

// User 用户实体
type User struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey"`
	Name         string      `gorm:"size:255"`
	Email        string      `gorm:"size:255;uniqueIndex"`
	PasswordHash string      `gorm:"column:password_hash"`
	Phone        string      `gorm:"size:64"`
	Roles        []role.Role `gorm:"many2many:user_roles;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	DateCreated  time.Time   `gorm:"column:date_created;autoCreateTime"`
	DateUpdated  time.Time   `gorm:"column:date_updated;autoUpdateTime"`
}

func (u *User) TableName() string {
	return "user"
}
