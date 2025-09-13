package models

import (
	"time"
)

type User struct {
	ID            string     `json:"id" db:"id"`
	PhoneNumber   string     `json:"phone_number" db:"phone_number"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"`
	NationalID    string     `json:"-" db:"national_id"`
	Status        UserStatus `json:"status" db:"status"`
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	PhoneVerified bool       `json:"phone_verified" db:"phone_verified"`
	KYCVerified   bool       `json:"kyc_verified" db:"kyc_verified"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin     *time.Time `json:"last_login" db:"last_login"`
	LoginAttempts int        `json:"login_attempts" db:"login_attempts"`
	LockedUntil   int64      `json:"locked_until" db:"locked_until"`
}

type UserStatus string

const (
	UserStatusActive              UserStatus = "active"
	UserStatusSuspended           UserStatus = "suspended"
	UserStatusPendingVerification UserStatus = "pending_verification"
	UserStatusDeactivated         UserStatus = "deactivated"
)

type UserEkycProgress struct {
	UserID         string     `json:"user_id" db:"user_id"`
	CicNo          string     `json:"cic_no" db:"cic_no"`
	IsOcrDone      bool       `json:"is_ocr_done" db:"is_ocr_done"`
	OcrDoneAt      *time.Time `json:"ocr_done_at" db:"ocr_done_at"`
	IsFaceVerified bool       `json:"is_face_verified" db:"is_face_verified"`
	FaceVerifiedAt *time.Time `json:"face_verified_at" db:"face_verified_at"`
}

type UserCard struct {
	NationalID        string `json:"national_id" db:"national_id"`
	Name              string `json:"name" db:"name"`
	Dob               string `json:"dob" db:"dob"`
	Sex               string `json:"sex" db:"sex"`
	Nationality       string `json:"nationality" db:"nationality"`
	Home              string `json:"home" db:"home"`
	Address           string `json:"address" db:"address"`
	Doe               string `json:"doe" db:"doe"`
	NumberOfNameLines string `json:"number_of_name_lines" db:"number_of_name_lines"`
	Features          string `json:"features" db:"features"`
	IssueDate         string `json:"issue_date" db:"issue_date"`
	Mrz               string `json:"mrz" db:"mrz"`
	IssueLoc          string `json:"issue_loc" db:"issue_loc"`
	ImageFront        string `json:"image_front" db:"image_front"`
	ImageBack         string `json:"image_back" db:"image_back"`
	UserID            string `json:"user_id" db:"user_id"`
}
