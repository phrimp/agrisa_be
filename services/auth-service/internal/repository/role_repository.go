package repository

import (
	"auth-service/internal/models"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// RoleRepository handles role-related database operations
type RoleRepository interface {
	// Role CRUD operations
	CreateRole(role *models.Role) error
	GetRoleByID(id int) (*models.Role, error)
	GetRoleByName(name string) (*models.Role, error)
	GetRoles(active *bool, limit, offset int) ([]*models.Role, error)
	UpdateRole(role *models.Role) error
	DeleteRole(id int) error
	ActivateRole(id int) error
	DeactivateRole(id int) error

	// Permission operations
	CreatePermission(permission *models.Permission) error
	GetPermissionByID(id int) (*models.Permission, error)
	GetPermissions(resource string, limit, offset int) ([]*models.Permission, error)
	DeletePermission(id int) error

	// Role-Permission operations
	GrantPermissionToRole(roleID, permissionID int) error
	RevokePermissionFromRole(roleID, permissionID int) error
	GetRolePermissions(roleID int) ([]*models.Permission, error)
	GetRolesWithPermission(permissionID int) ([]*models.Role, error)
	HasPermission(roleID int, resource, action string) (bool, error)

	// User-Role operations
	AssignRoleToUser(userID string, roleID int, assignedBy *string, expiresAt *time.Time) error
	RemoveRoleFromUser(userID string, roleID int) error
	GetUserRoles(userID string, activeOnly bool) ([]*models.Role, error)
	GetRoleUsers(roleID int, activeOnly bool) ([]string, error)
	GetUserPermissions(userID string) ([]*models.Permission, error)
	UserHasPermission(userID string, resource, action string) (bool, error)

	// Role hierarchy operations
	CreateRoleHierarchy(parentRoleID, childRoleID int) error
	DeleteRoleHierarchy(parentRoleID, childRoleID int) error
	GetRoleChildren(roleID int) ([]*models.Role, error)
	GetRoleParents(roleID int) ([]*models.Role, error)
	GetEffectiveRolePermissions(roleID int) ([]*models.Permission, error)
}

// roleRepository implements RoleRepository interface
type roleRepository struct {
	db *sqlx.DB
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *sqlx.DB) RoleRepository {
	return &roleRepository{db: db}
}

// CreateRole creates a new role
func (r *roleRepository) CreateRole(role *models.Role) error {
	query := `
		INSERT INTO roles (name, display_name, description, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := r.db.QueryRow(query, role.Name, role.DisplayName, role.Description, role.IsActive).
		Scan(&role.ID, &role.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// GetRoleByID retrieves a role by its ID
func (r *roleRepository) GetRoleByID(id int) (*models.Role, error) {
	role := &models.Role{}
	query := `
		SELECT id, name, display_name, description, is_active, created_at
		FROM roles
		WHERE id = $1`

	err := r.db.Get(role, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get role by ID: %w", err)
	}

	return role, nil
}

// GetRoleByName retrieves a role by its name
func (r *roleRepository) GetRoleByName(name string) (*models.Role, error) {
	role := &models.Role{}
	query := `
		SELECT id, name, display_name, description, is_active, created_at
		FROM roles
		WHERE name = $1`

	err := r.db.Get(role, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role with name '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	return role, nil
}

// GetRoles retrieves roles with optional filtering
func (r *roleRepository) GetRoles(active *bool, limit, offset int) ([]*models.Role, error) {
	var roles []*models.Role
	var query string
	var args []interface{}

	baseQuery := `
		SELECT id, name, display_name, description, is_active, created_at
		FROM roles`

	conditions := []string{}
	argIndex := 1

	if active != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *active)
		argIndex++
	}

	if len(conditions) > 0 {
		query = baseQuery + " WHERE " + strings.Join(conditions, " AND ")
	} else {
		query = baseQuery
	}

	query += " ORDER BY name"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++

		if offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}
	}

	err := r.db.Select(&roles, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}

	return roles, nil
}

// UpdateRole updates an existing role
func (r *roleRepository) UpdateRole(role *models.Role) error {
	query := `
		UPDATE roles
		SET name = $2, display_name = $3, description = $4, is_active = $5
		WHERE id = $1`

	result, err := r.db.Exec(query, role.ID, role.Name, role.DisplayName, role.Description, role.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role with ID %d not found", role.ID)
	}

	return nil
}

// DeleteRole deletes a role (soft delete by setting is_active = false)
func (r *roleRepository) DeleteRole(id int) error {
	query := `UPDATE roles SET is_active = false WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role with ID %d not found", id)
	}

	return nil
}

// ActivateRole activates a role
func (r *roleRepository) ActivateRole(id int) error {
	query := `UPDATE roles SET is_active = true WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to activate role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role with ID %d not found", id)
	}

	return nil
}

// DeactivateRole deactivates a role
func (r *roleRepository) DeactivateRole(id int) error {
	query := `UPDATE roles SET is_active = false WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role with ID %d not found", id)
	}

	return nil
}

// CreatePermission creates a new permission
func (r *roleRepository) CreatePermission(permission *models.Permission) error {
	query := `
		INSERT INTO permissions (name, resource, action, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := r.db.QueryRow(query, permission.Name, permission.Resource, permission.Action, permission.Description).
		Scan(&permission.ID, &permission.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	return nil
}

// GetPermissionByID retrieves a permission by its ID
func (r *roleRepository) GetPermissionByID(id int) (*models.Permission, error) {
	permission := &models.Permission{}
	query := `
		SELECT id, name, resource, action, description, created_at
		FROM permissions
		WHERE id = $1`

	err := r.db.Get(permission, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get permission by ID: %w", err)
	}

	return permission, nil
}

// GetPermissions retrieves permissions with optional resource filtering
func (r *roleRepository) GetPermissions(resource string, limit, offset int) ([]*models.Permission, error) {
	var permissions []*models.Permission
	var query string
	var args []interface{}

	baseQuery := `
		SELECT id, name, resource, action, description, created_at
		FROM permissions`

	argIndex := 1

	if resource != "" {
		query = baseQuery + " WHERE resource = $1"
		args = append(args, resource)
		argIndex++
	} else {
		query = baseQuery
	}

	query += " ORDER BY resource, action"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++

		if offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}
	}

	err := r.db.Select(&permissions, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	return permissions, nil
}

// DeletePermission deletes a permission
func (r *roleRepository) DeletePermission(id int) error {
	query := `DELETE FROM permissions WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("permission with ID %d not found", id)
	}

	return nil
}

// GrantPermissionToRole grants a permission to a role
func (r *roleRepository) GrantPermissionToRole(roleID, permissionID int) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING`

	_, err := r.db.Exec(query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to grant permission to role: %w", err)
	}

	return nil
}

// RevokePermissionFromRole revokes a permission from a role
func (r *roleRepository) RevokePermissionFromRole(roleID, permissionID int) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`

	result, err := r.db.Exec(query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to revoke permission from role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("permission not granted to role")
	}

	return nil
}

// GetRolePermissions retrieves all permissions for a role
func (r *roleRepository) GetRolePermissions(roleID int) ([]*models.Permission, error) {
	var permissions []*models.Permission
	query := `
		SELECT p.id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.resource, p.action`

	err := r.db.Select(&permissions, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	return permissions, nil
}

// GetRolesWithPermission retrieves all roles that have a specific permission
func (r *roleRepository) GetRolesWithPermission(permissionID int) ([]*models.Role, error) {
	var roles []*models.Role
	query := `
		SELECT r.id, r.name, r.display_name, r.description, r.is_active, r.created_at
		FROM roles r
		INNER JOIN role_permissions rp ON r.id = rp.role_id
		WHERE rp.permission_id = $1 AND r.is_active = true
		ORDER BY r.name`

	err := r.db.Select(&roles, query, permissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles with permission: %w", err)
	}

	return roles, nil
}

// HasPermission checks if a role has a specific permission
func (r *roleRepository) HasPermission(roleID int, resource, action string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM role_permissions rp
		INNER JOIN permissions p ON rp.permission_id = p.id
		WHERE rp.role_id = $1 AND p.resource = $2 AND p.action = $3`

	err := r.db.Get(&count, query, roleID, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check role permission: %w", err)
	}

	return count > 0, nil
}

// AssignRoleToUser assigns a role to a user
func (r *roleRepository) AssignRoleToUser(userID string, roleID int, assignedBy *string, expiresAt *time.Time) error {
	var expiresAtValue any
	if expiresAt != nil {
		expiresAtValue = expiresAt.Unix()
	} else {
		expiresAtValue = nil
	}
	assignedAt := time.Now().Unix()

	query := `
        INSERT INTO user_roles (user_id, role_id, assigned_by, assigned_at, expires_at, is_active)
        VALUES ($1, $2, $3, $4, $5, TRUE) 
        ON CONFLICT (user_id, role_id) DO UPDATE SET
            assigned_by = EXCLUDED.assigned_by,
            assigned_at = EXCLUDED.assigned_at,
            expires_at = EXCLUDED.expires_at,
            is_active = true`

	_, err := r.db.Exec(query, userID, roleID, assignedBy, assignedAt, expiresAtValue)
	return err
}

// RemoveRoleFromUser removes a role from a user
func (r *roleRepository) RemoveRoleFromUser(userID string, roleID int) error {
	query := `UPDATE user_roles SET is_active = false WHERE user_id = $1 AND role_id = $2`

	result, err := r.db.Exec(query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role not assigned to user")
	}

	return nil
}

// GetUserRoles retrieves all roles assigned to a user
func (r *roleRepository) GetUserRoles(userID string, activeOnly bool) ([]*models.Role, error) {
	var roles []*models.Role
	var query string

	if activeOnly {
		query = `
			SELECT r.id, r.name, r.display_name, r.description, r.is_active, r.created_at
			FROM roles r
			INNER JOIN user_roles ur ON r.id = ur.role_id
			WHERE ur.user_id = $1 AND ur.is_active = true AND r.is_active = true
			AND (ur.expires_at IS NULL OR ur.expires_at > EXTRACT(EPOCH FROM CURRENT_TIMESTAMP))
			ORDER BY r.name`
	} else {
		query = `
			SELECT r.id, r.name, r.display_name, r.description, r.is_active, r.created_at
			FROM roles r
			INNER JOIN user_roles ur ON r.id = ur.role_id
			WHERE ur.user_id = $1
			ORDER BY r.name`
	}

	err := r.db.Select(&roles, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return roles, nil
}

// GetRoleUsers retrieves all users assigned to a role
func (r *roleRepository) GetRoleUsers(roleID int, activeOnly bool) ([]string, error) {
	var userIDs []string
	var query string

	if activeOnly {
		query = `
			SELECT ur.user_id
			FROM user_roles ur
			WHERE ur.role_id = $1 AND ur.is_active = true
			AND (ur.expires_at IS NULL OR ur.expires_at > EXTRACT(EPOCH FROM CURRENT_TIMESTAMP))
			ORDER BY ur.assigned_at`
	} else {
		query = `
			SELECT ur.user_id
			FROM user_roles ur
			WHERE ur.role_id = $1
			ORDER BY ur.assigned_at`
	}

	err := r.db.Select(&userIDs, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role users: %w", err)
	}

	return userIDs, nil
}

// GetUserPermissions retrieves all permissions for a user through their roles
func (r *roleRepository) GetUserPermissions(userID string) ([]*models.Permission, error) {
	var permissions []*models.Permission
	query := `
		SELECT DISTINCT p.id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN user_roles ur ON rp.role_id = ur.role_id
		INNER JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 
		AND ur.is_active = true 
		AND r.is_active = true
		AND (ur.expires_at IS NULL OR ur.expires_at > EXTRACT(EPOCH FROM CURRENT_TIMESTAMP))
		ORDER BY p.resource, p.action`

	err := r.db.Select(&permissions, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	return permissions, nil
}

// UserHasPermission checks if a user has a specific permission
func (r *roleRepository) UserHasPermission(userID string, resource, action string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN user_roles ur ON rp.role_id = ur.role_id
		INNER JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 
		AND p.resource = $2 
		AND p.action = $3
		AND ur.is_active = true 
		AND r.is_active = true
		AND (ur.expires_at IS NULL OR ur.expires_at > EXTRACT(EPOCH FROM CURRENT_TIMESTAMP))`

	err := r.db.Get(&count, query, userID, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check user permission: %w", err)
	}

	return count > 0, nil
}

// CreateRoleHierarchy creates a parent-child relationship between roles
func (r *roleRepository) CreateRoleHierarchy(parentRoleID, childRoleID int) error {
	if parentRoleID == childRoleID {
		return fmt.Errorf("parent and child role cannot be the same")
	}

	// Check for circular dependencies
	exists, err := r.checkCircularDependency(parentRoleID, childRoleID)
	if err != nil {
		return fmt.Errorf("failed to check circular dependency: %w", err)
	}
	if exists {
		return fmt.Errorf("creating hierarchy would result in circular dependency")
	}

	query := `
		INSERT INTO role_hierarchy (parent_role_id, child_role_id)
		VALUES ($1, $2)
		ON CONFLICT (parent_role_id, child_role_id) DO NOTHING`

	_, err = r.db.Exec(query, parentRoleID, childRoleID)
	if err != nil {
		return fmt.Errorf("failed to create role hierarchy: %w", err)
	}

	return nil
}

// DeleteRoleHierarchy removes a parent-child relationship between roles
func (r *roleRepository) DeleteRoleHierarchy(parentRoleID, childRoleID int) error {
	query := `DELETE FROM role_hierarchy WHERE parent_role_id = $1 AND child_role_id = $2`

	result, err := r.db.Exec(query, parentRoleID, childRoleID)
	if err != nil {
		return fmt.Errorf("failed to delete role hierarchy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("hierarchy relationship not found")
	}

	return nil
}

// GetRoleChildren retrieves all direct child roles of a parent role
func (r *roleRepository) GetRoleChildren(roleID int) ([]*models.Role, error) {
	var roles []*models.Role
	query := `
		SELECT r.id, r.name, r.display_name, r.description, r.is_active, r.created_at
		FROM roles r
		INNER JOIN role_hierarchy rh ON r.id = rh.child_role_id
		WHERE rh.parent_role_id = $1 AND r.is_active = true
		ORDER BY r.name`

	err := r.db.Select(&roles, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role children: %w", err)
	}

	return roles, nil
}

// GetRoleParents retrieves all direct parent roles of a child role
func (r *roleRepository) GetRoleParents(roleID int) ([]*models.Role, error) {
	var roles []*models.Role
	query := `
		SELECT r.id, r.name, r.display_name, r.description, r.is_active, r.created_at
		FROM roles r
		INNER JOIN role_hierarchy rh ON r.id = rh.parent_role_id
		WHERE rh.child_role_id = $1 AND r.is_active = true
		ORDER BY r.name`

	err := r.db.Select(&roles, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role parents: %w", err)
	}

	return roles, nil
}

// GetEffectiveRolePermissions retrieves all permissions for a role including inherited ones
func (r *roleRepository) GetEffectiveRolePermissions(roleID int) ([]*models.Permission, error) {
	var permissions []*models.Permission
	query := `
		WITH RECURSIVE role_tree AS (
			-- Base case: direct permissions of the role
			SELECT role_id FROM (VALUES ($1)) AS t(role_id)
			
			UNION
			
			-- Recursive case: permissions from parent roles
			SELECT rh.parent_role_id
			FROM role_hierarchy rh
			INNER JOIN role_tree rt ON rh.child_role_id = rt.role_id
			INNER JOIN roles r ON rh.parent_role_id = r.id
			WHERE r.is_active = true
		)
		SELECT DISTINCT p.id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN role_tree rt ON rp.role_id = rt.role_id
		ORDER BY p.resource, p.action`

	err := r.db.Select(&permissions, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective role permissions: %w", err)
	}

	return permissions, nil
}

// checkCircularDependency checks if creating a hierarchy would result in circular dependency
func (r *roleRepository) checkCircularDependency(parentRoleID, childRoleID int) (bool, error) {
	var count int
	query := `
		WITH RECURSIVE role_tree AS (
			-- Start from the proposed child role
			SELECT parent_role_id, child_role_id, 1 as level
			FROM role_hierarchy
			WHERE child_role_id = $1
			
			UNION
			
			-- Recursively find all ancestors
			SELECT rh.parent_role_id, rh.child_role_id, rt.level + 1
			FROM role_hierarchy rh
			INNER JOIN role_tree rt ON rh.child_role_id = rt.parent_role_id
			WHERE rt.level < 10  -- Prevent infinite recursion
		)
		SELECT COUNT(*)
		FROM role_tree
		WHERE parent_role_id = $2`

	err := r.db.Get(&count, query, childRoleID, parentRoleID)
	if err != nil {
		return false, fmt.Errorf("failed to check circular dependency: %w", err)
	}

	return count > 0, nil
}
