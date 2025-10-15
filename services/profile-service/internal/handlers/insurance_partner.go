package handlers

import (
	"net/http"
	"profile-service/internal/services"

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

func (h *InsurancePartnerHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func (h *InsurancePartnerHandler) GetInsurancePartnersByID(c *gin.Context) {
	partnerID := c.Param("partner_id")
	result, err := h.InsurancePartnerService.GetInsurancePartnerByID(partnerID)
	if err != nil {
		internalServerErrorResponse := utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "Failed to get insurance partner by ID")
		c.JSON(http.StatusInternalServerError, internalServerErrorResponse)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	offset, err := utils.GetQueryParamAsInt(c, "offset", 1)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.InsurancePartnerService.GetPartnerReviews(partnerID, sortBy, sortDirection, limit, offset)
	if err != nil {
		internalServerErrorResponse := utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "Failed to get partner reviews")
		c.JSON(http.StatusInternalServerError, internalServerErrorResponse)
		return
	}
	response := utils.CreateSuccessResponse(result)
	c.JSON(http.StatusOK, response)
}
