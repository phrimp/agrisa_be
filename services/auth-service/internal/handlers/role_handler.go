package handlers

import (
	"auth-service/internal/services"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleService *services.RoleService
}

func NewRoleHandler(roleService *services.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

func (r *RoleHandler) RegisterRoutes(router *gin.Engine) {
	_ = router.Group("/auth/public/api/v2/role")
	_ = router.Group("/auth/protected/api/v2/role")
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
	return nil
}
