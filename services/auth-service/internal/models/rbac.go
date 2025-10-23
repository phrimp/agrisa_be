package models

import "time"

type Role struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UserRole struct {
	ID         int     `json:"id" db:"id"`
	UserID     string  `json:"user_id" db:"user_id"`
	RoleID     int     `json:"role_id" db:"role_id"`
	AssignedBy *string `json:"assigned_by" db:"assigned_by"`
	AssignedAt int64   `json:"assigned_at" db:"assigned_at"`
	ExpiresAt  int64   `json:"expires_at" db:"expires_at"`
	IsActive   bool    `json:"is_active" db:"is_active"`
}

type Permission struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type RolePermission struct {
	ID           int       `json:"id" db:"id"`
	RoleID       int       `json:"role_id" db:"role_id"`
	PermissionID int       `json:"permission_id" db:"permission_id"`
	GrantedAt    time.Time `json:"granted_at" db:"granted_at"`
}

type RoleHierarchy struct {
	ID           int       `json:"id" db:"id"`
	ParentRoleID int       `json:"parent_role_id" db:"parent_role_id"`
	ChildRoleID  int       `json:"child_role_id" db:"child_role_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

const (
	UserRoleID  int = 1
	AdminRoleID int = 2
)
