package models

import (
	"time"
)

// Authentication DTOs
type LoginRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email      string `json:"email" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Password   string `json:"password" binding:"required"`
	NationalID string `json:"national_id" binding:"required"`
}

type LoginResponse struct {
	User        *User        `json:"user"`
	Session     *UserSession `json:"session"`
	AccessToken string       `json:"access_token"`
}

// Role Management DTOs
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
}

type AssignRoleRequest struct {
	AssignedBy *string    `json:"assigned_by"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type PermissionCheckRequest struct {
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// Permission Management DTOs
type CreatePermissionRequest struct {
	Name        string `json:"name" binding:"required"`
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	Description string `json:"description"`
}

type UpdatePermissionRequest struct {
	Name        string `json:"name" binding:"required"`
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	Description string `json:"description"`
}

// Response DTOs
type PaginatedRolesResponse struct {
	Roles  []*Role `json:"roles"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

type PaginatedPermissionsResponse struct {
	Permissions []*Permission `json:"permissions"`
	Total       int           `json:"total"`
	Limit       int           `json:"limit"`
	Offset      int           `json:"offset"`
}

// Common Error Response (simple version for handlers)
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// GetAllUsersResponse struct cho response
type GetAllUsersResponse struct {
	Users  []*User `json:"users"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

type VerifyIdentifierRequest struct {
	Identifier string `json:"identifier" binding:"required"`
}

type UpdateUserCardRequest struct {
	NationalID        *string `json:"national_id" db:"national_id"`
	Name              *string `json:"name" db:"name"`
	DOB               *string `json:"dob" db:"dob"`
	Sex               *string `json:"sex" db:"sex"`
	Nationality       *string `json:"nationality" db:"nationality"`
	Home              *string `json:"home" db:"home"`
	Address           *string `json:"address" db:"address"`
	DOE               *string `json:"doe" db:"doe"`
	NumberOfNameLines *string `json:"number_of_name_lines" db:"number_of_name_lines"`
	Features          *string `json:"features" db:"features"`
	IssueDate         *string `json:"issue_date" db:"issue_date"`
	MRZ               *string `json:"mrz" db:"mrz"`
	IssueLoc          *string `json:"issue_loc" db:"issue_loc"`
	ImageFront        *string `json:"image_front" db:"image_front"`
	ImageBack         *string `json:"image_back" db:"image_back"`
}
