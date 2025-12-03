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

type ClaimRejectionHandler struct {
	claimRejectionService   *services.ClaimRejectionService
	registeredPolicyService *services.RegisteredPolicyService
}

func NewClaimRejectionHandler(claimRejectionService *services.ClaimRejectionService, registeredPolicyService *services.RegisteredPolicyService) *ClaimRejectionHandler {
	return &ClaimRejectionHandler{
		claimRejectionService:   claimRejectionService,
		registeredPolicyService: registeredPolicyService,
	}
}

func (h *ClaimRejectionHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Claim Rejection routes
	claimRejectionGroup := protectedGr.Group("/claim-rejections")

	// Partner routes - read partner's claim rejections
	partnerGroup := claimRejectionGroup.Group("/read-partner")
	partnerGroup.Get("/list", h.GetPartnerClaimRejections)      // GET /claim-rejections/read-partner/list
	partnerGroup.Get("/:id", h.GetPartnerClaimRejectionByID)    // GET /claim-rejections/read-partner/:id
	partnerGroup.Get("/claim/:claim_id", h.GetPartnerByClaimID) // GET /claim-rejections/read-partner/claim/:claim_id

	// Partner create routes - create claim rejection
	partnerCreateGroup := claimRejectionGroup.Group("/create-partner")
	partnerCreateGroup.Post("/", h.CreatePartnerClaimRejection) // POST /claim-rejections/create-partner/

	// Admin routes - full CRUD access
	adminGroup := claimRejectionGroup.Group("/admin")
	adminGroup.Get("/list", h.GetAllClaimRejections)   // GET /claim-rejections/admin/list
	adminGroup.Get("/:id", h.GetClaimRejectionByID)    // GET /claim-rejections/admin/:id
	adminGroup.Get("/claim/:claim_id", h.GetByClaimID) // GET /claim-rejections/admin/claim/:claim_id
	adminGroup.Post("/", h.CreateClaimRejection)       // POST /claim-rejections/admin/
	adminGroup.Put("/:id", h.UpdateClaimRejection)     // PUT /claim-rejections/admin/:id
	adminGroup.Delete("/:id", h.DeleteClaimRejection)  // DELETE /claim-rejections/admin/:id
}

// ============================================================================
// CRUD HANDLERS
// ============================================================================

// GetAllClaimRejections retrieves all claim rejections
func (h *ClaimRejectionHandler) GetAllClaimRejections(c fiber.Ctx) error {
	claimRejections, err := h.claimRejectionService.GetAll(c.Context())
	if err != nil {
		slog.Error("Failed to get claim rejections", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claim rejections"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejections": claimRejections,
		"count":            len(claimRejections),
	}))
}

// GetClaimRejectionByID retrieves a specific claim rejection by ID
func (h *ClaimRejectionHandler) GetClaimRejectionByID(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim rejection ID format"))
	}

	claimRejection, err := h.claimRejectionService.GetByID(c.Context(), id)
	if err != nil {
		slog.Error("Failed to get claim rejection", "id", id, "error", err)
		return c.Status(http.StatusNotFound).JSON(
			utils.CreateErrorResponse("NOT_FOUND", "Claim rejection not found"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejection": claimRejection,
	}))
}

// GetByClaimID retrieves claim rejection by claim ID
func (h *ClaimRejectionHandler) GetByClaimID(c fiber.Ctx) error {
	claimIDStr := c.Params("claim_id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim ID format"))
	}

	claimRejection, err := h.claimRejectionService.GetByClaimID(c.Context(), claimID)
	if err != nil {
		slog.Error("Failed to get claim rejection by claim ID", "claim_id", claimID, "error", err)
		return c.Status(http.StatusNotFound).JSON(
			utils.CreateErrorResponse("NOT_FOUND", "Claim rejection not found for this claim"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejection": claimRejection,
	}))
}

// CreateClaimRejection creates a new claim rejection
func (h *ClaimRejectionHandler) CreateClaimRejection(c fiber.Ctx) error {
	var req struct {
		ClaimID             string                    `json:"claim_id"`
		ValidationTimestamp int64                     `json:"validation_timestamp"`
		ClaimRejectionType  models.ClaimRejectionType `json:"claim_rejection_type"`
		Reason              *string                   `json:"reason,omitempty"`
		ReasonEvidence      map[string]interface{}    `json:"reason_evidence,omitempty"`
		ValidatedBy         *string                   `json:"validated_by,omitempty"`
		ValidationNotes     *string                   `json:"validation_notes,omitempty"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	// Validate required fields
	if req.ClaimID == "" {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("MISSING_FIELDS", "claim_id is required"))
	}

	claimID, err := uuid.Parse(req.ClaimID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim ID format"))
	}

	claimRejection := &models.ClaimRejection{
		ID:                  uuid.New(),
		ClaimID:             claimID,
		ValidationTimestamp: req.ValidationTimestamp,
		ClaimRejectionType:  req.ClaimRejectionType,
		Reason:              req.Reason,
		ReasonEvidence:      req.ReasonEvidence,
		ValidatedBy:         req.ValidatedBy,
		ValidationNotes:     req.ValidationNotes,
	}

	response, err := h.claimRejectionService.CreateNewClaimRejection(c.Context(), claimRejection, claimID)
	if err != nil {
		slog.Error("Failed to create claim rejection", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("CREATION_FAILED", "Failed to create claim rejection"))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejection_id": response.ClaimRejectionID,
		"message":            "Claim rejection created successfully",
	}))
}

// UpdateClaimRejection updates an existing claim rejection
func (h *ClaimRejectionHandler) UpdateClaimRejection(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim rejection ID format"))
	}

	var req struct {
		ClaimID             string                    `json:"claim_id"`
		ValidationTimestamp int64                     `json:"validation_timestamp"`
		ClaimRejectionType  models.ClaimRejectionType `json:"claim_rejection_type"`
		Reason              *string                   `json:"reason,omitempty"`
		ReasonEvidence      map[string]interface{}    `json:"reason_evidence,omitempty"`
		ValidatedBy         *string                   `json:"validated_by,omitempty"`
		ValidationNotes     *string                   `json:"validation_notes,omitempty"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	claimID, err := uuid.Parse(req.ClaimID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim ID format"))
	}

	claimRejection := &models.ClaimRejection{
		ID:                  id,
		ClaimID:             claimID,
		ValidationTimestamp: req.ValidationTimestamp,
		ClaimRejectionType:  req.ClaimRejectionType,
		Reason:              req.Reason,
		ReasonEvidence:      req.ReasonEvidence,
		ValidatedBy:         req.ValidatedBy,
		ValidationNotes:     req.ValidationNotes,
	}

	err = h.claimRejectionService.Update(c.Context(), claimRejection)
	if err != nil {
		slog.Error("Failed to update claim rejection", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("UPDATE_FAILED", "Failed to update claim rejection"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"message": "Claim rejection updated successfully",
	}))
}

// DeleteClaimRejection deletes a claim rejection by ID
func (h *ClaimRejectionHandler) DeleteClaimRejection(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim rejection ID format"))
	}

	err = h.claimRejectionService.Delete(c.Context(), id)
	if err != nil {
		slog.Error("Failed to delete claim rejection", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("DELETE_FAILED", "Failed to delete claim rejection"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"message": "Claim rejection deleted successfully",
	}))
}

// ============================================================================
// PARTNER PERMISSION HANDLERS (read-partner)
// ============================================================================

// GetPartnerClaimRejections retrieves all claim rejections for the authenticated insurance partner
func (h *ClaimRejectionHandler) GetPartnerClaimRejections(c fiber.Ctx) error {
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

	claimRejections, err := h.claimRejectionService.GetAllByProviderID(c.Context(), partnerID)
	if err != nil {
		slog.Error("Failed to get partner claim rejections", "partner_id", partnerID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claim rejections"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejections": claimRejections,
		"count":            len(claimRejections),
		"partner_id":       partnerID,
	}))
}

// GetPartnerClaimRejectionByID retrieves a specific claim rejection by ID for the insurance partner
func (h *ClaimRejectionHandler) GetPartnerClaimRejectionByID(c fiber.Ctx) error {
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

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim rejection ID format"))
	}

	claimRejection, err := h.claimRejectionService.GetByIDForPartner(c.Context(), id, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Claim rejection not found"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this claim rejection"))
		}
		slog.Error("Failed to get claim rejection", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claim rejection"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejection": claimRejection,
	}))
}

// GetPartnerByClaimID retrieves claim rejection by claim ID for the insurance partner
func (h *ClaimRejectionHandler) GetPartnerByClaimID(c fiber.Ctx) error {
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

	claimIDStr := c.Params("claim_id")
	claimID, err := uuid.Parse(claimIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim ID format"))
	}

	claimRejection, err := h.claimRejectionService.GetByClaimIDForPartner(c.Context(), claimID, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Claim rejection not found for this claim"))
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this claim rejection"))
		}
		slog.Error("Failed to get claim rejection by claim ID", "claim_id", claimID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve claim rejection"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejection": claimRejection,
	}))
}

// ============================================================================
// PARTNER CREATE HANDLERS (create-partner)
// ============================================================================

// CreatePartnerClaimRejection creates a new claim rejection for partner
// This will also update the claim status to "rejected"
func (h *ClaimRejectionHandler) CreatePartnerClaimRejection(c fiber.Ctx) error {
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

	var req struct {
		ClaimID             string                    `json:"claim_id"`
		ValidationTimestamp int64                     `json:"validation_timestamp"`
		ClaimRejectionType  models.ClaimRejectionType `json:"claim_rejection_type"`
		Reason              *string                   `json:"reason,omitempty"`
		ReasonEvidence      map[string]interface{}    `json:"reason_evidence,omitempty"`
		ValidatedBy         *string                   `json:"validated_by,omitempty"`
		ValidationNotes     *string                   `json:"validation_notes,omitempty"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	// Validate required fields
	if req.ClaimID == "" {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("MISSING_FIELDS", "claim_id is required"))
	}

	claimID, err := uuid.Parse(req.ClaimID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_ID", "Invalid claim ID format"))
	}

	claimRejection := &models.ClaimRejection{
		ID:                  uuid.New(),
		ClaimID:             claimID,
		ValidationTimestamp: req.ValidationTimestamp,
		ClaimRejectionType:  req.ClaimRejectionType,
		Reason:              req.Reason,
		ReasonEvidence:      req.ReasonEvidence,
		ValidatedBy:         req.ValidatedBy,
		ValidationNotes:     req.ValidationNotes,
	}

	// Create claim rejection with partner authorization
	response, err := h.claimRejectionService.CreateClaimRejectionForPartner(c.Context(), claimRejection, claimID, partnerID)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to reject this claim"))
		}
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Claim not found"))
		}
		if strings.Contains(err.Error(), "invalid status") {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_STATUS", err.Error()))
		}
		slog.Error("Failed to create partner claim rejection", "claim_id", claimID, "partner_id", partnerID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("CREATION_FAILED", "Failed to create claim rejection"))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"claim_rejection_id": response.ClaimRejectionID,
		"claim_id":           claimID,
		"message":            "Claim rejection created successfully and claim status updated to rejected",
	}))
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// getPartnerIDFromToken extracts and validates the partner ID from the authorization token
func (h *ClaimRejectionHandler) getPartnerIDFromToken(c fiber.Ctx) (string, error) {
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
