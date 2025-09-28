package service

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/repository"
)

// User represents a domain user.
type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	Roles        []string
	PasswordHash string
}

// CreateUserInput defines payload for creating a user.
type CreateUserInput struct {
	Name         string   `validate:"required"`
	Email        string   `validate:"required,email"`
	Roles        []string `validate:"required"`
	PasswordHash string   `validate:"required"`
}

// UpdateUserInput defines payload for updating a user.
type UpdateUserInput struct {
	Name         *string   `validate:"omitempty"`
	Email        *string   `validate:"omitempty,email"`
	Roles        *[]string `validate:"omitempty"`
	PasswordHash *string   `validate:"omitempty"`
}

// UserService coordinates user operations.
type UserService struct {
	repo      *repository.UserRepository
	validator *validator.Validate
}

// NewUserService constructs a UserService.
func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo, validator: validator.New()}
}

// List returns all users.
func (s *UserService) List(ctx context.Context) ([]User, error) {
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
func (s *UserService) Create(ctx context.Context, input CreateUserInput) (User, error) {
	if err := s.validator.Struct(input); err != nil {
		return User{}, err
	}

	record := repository.User{
		Name:         input.Name,
		Email:        strings.ToLower(input.Email),
		Roles:        strings.Join(input.Roles, ","),
		PasswordHash: input.PasswordHash,
	}

	if err := repository.HandleErrors(s.repo.Create(ctx, &record)); err != nil {
		return User{}, err
	}

	return toDomain(record), nil
}

// Update modifies existing user information.
func (s *UserService) Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) (User, error) {
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
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Get fetches a single user by ID.
func (s *UserService) Get(ctx context.Context, id uuid.UUID) (User, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return User{}, err
	}

	return toDomain(*record), nil
}

func toDomain(u repository.User) User {
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
