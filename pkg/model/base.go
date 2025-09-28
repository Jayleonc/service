package model

import (
	"time"

	"gorm.io/gorm"
)

// Base defines common timestamp fields for database models.
type Base struct {
	CreatedAt time.Time      `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"column:deleted_at;index"`
}
