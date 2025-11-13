package handlers

import (
	utils "agrisa_utils"
	"log/slog"
	"net/http"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type PolicyHandler struct {
	registeredPolicyService *services.RegisteredPolicyService
}

func NewPolicyHandler(registeredPolicyService *services.RegisteredPolicyService) *PolicyHandler {
	return &PolicyHandler{
		registeredPolicyService: registeredPolicyService,
	}
}

func (h *PolicyHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Registered Policy routes
	policyGroup := protectedGr.Group("/policies")

	// Policy registration endpoint
	policyGroup.Post("/register", h.RegisterPolicy) // POST /policies/register - Register a new policy
}

// ============================================================================
// POLICY REGISTRATION OPERATIONS
// ============================================================================

// RegisterPolicy handles the registration of a new policy for a farmer
func (h *PolicyHandler) RegisterPolicy(c fiber.Ctx) error {
	// Parse request body
	var req models.RegisterAPolicyAPIRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("Failed to bind request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body: "+err.Error()))
	}

	// Validate request
	if err := req.Validate(); err != nil {
		slog.Error("Request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Get user ID from header for authorization
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Build internal request structure
	registerRequest := models.RegisterAPolicyRequest{
		RegisteredPolicy: req.RegisteredPolicy,
		Farm:             req.Farm,
		PolicyTags:       req.PolicyTags,
	}

	// Determine if this is a new farm or existing farm
	if req.Farm.ID != uuid.Nil {
		// Existing farm - use farm ID
		registerRequest.FarmID = req.Farm.ID.String()
		registerRequest.IsNewFarm = false
	} else if req.Farm.FarmName != nil && *req.Farm.FarmName != "" {
		// New farm - farm details provided
		registerRequest.IsNewFarm = true
	} else {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Either farm ID or complete farm details must be provided"))
	}

	// Ensure the farmer ID matches the authenticated user
	if registerRequest.RegisteredPolicy.FarmerID != userID {
		slog.Warn("Authorization mismatch",
			"authenticated_user", userID,
			"requested_farmer_id", registerRequest.RegisteredPolicy.FarmerID)
		return c.Status(http.StatusForbidden).JSON(
			utils.CreateErrorResponse("FORBIDDEN", "Cannot register policy for another user"))
	}

	// Call service to register the policy
	response, err := h.registeredPolicyService.RegisterAPolicy(registerRequest, c.Context())
	if err != nil {
		// Parse error and return appropriate status code
		errMsg := err.Error()

		if strings.Contains(errMsg, "validation") || strings.Contains(errMsg, "invalid") {
			slog.Error("Validation failed", "error", err)
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("VALIDATION_FAILED", errMsg))
		}

		if strings.Contains(errMsg, "not found") {
			slog.Error("Resource not found", "error", err)
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", errMsg))
		}

		if strings.Contains(errMsg, "authorization") || strings.Contains(errMsg, "unauthorized") {
			slog.Error("Authorization error", "error", err)
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", errMsg))
		}

		if strings.Contains(errMsg, "unimplemented") || strings.Contains(errMsg, "feature") {
			slog.Error("Feature not available", "error", err)
			return c.Status(http.StatusNotImplemented).JSON(
				utils.CreateErrorResponse("NOT_IMPLEMENTED", errMsg))
		}

		// Generic internal server error
		slog.Error("Failed to register policy", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("REGISTRATION_FAILED", "Failed to register policy: "+errMsg))
	}

	// Success response
	slog.Info("Policy registered successfully",
		"policy_id", response.RegisterPolicyID,
		"farmer_id", userID)

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(response))
}
