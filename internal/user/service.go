package user

import (
	"context"
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/internal/auth"
)

var (
	// ErrEmailExists indicates that the email is already registered.
	ErrEmailExists = errors.New("user: email already exists")
	// ErrInvalidCredentials represents invalid login credentials.
	ErrInvalidCredentials = errors.New("user: invalid credentials")
)

// Service coordinates user operations.
type Service struct {
	repo        *Repository
	validator   *validator.Validate
	authService *auth.Service
}

// RegisterInput defines payload for creating a user.
type RegisterInput struct {
	Name     string `validate:"required"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8"`
	Phone    string `validate:"omitempty"`
}

// LoginInput defines payload for logging in.
type LoginInput struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

// UpdateProfileInput defines payload for updating profile information.
type UpdateProfileInput struct {
	Name  *string `validate:"omitempty"`
	Phone *string `validate:"omitempty"`
}

// LoginResult captures the outcome of a successful login.
type LoginResult struct {
	Profile Profile
	Tokens  auth.Tokens
}

// NewService constructs a Service.
func NewService(repo *Repository, authService *auth.Service) *Service {
	return &Service{repo: repo, validator: validator.New(), authService: authService}
}

// Register persists a new user record.
func (s *Service) Register(ctx context.Context, input RegisterInput) (Profile, error) {
	if err := s.validator.Struct(input); err != nil {
		return Profile{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Profile{}, err
	}

	user := User{
		ID:           uuid.New(),
		Name:         input.Name,
		Email:        strings.ToLower(input.Email),
		Roles:        "user",
		PasswordHash: string(passwordHash),
		Phone:        input.Phone,
	}

	if err := s.repo.Create(ctx, &user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Profile{}, ErrEmailExists
		}
		return Profile{}, err
	}

	return toProfile(user), nil
}

// Login validates credentials and issues a new token pair.
func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	if err := s.validator.Struct(input); err != nil {
		return LoginResult{}, err
	}

	record, err := s.repo.GetByEmail(ctx, strings.ToLower(input.Email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(input.Password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	roles := parseRoles(record.Roles)
	tokens, err := s.authService.IssueTokens(ctx, record.ID, roles)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Profile: toProfile(*record), Tokens: tokens}, nil
}

// Profile retrieves the profile information for a user.
func (s *Service) Profile(ctx context.Context, id uuid.UUID) (Profile, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	return toProfile(*record), nil
}

// UpdateProfile modifies the profile information for a user.
func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (Profile, error) {
	if err := s.validator.Struct(input); err != nil {
		return Profile{}, err
	}

	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}

	if input.Name != nil {
		record.Name = *input.Name
	}
	if input.Phone != nil {
		record.Phone = *input.Phone
	}

	if err := s.repo.Update(ctx, record); err != nil {
		return Profile{}, err
	}

	return toProfile(*record), nil
}

func toProfile(u User) Profile {
	return Profile{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Roles:       parseRoles(u.Roles),
		Phone:       u.Phone,
		DateCreated: u.DateCreated,
		DateUpdated: u.DateUpdated,
	}
}

func parseRoles(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
