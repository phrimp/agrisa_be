package handlers

import (
	utils "agrisa_utils"
	"fmt"
	"log/slog"
	"net/http"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"strconv"
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

	// ============================================================================
	// PERMISSION-BASED ROUTES
	// Format: /policies/{crud-permission}-{detail}/...
	// ============================================================================

	// Farmer routes - read own policies only
	farmerGroup := policyGroup.Group("/read-own")
	farmerGroup.Get("/list", h.GetFarmerOwnPolicies)                                                   // GET /policies/read-own/list
	farmerGroup.Get("/detail/:id", h.GetFarmerPolicyDetail)                                            // GET /policies/read-own/detail/:id
	farmerGroup.Get("/stats/overview", h.GetStatsOverview)                                             // GET /policies/read-own/stats/overview
	farmerGroup.Get("/monitoring-data/:farm_id", h.GetFarmerMonitoringData)                            // GET /policies/read-own/monitoring-data/:farm_id
	farmerGroup.Get("/monitoring-data/:farm_id/:parameter_name", h.GetFarmerMonitoringDataByParameter) // GET /policies/read-own/monitoring-data/:farm_id/:parameter_name

	// Insurance Partner routes - read/manage partner's policies
	partnerGroup := policyGroup.Group("/read-partner")
	partnerGroup.Get("/list", h.GetPartnerPolicies)                                           // GET /policies/read-partner/list
	partnerGroup.Get("/detail/:id", h.GetPartnerPolicyDetail)                                 // GET /policies/read-partner/detail/:id
	partnerGroup.Get("/stats", h.GetPartnerPolicyStats)                                       // GET /policies/read-partner/stats
	partnerGroup.Get("/monitoring-data/:farm_id/:parameter_name", h.GetPartnerMonitoringData) // GET /policies/read-partner/monitoring-data/:farm_id/:parameter_name
	partnerCreateGroup := policyGroup.Group("/create-partner")
	partnerCreateGroup.Post("/underwriting/:id", h.CreatePartnerPolicyUnderwriting) // PATCH /policies/update-partner/underwriting/:id]
	partnerGroup.Post("/monthly-data-cost", h.GetMonthlyDataCost)

	// Admin routes - full access to all policies
	adminReadGroup := policyGroup.Group("/read-all")
	adminReadGroup.Get("/list", h.GetAllPoliciesAdmin)                         // GET /policies/read-all/list
	adminReadGroup.Get("/detail/:id", h.GetPolicyDetailAdmin)                  // GET /policies/read-all/detail/:id
	adminReadGroup.Get("/stats", h.GetAllPolicyStatsAdmin)                     // GET /policies/read-all/stats
	adminReadGroup.Get("/filter", h.GetPoliciesWithFilter)                     // GET /policies/filter - Get policies with filters
	adminReadGroup.Get("/monitoring-data", h.GetAllMonitoringData)             // GET /policies/read-all/monitoring-data - Get all monitoring data with policy status
	adminReadGroup.Get("/monitoring-data/:farm_id", h.GetMonitoringDataByFarm) // GET /policies/read-all/monitoring-data/:farm_id - Get monitoring data by farm

	adminUpdateGroup := policyGroup.Group("/update-any")
	adminUpdateGroup.Patch("/status/:id", h.UpdatePolicyStatusAdmin)             // PATCH /policies/update-any/status/:id
	adminUpdateGroup.Patch("/underwriting/:id", h.UpdatePolicyUnderwritingAdmin) // PATCH /policies/update-any/underwriting/:id
}

// ============================================================================
// POLICY REGISTRATION OPERATIONS
// ============================================================================

// RegisterPolicy handles the registration of a new policy for a farmer
func (h *PolicyHandler) RegisterPolicy(c fiber.Ctx) error {
	// Parse request body
	var req models.RegisterAPolicyAPIRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
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

// GetPoliciesWithFilter retrieves registered policies with optional filters
func (h *PolicyHandler) GetPoliciesWithFilter(c fiber.Ctx) error {
	var filter models.RegisteredPolicyFilterRequest

	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Parse UUID fields
	if idParam := c.Query("policy_id"); idParam != "" {
		parsedID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_UUID", "Invalid policy_id format"))
		}
		filter.PolicyID = &parsedID
	}

	if basePolicyIDParam := c.Query("base_policy_id"); basePolicyIDParam != "" {
		parsedID, err := uuid.Parse(basePolicyIDParam)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_UUID", "Invalid base_policy_id format"))
		}
		filter.BasePolicyID = &parsedID
	}

	if farmIDParam := c.Query("farm_id"); farmIDParam != "" {
		parsedID, err := uuid.Parse(farmIDParam)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_UUID", "Invalid farm_id format"))
		}
		filter.FarmID = &parsedID
	}

	// Parse string fields
	filter.PolicyNumber = c.Query("policy_number")
	filter.FarmerID = c.Query("farmer_id")
	filter.InsuranceProviderID = c.Query("insurance_provider_id")

	// Parse status fields
	if statusParam := c.Query("status"); statusParam != "" {
		status := models.PolicyStatus(statusParam)
		filter.Status = &status
	}

	if underwritingStatusParam := c.Query("underwriting_status"); underwritingStatusParam != "" {
		status := models.UnderwritingStatus(underwritingStatusParam)
		filter.UnderwritingStatus = &status
	}

	// Parse presigned URL options
	if includePresignedParam := c.Query("include_presigned_url"); includePresignedParam != "" {
		filter.IncludePresignedURL = includePresignedParam == "true" || includePresignedParam == "1"
	}

	if expiryHoursParam := c.Query("url_expiry_hours"); expiryHoursParam != "" {
		hours, err := strconv.Atoi(expiryHoursParam)
		if err == nil && hours > 0 {
			filter.URLExpiryHours = hours
		}
	}

	// Validate filter
	if err := filter.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Call service
	response, err := h.registeredPolicyService.GetRegisteredPoliciesWithFilters(c.Context(), filter)
	if err != nil {
		slog.Error("Failed to get filtered policies", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", fmt.Sprintf("Failed to retrieve policies: %v", err)))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

// ============================================================================
// FARMER PERMISSION HANDLERS (read-own)
// ============================================================================

// GetFarmerOwnPolicies retrieves all policies for the authenticated farmer
func (h *PolicyHandler) GetFarmerOwnPolicies(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policies, err := h.registeredPolicyService.GetPoliciesByFarmerID(userID)
	if err != nil {
		slog.Error("Failed to get farmer policies", "farmer_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve policies"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"policies":  policies,
		"count":     len(policies),
		"farmer_id": userID,
	}))
}

// GetFarmerPolicyDetail retrieves a specific policy detail for the authenticated farmer
func (h *PolicyHandler) GetFarmerPolicyDetail(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policyIDStr := c.Params("id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	policy, err := h.registeredPolicyService.GetPolicyByID(policyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to get policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve policy"))
	}

	// Authorization check - farmer can only view their own policies
	if policy.FarmerID != userID {
		return c.Status(http.StatusForbidden).JSON(
			utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this policy"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(policy))
}

// ============================================================================
// INSURANCE PARTNER PERMISSION HANDLERS (read-partner, update-partner)
// ============================================================================

// GetPartnerPolicies retrieves all policies for the authenticated insurance partner
func (h *PolicyHandler) GetPartnerPolicies(c fiber.Ctx) error {
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "Authorization token is required"))
	}

	token := strings.TrimPrefix(tokenString, "Bearer ")

	slog.Info("Fetching partner policies token: ", "token", token)
	// calling api to get profile by token
	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	// log the partner profile data
	slog.Info("Partner profile data", "data", partnerProfileData)

	// get partner id from profile data
	partnerID, err := h.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
	}

	// partnerProfileID := partnerProfileData["partner_id"].(string)

	// userID is the insurance provider ID for partners
	policies, err := h.registeredPolicyService.GetPoliciesByProviderID(partnerID)
	if err != nil {
		slog.Error("Failed to get partner policies", "provider_id", partnerID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve policies"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"policies":    policies,
		"count":       len(policies),
		"provider_id": partnerID,
	}))
}

// GetPartnerPolicyDetail retrieves a specific policy detail for the insurance partner
func (h *PolicyHandler) GetPartnerPolicyDetail(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	partnerProfileID, err := h.getPartnerIDFromToken(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	policyIDStr := c.Params("id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	policy, err := h.registeredPolicyService.GetPolicyByID(policyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to get policy", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve policy"))
	}

	// Authorization check - partner can only view their provider's policies
	if policy.InsuranceProviderID != partnerProfileID {
		return c.Status(http.StatusForbidden).JSON(
			utils.CreateErrorResponse("FORBIDDEN", "You do not have permission to view this policy"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(policy))
}

// GetPartnerPolicyStats retrieves policy statistics for the insurance partner
func (h *PolicyHandler) GetPartnerPolicyStats(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	stats, err := h.registeredPolicyService.GetPolicyStats(userID)
	if err != nil {
		slog.Error("Failed to get partner policy stats", "provider_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve statistics"))
	}

	stats["provider_id"] = userID
	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(stats))
}

// GetPartnerMonitoringData retrieves monitoring data for a farm by parameter name (partner access)
func (h *PolicyHandler) GetPartnerMonitoringData(c fiber.Ctx) error {
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

	parameterName := models.DataSourceParameterName(c.Params("parameter_name"))
	if parameterName == "" {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_PARAMETER", "Parameter name is required"))
	}

	// Parse optional time range parameters
	var startTimestamp, endTimestamp *int64
	if startParam := c.Query("start_timestamp"); startParam != "" {
		start, err := strconv.ParseInt(startParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid start_timestamp format"))
		}
		startTimestamp = &start
	}
	if endParam := c.Query("end_timestamp"); endParam != "" {
		end, err := strconv.ParseInt(endParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid end_timestamp format"))
		}
		endTimestamp = &end
	}

	data, err := h.registeredPolicyService.GetMonitoringDataByFarmAndParameter(c.Context(), farmID, parameterName, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get partner monitoring data",
			"farm_id", farmID,
			"parameter_name", parameterName,
			"provider_id", userID,
			"error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve monitoring data"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"monitoring_data": data,
		"count":           len(data),
		"farm_id":         farmID,
		"parameter_name":  parameterName,
		"provider_id":     userID,
	}))
}

// CreatePartnerPolicyUnderwriting creates an underwriting record for partner's policy
func (h *PolicyHandler) CreatePartnerPolicyUnderwriting(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policyIDStr := c.Params("id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	// Parse request body
	var req models.CreatePartnerPolicyUnderwritingRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body: "+err.Error()))
	}

	// Validate request
	if err := req.Validate(); err != nil {
		slog.Error("Request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Verify policy exists
	policy, err := h.registeredPolicyService.GetPolicyByID(policyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve policy"))
	}

	// TODO: Add authorization check - verify user belongs to the insurance provider
	// This will need a profile service call to check if user exists in insurance provider profile
	// For now, we log a warning but allow the operation

	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	slog.Info("Fetching partner policies token: ", "token", token)
	// calling api to get profile by token
	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	if policy.InsuranceProviderID != partnerProfileData["partner_id"].(string) {
		return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", "Cannot underwrite others policies"))
	}

	// Call service to create underwriting
	response, err := h.registeredPolicyService.CreatePartnerPolicyUnderwriting(
		c.Context(),
		policyID,
		req,
		userID,
	)
	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "not found") {
			slog.Error("Resource not found", "error", err)
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", errMsg))
		}

		slog.Error("Failed to create underwriting", "policy_id", policyID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("UNDERWRITING_FAILED", "Failed to create underwriting: "+errMsg))
	}

	slog.Info("Underwriting created successfully",
		"policy_id", policyID,
		"underwriting_id", response.UnderwritingID,
		"status", response.UnderwritingStatus,
		"validated_by", userID)

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(response))
}

// ============================================================================
// ADMIN PERMISSION HANDLERS (read-all, update-any)
// ============================================================================

// GetAllPoliciesAdmin retrieves all policies (admin access)
func (h *PolicyHandler) GetAllPoliciesAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policies, err := h.registeredPolicyService.GetAllPolicies()
	if err != nil {
		slog.Error("Failed to get all policies", "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve policies"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"policies":     policies,
		"count":        len(policies),
		"requested_by": userID,
	}))
}

// GetPolicyDetailAdmin retrieves any policy detail (admin access)
func (h *PolicyHandler) GetPolicyDetailAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policyIDStr := c.Params("id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	policy, err := h.registeredPolicyService.GetPolicyByID(policyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to get policy", "policy_id", policyID, "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve policy"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(policy))
}

// GetAllPolicyStatsAdmin retrieves all policy statistics (admin access)
func (h *PolicyHandler) GetAllPolicyStatsAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Empty provider ID means all policies
	stats, err := h.registeredPolicyService.GetPolicyStats("")
	if err != nil {
		slog.Error("Failed to get all policy stats", "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve statistics"))
	}

	stats["requested_by"] = userID
	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(stats))
}

// UpdatePolicyStatusAdmin updates policy status (admin access)
func (h *PolicyHandler) UpdatePolicyStatusAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policyIDStr := c.Params("id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	var req struct {
		Status models.PolicyStatus `json:"status"`
	}
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	err = h.registeredPolicyService.UpdatePolicyStatus(policyID, req.Status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to update policy status", "policy_id", policyID, "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("UPDATE_FAILED", "Failed to update policy status"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"policy_id":  policyID,
		"status":     req.Status,
		"updated_by": userID,
	}))
}

// UpdatePolicyUnderwritingAdmin updates underwriting status (admin access)
func (h *PolicyHandler) UpdatePolicyUnderwritingAdmin(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	policyIDStr := c.Params("id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	var req struct {
		UnderwritingStatus models.UnderwritingStatus `json:"underwriting_status"`
	}
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	err = h.registeredPolicyService.UpdateUnderwritingStatus(policyID, req.UnderwritingStatus)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("NOT_FOUND", "Policy not found"))
		}
		slog.Error("Failed to update underwriting status", "policy_id", policyID, "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("UPDATE_FAILED", "Failed to update underwriting status"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"policy_id":           policyID,
		"underwriting_status": req.UnderwritingStatus,
		"updated_by":          userID,
	}))
}

func (h *PolicyHandler) GetStatsOverview(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	stats, err := h.registeredPolicyService.GetStatsOverview(userID)
	if err != nil {
		slog.Error("Failed to get stats overview", "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve statistics"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(stats))
}

// GetFarmerMonitoringData retrieves monitoring data for a farmer's own farm
func (h *PolicyHandler) GetFarmerMonitoringData(c fiber.Ctx) error {
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

	// Parse optional time range parameters
	var startTimestamp, endTimestamp *int64
	if startParam := c.Query("start_timestamp"); startParam != "" {
		start, err := strconv.ParseInt(startParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid start_timestamp format"))
		}
		startTimestamp = &start
	}
	if endParam := c.Query("end_timestamp"); endParam != "" {
		end, err := strconv.ParseInt(endParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid end_timestamp format"))
		}
		endTimestamp = &end
	}

	// Get monitoring data - service will verify farm ownership
	data, err := h.registeredPolicyService.GetFarmerMonitoringData(c.Context(), userID, farmID, startTimestamp, endTimestamp)
	if err != nil {
		if strings.Contains(err.Error(), "not authorized") || strings.Contains(err.Error(), "does not own") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have access to this farm's monitoring data"))
		}
		slog.Error("Failed to get farmer monitoring data", "farm_id", farmID, "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve monitoring data"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"monitoring_data": data,
		"count":           len(data),
		"farm_id":         farmID,
	}))
}

// GetFarmerMonitoringDataByParameter retrieves monitoring data for a specific parameter from a farmer's own farm
func (h *PolicyHandler) GetFarmerMonitoringDataByParameter(c fiber.Ctx) error {
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

	parameterName := models.DataSourceParameterName(c.Params("parameter_name"))
	if parameterName == "" {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_PARAMETER", "Parameter name is required"))
	}

	// Parse optional time range parameters
	var startTimestamp, endTimestamp *int64
	if startParam := c.Query("start_timestamp"); startParam != "" {
		start, err := strconv.ParseInt(startParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid start_timestamp format"))
		}
		startTimestamp = &start
	}
	if endParam := c.Query("end_timestamp"); endParam != "" {
		end, err := strconv.ParseInt(endParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid end_timestamp format"))
		}
		endTimestamp = &end
	}

	// Get monitoring data - service will verify farm ownership
	data, err := h.registeredPolicyService.GetFarmerMonitoringDataByParameter(c.Context(), userID, farmID, parameterName, startTimestamp, endTimestamp)
	if err != nil {
		if strings.Contains(err.Error(), "not authorized") || strings.Contains(err.Error(), "does not own") {
			return c.Status(http.StatusForbidden).JSON(
				utils.CreateErrorResponse("FORBIDDEN", "You do not have access to this farm's monitoring data"))
		}
		slog.Error("Failed to get farmer monitoring data by parameter",
			"farm_id", farmID,
			"parameter_name", parameterName,
			"user_id", userID,
			"error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve monitoring data"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"monitoring_data": data,
		"count":           len(data),
		"farm_id":         farmID,
		"parameter_name":  parameterName,
	}))
}

// ============================================================================
// MONITORING DATA ENDPOINTS
// ============================================================================

// GetAllMonitoringData retrieves all farm monitoring data with policy status (admin access)
func (h *PolicyHandler) GetAllMonitoringData(c fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "User ID is required"))
	}

	// Parse optional time range parameters
	var startTimestamp, endTimestamp *int64
	if startParam := c.Query("start_timestamp"); startParam != "" {
		start, err := strconv.ParseInt(startParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid start_timestamp format"))
		}
		startTimestamp = &start
	}
	if endParam := c.Query("end_timestamp"); endParam != "" {
		end, err := strconv.ParseInt(endParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid end_timestamp format"))
		}
		endTimestamp = &end
	}

	data, err := h.registeredPolicyService.GetAllMonitoringDataWithPolicyStatus(c.Context(), startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get all monitoring data", "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve monitoring data"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"monitoring_data": data,
		"count":           len(data),
		"requested_by":    userID,
	}))
}

// GetMonitoringDataByFarm retrieves farm monitoring data with policy status for a specific farm (admin access)
func (h *PolicyHandler) GetMonitoringDataByFarm(c fiber.Ctx) error {
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

	// Parse optional time range parameters
	var startTimestamp, endTimestamp *int64
	if startParam := c.Query("start_timestamp"); startParam != "" {
		start, err := strconv.ParseInt(startParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid start_timestamp format"))
		}
		startTimestamp = &start
	}
	if endParam := c.Query("end_timestamp"); endParam != "" {
		end, err := strconv.ParseInt(endParam, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_PARAMETER", "Invalid end_timestamp format"))
		}
		endTimestamp = &end
	}

	data, err := h.registeredPolicyService.GetMonitoringDataWithPolicyStatusByFarmID(c.Context(), farmID, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get monitoring data by farm", "farm_id", farmID, "admin_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve monitoring data"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"monitoring_data": data,
		"count":           len(data),
		"farm_id":         farmID,
		"requested_by":    userID,
	}))
}

func (h *PolicyHandler) GetMonthlyDataCost(c fiber.Ctx) error {
	var request models.MonthlyDataCostRequest

	if err := c.Bind().Body(&request); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "Authorization token is required"))
	}

	token := strings.TrimPrefix(tokenString, "Bearer ")

	// calling api to get profile by token
	partnerProfileData, err := h.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	// get partner id from profile data
	partnerProfileID, err := h.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
	}

	// Call service
	response, err := h.registeredPolicyService.GetMonthlyDataCost(
		request, partnerProfileID,
	)
	if err != nil {
		slog.Error("Failed to calculate monthly data cost",
			"provider_id", partnerProfileID,
			"error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("CALCULATION_FAILED",
				"Failed to calculate monthly data cost"))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

// Helper function to extract partner ID from authorization token
func (h *PolicyHandler) getPartnerIDFromToken(c fiber.Ctx) (string, error) {
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
