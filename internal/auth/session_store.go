package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Jayleonc/service/internal/feature"
)

type sessionPayload struct {
	UserID       string   `json:"userId"`
	Roles        []string `json:"roles"`
	RefreshToken string   `json:"refreshToken"`
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
func (s *SessionStore) Save(ctx context.Context, data feature.AuthContext, ttl time.Duration) error {
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
func (s *SessionStore) Get(ctx context.Context, sessionID string) (feature.AuthContext, error) {
	raw, err := s.client.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return feature.AuthContext{}, ErrSessionNotFound
		}
		return feature.AuthContext{}, err
	}

	var payload sessionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return feature.AuthContext{}, err
	}

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return feature.AuthContext{}, err
	}

	return feature.AuthContext{
		SessionID:    sessionID,
		UserID:       userID,
		Roles:        payload.Roles,
		RefreshToken: payload.RefreshToken,
	}, nil
}

// GetByRefreshToken locates a session using the refresh token mapping.
func (s *SessionStore) GetByRefreshToken(ctx context.Context, refreshToken string) (feature.AuthContext, error) {
	sessionID, err := s.client.Get(ctx, s.refreshKey(refreshToken)).Result()
	if err != nil {
		if err == redis.Nil {
			return feature.AuthContext{}, ErrInvalidRefreshToken
		}
		return feature.AuthContext{}, err
	}

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		if err == ErrSessionNotFound {
			return feature.AuthContext{}, ErrInvalidRefreshToken
		}
		return feature.AuthContext{}, err
	}

	return session, nil
}

// ReplaceRefreshToken rotates the refresh token associated with a session.
func (s *SessionStore) ReplaceRefreshToken(ctx context.Context, data feature.AuthContext, previousToken string, ttl time.Duration) error {
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
