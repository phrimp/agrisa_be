package handlers

import (
	"auth-service/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userService services.IUserService
}

func NewAuthHandler(userService services.IUserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

func (a *AuthHandler) RegisterRoutes(router *gin.Engine, authHander *AuthHandler) {
	authGrPub := router.Group("/api/v2/auth/public")

	authGrPub.POST("register")
	authGrPub.POST("login")

	authGrPro := router.Group("/api/v2/auth/protected")
	sessionGr := authGrPro.Group("/session")
	// User manage their own session
	sessionGr.GET("/me")
	// Admin manage all sessions
}
