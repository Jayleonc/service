package role

import (
	"time"

	"github.com/google/uuid"
)

// Role 定义系统角色模型
type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"size:64;uniqueIndex"`
	Description string    `gorm:"size:255"`
	DateCreated time.Time `gorm:"column:date_created;autoCreateTime"`
	DateUpdated time.Time `gorm:"column:date_updated;autoUpdateTime"`
}

func (Role) TableName() string {
	return "roles"
}
