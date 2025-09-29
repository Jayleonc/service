package rbac

import (
	"github.com/google/uuid"

	"github.com/Jayleonc/service/pkg/model"
)

// Permission represents an action that can be executed on a specific resource.
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Resource    string    `gorm:"size:255;index:idx_permissions_resource_action,unique"`
	Action      string    `gorm:"size:255;index:idx_permissions_resource_action,unique"`
	Description string    `gorm:"size:512"`
	model.Base
}

// TableName overrides the default gorm table name.
func (Permission) TableName() string {
	return "permission"
}

// Role groups permissions and can be attached to a user.
type Role struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Name        string        `gorm:"size:255;uniqueIndex"`
	Description string        `gorm:"size:512"`
	Permissions []*Permission `gorm:"many2many:role_permissions;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	model.Base
}

// TableName overrides the default gorm table name.
func (Role) TableName() string {
	return "role"
}
