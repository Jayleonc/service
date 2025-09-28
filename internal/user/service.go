package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/role"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/paginator"
	"github.com/Jayleonc/service/pkg/request"
	"github.com/Jayleonc/service/pkg/response"
)

var (
	// ErrEmailExists indicates that the email is already registered.
	ErrEmailExists = errors.New("user: email already exists")
	// ErrInvalidCredentials represents invalid login credentials.
	ErrInvalidCredentials = errors.New("user: invalid credentials")
	// ErrRolesRequired indicates that at least one role must be assigned.
	ErrRolesRequired = errors.New("user: at least one role must be assigned")
)

// Service coordinates user operations.
type Service struct {
	repo        *Repository
	roleRepo    *role.Repository
	validator   *validator.Validate
	authService *auth.Service
}

// RegisterInput defines payload for creating a user.
type RegisterInput struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Phone    string `json:"phone" validate:"omitempty"`
}

// LoginInput defines payload for logging in.
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UpdateProfileInput defines payload for updating profile information.
type UpdateProfileInput struct {
	Name  *string `json:"name" validate:"omitempty"`
	Phone *string `json:"phone" validate:"omitempty"`
}

// CreateUserRequest 管理员创建用户请求
type CreateUserRequest struct {
	Name     string   `json:"name" validate:"required"`
	Email    string   `json:"email" validate:"required,email"`
	Password string   `json:"password" validate:"required,min=8"`
	Phone    string   `json:"phone" validate:"omitempty"`
	Roles    []string `json:"roles" validate:"omitempty,dive,required"`
}

// UpdateUserRequest 管理员更新用户信息
type UpdateUserRequest struct {
	ID    uuid.UUID `json:"id" validate:"required"`
	Name  *string   `json:"name" validate:"omitempty"`
	Phone *string   `json:"phone" validate:"omitempty"`
}

// DeleteUserRequest 管理员删除用户
type DeleteUserRequest struct {
	ID uuid.UUID `json:"id" validate:"required"`
}

// AssignRolesRequest 管理员分配角色
type AssignRolesRequest struct {
	ID    uuid.UUID `json:"id" validate:"required"`
	Roles []string  `json:"roles" validate:"required,min=1,dive,required"`
}

// ListUsersRequest 用户分页请求
type ListUsersRequest struct {
	Pagination request.Pagination `json:"pagination"`
	Name       string             `json:"name"`
	Email      string             `json:"email"`
}

// LoginResult captures the outcome of a successful login.
type LoginResult struct {
	Profile Profile
	Tokens  auth.Tokens
}

// NewService constructs a Service.
func NewService(repo *Repository, roleRepo *role.Repository, validate *validator.Validate, authService *auth.Service) *Service {
	return &Service{repo: repo, roleRepo: roleRepo, validator: validate, authService: authService}
}

// Register persists a new user record.
func (s *Service) Register(ctx context.Context, input RegisterInput) (Profile, error) {
	if err := s.validator.Struct(input); err != nil {
		return Profile{}, err
	}

	roles, err := s.rolesByNames(ctx, []string{constant.RoleUser})
	if err != nil {
		return Profile{}, err
	}
	if len(roles) == 0 {
		return Profile{}, ErrRolesRequired
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Profile{}, err
	}

	user := &User{
		ID:           uuid.New(),
		Name:         input.Name,
		Email:        strings.ToLower(input.Email),
		PasswordHash: string(passwordHash),
		Phone:        input.Phone,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Profile{}, ErrEmailExists
		}
		return Profile{}, err
	}

	if err := s.repo.ReplaceRoles(ctx, user, roles); err != nil {
		return Profile{}, err
	}
	user.Roles = roles

	return toProfile(*user), nil
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

	roles := roleNames(record.Roles)
	if len(roles) == 0 {
		return LoginResult{}, ErrRolesRequired
	}

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

// CreateUser 管理员创建用户
func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (Profile, error) {
	if err := s.validator.Struct(req); err != nil {
		return Profile{}, err
	}

	targetRoles := req.Roles
	if len(targetRoles) == 0 {
		targetRoles = []string{constant.RoleUser}
	}

	roles, err := s.rolesByNames(ctx, targetRoles)
	if err != nil {
		return Profile{}, err
	}
	if len(roles) == 0 {
		return Profile{}, ErrRolesRequired
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Profile{}, err
	}

	user := &User{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(passwordHash),
		Phone:        req.Phone,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Profile{}, ErrEmailExists
		}
		return Profile{}, err
	}

	if err := s.repo.ReplaceRoles(ctx, user, roles); err != nil {
		return Profile{}, err
	}
	user.Roles = roles

	return toProfile(*user), nil
}

// UpdateUser 管理员更新用户
func (s *Service) UpdateUser(ctx context.Context, req UpdateUserRequest) (Profile, error) {
	if err := s.validator.Struct(req); err != nil {
		return Profile{}, err
	}

	record, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return Profile{}, err
	}

	if req.Name != nil {
		record.Name = *req.Name
	}
	if req.Phone != nil {
		record.Phone = *req.Phone
	}

	if err := s.repo.Update(ctx, record); err != nil {
		return Profile{}, err
	}

	return toProfile(*record), nil
}

// DeleteUser 管理员删除用户
func (s *Service) DeleteUser(ctx context.Context, req DeleteUserRequest) error {
	if err := s.validator.Struct(req); err != nil {
		return err
	}
	return s.repo.Delete(ctx, req.ID)
}

// ListUsers 使用统一分页返回用户列表
func (s *Service) ListUsers(ctx context.Context, req ListUsersRequest) (*response.PageResult, error) {
	query := s.repo.Query(ctx)
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Email != "" {
		query = query.Where("email LIKE ?", "%"+req.Email+"%")
	}

	var users []User
	pageResult, err := paginator.Paginate(query, &req.Pagination, &users)
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(users))
	for _, user := range users {
		profiles = append(profiles, toProfile(user))
	}

	pageResult.List = profiles
	return pageResult, nil
}

// AssignRoles 管理员分配角色
func (s *Service) AssignRoles(ctx context.Context, req AssignRolesRequest) (Profile, error) {
	if err := s.validator.Struct(req); err != nil {
		return Profile{}, err
	}

	record, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return Profile{}, err
	}

	roles, err := s.rolesByNames(ctx, req.Roles)
	if err != nil {
		return Profile{}, err
	}
	if len(roles) == 0 {
		return Profile{}, ErrRolesRequired
	}

	if err := s.repo.ReplaceRoles(ctx, record, roles); err != nil {
		return Profile{}, err
	}

	record.Roles = roles
	return toProfile(*record), nil
}

func (s *Service) rolesByNames(ctx context.Context, names []string) ([]role.Role, error) {
	roles, err := s.roleRepo.FindByNames(ctx, names)
	if err != nil {
		return nil, err
	}

	if len(roles) != len(uniqueNormalized(names)) {
		return nil, ErrRolesRequired
	}

	return roles, nil
}

func uniqueNormalized(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := strings.ToUpper(strings.TrimSpace(name))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func roleNames(roles []role.Role) []string {
	if len(roles) == 0 {
		return nil
	}
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		out = append(out, role.Name)
	}
	return out
}

// Profile 表示返回给客户端的安全用户信息。
type Profile struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Phone       string    `json:"phone"`
	DateCreated time.Time `json:"date_created"`
	DateUpdated time.Time `json:"date_updated"`
}

func toProfile(u User) Profile {
	return Profile{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Roles:       roleNames(u.Roles),
		Phone:       u.Phone,
		DateCreated: u.DateCreated,
		DateUpdated: u.DateUpdated,
	}
}
