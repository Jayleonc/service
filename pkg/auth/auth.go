package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Config holds authentication configuration.
type Config struct {
	Issuer   string
	Audience string
	Secret   string
	Duration time.Duration
}

// Manager handles JWT creation and validation.
type Manager struct {
	issuer   string
	audience string
	secret   []byte
	dur      time.Duration
}

// Claims represents a JWT claim set.
type Claims struct {
	Subject string   `json:"sub"`
	Roles   []string `json:"roles"`
	jwt.RegisteredClaims
}

// NewManager constructs a new Manager.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Secret == "" {
		return nil, errors.New("auth: secret is required")
	}

	return &Manager{
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		secret:   []byte(cfg.Secret),
		dur:      cfg.Duration,
	}, nil
}

// GenerateToken creates a signed JWT using the provided subject and roles.
func (m *Manager) GenerateToken(subject string, roles []string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Subject: subject,
		Roles:   roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.dur)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
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
