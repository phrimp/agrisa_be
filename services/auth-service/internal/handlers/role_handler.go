package handlers

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"auth-service/utils"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleService *services.RoleService
}

// DTOs are now imported from models package

func NewRoleHandler(roleService *services.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

func (r *RoleHandler) RegisterRoutes(router *gin.Engine) {
	// Public routes
	publicGroup := router.Group("/auth/public/api/v2/role")
	{
		publicGroup.GET("/:id", r.GetRole)
		publicGroup.GET("/name/:name", r.GetRoleByName)
	}

	// Protected routes
	protectedGroup := router.Group("/auth/protected/api/v2/role")
	{
		// Role CRUD
		protectedGroup.POST("", r.CreateRole)
		protectedGroup.PUT("/:id", r.UpdateRole)
		protectedGroup.DELETE("/:id", r.DeleteRole)
		protectedGroup.PATCH("/:id/activate", r.ActivateRole)
		protectedGroup.PATCH("/:id/deactivate", r.DeactivateRole)
		protectedGroup.GET("", r.GetAllRoles)

		// Role-Permission Management
		protectedGroup.POST("/:id/permissions/:permissionId", r.GrantPermissionToRole)
		protectedGroup.DELETE("/:id/permissions/:permissionId", r.RevokePermissionFromRole)
		protectedGroup.GET("/:id/permissions", r.GetRolePermissions)
		protectedGroup.GET("/:id/permissions/effective", r.GetEffectiveRolePermissions)

		// User-Role Management
		protectedGroup.POST("/:id/users/:userId", r.AssignRoleToUser)
		protectedGroup.DELETE("/:id/users/:userId", r.RemoveRoleFromUser)
		protectedGroup.GET("/users/:userId/roles", r.GetUserRoles)
		protectedGroup.GET("/users/:userId/permissions", r.GetUserPermissions)
		protectedGroup.POST("/users/:userId/permissions/check", r.CheckUserPermission)

		// Role Hierarchy
		protectedGroup.POST("/hierarchy/:parentId/children/:childId", r.CreateRoleHierarchy)
		protectedGroup.DELETE("/hierarchy/:parentId/children/:childId", r.DeleteRoleHierarchy)
	}
}

// Public Endpoints

func (r *RoleHandler) GetAllRoles(c *gin.Context) {
	activeOnly := c.DefaultQuery("active_only", "true") == "true"
	limit, offset := utils.ParsePaginationParams(c)

	roles, err := r.roleService.GetAllRoles(activeOnly, limit, offset)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to retrieve roles", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, models.PaginatedRolesResponse{
		Roles:  roles,
		Total:  len(roles),
		Limit:  limit,
		Offset: offset,
	})
}

func (r *RoleHandler) GetRole(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	role, err := r.roleService.GetRole(id)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "role not found", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, role)
}

func (r *RoleHandler) GetRoleByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		utils.SendError(c, http.StatusBadRequest, "invalid role name", "role name cannot be empty")
		return
	}

	role, err := r.roleService.GetRoleByName(name)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "role not found", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, role)
}

// Protected Endpoints - Role CRUD

func (r *RoleHandler) CreateRole(c *gin.Context) {
	var req models.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	role, err := r.roleService.CreateRole(req.Name, req.DisplayName, req.Description)
	if err != nil {
		utils.SendError(c, http.StatusConflict, "failed to create role", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusCreated, role)
}

func (r *RoleHandler) UpdateRole(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	var req models.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Get existing role first
	existingRole, err := r.roleService.GetRole(id)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "role not found", err.Error())
		return
	}

	// Update the role
	existingRole.DisplayName = req.DisplayName
	existingRole.Description = req.Description

	err = r.roleService.UpdateRole(existingRole)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to update role", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, existingRole)
}

func (r *RoleHandler) DeleteRole(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	err = r.roleService.DeleteRole(id)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to delete role", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "role deleted successfully")
}

func (r *RoleHandler) ActivateRole(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	err = r.roleService.ActivateRole(id)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to activate role", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "role activated successfully")
}

func (r *RoleHandler) DeactivateRole(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	err = r.roleService.DeactivateRole(id)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to deactivate role", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "role deactivated successfully")
}

// Role-Permission Management

func (r *RoleHandler) GrantPermissionToRole(c *gin.Context) {
	roleID, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	permissionID, err := utils.ParseIDParam(c, "permissionId")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid permission ID", err.Error())
		return
	}

	err = r.roleService.GrantPermissionToRole(roleID, permissionID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to grant permission to role", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "permission granted to role successfully")
}

func (r *RoleHandler) RevokePermissionFromRole(c *gin.Context) {
	roleID, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	permissionID, err := utils.ParseIDParam(c, "permissionId")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid permission ID", err.Error())
		return
	}

	err = r.roleService.RevokePermissionFromRole(roleID, permissionID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to revoke permission from role", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "permission revoked from role successfully")
}

func (r *RoleHandler) GetRolePermissions(c *gin.Context) {
	roleID, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	permissions, err := r.roleService.GetRolePermissions(roleID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to get role permissions", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, gin.H{"permissions": permissions})
}

func (r *RoleHandler) GetEffectiveRolePermissions(c *gin.Context) {
	roleID, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	permissions, err := r.roleService.GetEffectiveRolePermissions(roleID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to get effective role permissions", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, gin.H{"effective_permissions": permissions})
}

// User-Role Management

func (r *RoleHandler) AssignRoleToUser(c *gin.Context) {
	roleID, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	userID := c.Param("userId")
	if userID == "" {
		utils.SendError(c, http.StatusBadRequest, "invalid user ID", "user ID cannot be empty")
		return
	}

	var req models.AssignRoleRequest
	c.ShouldBindJSON(&req) // Optional body

	err = r.roleService.AssignRoleToUser(userID, roleID, req.AssignedBy, req.ExpiresAt)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to assign role to user", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "role assigned to user successfully")
}

func (r *RoleHandler) RemoveRoleFromUser(c *gin.Context) {
	roleID, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid role ID", err.Error())
		return
	}

	userID := c.Param("userId")
	if userID == "" {
		utils.SendError(c, http.StatusBadRequest, "invalid user ID", "user ID cannot be empty")
		return
	}

	err = r.roleService.RemoveRoleFromUser(userID, roleID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to remove role from user", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "role removed from user successfully")
}

func (r *RoleHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		utils.SendError(c, http.StatusBadRequest, "invalid user ID", "user ID cannot be empty")
		return
	}

	activeOnly := c.DefaultQuery("active_only", "true") == "true"

	roles, err := r.roleService.GetUserRoles(userID, activeOnly)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to get user roles", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, gin.H{"roles": roles})
}

func (r *RoleHandler) GetUserPermissions(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		utils.SendError(c, http.StatusBadRequest, "invalid user ID", "user ID cannot be empty")
		return
	}

	permissions, err := r.roleService.GetUserPermissions(userID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to get user permissions", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, gin.H{"permissions": permissions})
}

func (r *RoleHandler) CheckUserPermission(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		utils.SendError(c, http.StatusBadRequest, "invalid user ID", "user ID cannot be empty")
		return
	}

	var req models.PermissionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	hasPermission, err := r.roleService.UserHasPermission(userID, req.Resource, req.Action)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to check user permission", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, gin.H{
		"has_permission": hasPermission,
		"resource":       req.Resource,
		"action":         req.Action,
	})
}

// Role Hierarchy

func (r *RoleHandler) CreateRoleHierarchy(c *gin.Context) {
	parentRoleID, err := utils.ParseIDParam(c, "parentId")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid parent role ID", err.Error())
		return
	}

	childRoleID, err := utils.ParseIDParam(c, "childId")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid child role ID", err.Error())
		return
	}

	err = r.roleService.CreateRoleHierarchy(parentRoleID, childRoleID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to create role hierarchy", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "role hierarchy created successfully")
}

func (r *RoleHandler) DeleteRoleHierarchy(c *gin.Context) {
	parentRoleID, err := utils.ParseIDParam(c, "parentId")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid parent role ID", err.Error())
		return
	}

	childRoleID, err := utils.ParseIDParam(c, "childId")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid child role ID", err.Error())
		return
	}

	err = r.roleService.DeleteRoleHierarchy(parentRoleID, childRoleID)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to delete role hierarchy", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "role hierarchy deleted successfully")
}

func (r *RoleHandler) InitDefaultRole() error {
	userRole, err := r.roleService.CreateRole("user_default", "User", "Default role for new user")
	if err != nil {
		return fmt.Errorf("default user role creation failed: %s", err)
	}
	log.Println("default user role created successfully: ", userRole)
	adminRole, err := r.roleService.CreateRole("admin", "Admin", "Admin")
	if err != nil {
		return fmt.Errorf("default admin role creation failed: %s", err)
	}
	log.Println("default admin role created successfully: ", adminRole)
	adminPartnerRole, err := r.roleService.CreateRole("admin_partner", "Admin Partner", "Admin for Insurance Partner")
	if err != nil {
		return fmt.Errorf("default admin partner role creation failed: %s", err)
	}
	log.Println("default admin role created successfully: ", adminPartnerRole)
	farmerRole, err := r.roleService.CreateRole("farmer", "Farmer", "Farmer")
	if err != nil {
		return fmt.Errorf("default farmer role creation failed: %s", err)
	}
	log.Println("default admin role created successfully: ", farmerRole)

	return nil
}
