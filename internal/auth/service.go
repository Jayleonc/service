package auth

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// User represents a domain user.
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	PasswordHash string    `json:"password_hash"`
}

// CreateUserInput defines payload for creating a user.
type CreateUserInput struct {
	Name         string   `json:"name" validate:"required"`
	Email        string   `json:"email" validate:"required,email"`
	Roles        []string `json:"roles" validate:"required"`
	PasswordHash string   `json:"password_hash" validate:"required"`
}

// UpdateUserInput defines payload for updating a user.
type UpdateUserInput struct {
	Name         *string   `json:"name" validate:"omitempty"`
	Email        *string   `json:"email" validate:"omitempty,email"`
	Roles        *[]string `json:"roles" validate:"omitempty"`
	PasswordHash *string   `json:"password_hash" validate:"omitempty"`
}

// Service coordinates user operations.
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService constructs a Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo, validator: validator.New()}
}

// List returns all users.
func (s *Service) List(ctx context.Context) ([]User, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(records))
	for _, rec := range records {
		users = append(users, toDomain(rec))
	}

	return users, nil
}

// Create persists a new user record.
func (s *Service) Create(ctx context.Context, input CreateUserInput) (User, error) {
	if err := s.validator.Struct(input); err != nil {
		return User{}, err
	}

	record := userRecord{
		Name:         input.Name,
		Email:        strings.ToLower(input.Email),
		Roles:        strings.Join(input.Roles, ","),
		PasswordHash: input.PasswordHash,
	}

	if err := handleErrors(s.repo.Create(ctx, &record)); err != nil {
		return User{}, err
	}

	return toDomain(record), nil
}

// Update modifies existing user information.
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) (User, error) {
	if err := s.validator.Struct(input); err != nil {
		return User{}, err
	}

	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return User{}, err
	}

	if input.Name != nil {
		record.Name = *input.Name
	}
	if input.Email != nil {
		record.Email = strings.ToLower(*input.Email)
	}
	if input.Roles != nil {
		record.Roles = strings.Join(*input.Roles, ",")
	}
	if input.PasswordHash != nil {
		record.PasswordHash = *input.PasswordHash
	}

	if err := s.repo.Update(ctx, record); err != nil {
		return User{}, err
	}

	return toDomain(*record), nil
}

// Delete removes a user by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Get fetches a single user by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (User, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return User{}, err
	}

	return toDomain(*record), nil
}

func toDomain(u userRecord) User {
	roles := []string{}
	if u.Roles != "" {
		roles = strings.Split(u.Roles, ",")
	}

	return User{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Roles:        roles,
		PasswordHash: u.PasswordHash,
	}
}
