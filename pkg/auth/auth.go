package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Config 表示认证相关的配置。
type Config struct {
	Issuer     string
	Audience   string
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Manager 负责创建和校验 JWT。
type Manager struct {
	issuer     string
	audience   string
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// Claims 表示 JWT 的载荷集合。
type Claims struct {
	SessionID string   `json:"sid"`
	Subject   string   `json:"sub"`
	Roles     []string `json:"roles"`
	jwt.RegisteredClaims
}

var (
	mu      sync.RWMutex
	current *Manager
)

// NewManager 根据配置构造一个新的 Manager。
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Secret == "" {
		return nil, errors.New("auth: secret is required")
	}

	return &Manager{
		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
		secret:     []byte(cfg.Secret),
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
	}, nil
}

// Init 创建全局可用的认证管理器实例并完成初始化。
func Init(cfg Config) (*Manager, error) {
	manager, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}
	SetDefault(manager)
	return manager, nil
}

// SetDefault 将传入的 manager 记录为全局默认的认证管理器。
func SetDefault(manager *Manager) {
	mu.Lock()
	defer mu.Unlock()
	current = manager
}

// Default 返回已经配置的全局认证管理器。
func Default() *Manager {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// GenerateToken 根据会话 ID、主体和角色生成签名后的 JWT。
func (m *Manager) GenerateToken(sessionID string, subject string, roles []string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.accessTTL)
	claims := Claims{
		SessionID: sessionID,
		Subject:   subject,
		Roles:     roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, exp, nil
}

// ParseToken 校验传入的 JWT 字符串并返回解析后的载荷。
func (m *Manager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("auth: invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("auth: unexpected claim type")
	}

	return claims, nil
}

// AccessTTL 返回访问令牌的有效期配置。
func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}

// RefreshTTL 返回刷新令牌的有效期配置。
func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}
