package handlers

import (
	"net/http"
	"profile-service/internal/services"

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
}

func (h *InsurancePartnerHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func (h *InsurancePartnerHandler) GetInsurancePartnersByID(c *gin.Context) {

}
