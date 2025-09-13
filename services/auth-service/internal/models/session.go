package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserSession struct {
	ID               string    `json:"id" db:"id"`
	UserID           string    `json:"user_id" db:"user_id"`
	TokenHash        string    `json:"-" db:"token_hash"`
	RefreshTokenHash *string   `json:"-" db:"refresh_token_hash"`
	DeviceInfo       *string   `json:"device_info" db:"device_info"`
	IPAddress        *string   `json:"ip_address" db:"ip_address"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	IsActive         bool      `json:"is_active" db:"is_active"`
}

type APIKey struct {
	ID        int        `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	KeyHash   string     `json:"-" db:"key_hash"`
	KeyName   string     `json:"key_name" db:"key_name"`
	RateLimit int        `json:"rate_limit" db:"rate_limit"`
	ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	LastUsed  *time.Time `json:"last_used" db:"last_used"`
	IsActive  bool       `json:"is_active" db:"is_active"`
}

type APIKeyPermission struct {
	ID             int       `json:"id" db:"id"`
	APIKeyID       int       `json:"api_key_id" db:"api_key_id"`
	PermissionName string    `json:"permission_name" db:"permission_name"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type Claims struct {
	jwt.RegisteredClaims
	Id     string
	UserID string
	Email  string
	Phone  string
	Roles  []string
}
