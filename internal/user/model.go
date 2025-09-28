package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents the persisted user entity.
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

// Profile represents the safe user information returned to clients.
type Profile struct {
	ID          uuid.UUID
	Name        string
	Email       string
	Roles       []string
	Phone       string
	DateCreated time.Time
	DateUpdated time.Time
}
