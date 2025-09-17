package repository

import (
	"auth-service/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionRepository handles session-related Redis operations
type SessionRepository interface {
	// Session CRUD operations
	CreateSession(ctx context.Context, session *models.UserSession) error
	GetSession(ctx context.Context, sessionID string) (*models.UserSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteUserSessions(ctx context.Context, userID string) error

	// Session management
	RenewSession(ctx context.Context, sessionID string) error
	IsSessionActive(ctx context.Context, sessionID string) (bool, error)
	GetUserSessions(ctx context.Context, userID string) ([]*models.UserSession, error)
}

// sessionRepository implements SessionRepository interface
type sessionRepository struct {
	client     *redis.Client
	expiration time.Duration
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(client *redis.Client) SessionRepository {
	return &sessionRepository{
		client:     client,
		expiration: 5 * time.Minute, // 5 minutes as requested
	}
}

func (r *sessionRepository) CreateSession(ctx context.Context, session *models.UserSession) error {
	if session.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if session.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Set session expiration time
	session.ExpiresAt = time.Now().Add(r.expiration)
	session.IsActive = true

	// Serialize session to JSON
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store session in Redis with expiration
	sessionKey := r.getSessionKey(session.ID)
	if err := r.client.Set(ctx, sessionKey, sessionData, r.expiration).Err(); err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Add session to user's session set
	userSessionsKey := r.getUserSessionsKey(session.UserID)
	if err := r.client.SAdd(ctx, userSessionsKey, session.ID).Err(); err != nil {
		return fmt.Errorf("failed to add session to user sessions: %w", err)
	}

	// Set expiration on user sessions set
	if err := r.client.Expire(ctx, userSessionsKey, r.expiration).Err(); err != nil {
		return fmt.Errorf("failed to set expiration on user sessions: %w", err)
	}

	return nil
}

// GetSession retrieves a session from Redis
func (r *sessionRepository) GetSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	sessionKey := r.getSessionKey(sessionID)
	sessionData, err := r.client.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session models.UserSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		r.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// DeleteSession removes a session from Redis
func (r *sessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Get session to find user ID
	session, err := r.GetSession(ctx, sessionID)
	if err != nil {
		// Session might already be deleted or expired, not an error
		return nil
	}

	sessionKey := r.getSessionKey(sessionID)
	userSessionsKey := r.getUserSessionsKey(session.UserID)

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	pipe.Del(ctx, sessionKey)
	pipe.SRem(ctx, userSessionsKey, sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteUserSessions removes all sessions for a user
func (r *sessionRepository) DeleteUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	userSessionsKey := r.getUserSessionsKey(userID)

	// Get all session IDs for the user
	sessionIDs, err := r.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // No sessions to delete
		}
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	// Delete all sessions
	pipe := r.client.Pipeline()
	for _, sessionID := range sessionIDs {
		sessionKey := r.getSessionKey(sessionID)
		pipe.Del(ctx, sessionKey)
	}
	pipe.Del(ctx, userSessionsKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// RenewSession extends the session expiration time
func (r *sessionRepository) RenewSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Get current session
	session, err := r.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session for renewal: %w", err)
	}

	// Update expiration time
	session.ExpiresAt = time.Now().Add(r.expiration)

	// Serialize updated session
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Update session in Redis with new expiration
	sessionKey := r.getSessionKey(sessionID)
	userSessionsKey := r.getUserSessionsKey(session.UserID)

	pipe := r.client.Pipeline()
	pipe.Set(ctx, sessionKey, sessionData, r.expiration)
	pipe.Expire(ctx, userSessionsKey, r.expiration)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to renew session: %w", err)
	}

	return nil
}

// IsSessionActive checks if a session exists and is active
func (r *sessionRepository) IsSessionActive(ctx context.Context, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("session ID cannot be empty")
	}

	session, err := r.GetSession(ctx, sessionID)
	if err != nil {
		return false, nil // Session doesn't exist or is expired
	}

	return session.IsActive, nil
}

// GetUserSessions retrieves all active sessions for a user
func (r *sessionRepository) GetUserSessions(ctx context.Context, userID string) ([]*models.UserSession, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	userSessionsKey := r.getUserSessionsKey(userID)
	sessionIDs, err := r.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		if err == redis.Nil {
			return []*models.UserSession{}, nil
		}
		return nil, fmt.Errorf("failed to get user session IDs: %w", err)
	}

	sessions := make([]*models.UserSession, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		session, err := r.GetSession(ctx, sessionID)
		if err != nil {
			// Session might be expired, skip it
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Helper methods

// getSessionKey generates Redis key for session
func (r *sessionRepository) getSessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

// getUserSessionsKey generates Redis key for user sessions set
func (r *sessionRepository) getUserSessionsKey(userID string) string {
	return fmt.Sprintf("user_sessions:%s", userID)
}
