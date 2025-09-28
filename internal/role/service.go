package role

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/pkg/database"
)

// ErrUnavailable indicates that the role subsystem is not ready for use.
var ErrUnavailable = errors.New("role: service unavailable")

// Assign replaces the roles assigned to a user with the provided collection.
func Assign(ctx context.Context, userID uuid.UUID, roles []string) error {
	db := database.Default()
	if db == nil {
		return ErrUnavailable
	}

	cleaned := normalizeRoles(roles)

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&Role{}).Error; err != nil {
			return err
		}
		if len(cleaned) == 0 {
			return nil
		}

		records := make([]Role, 0, len(cleaned))
		for _, roleName := range cleaned {
			records = append(records, Role{ID: uuid.New(), UserID: userID, Role: roleName})
		}

		return tx.Create(&records).Error
	})
}

// List retrieves the roles associated with the provided user identifier.
func List(ctx context.Context, userID uuid.UUID) ([]string, error) {
	db := database.Default()
	if db == nil {
		return nil, ErrUnavailable
	}

	var records []Role
	if err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("date_created ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	roles := make([]string, 0, len(records))
	for _, record := range records {
		roles = append(roles, record.Role)
	}
	return roles, nil
}

func normalizeRoles(roles []string) []string {
	if len(roles) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(roles))
	cleaned := make([]string, 0, len(roles))
	for _, roleName := range roles {
		trimmed := strings.ToLower(strings.TrimSpace(roleName))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}

	return cleaned
}
