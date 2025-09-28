package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type sessionPayload struct {
	UserID       string   `json:"user_id"`
	Roles        []string `json:"roles"`
	RefreshToken string   `json:"refresh_token"`
}

// SessionData captures the authenticated user context persisted in Redis.
type SessionData struct {
	SessionID    string
	UserID       uuid.UUID
	Roles        []string
	RefreshToken string
}

// SessionStore encapsulates Redis interactions for authentication sessions.
type SessionStore struct {
	client *redis.Client
}

// NewSessionStore constructs a new SessionStore.
func NewSessionStore(client *redis.Client) *SessionStore {
	return &SessionStore{client: client}
}

func (s *SessionStore) sessionKey(id string) string {
	return fmt.Sprintf("session:%s", id)
}

func (s *SessionStore) refreshKey(token string) string {
	return fmt.Sprintf("refresh:%s", token)
}

// Save persists a new session and its refresh token mapping.
func (s *SessionStore) Save(ctx context.Context, data SessionData, ttl time.Duration) error {
	payload := sessionPayload{
		UserID:       data.UserID.String(),
		Roles:        data.Roles,
		RefreshToken: data.RefreshToken,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, s.sessionKey(data.SessionID), raw, ttl)
	pipe.Set(ctx, s.refreshKey(data.RefreshToken), data.SessionID, ttl)

	_, err = pipe.Exec(ctx)
	return err
}

// Get retrieves a session by its identifier.
func (s *SessionStore) Get(ctx context.Context, sessionID string) (SessionData, error) {
	raw, err := s.client.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return SessionData{}, ErrSessionNotFound
		}
		return SessionData{}, err
	}

	var payload sessionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return SessionData{}, err
	}

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return SessionData{}, err
	}

	return SessionData{
		SessionID:    sessionID,
		UserID:       userID,
		Roles:        payload.Roles,
		RefreshToken: payload.RefreshToken,
	}, nil
}

// GetByRefreshToken locates a session using the refresh token mapping.
func (s *SessionStore) GetByRefreshToken(ctx context.Context, refreshToken string) (SessionData, error) {
	sessionID, err := s.client.Get(ctx, s.refreshKey(refreshToken)).Result()
	if err != nil {
		if err == redis.Nil {
			return SessionData{}, ErrInvalidRefreshToken
		}
		return SessionData{}, err
	}

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		if err == ErrSessionNotFound {
			return SessionData{}, ErrInvalidRefreshToken
		}
		return SessionData{}, err
	}

	return session, nil
}

// ReplaceRefreshToken rotates the refresh token associated with a session.
func (s *SessionStore) ReplaceRefreshToken(ctx context.Context, data SessionData, previousToken string, ttl time.Duration) error {
	payload := sessionPayload{
		UserID:       data.UserID.String(),
		Roles:        data.Roles,
		RefreshToken: data.RefreshToken,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, s.sessionKey(data.SessionID), raw, ttl)
	pipe.Set(ctx, s.refreshKey(data.RefreshToken), data.SessionID, ttl)
	if previousToken != "" {
		pipe.Del(ctx, s.refreshKey(previousToken))
	}

	_, err = pipe.Exec(ctx)
	return err
}
