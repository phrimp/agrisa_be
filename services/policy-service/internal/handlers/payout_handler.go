package handlers

import (
	utils "agrisa_utils"
	"fmt"
	"log/slog"
	"net/http"
	"policy-service/internal/services"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type PayoutHandler struct {
	payoutService           *services.PayoutService
	registeredPolicyService *services.RegisteredPolicyService
}

func NewPayoutHandler(payoutService *services.PayoutService, registeredPolicyService *services.RegisteredPolicyService) *PayoutHandler {
	return &PayoutHandler{
		payoutService:           payoutService,
		registeredPolicyService: registeredPolicyService,
	}
}

func (h *PayoutHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Payout routes
	payoutGroup := protectedGr.Group("/payouts")

	// ============================================================================
	// PERMISSION-BASED ROUTES
	// Format: /payouts/{crud-permission}-{detail}/...
	// ============================================================================

	// Farmer routes - read own payouts only
	farmerGroup := payoutGroup.Group("/read-own")
	farmerGroup.Get("/list", h.GetFarmerOwnPayouts)                      // GET /payouts/read-own/list
	farmerGroup.Get("/detail/:id", h.GetFarmerPayoutDetail)              // GET /payouts/read-own/detail/:id
	farmerGroup.Get("/by-claim/:claim_id", h.GetFarmerPayoutByClaim)     // GET /payouts/read-own/by-claim/:claim_id
	farmerGroup.Get("/by-policy/:policy_id", h.GetFarmerPayoutsByPolicy) // GET /payouts/read-own/by-policy/:policy_id
	farmerGroup.Get("/by-farm/:farm_id", h.GetFarmerPayoutsByFarm)       // GET /payouts/read-own/by-farm/:farm_id

	// Insurance Partner routes - read partner's payouts
	partnerGroup := payoutGroup.Group("/read-partner")
	partnerGroup.Get("/list", h.GetPartnerPayouts)                         // GET /payouts/read-partner/list (not implemented yet, would need service method)
	partnerGroup.Get("/detail/:id", h.GetPartnerPayoutDetail)              // GET /payouts/read-partner/detail/:id
	partnerGroup.Get("/by-policy/:policy_id", h.GetPartnerPayoutsByPolicy) // GET /payouts/read-partner/by-policy/:policy_id
	partnerGroup.Get("/by-farm/:farm_id", h.GetPartnerPayoutsByFarm)       // GET /payouts/read-partner/by-farm/:farm_id

	// Admin routes - full access to all payouts
	adminReadGroup := payoutGroup.Group("/read-all")
	adminReadGroup.Get("/detail/:id", h.GetPayoutDetailAdmin)              // GET /payouts/read-all/detail/:id
	adminReadGroup.Get("/by-claim/:claim_id", h.GetPayoutByClaimAdmin)     // GET /payouts/read-all/by-claim/:claim_id
	adminReadGroup.Get("/by-policy/:policy_id", h.GetPayoutsByPolicyAdmin) // GET /payouts/read-all/by-policy/:policy_id
	adminReadGroup.Get("/by-farm/:farm_id", h.GetPayoutsByFarmAdmin)       // GET /payouts/read-all/by-farm/:farm_id
	adminReadGroup.Get("/by-farmer/:farmer_id", h.GetPayoutsByFarmerAdmin) // GET /payouts/read-all/by-farmer/:farmer_id
}

// ============================================================================
// FARMER PERMISSION HANDLERS (read-own)
// ============================================================================

// GetFarmerOwnPayouts retrieves all payouts for the authenticated farmer
func (h *PayoutHandler) GetFarmerOwnPayouts(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	payouts, err := h.payoutService.GetPayoutsByFarmerID(c.Context(), userID)
	if err != nil {
		slog.Error("Failed to get farmer payouts", "farmer_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts":   payouts,
		"count":     len(payouts),
		"farmer_id": userID,
	}))
}

// GetFarmerPayoutDetail retrieves a specific payout detail for the authenticated farmer
func (h *PayoutHandler) GetFarmerPayoutDetail(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	payoutIDStr := c.Params("id")
	payoutID, err := uuid.Parse(payoutIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid payout ID format"))
	}

	payout, err := h.payoutService.GetPayoutByIDForFarmer(c.Context(), payoutID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Payout not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this payout"))
		}
		slog.Error("Failed to get payout", "payout_id", payoutID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payout"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(payout))
}

// GetFarmerPayoutByClaim retrieves a payout for a specific claim owned by the farmer
func (h *PayoutHandler) GetFarmerPayoutByClaim(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	claimIDStr := c.Params("claim_id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid claim ID format"))
	}

	// Get payout by claim ID (no specific farmer authorization method, using direct query)
	payout, err := h.payoutService.GetPayoutByClaimID(c.Context(), claimID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Payout not found for this claim"))
		}
		slog.Error("Failed to get payout by claim", "claim_id", claimID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payout"))
	}

	// Verify the payout belongs to the farmer
	if payout.FarmerID != userID {
		return c.Status(http.StatusForbidden).JSON(
			utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this payout"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(payout))
}

// GetFarmerPayoutsByPolicy retrieves payouts for a specific policy owned by the farmer
func (h *PayoutHandler) GetFarmerPayoutsByPolicy(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policyIDStr := c.Params("policy_id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	payouts, err := h.payoutService.GetPayoutsByRegisteredPolicyIDForFarmer(c.Context(), policyID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view these payouts"))
		}
		slog.Error("Failed to get payouts by policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts":   payouts,
		"count":     len(payouts),
		"policy_id": policyID,
	}))
}

// GetFarmerPayoutsByFarm retrieves payouts for a specific farm owned by the farmer
func (h *PayoutHandler) GetFarmerPayoutsByFarm(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	farmIDStr := c.Params("farm_id")
	farmID, err := uuid.Parse(farmIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid farm ID format"))
	}

	payouts, err := h.payoutService.GetPayoutsByFarmIDForFarmer(c.Context(), farmID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Farm not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view these payouts"))
		}
		slog.Error("Failed to get payouts by farm", "farm_id", farmID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts": payouts,
		"count":   len(payouts),
		"farm_id": farmID,
	}))
}

// ============================================================================
// INSURANCE PARTNER PERMISSION HANDLERS (read-partner)
// ============================================================================

// GetPartnerPayouts retrieves all payouts for the authenticated insurance partner
// Note: This would require a service method like GetPayoutsByProviderID, which doesn't exist yet
func (h *PayoutHandler) GetPartnerPayouts(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Get partner ID from token
	_, err := h.getPartnerIDFromToken(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	// TODO: Implement GetPayoutsByProviderID in service layer if needed
	// For now, return a not implemented response
	return c.Status(http.StatusNotImplemented).JSON(
		utils.CreateErrorResponse("NOT_IMPLEMENTED", "Getting all payouts by provider ID is not yet implemented. Use specific queries instead (by-policy, by-farm)"))
}

// GetPartnerPayoutDetail retrieves a specific payout detail for the insurance partner
func (h *PayoutHandler) GetPartnerPayoutDetail(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Get partner ID from token
	partnerID, err := h.getPartnerIDFromToken(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	payoutIDStr := c.Params("id")
	payoutID, err := uuid.Parse(payoutIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid payout ID format"))
	}

	payout, err := h.payoutService.GetPayoutByIDForPartner(c.Context(), payoutID, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Payout not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this payout"))
		}
		slog.Error("Failed to get payout", "payout_id", payoutID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payout"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(payout))
}

// GetPartnerPayoutsByPolicy retrieves payouts for a specific policy managed by the partner
func (h *PayoutHandler) GetPartnerPayoutsByPolicy(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Get partner ID from token
	partnerID, err := h.getPartnerIDFromToken(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	policyIDStr := c.Params("policy_id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	payouts, err := h.payoutService.GetPayoutsByRegisteredPolicyIDForPartner(c.Context(), policyID, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view these payouts"))
		}
		slog.Error("Failed to get payouts by policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts":   payouts,
		"count":     len(payouts),
		"policy_id": policyID,
	}))
}

// GetPartnerPayoutsByFarm retrieves payouts for a specific farm managed by the partner
func (h *PayoutHandler) GetPartnerPayoutsByFarm(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Get partner ID from token
	partnerID, err := h.getPartnerIDFromToken(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	farmIDStr := c.Params("farm_id")
	farmID, err := uuid.Parse(farmIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid farm ID format"))
	}

	payouts, err := h.payoutService.GetPayoutsByFarmIDForPartner(c.Context(), farmID, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Farm not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view these payouts"))
		}
		slog.Error("Failed to get payouts by farm", "farm_id", farmID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts": payouts,
		"count":   len(payouts),
		"farm_id": farmID,
	}))
}

// ============================================================================
// ADMIN PERMISSION HANDLERS (read-all)
// ============================================================================

// GetPayoutDetailAdmin retrieves any payout detail (admin access)
func (h *PayoutHandler) GetPayoutDetailAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	payoutIDStr := c.Params("id")
	payoutID, err := uuid.Parse(payoutIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid payout ID format"))
	}

	payout, err := h.payoutService.GetPayoutByID(c.Context(), payoutID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Payout not found"))
		}
		slog.Error("Failed to get payout", "payout_id", payoutID, "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payout"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(payout))
}

// GetPayoutByClaimAdmin retrieves payout by claim ID (admin access)
func (h *PayoutHandler) GetPayoutByClaimAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	claimIDStr := c.Params("claim_id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid claim ID format"))
	}

	payout, err := h.payoutService.GetPayoutByClaimID(c.Context(), claimID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Payout not found for this claim"))
		}
		slog.Error("Failed to get payout by claim", "claim_id", claimID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payout"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(payout))
}

// GetPayoutsByPolicyAdmin retrieves payouts for a specific policy (admin access)
func (h *PayoutHandler) GetPayoutsByPolicyAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policyIDStr := c.Params("policy_id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	payouts, err := h.payoutService.GetPayoutsByRegisteredPolicyID(c.Context(), policyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to get payouts by policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts":   payouts,
		"count":     len(payouts),
		"policy_id": policyID,
	}))
}

// GetPayoutsByFarmAdmin retrieves payouts for a specific farm (admin access)
func (h *PayoutHandler) GetPayoutsByFarmAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	farmIDStr := c.Params("farm_id")
	farmID, err := uuid.Parse(farmIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid farm ID format"))
	}

	payouts, err := h.payoutService.GetPayoutsByFarmID(c.Context(), farmID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Farm not found"))
		}
		slog.Error("Failed to get payouts by farm", "farm_id", farmID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts": payouts,
		"count":   len(payouts),
		"farm_id": farmID,
	}))
}

// GetPayoutsByFarmerAdmin retrieves payouts for a specific farmer (admin access)
func (h *PayoutHandler) GetPayoutsByFarmerAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	farmerID := c.Params("farmer_id")
	if farmerID == "" {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_PARAMETER", "Farmer ID is required"))
	}

	payouts, err := h.payoutService.GetPayoutsByFarmerID(c.Context(), farmerID)
	if err != nil {
		slog.Error("Failed to get payouts by farmer", "farmer_id", farmerID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve payouts"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"payouts":   payouts,
		"count":     len(payouts),
		"farmer_id": farmerID,
	}))
}

// Helper function to extract partner ID from authorization token
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
