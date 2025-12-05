package handlers

import (
	"log"
	"net/http"
	"profile-service/internal/models"
	"profile-service/internal/services"
	"strings"
	"utils"

	"github.com/gin-gonic/gin"
)

type InsurancePartnerHandler struct {
	InsurancePartnerService services.IInsurancePartnerService
}

func NewInsurancePartnerHandler(insurancePartnerService services.IInsurancePartnerService) *InsurancePartnerHandler {
	return &InsurancePartnerHandler{
		InsurancePartnerService: insurancePartnerService,
	}
}

func (h *InsurancePartnerHandler) RegisterRoutes(router *gin.Engine) {
	insurancePartnerProfileGrPub := router.Group("/profile/public/api/v1")
	insurancePartnerProfileGrPub.GET("/ping", h.Ping)
	insurancePartnerProfileGrPub.GET("/insurance-partners/:partner_id/profile", h.GetInsurancePartnerPublicByID)
	insurancePartnerProfileGrPub.GET("/insurance-partners/:partner_id/reviews", h.GetPartnerReviews)
	insurancePartnerProfileGrPub.GET("/insurance-partners", h.GetAllInsurancePartnersPublicProfiles)
	insurancePartnerProfileGrPub.GET("/insurance-partners/:partner_id", h.GetPrivateProfileByPartnerID)

	insurancePartnerProtectedGrPub := router.Group("/profile/protected/api/v1")
	insurancePartnerProtectedGrPub.POST("/insurance-partners", h.CreateInsurancePartner) // featurea: insu
	insurancePartnerProtectedGrPub.GET("/insurance-partners/me/profile", h.GetInsurancePartnerPrivateByID)
	insurancePartnerProtectedGrPub.PUT("/insurance-partners/me/profile", h.UpdateInsurancePartnerProfile)

	// ======= PARTNER DELETION REQUESTS =======
	partnerGr := insurancePartnerProtectedGrPub.Group("/insurance-partners")
	// partner endpoint
	partnerGr.POST("/deletion-requests", h.CreatePartnerDeletionRequest)
	partnerGr.GET("/:partner_admin_id/deletion-requests", h.GetPartnerDeletionRequestsByPartnerAdminID)

	//admin endpoint
	partnerAdminGr := insurancePartnerProtectedGrPub.Group("/insurance-partners/admin")
	partnerAdminGr.POST("/process-request", h.ProcessPartnerDeletionRequestReview)
}

func MapErrorToHTTPStatusExtended(errorString string) (errorCode string, httpStatus int) {
	errorLower := strings.ToLower(errorString)

	switch {
	case strings.Contains(errorLower, "no rows in result set"):
		return "NOT_FOUND", http.StatusNotFound
	case strings.Contains(errorLower, "duplicate"):
		return "CONFLICT", http.StatusConflict
	case strings.Contains(errorLower, "invalid") || strings.Contains(errorLower, "validation errors occurred"):
		return "BAD_REQUEST", http.StatusBadRequest
	case strings.Contains(errorLower, "unauthorized"):
		return "UNAUTHORIZED", http.StatusUnauthorized
	case strings.Contains(errorLower, "forbidden"):
		return "FORBIDDEN", http.StatusForbidden
	case strings.Contains(errorLower, "timeout"):
		return "REQUEST_TIMEOUT", http.StatusRequestTimeout
	default:
		return "INTERNAL_SERVER_ERROR", http.StatusInternalServerError
	}
}

func (h *InsurancePartnerHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func (h *InsurancePartnerHandler) GetAllInsurancePartnersPublicProfiles(c *gin.Context) {
	result, err := h.InsurancePartnerService.GetAllPartnersPublicProfiles()
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	response := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}

func (h *InsurancePartnerHandler) GetInsurancePartnerPublicByID(c *gin.Context) {
	partnerID := c.Param("partner_id")
	result, err := h.InsurancePartnerService.GetPublicProfile(partnerID)
	if err != nil {
		log.Printf("Error getting insurance partner by ID %s: %s", partnerID, err.Error())
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	response := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}

// GetPartnerReviews handles GET /insurance-partners/:partner_id/reviews
func (h *InsurancePartnerHandler) GetPartnerReviews(c *gin.Context) {
	partnerID := c.Param("partner_id")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "asc")
	limit, err := utils.GetQueryParamAsInt(c, "limit", 10)
	if err != nil {
		badRequestResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid limit parameter")
		c.JSON(http.StatusBadRequest, badRequestResponse)
		return
	}
	offset, err := utils.GetQueryParamAsInt(c, "offset", 1)
	if err != nil {
		badRequestResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid offset parameter")
		c.JSON(http.StatusBadRequest, badRequestResponse)
		return
	}

	result, err := h.InsurancePartnerService.GetPartnerReviews(partnerID, sortBy, sortDirection, limit, offset)
	if err != nil {
		log.Printf("Error getting reviews for partner ID %s: %s", partnerID, err.Error())
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	response := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}

// Create insurance partner profile
func (h *InsurancePartnerHandler) CreateInsurancePartner(c *gin.Context) {
	log.Printf("Received POST request for path: %s", c.Request.URL.Path)
	var req models.CreateInsurancePartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for CreateInsurancePartner: %s", err.Error())
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}
	createdBy := c.GetHeader("X-User-ID")
	result := h.InsurancePartnerService.CreateInsurancePartner(&req, createdBy)
	if result.Message == "Validation errors occurred" {
		errorResponse := utils.CreateSuccessResponse(result.Data)
		errorResponse.Success = false
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *InsurancePartnerHandler) GetInsurancePartnerPrivateByID(c *gin.Context) {
	staffID := c.GetHeader("X-User-ID")
	log.Printf("Fetching private profile for staffID: %s", staffID)
	result, err := h.InsurancePartnerService.GetPrivateProfile(staffID)
	if err != nil {
		log.Printf("Error getting insurance partner private by staffID %s: %s", staffID, err.Error())
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	response := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}

func (h *InsurancePartnerHandler) UpdateInsurancePartnerProfile(c *gin.Context) {
	updateBy := c.GetHeader("X-User-ID")
	var requestBody map[string]interface{}
	log.Printf("request go hereeeeee")
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		log.Printf("Error binding JSON for UpdateInsurancePartnerProfile: %s", err.Error())
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	log.Printf("request through parse json to requestBody")

	if requestBody["partner_id"] == nil {
		log.Printf("partner_id is missing in the request body")
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "partner_id is required")
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	log.Printf("request through partner_id check")

	dataResponse, err := h.InsurancePartnerService.UpdateInsurancePartner(requestBody, updateBy, "Admin")
	if err != nil {
		log.Printf("Error updating insurance partner profile: %s", err.Error())
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	successResponse := utils.CreateSuccessResponse(dataResponse)
	c.JSON(http.StatusOK, successResponse)
}

func (h *InsurancePartnerHandler) GetPrivateProfileByPartnerID(c *gin.Context) {
	partnerID := c.Param("partner_id")
	result, err := h.InsurancePartnerService.GetPrivateProfileByPartnerID(partnerID)
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	response := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}

func (h *InsurancePartnerHandler) CreatePartnerDeletionRequest(c *gin.Context) {
	adminPartnerID := c.GetHeader("X-User-ID")
	var req models.PartnerDeletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for CreatePartnerDeletionRequest: %s", err.Error())
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}
	result, err := h.InsurancePartnerService.CreatePartnerDeletionRequest(&req, adminPartnerID)
	if err != nil {
		log.Printf("Error creating partner deletion request: %s", err.Error())
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	successResponse := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusCreated, successResponse)
}

func (h *InsurancePartnerHandler) GetPartnerDeletionRequestsByPartnerAdminID(c *gin.Context) {
	partnerAdminID := c.Param("partner_admin_id")
	result, err := h.InsurancePartnerService.GetDeletionRequestsByRequesterID(partnerAdminID)
	if err != nil {
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
	response := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}

func (h *InsurancePartnerHandler) ProcessPartnerDeletionRequestReview(c *gin.Context) {
	var req models.ProcessRequestReviewDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for ProcessPartnerDeletionRequestReview: %s", err.Error())
		errorResponse := utils.CreateErrorResponse("BAD_REQUEST", "Invalid request payload")
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}
	err := h.InsurancePartnerService.ValidateDeletionRequestProcess(req)
	if err != nil {
		log.Printf("Error validating deletion request process: %s", err.Error())
		errorCode, httpStatus := MapErrorToHTTPStatusExtended(err.Error())
		errorResponse := utils.CreateErrorResponse(errorCode, err.Error())
		c.JSON(httpStatus, errorResponse)
		return
	}
}
