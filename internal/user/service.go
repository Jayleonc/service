package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/rbac"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/ginx/paginator"
	"github.com/Jayleonc/service/pkg/ginx/request"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// Service 协调用户相关的业务操作。
type Service struct {
	repo        *Repository
	authService *auth.Service
	rbacService *rbac.Service
}

// RegisterInput 定义注册用户所需的入参结构。
type RegisterInput struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Phone    string `json:"phone" validate:"omitempty"`
}

// LoginInput 定义用户登录时的入参结构。
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UpdateProfileInput 定义用户自助更新资料的入参。
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

// LoginResult 描述登录成功后的返回结果。
type LoginResult struct {
	Profile Profile
	Tokens  auth.Tokens
}

// NewService 创建 Service 实例。
func NewService(repo *Repository, authService *auth.Service, rbacService *rbac.Service) *Service {
	return &Service{repo: repo, authService: authService, rbacService: rbacService}
}

// Register 持久化新用户。
func (s *Service) Register(ctx context.Context, input RegisterInput) (Profile, error) {
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
		ID:           uuid.Must(uuid.NewV7()),
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

// Login 校验凭证并签发新的令牌对。
func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
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

// Profile 查询用户的个人资料。
func (s *Service) Profile(ctx context.Context, id uuid.UUID) (Profile, error) {
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	return toProfile(*record), nil
}

// UpdateProfile 修改用户个人资料。
func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (Profile, error) {
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
		ID:           uuid.Must(uuid.NewV7()),
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
	return s.repo.Delete(ctx, req.ID)
}

// ListUsers 使用统一分页返回用户列表
func (s *Service) ListUsers(ctx context.Context, req ListUsersRequest) (*response.PageResult[Profile], error) {
	query := s.repo.Query(ctx)
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Email != "" {
		query = query.Where("email LIKE ?", "%"+req.Email+"%")
	}

	pageResult, err := paginator.Paginate[User](query, &req.Pagination)
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(pageResult.List))
	for _, user := range pageResult.List {
		profiles = append(profiles, toProfile(user))
	}

	return &response.PageResult[Profile]{
		List:     profiles,
		Total:    pageResult.Total,
		Page:     pageResult.Page,
		PageSize: pageResult.PageSize,
	}, nil
}

// AssignRoles 管理员分配角色
func (s *Service) AssignRoles(ctx context.Context, req AssignRolesRequest) (Profile, error) {
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

func (s *Service) rolesByNames(ctx context.Context, names []string) ([]*rbac.Role, error) {
	roles, err := s.rbacService.GetRolesByNames(ctx, names)
	if err != nil {
		if errors.Is(err, rbac.ErrResourceNotFound) {
			return nil, ErrRolesRequired
		}
		return nil, err
	}

	if len(roles) == 0 {
		return nil, ErrRolesRequired
	}

	return roles, nil
}

func roleNames(roles []*rbac.Role) []string {
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
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toProfile(u User) Profile {
	return Profile{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Roles:     roleNames(u.Roles),
		Phone:     u.Phone,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
