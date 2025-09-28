package role

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;index"`
	Role        string
	DateCreated time.Time `gorm:"column:date_created;autoCreateTime"`
}

func (r *Role) TableName() string {
	return "role"
}
