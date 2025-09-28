package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// userRecord represents the persisted user model.
type userRecord struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string
	Email        string `gorm:"uniqueIndex"`
	Roles        string
	PasswordHash string
	DateCreated  time.Time
	DateUpdated  time.Time
}

// BeforeCreate hook to set defaults.
func (u *userRecord) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	if u.DateCreated.IsZero() {
		u.DateCreated = now
	}
	u.DateUpdated = now
	return nil
}

// BeforeUpdate ensures DateUpdated is refreshed.
func (u *userRecord) BeforeUpdate(_ *gorm.DB) error {
	u.DateUpdated = time.Now().UTC()
	return nil
}

// Repository provides database access for users.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create persists a new user.
func (r *Repository) Create(ctx context.Context, user *userRecord) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user record.
func (r *Repository) Update(ctx context.Context, user *userRecord) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete removes a user by ID.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&userRecord{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Get retrieves a user by ID.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*userRecord, error) {
	var user userRecord
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*userRecord, error) {
	var user userRecord
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// List returns all users.
func (r *Repository) List(ctx context.Context) ([]userRecord, error) {
	var users []userRecord
	if err := r.db.WithContext(ctx).Order("date_created ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Migrate performs the schema migration for user repository.
func (r *Repository) Migrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&userRecord{})
}

// ErrDuplicateEmail is returned when an email already exists.
var ErrDuplicateEmail = errors.New("auth: repository: duplicate email")

// handleErrors translate common database errors to domain errors.
func handleErrors(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicateEmail
	}
	return err
}
