package handlers

import (
	utils "agrisa_utils"
	"log/slog"
	"net/http"
	"policy-service/internal/services"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type RiskAnalysisHandler struct {
	riskAnalysisService *services.RiskAnalysisCRUDService
}

func NewRiskAnalysisHandler(riskAnalysisService *services.RiskAnalysisCRUDService) *RiskAnalysisHandler {
	return &RiskAnalysisHandler{
		riskAnalysisService: riskAnalysisService,
	}
}

func (h *RiskAnalysisHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Risk Analysis routes
	riskGroup := protectedGr.Group("/risk-analysis")

	// ============================================================================
	// PERMISSION-BASED ROUTES
	// Format: /risk-analysis/{crud-permission}-{detail}/...
	// ============================================================================

	// Farmer routes - read own risk analyses only
	farmerGroup := riskGroup.Group("/read-own")
	farmerGroup.Get("/by-policy/:policy_id", h.GetByPolicyIDOwn)    // GET /risk-analysis/read-own/by-policy/:policy_id
	farmerGroup.Get("/latest/:policy_id", h.GetLatestByPolicyIDOwn) // GET /risk-analysis/read-own/latest/:policy_id

	// Partner routes - read partner's risk analyses
	partnerGroup := riskGroup.Group("/read-partner")
	partnerGroup.Get("/by-policy/:policy_id", h.GetByPolicyID)    // GET /risk-analysis/read-partner/by-policy/:policy_id
	partnerGroup.Get("/latest/:policy_id", h.GetLatestByPolicyID) // GET /risk-analysis/read-partner/latest/:policy_id
	partnerGroup.Get("/:id", h.GetByID)                           // GET /risk-analysis/read-partner/:id

	// Admin routes - full access to all risk analyses
	adminReadGroup := riskGroup.Group("/read-all")
	adminReadGroup.Get("/", h.GetAll)                               // GET /risk-analysis/read-all
	adminReadGroup.Get("/by-policy/:policy_id", h.GetByPolicyID)    // GET /risk-analysis/read-all/by-policy/:policy_id
	adminReadGroup.Get("/latest/:policy_id", h.GetLatestByPolicyID) // GET /risk-analysis/read-all/latest/:policy_id
	adminReadGroup.Get("/:id", h.GetByID)                           // GET /risk-analysis/read-all/:id

	// Admin delete routes
	adminDeleteGroup := riskGroup.Group("/delete-any")
	adminDeleteGroup.Delete("/:id", h.Delete) // DELETE /risk-analysis/delete-any/:id
}

// ============================================================================
// FARMER PERMISSION HANDLERS (read-own)
// ============================================================================

// GetByPolicyIDOwn retrieves all risk analyses for a farmer's own policy
func (h *RiskAnalysisHandler) GetByPolicyIDOwn(c fiber.Ctx) error {
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

	analyses, err := h.riskAnalysisService.GetByPolicyIDOwn(c.Context(), userID, policyID)
	if err != nil {
		if strings.Contains(err.Error(), "policy not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		if strings.Contains(err.Error(), "does not own") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have access to this policy's risk analyses"))
		}
		slog.Error("Failed to get risk analyses by policy ID (own)", "policy_id", policyID, "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve risk analyses"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"risk_analyses":        analyses,
		"count":                len(analyses),
		"registered_policy_id": policyID,
	}))
}

// GetLatestByPolicyIDOwn retrieves the most recent risk analysis for a farmer's own policy
func (h *RiskAnalysisHandler) GetLatestByPolicyIDOwn(c fiber.Ctx) error {
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

	analysis, err := h.riskAnalysisService.GetLatestByPolicyIDOwn(c.Context(), userID, policyID)
	if err != nil {
		if strings.Contains(err.Error(), "policy not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		if strings.Contains(err.Error(), "does not own") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have access to this policy's risk analyses"))
		}
		if strings.Contains(err.Error(), "no rows") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "No risk analysis found for this policy"))
		}
		slog.Error("Failed to get latest risk analysis (own)", "policy_id", policyID, "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve latest risk analysis"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(analysis))
}

// ============================================================================
// PARTNER/ADMIN PERMISSION HANDLERS (read-partner, read-all)
// ============================================================================

// GetByID retrieves a specific risk analysis by ID
func (h *RiskAnalysisHandler) GetByID(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid risk analysis ID format"))
	}

	analysis, err := h.riskAnalysisService.GetByID(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Risk analysis not found"))
		}
		slog.Error("Failed to get risk analysis", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve risk analysis"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(analysis))
}

// GetByPolicyID retrieves all risk analyses for a specific policy
func (h *RiskAnalysisHandler) GetByPolicyID(c fiber.Ctx) error {
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

	analyses, err := h.riskAnalysisService.GetByPolicyID(c.Context(), policyID)
	if err != nil {
		if strings.Contains(err.Error(), "policy not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to get risk analyses by policy ID", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve risk analyses"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"risk_analyses":        analyses,
		"count":                len(analyses),
		"registered_policy_id": policyID,
	}))
}

// GetLatestByPolicyID retrieves the most recent risk analysis for a policy
func (h *RiskAnalysisHandler) GetLatestByPolicyID(c fiber.Ctx) error {
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

	analysis, err := h.riskAnalysisService.GetLatestByPolicyID(c.Context(), policyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no rows") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "No risk analysis found for this policy"))
		}
		slog.Error("Failed to get latest risk analysis", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve latest risk analysis"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(analysis))
}

// GetAll retrieves all risk analyses with optional pagination
func (h *RiskAnalysisHandler) GetAll(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Parse pagination parameters
	limit := 50 // default
	offset := 0

	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetParam := c.Query("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	analyses, err := h.riskAnalysisService.GetAll(c.Context(), limit, offset)
	if err != nil {
		slog.Error("Failed to get all risk analyses", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve risk analyses"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"risk_analyses": analyses,
		"count":         len(analyses),
		"limit":         limit,
		"offset":        offset,
	}))
}

// Delete removes a risk analysis by ID
func (h *RiskAnalysisHandler) Delete(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid risk analysis ID format"))
	}

	err = h.riskAnalysisService.Delete(c.Context(), id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Risk analysis not found"))
		}
		slog.Error("Failed to delete risk analysis", "id", id, "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("DELETE_FAILED", "Failed to delete risk analysis"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"message":    "Risk analysis deleted successfully",
		"deleted_id": id,
	}))
}
