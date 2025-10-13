package handlers

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"auth-service/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct {
	roleService *services.RoleService
}

// DTOs are now imported from models package

func NewPermissionHandler(roleService *services.RoleService) *PermissionHandler {
	return &PermissionHandler{
		roleService: roleService,
	}
}

func (p *PermissionHandler) RegisterRoutes(router *gin.Engine) {
	// Public routes
	publicGroup := router.Group("/auth/public/api/v2/permission")
	{
		publicGroup.GET("/permissions", p.GetAllPermissions)
	}

	// Protected routes
	protectedGroup := router.Group("/auth/protected/api/v2/permission")
	{
		protectedGroup.POST("/permissions", p.CreatePermission)
		protectedGroup.GET("/permissions/:id", p.GetPermission)
		protectedGroup.PUT("/permissions/:id", p.UpdatePermission)
		protectedGroup.DELETE("/permissions/:id", p.DeletePermission)
	}
}

// Public Endpoints

func (p *PermissionHandler) GetAllPermissions(c *gin.Context) {
	resource := c.Query("resource") // Optional resource filter
	limit, offset := utils.ParsePaginationParams(c)

	permissions, err := p.roleService.GetAllPermissions(resource, limit, offset)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to retrieve permissions", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, models.PaginatedPermissionsResponse{
		Permissions: permissions,
		Total:       len(permissions),
		Limit:       limit,
		Offset:      offset,
	})
}

// Protected Endpoints

func (p *PermissionHandler) CreatePermission(c *gin.Context) {
	var req models.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	permission, err := p.roleService.CreatePermission(req.Name, req.Resource, req.Action, req.Description)
	if err != nil {
		utils.SendError(c, http.StatusConflict, "failed to create permission", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusCreated, permission)
}

func (p *PermissionHandler) GetPermission(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid permission ID", err.Error())
		return
	}

	permission, err := p.roleService.GetPermission(id)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "permission not found", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, permission)
}

func (p *PermissionHandler) UpdatePermission(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid permission ID", err.Error())
		return
	}

	var req models.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Get existing permission first
	existingPermission, err := p.roleService.GetPermission(id)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "permission not found", err.Error())
		return
	}

	// Update the permission
	existingPermission.Name = req.Name
	existingPermission.Resource = req.Resource
	existingPermission.Action = req.Action
	existingPermission.Description = req.Description

	err = p.roleService.UpdatePermission(existingPermission)
	if err != nil {
		// Check if it's an "not implemented" error from service
		if err.Error() == "UpdatePermission not implemented in repository" {
			utils.SendError(c, http.StatusNotImplemented, "endpoint not fully implemented", "UpdatePermission method needs to be implemented in repository layer")
			return
		}
		utils.SendError(c, http.StatusInternalServerError, "failed to update permission", err.Error())
		return
	}

	utils.SendSuccess(c, http.StatusOK, existingPermission)
}

func (p *PermissionHandler) DeletePermission(c *gin.Context) {
	id, err := utils.ParseIDParam(c, "id")
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "invalid permission ID", err.Error())
		return
	}

	err = p.roleService.DeletePermission(id)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "failed to delete permission", err.Error())
		return
	}

	utils.SendMessage(c, http.StatusOK, "permission deleted successfully")
}
