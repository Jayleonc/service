package user

import (
	"time"

	"github.com/google/uuid"
)

// User 用户实体
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string    `gorm:"size:255"`
	Email        string    `gorm:"size:255;uniqueIndex"`
	Roles        string    `gorm:"size:512"`
	PasswordHash string    `gorm:"column:password_hash"`
	Phone        string    `gorm:"size:64"`
	DateCreated  time.Time `gorm:"column:date_created;autoCreateTime"`
	DateUpdated  time.Time `gorm:"column:date_updated;autoUpdateTime"`
}

func (u *User) TableName() string {
	return "user"
}

// Profile 表示返回给客户端的安全用户信息。
type Profile struct {
	ID          uuid.UUID
	Name        string
	Email       string
	Roles       []string
	Phone       string
	DateCreated time.Time
	DateUpdated time.Time
}
