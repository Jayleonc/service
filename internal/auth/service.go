package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/feature"
	authpkg "github.com/Jayleonc/service/pkg/auth"
)

var (
	// ErrSessionNotFound 表示会话数据在存储中不存在。
	ErrSessionNotFound = errors.New("auth: session not found")
)

// Tokens 表示一对签发完成的访问令牌与刷新令牌。
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// Service 基于 Redis 会话存储，负责无状态 JWT 的签发与校验。
type Service struct {
	manager    *authpkg.Manager
	store      *SessionStore
	refreshTTL time.Duration
}

// NewService 构造 Service 实例。
func NewService(manager *authpkg.Manager, store *SessionStore) *Service {
	return &Service{
		manager:    manager,
		store:      store,
		refreshTTL: manager.RefreshTTL(),
	}
}

// IssueTokens 创建新的认证会话并返回令牌对。
func (s *Service) IssueTokens(ctx context.Context, userID uuid.UUID, roles []string) (Tokens, error) {
	// 生成访问令牌和刷新令牌需要独立的随机标识符，保证每次登录互不干扰。
	sessionID := uuid.NewString()
	refreshToken := uuid.NewString()

	accessToken, _, err := s.manager.GenerateToken(sessionID, userID.String(), roles)
	if err != nil {
		return Tokens{}, err
	}

	// 将会话上下文写入 Redis，后续校验和刷新都会依赖这份数据。
	session := feature.AuthContext{
		SessionID:    sessionID,
		UserID:       userID,
		Roles:        roles,
		RefreshToken: refreshToken,
	}

	if err := s.store.Save(ctx, session, s.refreshTTL); err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.manager.AccessTTL(),
	}, nil
}

// Refresh 根据旧的刷新令牌生成新的访问令牌，并同时轮换刷新令牌。
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	session, err := s.store.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return Tokens{}, err
	}

	// 为防止刷新令牌被重放，每次刷新都生成新的随机值并立即替换旧值。
	newRefreshToken := uuid.NewString()
	session.RefreshToken = newRefreshToken

	accessToken, _, err := s.manager.GenerateToken(session.SessionID, session.UserID.String(), session.Roles)
	if err != nil {
		return Tokens{}, err
	}

	if err := s.store.ReplaceRefreshToken(ctx, session, refreshToken, s.refreshTTL); err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.manager.AccessTTL(),
	}, nil
}

// Validate 根据访问令牌解析出会话上下文。
func (s *Service) Validate(ctx context.Context, token string) (feature.AuthContext, error) {
	claims, err := s.manager.ParseToken(token)
	if err != nil {
		return feature.AuthContext{}, err
	}

	// 结合会话 ID 从 Redis 读取完整上下文，确保权限信息实时可控。
	session, err := s.store.Get(ctx, claims.SessionID)
	if err != nil {
		return feature.AuthContext{}, err
	}

	return session, nil
}
