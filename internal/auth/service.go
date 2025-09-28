package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	authpkg "github.com/Jayleonc/service/pkg/auth"
)

var (
	// ErrInvalidRefreshToken indicates that the provided refresh token is no longer valid.
	ErrInvalidRefreshToken = errors.New("auth: invalid refresh token")
	// ErrSessionNotFound is returned when a session cannot be located in the store.
	ErrSessionNotFound = errors.New("auth: session not found")
)

// Tokens represents an issued access and refresh token pair.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// Service manages stateless JWT creation backed by a Redis session store.
type Service struct {
	manager    *authpkg.Manager
	store      *SessionStore
	refreshTTL time.Duration
}

// NewService constructs a Service instance.
func NewService(manager *authpkg.Manager, store *SessionStore) *Service {
	return &Service{
		manager:    manager,
		store:      store,
		refreshTTL: manager.RefreshTTL(),
	}
}

// IssueTokens creates a new authenticated session and returns the token pair.
func (s *Service) IssueTokens(ctx context.Context, userID uuid.UUID, roles []string) (Tokens, error) {
	sessionID := uuid.NewString()
	refreshToken := uuid.NewString()

	accessToken, _, err := s.manager.GenerateToken(sessionID, userID.String(), roles)
	if err != nil {
		return Tokens{}, err
	}

	session := SessionData{
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

// Refresh rotates the refresh token and issues a new access token for the provided refresh token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	session, err := s.store.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return Tokens{}, err
	}

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

// Validate extracts the session from the access token.
func (s *Service) Validate(ctx context.Context, token string) (SessionData, error) {
	claims, err := s.manager.ParseToken(token)
	if err != nil {
		return SessionData{}, err
	}

	session, err := s.store.Get(ctx, claims.SessionID)
	if err != nil {
		return SessionData{}, err
	}

	return session, nil
}
