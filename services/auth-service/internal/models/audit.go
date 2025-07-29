package models

import "time"

type AuditLog struct {
	ID           int       `json:"id" db:"id"`
	UserID       *string   `json:"user_id" db:"user_id"`
	Action       string    `json:"action" db:"action"`
	ResourceType *string   `json:"resource_type" db:"resource_type"`
	ResourceID   *string   `json:"resource_id" db:"resource_id"`
	IPAddress    *string   `json:"ip_address" db:"ip_address"`
	Success      bool      `json:"success" db:"success"`
	ErrorMessage *string   `json:"error_message" db:"error_message"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

type PasswordHistory struct {
	ID           int       `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
