package handlers

import (
	utils "agrisa_utils"
	"fmt"
	"log/slog"
	"net/http"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ClaimHandler struct {
	claimService            *services.ClaimService
	registeredPolicyService *services.RegisteredPolicyService
}

func NewClaimHandler(claimService *services.ClaimService, registeredPolicyService *services.RegisteredPolicyService) *ClaimHandler {
	return &ClaimHandler{
		claimService:            claimService,
		registeredPolicyService: registeredPolicyService,
	}
}

func (h *ClaimHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Claim routes
	claimGroup := protectedGr.Group("/claims")

	// ============================================================================
	// PERMISSION-BASED ROUTES
	// Format: /claims/{crud-permission}-{detail}/...
	// ============================================================================

	// Farmer routes - read own claims only
	farmerGroup := claimGroup.Group("/read-own")
	farmerGroup.Get("/list", h.GetFarmerOwnClaims)                      // GET /claims/read-own/list
	farmerGroup.Get("/detail/:id", h.GetFarmerClaimDetail)              // GET /claims/read-own/detail/:id
	farmerGroup.Get("/by-policy/:policy_id", h.GetFarmerClaimsByPolicy) // GET /claims/read-own/by-policy/:policy_id
	farmerGroup.Get("/by-farm/:farm_id", h.GetFarmerClaimsByFarm)       // GET /claims/read-own/by-farm/:farm_id

	// Insurance Partner routes - read partner's claims
	partnerGroup := claimGroup.Group("/read-partner")
	partnerGroup.Get("/list", h.GetPartnerClaims)                         // GET /claims/read-partner/list
	partnerGroup.Get("/detail/:id", h.GetPartnerClaimDetail)              // GET /claims/read-partner/detail/:id
	partnerGroup.Get("/by-policy/:policy_id", h.GetPartnerClaimsByPolicy) // GET /claims/read-partner/by-policy/:policy_id

	// Admin routes - full access to all claims
	adminReadGroup := claimGroup.Group("/read-all")
	adminReadGroup.Get("/list", h.GetAllClaimsAdmin)                      // GET /claims/read-all/list
	adminReadGroup.Get("/detail/:id", h.GetClaimDetailAdmin)              // GET /claims/read-all/detail/:id
	adminReadGroup.Get("/by-policy/:policy_id", h.GetClaimsByPolicyAdmin) // GET /claims/read-all/by-policy/:policy_id
	adminReadGroup.Get("/by-farm/:farm_id", h.GetClaimsByFarmAdmin)       // GET /claims/read-all/by-farm/:farm_id

	adminDeleteGroup := claimGroup.Group("/delete-any")
	adminDeleteGroup.Delete("/:id", h.DeleteClaimAdmin) // DELETE /claims/delete-any/:id
}

// ============================================================================
// FARMER PERMISSION HANDLERS (read-own)
// ============================================================================

// GetFarmerOwnClaims retrieves all claims for the authenticated farmer
func (h *ClaimHandler) GetFarmerOwnClaims(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	claims, err := h.claimService.GetClaimsByFarmerID(c.Context(), userID)
	if err != nil {
		slog.Error("Failed to get farmer claims", "farmer_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":    claims,
		"count":     len(claims),
		"farmer_id": userID,
	}))
}

// GetFarmerClaimDetail retrieves a specific claim detail for the authenticated farmer
func (h *ClaimHandler) GetFarmerClaimDetail(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	claimIDStr := c.Params("id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid claim ID format"))
	}

	claim, err := h.claimService.GetClaimByIDForFarmer(c.Context(), claimID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Claim not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this claim"))
		}
		slog.Error("Failed to get claim", "claim_id", claimID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claim"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(claim))
}

// GetFarmerClaimsByPolicy retrieves claims for a specific policy owned by the farmer
func (h *ClaimHandler) GetFarmerClaimsByPolicy(c fiber.Ctx) error {
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

	claims, err := h.claimService.GetClaimsByPolicyIDForFarmer(c.Context(), policyID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view these claims"))
		}
		slog.Error("Failed to get claims by policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":    claims,
		"count":     len(claims),
		"policy_id": policyID,
	}))
}

// GetFarmerClaimsByFarm retrieves claims for a specific farm owned by the farmer
func (h *ClaimHandler) GetFarmerClaimsByFarm(c fiber.Ctx) error {
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

	claims, err := h.claimService.GetClaimsByFarmIDForFarmer(c.Context(), farmID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Farm not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view these claims"))
		}
		slog.Error("Failed to get claims by farm", "farm_id", farmID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":  claims,
		"count":   len(claims),
		"farm_id": farmID,
	}))
}

// ============================================================================
// INSURANCE PARTNER PERMISSION HANDLERS (read-partner)
// ============================================================================

// GetPartnerClaims retrieves all claims for the authenticated insurance partner
func (h *ClaimHandler) GetPartnerClaims(c fiber.Ctx) error {
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

	claims, err := h.claimService.GetClaimsByProviderID(c.Context(), partnerID)
	if err != nil {
		slog.Error("Failed to get partner claims", "partner_id", partnerID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":     claims,
		"count":      len(claims),
		"partner_id": partnerID,
	}))
}

// GetPartnerClaimDetail retrieves a specific claim detail for the insurance partner
func (h *ClaimHandler) GetPartnerClaimDetail(c fiber.Ctx) error {
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

	claimIDStr := c.Params("id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid claim ID format"))
	}

	claim, err := h.claimService.GetClaimByIDForPartner(c.Context(), claimID, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Claim not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this claim"))
		}
		slog.Error("Failed to get claim", "claim_id", claimID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claim"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(claim))
}

// GetPartnerClaimsByPolicy retrieves claims for a specific policy managed by the partner
func (h *ClaimHandler) GetPartnerClaimsByPolicy(c fiber.Ctx) error {
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

	claims, err := h.claimService.GetClaimsByPolicyIDForPartner(c.Context(), policyID, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view these claims"))
		}
		slog.Error("Failed to get claims by policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":    claims,
		"count":     len(claims),
		"policy_id": policyID,
	}))
}

// ============================================================================
// ADMIN PERMISSION HANDLERS (read-all, delete-any)
// ============================================================================

// GetAllClaimsAdmin retrieves all claims (admin access)
func (h *ClaimHandler) GetAllClaimsAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Parse optional filters
	filters := make(map[string]interface{})

	if statusParam := c.Query("status"); statusParam != "" {
		filters["status"] = models.ClaimStatus(statusParam)
	}

	if policyIDParam := c.Query("policy_id"); policyIDParam != "" {
		policyID, err := uuid.Parse(policyIDParam)
		if err == nil {
			filters["registered_policy_id"] = policyID
		}
	}

	if farmIDParam := c.Query("farm_id"); farmIDParam != "" {
		farmID, err := uuid.Parse(farmIDParam)
		if err == nil {
			filters["farm_id"] = farmID
		}
	}

	claims, err := h.claimService.GetAllClaims(c.Context(), filters)
	if err != nil {
		slog.Error("Failed to get all claims", "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":       claims,
		"count":        len(claims),
		"requested_by": userID,
	}))
}

// GetClaimDetailAdmin retrieves any claim detail (admin access)
func (h *ClaimHandler) GetClaimDetailAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	claimIDStr := c.Params("id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid claim ID format"))
	}

	claim, err := h.claimService.GetClaimByID(c.Context(), claimID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Claim not found"))
		}
		slog.Error("Failed to get claim", "claim_id", claimID, "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claim"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(claim))
}

// GetClaimsByPolicyAdmin retrieves claims for a specific policy (admin access)
func (h *ClaimHandler) GetClaimsByPolicyAdmin(c fiber.Ctx) error {
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

	claims, err := h.claimService.GetClaimsByPolicyID(c.Context(), policyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to get claims by policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":    claims,
		"count":     len(claims),
		"policy_id": policyID,
	}))
}

// GetClaimsByFarmAdmin retrieves claims for a specific farm (admin access)
func (h *ClaimHandler) GetClaimsByFarmAdmin(c fiber.Ctx) error {
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

	claims, err := h.claimService.GetClaimsByFarmID(c.Context(), farmID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Farm not found"))
		}
		slog.Error("Failed to get claims by farm", "farm_id", farmID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claims"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claims":  claims,
		"count":   len(claims),
		"farm_id": farmID,
	}))
}

// DeleteClaimAdmin deletes a claim (admin access only)
func (h *ClaimHandler) DeleteClaimAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	claimIDStr := c.Params("id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid claim ID format"))
	}

	err = h.claimService.DeleteClaim(c.Context(), claimID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Claim not found"))
		}
		slog.Error("Failed to delete claim", "claim_id", claimID, "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("DELETE_FAILED", "Failed to delete claim"))
	}

	slog.Info("Claim deleted successfully", "claim_id", claimID, "deleted_by", userID)

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"message":    "Claim deleted successfully",
		"claim_id":   claimID,
		"deleted_by": userID,
	}))
}

// Helper function to extract partner ID from authorization token
func (h *ClaimHandler) getPartnerIDFromToken(c fiber.Ctx) (string, error) {
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
