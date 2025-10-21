package handlers

import (
	"profile-service/internal/services"
	"utils"

	"github.com/gin-gonic/gin"
)

type UserProfileHandler struct {
	UserService services.IUserService
}

func NewUserProfileHandler(userService services.IUserService) *UserProfileHandler {
	return &UserProfileHandler{
		UserService: userService,
	}
}

func (h *UserProfileHandler) RegisterRoutes(router *gin.Engine) {
	userProfileGrPub := router.Group("/profile/protected/api/v1")
	userProfileGrPub.GET("/me", h.GetUserProfileByUserID)
}

func (h *UserProfileHandler) GetUserProfileByUserID(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	profile, err := h.UserService.GetUserProfileByUserID(userID)
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	successResponse := utils.CreateSuccessResponse(profile)
	c.JSON(200, successResponse)
}
