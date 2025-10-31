package handlers

import (
	"fmt"
	"net/http"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"strings"
	"time"

	utils "agrisa_utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type BasePolicyHandler struct {
	basePolicyService *services.BasePolicyService
	minioClient       *minio.MinioClient
}

func NewBasePolicyHandler(basePolicyService *services.BasePolicyService, minioClient *minio.MinioClient) *BasePolicyHandler {
	return &BasePolicyHandler{
		basePolicyService: basePolicyService,
		minioClient:       minioClient,
	}
}

func (bph *BasePolicyHandler) Register(app *fiber.App) {
	protectedGr := app.Group("policy/protected/api/v2")

	// Base Policy routes - Business Process Endpoints
	policyGroup := protectedGr.Group("/base-policies")

	// Core business process operations
	policyGroup.Post("/complete", bph.CreateCompletePolicy)                        // POST /base-policies/complete - Create complete policy in Redis
	policyGroup.Get("/draft/provider/:providerID", bph.GetDraftPoliciesByProvider) // GET  /base-policies/draft/provider/{id} - Get provider's draft policies
	policyGroup.Get("/draft/filter", bph.GetDraftPoliciesWithFilter)               // GET  /base-policies/draft/filter - Get policies with flexible filters
	policyGroup.Post("/validate", bph.ValidatePolicy)                              // POST /base-policies/validate - Validate policy & auto-commit
	policyGroup.Post("/commit", bph.CommitPolicies)                                // POST /base-policies/commit - Manual commit policies to DB
	policyGroup.Get("/active", bph.GetAllActivePolicy)
	policyGroup.Get("/detail", bph.GetCompletePolicyDetail) // GET  /base-policies/detail - Get complete policy details with PDF

	// Utility routes
	policyGroup.Get("/count", bph.GetBasePolicyCount)                                 // GET  /base-policies/count - Total policy count
	policyGroup.Get("/count/status/:status", bph.GetBasePolicyCountByStatus)          // GET  /base-policies/count/status/{status} - Count by status
	policyGroup.Patch("/:id/validation-status", bph.UpdateBasePolicyValidationStatus) // PATCH /base-policies/{id}/validation-status - Update validation

	policyManagementGroup := protectedGr.Group("/base-policies-management")
	policyManagementGroup.Get("/base-policies/complete-response", bph.GetAllCompletePolicyCreations)
}

// ============================================================================
// BUSINESS PROCESS OPERATIONS
// ============================================================================

func (bhp *BasePolicyHandler) GetAllCompletePolicyCreations(c fiber.Ctx) error {
	response, err := bhp.basePolicyService.GetAllPolicyCreationResponse(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to retrive policy creation reponse"))
	}
	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

func (bph *BasePolicyHandler) GetAllActivePolicy(c fiber.Ctx) error {
	activePolicies, err := bph.basePolicyService.GetActivePolicies(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to retrive active policies"))
	}
	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(activePolicies))
}

// CreateCompletePolicy creates a complete policy (BasePolicy + Trigger + Conditions) in Redis
func (bph *BasePolicyHandler) CreateCompletePolicy(c fiber.Ctx) error {
	var req models.CompletePolicyCreationRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}
	createdBy := c.Get("X-User-ID")
	req.BasePolicy.CreatedBy = &createdBy

	// Set default expiration if not provided (24 hours)
	expiration := 24 * time.Hour
	if expirationParam := c.Query("expiration_hours"); expirationParam != "" {
		if hours, err := time.ParseDuration(expirationParam + "h"); err == nil {
			expiration = hours
		}
	}
	err := req.Validate()
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	response, err := bph.basePolicyService.CreateCompletePolicy(c.Context(), &req, expiration)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("CREATION_FAILED", err.Error()))
	}

	pathName := "/document/" + req.PolicyDocument.Name + "-" + response.BasePolicyID.String()
	err = bph.minioClient.UploadBytes(c, minio.Storage.PolicyDocuments, pathName, []byte(req.PolicyDocument.Data), "application/pdf")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("FILE_UPLOAD_FAILED", err.Error()))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(response))
}

// GetDraftPoliciesByProvider retrieves all draft policies for a specific provider
func (bph *BasePolicyHandler) GetDraftPoliciesByProvider(c fiber.Ctx) error {
	providerID := c.Params("providerID")
	if providerID == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PARAMETER", "Provider ID is required"))
	}

	archiveStatus := c.Query("archive_status", "false") // Default to non-archived

	policies, err := bph.basePolicyService.GetAllDraftPolicyWFilter(c.Context(), providerID, "", archiveStatus)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"policies":       policies,
		"count":          len(policies),
		"provider_id":    providerID,
		"archive_status": archiveStatus,
	}))
}

// GetDraftPoliciesWithFilter retrieves draft policies with flexible filtering
func (bph *BasePolicyHandler) GetDraftPoliciesWithFilter(c fiber.Ctx) error {
	providerID := c.Query("provider_id")
	basePolicyID := c.Query("base_policy_id")
	archiveStatus := c.Query("archive_status")

	// At least one parameter is required
	if providerID == "" && basePolicyID == "" && archiveStatus == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PARAMETERS", "At least one filter parameter is required"))
	}

	policies, err := bph.basePolicyService.GetAllDraftPolicyWFilter(c.Context(), providerID, basePolicyID, archiveStatus)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("RETRIEVAL_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"policies":       policies,
		"count":          len(policies),
		"provider_id":    providerID,
		"base_policy_id": basePolicyID,
		"archive_status": archiveStatus,
	}))
}

// ValidatePolicy performs manual policy validation and commits to database
func (bph *BasePolicyHandler) ValidatePolicy(c fiber.Ctx) error {
	var req models.ValidatePolicyRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	if err := req.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	validation, err := bph.basePolicyService.ValidatePolicy(c.Context(), &req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_PROCESS_FAILED", err.Error()))
	}

	return c.Status(http.StatusCreated).JSON(utils.CreateSuccessResponse(validation))
}

// CommitPolicies transfers policies from Redis to PostgreSQL database
func (bph *BasePolicyHandler) CommitPolicies(c fiber.Ctx) error {
	var req models.CommitPoliciesRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	if err := req.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	response, err := bph.basePolicyService.CommitPolicies(c.Context(), &req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("COMMIT_FAILED", err.Error()))
	}

	// Return appropriate status based on results
	if response.TotalFailed > 0 && response.TotalCommitted == 0 {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("ALL_COMMITS_FAILED", "All policy commits failed"))
	} else if response.TotalFailed > 0 {
		return c.Status(http.StatusMultiStatus).JSON(utils.CreateSuccessResponse(response)) // Partial success
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(response))
}

// ============================================================================
// UTILITY OPERATIONS
// ============================================================================

// GetBasePolicyCount returns the total count of base policies
func (bph *BasePolicyHandler) GetBasePolicyCount(c fiber.Ctx) error {
	count, err := bph.basePolicyService.GetBasePolicyCount()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"total_count": count,
	}))
}

// GetBasePolicyCountByStatus returns count of base policies by status
func (bph *BasePolicyHandler) GetBasePolicyCountByStatus(c fiber.Ctx) error {
	statusParam := c.Params("status")
	if statusParam == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PARAMETER", "Status parameter is required"))
	}

	status := models.BasePolicyStatus(statusParam)
	count, err := bph.basePolicyService.GetBasePolicyCountByStatus(status)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"status": status,
		"count":  count,
	}))
}

// UpdateBasePolicyValidationStatus updates the validation status of a base policy
func (bph *BasePolicyHandler) UpdateBasePolicyValidationStatus(c fiber.Ctx) error {
	idParam := c.Params("id")
	if idParam == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PARAMETER", "Policy ID is required"))
	}

	basePolicyID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_UUID", "Invalid policy ID format"))
	}

	var updateReq struct {
		ValidationStatus models.ValidationStatus `json:"validation_status"`
		ValidationScore  *float64                `json:"validation_score,omitempty"`
	}

	if err := c.Bind().Body(&updateReq); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	err = bph.basePolicyService.UpdateBasePolicyValidationStatus(c.Context(), basePolicyID, updateReq.ValidationStatus, updateReq.ValidationScore)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UPDATE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]interface{}{
		"base_policy_id":    basePolicyID,
		"validation_status": updateReq.ValidationStatus,
		"validation_score":  updateReq.ValidationScore,
		"updated_at":        time.Now(),
	}))
}

// ============================================================================
// COMPLETE POLICY DETAIL OPERATION
// ============================================================================

// GetCompletePolicyDetail retrieves complete base policy details with document
func (bph *BasePolicyHandler) GetCompletePolicyDetail(c fiber.Ctx) error {
	var filter models.PolicyDetailFilterRequest

	// Parse UUID if provided
	if idParam := c.Query("id"); idParam != "" {
		parsedID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(
				utils.CreateErrorResponse("INVALID_UUID",
					"Invalid policy ID format"))
		}
		filter.ID = &parsedID
	}

	// Parse other query parameters
	filter.ProviderID = c.Query("provider_id")
	filter.CropType = c.Query("crop_type")
	if statusParam := c.Query("status"); statusParam != "" {
		filter.Status = models.BasePolicyStatus(statusParam)
	}

	// Parse boolean parameters
	includePDFParam := c.Query("include_pdf", "true")
	filter.IncludePDF = includePDFParam != "false" && includePDFParam != "0"

	// Parse PDF expiry hours (default 24)
	filter.PDFExpiryHours = 24 // Default 24 hours
	if expiryParam := c.Query("pdf_expiry_hours"); expiryParam != "" {
		var expiry int
		if _, err := fmt.Sscanf(expiryParam, "%d", &expiry); err == nil && expiry > 0 {
			filter.PDFExpiryHours = expiry
		}
	}

	// Validate filter
	if err := filter.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(
			utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	// Get complete policy detail
	detail, err := bph.basePolicyService.GetCompletePolicyDetail(c.Context(), filter)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(http.StatusNotFound).JSON(
				utils.CreateErrorResponse("POLICY_NOT_FOUND",
					"No base policy found matching the criteria"))
		}
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED",
				fmt.Sprintf("Failed to retrieve policy details: %v", err)))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(detail))
}
