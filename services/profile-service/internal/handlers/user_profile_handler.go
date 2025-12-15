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

	userProfilePubGr := router.Group("/profile/public/api/v1")
	userProfilePubGr.POST("/farmers", h.CreateFarmerProfile)
	userProfilePubGr.GET("/users/:partner_id", h.GetUserProfilesByPartnerID)
	userProfilePubGr.GET("/users/own/:user_id", h.GetUserProfileByUserIDPublic)

	userProfileProGr := router.Group("/profile/protected/api/v1")
	userProfileProGr.PUT("/users", h.UpdateUserProfile)
	userProfileProGr.GET("/me", h.GetUserProfileByUserID)
	userProfileProGr.POST("/users", h.CreateUserProfile)

	// admin endpoint
	userProfileProGr.POST("/users/bank-info", h.GetUserBankInfoByUserIDs)
	userProfileProGr.PUT("/users/admin/:user_id", h.UpdateUserProfileByAdmin)
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

func (h *UserProfileHandler) CreateFarmerProfile(c *gin.Context) {
	var req models.CreateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(400, errorResponse)
		return
	}

	err := h.UserService.CreateUserProfile(&req, "", "")
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	successResponse := utils.CreateSuccessResponse("Farmer profile created successfully")
	c.JSON(201, successResponse)
}

func (h *UserProfileHandler) UpdateUserProfile(c *gin.Context) {
	var updateProfileRequestBody map[string]any
	if err := c.ShouldBindJSON(&updateProfileRequestBody); err != nil {
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(400, errorResponse)
		return
	}

	userID := c.GetHeader("X-User-ID")

	updatedProfile, err := h.UserService.UpdateUserProfile(updateProfileRequestBody, userID, "")
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}

	successResponse := utils.CreateSuccessResponse(updatedProfile)
	c.JSON(200, successResponse)
}

func (h *UserProfileHandler) GetUserProfilesByPartnerID(c *gin.Context) {
	partnerID := c.Param("partner_id")
	userProfiles, err := h.UserService.GetUserProfilesByPartnerID(partnerID)
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	successResponse := utils.CreateSuccessResponse(userProfiles)
	c.JSON(200, successResponse)
}

func (h *UserProfileHandler) GetUserBankInfoByUserIDs(c *gin.Context) {
	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(400, errorResponse)
		return
	}
	bankInfos, err := h.UserService.GetUserBankInfoByUserIDs(req.UserIDs)
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	successResponse := utils.CreateSuccessResponse(bankInfos)
	c.JSON(200, successResponse)
}

func (h *UserProfileHandler) UpdateUserProfileByAdmin(c *gin.Context) {
	adminID := c.GetHeader("X-User-ID")
	if adminID == "" {
		errorResponse := utils.CreateErrorResponse("UNAUTHORIZED", "Admin ID is required")
		c.JSON(401, errorResponse)
		return
	}

	var updateProfileRequestBody map[string]any
	if err := c.ShouldBindJSON(&updateProfileRequestBody); err != nil {
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(400, errorResponse)
		return
	}

	userID := c.Param("user_id")

	updatedProfile, err := h.UserService.UpdateUserProfile(updateProfileRequestBody, userID, "Admin")
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}

	successResponse := utils.CreateSuccessResponse(updatedProfile)
	c.JSON(200, successResponse)
}

func (h *UserProfileHandler) GetUserProfileByUserIDPublic(c *gin.Context) {
	userID := c.Param("user_id")
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
