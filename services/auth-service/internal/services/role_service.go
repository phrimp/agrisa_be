package services

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"fmt"
	"time"
)

// RoleService provides business logic for role management
type RoleService struct {
	roleRepo repository.RoleRepository
}

// NewRoleService creates a new role service
func NewRoleService(roleRepo repository.RoleRepository) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
	}
}

// CreateRole creates a new role with validation
func (s *RoleService) CreateRole(name, displayName, description string) (*models.Role, error) {
	if name == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}
	if displayName == "" {
		return nil, fmt.Errorf("role display name cannot be empty")
	}

	// Check if role with same name already exists
	existing, err := s.roleRepo.GetRoleByName(name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("role with name '%s' already exists", name)
	}

	role := &models.Role{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		IsActive:    true,
	}

	err = s.roleRepo.CreateRole(role)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

// GetRole retrieves a role by ID
func (s *RoleService) GetRole(id int) (*models.Role, error) {
	return s.roleRepo.GetRoleByID(id)
}

// GetRoleByName retrieves a role by name
func (s *RoleService) GetRoleByName(name string) (*models.Role, error) {
	return s.roleRepo.GetRoleByName(name)
}

// GetAllRoles retrieves all roles with optional filtering
func (s *RoleService) GetAllRoles(activeOnly bool, limit, offset int) ([]*models.Role, error) {
	var active *bool
	if activeOnly {
		active = &activeOnly
	}
	return s.roleRepo.GetRoles(active, limit, offset)
}

// UpdateRole updates an existing role
func (s *RoleService) UpdateRole(role *models.Role) error {
	if role.ID <= 0 {
		return fmt.Errorf("invalid role ID")
	}
	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}
	if role.DisplayName == "" {
		return fmt.Errorf("role display name cannot be empty")
	}

	return s.roleRepo.UpdateRole(role)
}

// DeleteRole soft deletes a role
func (s *RoleService) DeleteRole(id int) error {
	return s.roleRepo.DeleteRole(id)
}

// ActivateRole activates a role
func (s *RoleService) ActivateRole(id int) error {
	return s.roleRepo.ActivateRole(id)
}

// DeactivateRole deactivates a role
func (s *RoleService) DeactivateRole(id int) error {
	return s.roleRepo.DeactivateRole(id)
}

// CreatePermission creates a new permission
func (s *RoleService) CreatePermission(name, resource, action, description string) (*models.Permission, error) {
	if name == "" {
		return nil, fmt.Errorf("permission name cannot be empty")
	}
	if resource == "" {
		return nil, fmt.Errorf("resource cannot be empty")
	}
	if action == "" {
		return nil, fmt.Errorf("action cannot be empty")
	}

	permission := &models.Permission{
		Name:        name,
		Resource:    resource,
		Action:      action,
		Description: description,
	}

	err := s.roleRepo.CreatePermission(permission)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return permission, nil
}

// GetAllPermissions retrieves all permissions with optional filtering
func (s *RoleService) GetAllPermissions(resource string, limit, offset int) ([]*models.Permission, error) {
	return s.roleRepo.GetPermissions(resource, limit, offset)
}

// GrantPermissionToRole grants a permission to a role
func (s *RoleService) GrantPermissionToRole(roleID, permissionID int) error {
	// Validate that role exists
	_, err := s.roleRepo.GetRoleByID(roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Validate that permission exists
	_, err = s.roleRepo.GetPermissionByID(permissionID)
	if err != nil {
		return fmt.Errorf("permission not found: %w", err)
	}

	return s.roleRepo.GrantPermissionToRole(roleID, permissionID)
}

// RevokePermissionFromRole revokes a permission from a role
func (s *RoleService) RevokePermissionFromRole(roleID, permissionID int) error {
	return s.roleRepo.RevokePermissionFromRole(roleID, permissionID)
}

// GetRolePermissions retrieves all permissions for a role
func (s *RoleService) GetRolePermissions(roleID int) ([]*models.Permission, error) {
	return s.roleRepo.GetRolePermissions(roleID)
}

// AssignRoleToUser assigns a role to a user
func (s *RoleService) AssignRoleToUser(userID string, roleID int, assignedBy *string, expiresAt *time.Time) error {
	// Validate that role exists and is active
	role, err := s.roleRepo.GetRoleByID(roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}
	if !role.IsActive {
		return fmt.Errorf("cannot assign inactive role")
	}

	return s.roleRepo.AssignRoleToUser(userID, roleID, assignedBy, expiresAt)
}

// RemoveRoleFromUser removes a role from a user
func (s *RoleService) RemoveRoleFromUser(userID string, roleID int) error {
	return s.roleRepo.RemoveRoleFromUser(userID, roleID)
}

// GetUserRoles retrieves all roles assigned to a user
func (s *RoleService) GetUserRoles(userID string, activeOnly bool) ([]*models.Role, error) {
	return s.roleRepo.GetUserRoles(userID, activeOnly)
}

// GetUserPermissions retrieves all permissions for a user through their roles
func (s *RoleService) GetUserPermissions(userID string) ([]*models.Permission, error) {
	return s.roleRepo.GetUserPermissions(userID)
}

// UserHasPermission checks if a user has a specific permission
func (s *RoleService) UserHasPermission(userID string, resource, action string) (bool, error) {
	return s.roleRepo.UserHasPermission(userID, resource, action)
}

// CreateRoleHierarchy creates a parent-child relationship between roles
func (s *RoleService) CreateRoleHierarchy(parentRoleID, childRoleID int) error {
	// Validate that both roles exist
	_, err := s.roleRepo.GetRoleByID(parentRoleID)
	if err != nil {
		return fmt.Errorf("parent role not found: %w", err)
	}

	_, err = s.roleRepo.GetRoleByID(childRoleID)
	if err != nil {
		return fmt.Errorf("child role not found: %w", err)
	}

	return s.roleRepo.CreateRoleHierarchy(parentRoleID, childRoleID)
}

// DeleteRoleHierarchy removes a parent-child relationship between roles
func (s *RoleService) DeleteRoleHierarchy(parentRoleID, childRoleID int) error {
	return s.roleRepo.DeleteRoleHierarchy(parentRoleID, childRoleID)
}

// GetEffectiveRolePermissions retrieves all permissions for a role including inherited ones
func (s *RoleService) GetEffectiveRolePermissions(roleID int) ([]*models.Permission, error) {
	return s.roleRepo.GetEffectiveRolePermissions(roleID)
}
