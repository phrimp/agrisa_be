package handlers

import (
	utils "agrisa_utils"
	"fmt"
	"log/slog"
	"net/http"
	"policy-service/internal/services"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type PayoutHandler struct {
	claimService            *services.ClaimService
	registeredPolicyService *services.RegisteredPolicyService
	payoutService           *services.PayoutService
}

func NewPayoutHandler(claimService *services.ClaimService, registeredPolicyService *services.RegisteredPolicyService, payoutService *services.PayoutService) *PayoutHandler {
	return &PayoutHandler{
		claimService:            claimService,
		registeredPolicyService: registeredPolicyService,
		payoutService:           payoutService,
	}
}

func (h *PayoutHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// routes
	payoutGroup := protectedGr.Group("/payouts")

	farmerGroup := payoutGroup.Group("/read-own")
	farmerGroup.Get("/list", h.GetFarmerOwn)

	partnerGroup := payoutGroup.Group("/read-partner")
	partnerGroup.Get("/list", h.GetPartnerPayout)
}

func (h *PayoutHandler) GetFarmerOwn(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	payouts, err := h.payoutService.GetPayoutsByFarmerID(c.Context(), userID)
	if err != nil {
		slog.Error("Failed to get farmer payouts", "farmer_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
		"payouts":   payouts,
		"count":     len(payouts),
		"farmer_id": userID,
	}))
}

func (h *PayoutHandler) GetPartnerPayout(c fiber.Ctx) error {
	_, err := h.getPartnerIDFromToken(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	return nil
}

func (h *PayoutHandler) getPartnerIDFromToken(c fiber.Ctx) (string, error) {
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return "", fmt.Errorf("authorization token is required")
	}

	token := strings.TrimPrefix(tokenString, "Bearer ")

	// Get partner profile from token
	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve insurance partner profile: %w", err)
	}

	// Extract partner ID from profile data
	partnerID, err := h.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve partner ID: %w", err)
	}

	return partnerID, nil
}
