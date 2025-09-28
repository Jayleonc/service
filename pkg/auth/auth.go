package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Config holds authentication configuration.
type Config struct {
	Issuer     string
	Audience   string
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Manager handles JWT creation and validation.
type Manager struct {
	issuer     string
	audience   string
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// Claims represents a JWT claim set.
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

// NewManager constructs a new Manager.
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

// Init constructs a new manager and stores it as the global instance.
func Init(cfg Config) (*Manager, error) {
	manager, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}
	SetDefault(manager)
	return manager, nil
}

// SetDefault records manager as the global authentication manager.
func SetDefault(manager *Manager) {
	mu.Lock()
	defer mu.Unlock()
	current = manager
}

// Default returns the global authentication manager, if one has been configured.
func Default() *Manager {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// GenerateToken creates a signed JWT using the provided subject and roles.
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

// ParseToken validates the JWT string and returns the claims.
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

// AccessTTL returns the configured lifespan for access tokens.
func (m *Manager) AccessTTL() time.Duration {
	return m.accessTTL
}

// RefreshTTL returns the configured lifespan for refresh tokens.
func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}
