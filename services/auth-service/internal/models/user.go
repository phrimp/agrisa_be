package models

import "time"

type User struct {
	ID            string     `json:"id" db:"id"`
	PhoneNumber   *string    `json:"phone_number" db:"phone_number"`
	Email         *string    `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"`
	NationalID    *string    `json:"-" db:"national_id"`
	Status        UserStatus `json:"status" db:"status"`
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	PhoneVerified bool       `json:"phone_verified" db:"phone_verified"`
	KYCVerified   bool       `json:"kyc_verified" db:"kyc_verified"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin     *time.Time `json:"last_login" db:"last_login"`
	LoginAttempts int        `json:"login_attempts" db:"login_attempts"`
	LockedUntil   *time.Time `json:"locked_until" db:"locked_until"`
}

type UserStatus string

const (
	UserStatusActive              UserStatus = "active"
	UserStatusSuspended           UserStatus = "suspended"
	UserStatusPendingVerification UserStatus = "pending_verification"
	UserStatusDeactivated         UserStatus = "deactivated"
)
