package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents the persisted user model.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string
	Email        string `gorm:"uniqueIndex"`
	Roles        string
	PasswordHash string
	DateCreated  time.Time
	DateUpdated  time.Time
}

// BeforeCreate hook to set defaults.
func (u *User) BeforeCreate(tx *gorm.DB) error {
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
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.DateUpdated = time.Now().UTC()
	return nil
}

// UserRepository provides database access for users.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository constructs a UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create persists a new user.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user record.
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&User{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Get retrieves a user by ID.
func (r *UserRepository) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// List returns all users.
func (r *UserRepository) List(ctx context.Context) ([]User, error) {
	var users []User
	if err := r.db.WithContext(ctx).Order("date_created ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Migrate performs the schema migration for user repository.
func (r *UserRepository) Migrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&User{})
}

// ErrDuplicateEmail is returned when an email already exists.
var ErrDuplicateEmail = errors.New("repository: duplicate email")

// HandleErrors translate common database errors to domain errors.
func HandleErrors(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicateEmail
	}
	return err
}
