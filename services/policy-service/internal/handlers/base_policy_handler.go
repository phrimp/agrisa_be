package handlers

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"policy-service/internal/services"
	"policy-service/internal/worker"
	"strconv"
	"strings"
	"time"

	utils "agrisa_utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type BasePolicyHandler struct {
	basePolicyService       *services.BasePolicyService
	minioClient             *minio.MinioClient
	workerManager           *worker.WorkerManagerV2
	registeredPolicyService *services.RegisteredPolicyService
}

func NewBasePolicyHandler(basePolicyService *services.BasePolicyService, minioClient *minio.MinioClient, workerManager *worker.WorkerManagerV2, registeredPolicyService *services.RegisteredPolicyService) *BasePolicyHandler {
	return &BasePolicyHandler{
		basePolicyService:       basePolicyService,
		minioClient:             minioClient,
		workerManager:           workerManager,
		registeredPolicyService: registeredPolicyService,
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
	policyGroup.Get("/all", bph.GetAllBasePolicies)         // GET /base-policies/all - Get all base policies
	policyGroup.Get("/detail", bph.GetCompletePolicyDetail) // GET  /base-policies/detail - Get complete policy details with PDF
	policyGroup.Get("/by-provider", bph.GetByProvider)
	policyGroup.Put("/cancel/:id", bph.CancelBasePolicy)

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

// func (bph *BasePolicyHandler) GetAllActivePolicy(c fiber.Ctx) error {
// 	activePolicies, err := bph.basePolicyService.GetActivePolicies(c)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to retrive active policies"))
// 	}
// 	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(activePolicies))
// }

func (bph *BasePolicyHandler) GetAllActivePolicy(c fiber.Ctx) error {
	providerID := c.Query("provider_id")
	cropType := c.Query("crop_type")

	activePolicies, err := bph.basePolicyService.GetActivePolicies(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to retrieve active policies"))
	}

	// Filter in memory
	filtered := []models.BasePolicy{}
	for _, p := range activePolicies {
		if (providerID == "" || p.InsuranceProviderID == providerID) && (cropType == "" || p.CropType == cropType) {
			filtered = append(filtered, p)
		}
	}

	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(filtered))
}

// CreateCompletePolicy creates a complete policy (BasePolicy + Trigger + Conditions) in Redis
func (bph *BasePolicyHandler) CreateCompletePolicy(c fiber.Ctx) error {
	var req models.CompletePolicyCreationRequest
	if err := c.Bind().Body(&req); err != nil {
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}
	createdBy := c.Get("X-User-ID")
	req.BasePolicy.CreatedBy = &createdBy

	// Set default expiration if not provided (24 hours)
	expiration := 10 * time.Minute // TODO: update to 24h
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
		slog.Error("base policy creation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("CREATION_FAILED", err.Error()))
	}

	// Decode base64 PDF data before uploading
	pathName := response.FilePath
	pdfData, err := base64.StdEncoding.DecodeString(req.PolicyDocument.Data)
	if err != nil {
		slog.Error("Failed to decode base64 PDF data",
			"base_policy_id", response.BasePolicyID,
			"error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PDF_DATA", "Failed to decode base64 PDF data"))
	}

	err = bph.minioClient.UploadBytes(c.Context(), minio.Storage.PolicyDocuments, pathName, pdfData, "application/pdf")
	if err != nil {
		slog.Error("Failed to upload PDF to MinIO",
			"base_policy_id", response.BasePolicyID,
			"path", pathName,
			"error", err)
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("FILE_UPLOAD_FAILED", err.Error()))
	}

	slog.Info("Successfully uploaded policy document",
		"base_policy_id", response.BasePolicyID,
		"path", pathName,
		"size_bytes", len(pdfData))
	// send job to AI
	job := worker.JobPayload{
		JobID:      uuid.NewString(),
		Type:       "document-validation",
		Params:     map[string]any{"fileName": pathName, "base_policy_id": response.BasePolicyID},
		MaxRetries: 100,
		OneTime:    true,
	}
	scheduler, ok := bph.workerManager.GetSchedulerByPolicyID(*worker.AIWorkerPoolUUID)
	if !ok {
		slog.Error("error get AI scheduler", "error", "scheduler doesn't exist")
	}
	scheduler.AddJob(job)

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

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
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

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
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
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	if err := req.Validate(); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("VALIDATION_FAILED", err.Error()))
	}

	validateBy := c.Get("X-User-ID")
	req.ValidatedBy = validateBy

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
		slog.Error("error parsing request", "error", err)
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
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	slog.Info("Fetching partner policies token: ", "token", token)
	// calling api to get profile by token
	partnerProfileData, err := bph.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	// get partner id from profile data
	partnerID, err := bph.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
	}

	profileData, ok := partnerProfileData["data"].(map[string]any)
	if ok {
		partnerIDProfile, ok := profileData["partner_id"].(string)
		if ok {
			if partnerID != partnerIDProfile {
				return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", "Cannot underwrite others policies"))
			}
		} else {
			return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "partner id not found"))
		}
	} else {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "profile data not fould"))
	}

	providerID := partnerID

	count, err := bph.basePolicyService.GetBasePolicyCount(providerID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
		"total_count": count,
	}))
}

// GetBasePolicyCountByStatus returns count of base policies by status
func (bph *BasePolicyHandler) GetBasePolicyCountByStatus(c fiber.Ctx) error {
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	slog.Info("Fetching partner policies token: ", "token", token)
	// calling api to get profile by token
	partnerProfileData, err := bph.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	// get partner id from profile data
	partnerID, err := bph.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
	}

	profileData, ok := partnerProfileData["data"].(map[string]any)
	if ok {
		partnerIDProfile, ok := profileData["partner_id"].(string)
		if ok {
			if partnerID != partnerIDProfile {
				return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", "Cannot underwrite others policies"))
			}
		} else {
			return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "partner id not found"))
		}
	} else {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "profile data not fould"))
	}

	providerID := partnerID

	statusParam := c.Params("status")
	if statusParam == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_PARAMETER", "Status parameter is required"))
	}

	status := models.BasePolicyStatus(statusParam)
	count, err := bph.basePolicyService.GetBasePolicyCountByStatus(status, providerID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("COUNT_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
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
		slog.Error("error parsing request", "error", err)
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	err = bph.basePolicyService.UpdateBasePolicyValidationStatus(c.Context(), basePolicyID, updateReq.ValidationStatus, updateReq.ValidationScore)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.CreateErrorResponse("UPDATE_FAILED", err.Error()))
	}

	return c.Status(http.StatusOK).JSON(utils.CreateSuccessResponse(map[string]any{
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

func (bph *BasePolicyHandler) GetByProvider(c fiber.Ctx) error {
	tokenString := c.Get("Authorization")

	token := strings.TrimPrefix(tokenString, "Bearer ")

	slog.Info("Fetching partner policies token: ", "token", token)
	// calling api to get profile by token
	partnerProfileData, err := bph.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	// get partner id from profile data
	partnerID, err := bph.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
	}

	profileData, ok := partnerProfileData["data"].(map[string]any)
	if ok {
		partnerIDProfile, ok := profileData["partner_id"].(string)
		if ok {
			if partnerID != partnerIDProfile {
				return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", "Cannot underwrite others policies"))
			}
		} else {
			return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "partner id not found"))
		}
	} else {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "profile data not fould"))
	}

	providerID := partnerID
	policies, err := bph.basePolicyService.GetByProvider(providerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "error get all base policies by provider partner"))
	}
	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(policies))
}

func (bph *BasePolicyHandler) CancelBasePolicy(c fiber.Ctx) error {
	basePolicyIDStr := c.Params("id")
	if basePolicyIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", "id is required"))
	}
	basePolicyID, err := uuid.Parse(basePolicyIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", "Invalid id format"))
	}
	keepRegisterPolicyStr := c.Query("keep_registered_policy")
	keepRegisterPolicy, err := strconv.ParseBool(keepRegisterPolicyStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.CreateErrorResponse("BAD_REQUEST", "keep_registered_policy value"))
	}

	tokenString := c.Get("Authorization")
	if tokenString == "" {
		return c.Status(http.StatusUnauthorized).JSON(
			utils.CreateErrorResponse("UNAUTHORIZED", "Authorization token is required"))
	}

	token := strings.TrimPrefix(tokenString, "Bearer ")

	slog.Info("Fetching partner policies token: ", "token", token)
	// calling api to get profile by token
	partnerProfileData, err := bph.registeredPolicyService.GetInsurancePartnerProfile(token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve insurance partner profile"))
	}

	// get partner id from profile data
	partnerID, err := bph.registeredPolicyService.GetPartnerID(partnerProfileData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("RETRIEVAL_FAILED", "Failed to retrieve partner ID"))
	}

	profileData, ok := partnerProfileData["data"].(map[string]any)
	if ok {
		partnerIDProfile, ok := profileData["partner_id"].(string)
		if ok {
			if partnerID != partnerIDProfile {
				return c.Status(http.StatusUnauthorized).JSON(utils.CreateErrorResponse("UNAUTHORIZED", "Cannot underwrite others policies"))
			}
		} else {
			return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "partner id not found"))
		}
	} else {
		return c.Status(http.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL", "profile data not fould"))
	}

	providerID := partnerID
	res, err := bph.basePolicyService.CancelBasePolicy(c.Context(), basePolicyID, providerID, keepRegisterPolicy)
	if err != nil {
		slog.Error("Failed to cancel base policy", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(
			utils.CreateErrorResponse("CANCEL_FAILED", "Failed to cancel base policy"))
	}
	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(res))
}

func (bph *BasePolicyHandler) GetAllBasePolicies(c fiber.Ctx) error {
	basePolicies, err := bph.basePolicyService.GetAllBasePolicies(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.CreateErrorResponse("INTERNAL_SERVER_ERROR", "failed to retrieve all base policies"))
	}
	return c.Status(fiber.StatusOK).JSON(utils.CreateSuccessResponse(basePolicies))
}
