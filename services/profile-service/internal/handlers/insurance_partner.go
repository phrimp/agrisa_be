package handlers

import (
	"log"
	"net/http"
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
	insurancePartnerProfileGrPub.GET("/insurance-partners/:partner_id", h.GetInsurancePartnersByID)
	insurancePartnerProfileGrPub.GET("/insurance-partners/:partner_id/reviews", h.GetPartnerReviews)
}

func MapErrorToHTTPStatusExtended(errorString string) (errorCode string, httpStatus int) {
	errorLower := strings.ToLower(errorString)

	switch {
	case strings.Contains(errorLower, "no rows in result set"):
		return "NOT_FOUND", http.StatusNotFound
	case strings.Contains(errorLower, "duplicate"):
		return "CONFLICT", http.StatusConflict
	case strings.Contains(errorLower, "invalid"):
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

func (h *InsurancePartnerHandler) GetInsurancePartnersByID(c *gin.Context) {
	partnerID := c.Param("partner_id")
	result, err := h.InsurancePartnerService.GetInsurancePartnerByID(partnerID)
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
