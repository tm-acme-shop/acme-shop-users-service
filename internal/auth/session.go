package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
)

const (
	sessionPrefix = "session:"
	sessionTTL    = 24 * time.Hour
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionInvalid  = errors.New("session invalid")
)

// Session represents a user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Active    bool      `json:"active"`
}

// SessionV1 represents a legacy session format.
// Deprecated: Use Session instead.
type SessionV1 struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	// TODO(TEAM-SEC): Remove legacy session format
}

// SessionService handles user session management.
type SessionService struct {
	client *redis.Client
	logger *logging.LoggerV2
}

// NewSessionService creates a new session service.
func NewSessionService(cfg config.RedisConfig) *SessionService {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &SessionService{
		client: client,
		logger: logging.NewLoggerV2("session-service"),
	}
}

// Create creates a new session for a user.
func (s *SessionService) Create(ctx context.Context, userID, email, role, ipAddress, userAgent string) (*Session, error) {
	sessionID := generateSessionID()
	now := time.Now()

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Email:     email,
		Role:      role,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Active:    true,
	}

	s.logger.Info("creating session", logging.Fields{
		"session_id": sessionID,
		"user_id":    userID,
	})

	data, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	key := sessionPrefix + sessionID
	if err := s.client.Set(ctx, key, data, sessionTTL).Err(); err != nil {
		s.logger.Error("failed to create session", logging.Fields{
			"error": err.Error(),
		})
		return nil, err
	}

	// Also track sessions by user ID for listing
	userSessionKey := "user_sessions:" + userID
	s.client.SAdd(ctx, userSessionKey, sessionID)

	return session, nil
}

// Get retrieves a session by ID.
func (s *SessionService) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := sessionPrefix + sessionID

	s.logger.Debug("getting session", logging.Fields{"session_id": sessionID})

	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		logging.Errorf("failed to get session %s: %v", sessionID, err)
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, ErrSessionInvalid
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	if !session.Active {
		return nil, ErrSessionInvalid
	}

	return &session, nil
}

// Delete deletes a session (logout).
func (s *SessionService) Delete(ctx context.Context, sessionID string) error {
	s.logger.Info("deleting session", logging.Fields{"session_id": sessionID})

	key := sessionPrefix + sessionID
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}

// DeleteAllForUser deletes all sessions for a user.
func (s *SessionService) DeleteAllForUser(ctx context.Context, userID string) error {
	s.logger.Info("deleting all sessions for user", logging.Fields{"user_id": userID})

	userSessionKey := "user_sessions:" + userID
	sessionIDs, err := s.client.SMembers(ctx, userSessionKey).Result()
	if err != nil {
		return err
	}

	for _, sessionID := range sessionIDs {
		key := sessionPrefix + sessionID
		s.client.Del(ctx, key)
	}

	s.client.Del(ctx, userSessionKey)

	return nil
}

// ListForUser lists all active sessions for a user.
func (s *SessionService) ListForUser(ctx context.Context, userID string) ([]*Session, error) {
	s.logger.Debug("listing sessions for user", logging.Fields{"user_id": userID})

	userSessionKey := "user_sessions:" + userID
	sessionIDs, err := s.client.SMembers(ctx, userSessionKey).Result()
	if err != nil {
		return nil, err
	}

	sessions := []*Session{}
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			// Remove expired/invalid sessions from set
			s.client.SRem(ctx, userSessionKey, sessionID)
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Refresh extends a session's expiration.
func (s *SessionService) Refresh(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ExpiresAt = time.Now().Add(sessionTTL)

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := sessionPrefix + sessionID
	return s.client.Set(ctx, key, data, sessionTTL).Err()
}

// Revoke marks a session as inactive without deleting it.
func (s *SessionService) Revoke(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Active = false

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := sessionPrefix + sessionID
	ttl := time.Until(session.ExpiresAt)
	return s.client.Set(ctx, key, data, ttl).Err()
}

func generateSessionID() string {
	// Simple session ID generation for demo
	return "sess-" + randomSessionString(24)
}

func randomSessionString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

// ValidateSessionLegacy validates a legacy session.
// Deprecated: Use Get instead.
// TODO(TEAM-SEC): Remove after session migration
func (s *SessionService) ValidateSessionLegacy(ctx context.Context, sessionID string) (string, error) {
	logging.Infof("validating legacy session: %s", sessionID)

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return "", err
	}

	return session.UserID, nil
}

// CreateSessionLegacy creates a legacy session.
// Deprecated: Use Create instead.
// TODO(TEAM-SEC): Remove after session migration
func (s *SessionService) CreateSessionLegacy(ctx context.Context, userID string) (string, error) {
	logging.Infof("creating legacy session for user: %s", userID)

	session, err := s.Create(ctx, userID, "", "", "", "")
	if err != nil {
		return "", err
	}

	return session.ID, nil
}

// Ping checks if the session store is accessible.
func (s *SessionService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
