package handlers

import (
	"profile-service/internal/models"
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
	userProfileGrPub.POST("/users", h.CreateUserProfile)
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

func (h *UserProfileHandler) CreateUserProfile(c *gin.Context) {
	var req models.CreateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(400, errorResponse)
		return
	}
	createdByID := c.GetHeader("X-User-ID")

	err := h.UserService.CreateUserProfile(&req, createdByID, "Admin")
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	successResponse := utils.CreateSuccessResponse("User profile created successfully")
	c.JSON(201, successResponse)
}
