package role

import (
	"github.com/Jayleonc/service/pkg/model"
	"github.com/google/uuid"
)

// Role 定义系统角色模型
type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"size:64;uniqueIndex"`
	Description string    `gorm:"size:255"`
	model.Base
}

func (Role) TableName() string {
	return "role"
}
