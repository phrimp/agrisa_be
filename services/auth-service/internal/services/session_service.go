package services

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionService provides business logic for session management
type SessionService struct {
	sessionRepo repository.SessionRepository
}

// NewSessionService creates a new session service
func NewSessionService(sessionRepo repository.SessionRepository) *SessionService {
	return &SessionService{
		sessionRepo: sessionRepo,
	}
}

// CreateSession creates a new user session
func (s *SessionService) CreateSession(ctx context.Context, userID, tokenHash string, refreshTokenHash *string, deviceInfo, ipAddress *string) (*models.UserSession, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if tokenHash == "" {
		return nil, fmt.Errorf("token hash cannot be empty")
	}

	session := &models.UserSession{
		ID:               uuid.New().String(),
		UserID:           userID,
		TokenHash:        tokenHash,
		RefreshTokenHash: refreshTokenHash,
		DeviceInfo:       deviceInfo,
		IPAddress:        ipAddress,
		CreatedAt:        time.Now(),
		IsActive:         true,
	}

	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	return s.sessionRepo.GetSession(ctx, sessionID)
}

// ValidateSession checks if a session exists and is active
func (s *SessionService) ValidateSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	session, err := s.sessionRepo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	if !session.IsActive {
		return nil, fmt.Errorf("session is inactive")
	}

	return session, nil
}

// RenewSession extends the session expiration time
func (s *SessionService) RenewSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// First validate the session exists
	_, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("cannot renew session: %w", err)
	}

	return s.sessionRepo.RenewSession(ctx, sessionID)
}

// InvalidateSession marks a session as inactive and removes it
func (s *SessionService) InvalidateSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	return s.sessionRepo.DeleteSession(ctx, sessionID)
}

// InvalidateUserSessions removes all sessions for a user (useful for logout all devices)
func (s *SessionService) InvalidateUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	return s.sessionRepo.DeleteUserSessions(ctx, userID)
}

// GetUserSessions retrieves all active sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID string) ([]*models.UserSession, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	return s.sessionRepo.GetUserSessions(ctx, userID)
}

// GetSessionCount returns the number of active sessions for a user
func (s *SessionService) GetSessionCount(ctx context.Context, userID string) (int, error) {
	sessions, err := s.GetUserSessions(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user sessions: %w", err)
	}

	return len(sessions), nil
}

// CleanupExpiredSessions is a utility method to clean up expired sessions
// This would typically be called by a background job
func (s *SessionService) CleanupExpiredSessions(ctx context.Context, userID string) error {
	sessions, err := s.sessionRepo.GetUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user sessions for cleanup: %w", err)
	}

	now := time.Now()
	for _, session := range sessions {
		if now.After(session.ExpiresAt) {
			// Session is expired, delete it
			if err := s.sessionRepo.DeleteSession(ctx, session.ID); err != nil {
				// Log error but continue cleanup
				continue
			}
		}
	}

	return nil
}
