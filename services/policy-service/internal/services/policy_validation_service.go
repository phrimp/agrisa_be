package services

import (
	utils "agrisa_utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"policy-service/internal/ai/gemini"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
)

// ValidatePolicy performs manual policy validation with user-controlled metrics
func (s *BasePolicyService) ValidatePolicy(ctx context.Context, request *models.ValidatePolicyRequest) (*models.BasePolicyDocumentValidation, error) {
	slog.Info("Starting policy validation",
		"base_policy_id", request.BasePolicyID,
		"validation_status", request.ValidationStatus,
		"validated_by", request.ValidatedBy,
		"total_checks", request.TotalChecks,
		"passed_checks", request.PassedChecks,
		"failed_checks", request.FailedChecks,
		"warning_count", request.WarningCount)
	start := time.Now()

	// Validate input parameters
	if err := request.Validate(); err != nil {
		slog.Error("Input validation failed",
			"base_policy_id", request.BasePolicyID,
			"error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify policy exists
	basePolicy := &models.BasePolicy{}

	policyPattern := fmt.Sprintf("*--%s--BasePolicy--*", request.BasePolicyID)
	slog.Info("DEBUG pattern", "pattern", policyPattern)
	policyKeys, err := s.basePolicyRepo.FindKeysByPattern(ctx, policyPattern, "--COMMIT_EVENT")
	slog.Info("DEBUG key", "keys", policyKeys)
	if err != nil || len(policyKeys) == 0 {
		slog.Error("Failed to find policy keys", "policy id", request.BasePolicyID, "error", err)
		basePolicy, err = s.basePolicyRepo.GetBasePolicyByID(request.BasePolicyID)
		if err != nil {
			slog.Error("Failed to get base policy",
				"base_policy_id", request.BasePolicyID,
				"error", err)
			return nil, fmt.Errorf("failed to get base policy: %w", err)
		}
	} else {
		if len(policyKeys) > 1 {
			return nil, fmt.Errorf("logic error: many matching policies exist in cache: %v", policyKeys)
		}
		basePolicyByte, err := s.basePolicyRepo.GetTempBasePolicyModels(ctx, policyKeys[0])
		slog.Info("DEBUG data", "data", basePolicyByte)
		if err != nil {
			slog.Info("Failed to get base policy data", "key", policyKeys[0], "error", err)
			return nil, fmt.Errorf("failed to get base policy: %w", err)
		}

		if err := utils.DeserializeModel(basePolicyByte, &basePolicy); err != nil {
			slog.Info("Failed to deserialize base policy", "key", policyKeys[0], "error", err)
			return nil, fmt.Errorf("failed to deserialize base policy: %w", err)
		}
	}

	slog.Info("Retrieved base policy for validation",
		"base_policy_id", request.BasePolicyID,
		"product_name", basePolicy.ProductName,
		"current_status", basePolicy.DocumentValidationStatus)

	// Begin transaction
	tx, err := s.basePolicyRepo.BeginTransaction()
	if err != nil {
		slog.Error("Failed to begin transaction",
			"base_policy_id", request.BasePolicyID,
			"error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create validation record
	validation := &models.BasePolicyDocumentValidation{
		ID:                  uuid.New(),
		BasePolicyID:        request.BasePolicyID,
		ValidationTimestamp: time.Now().Unix(),
		ValidationStatus:    request.ValidationStatus,
		OverallScore:        nil, // Deprecated - always nil
		TotalChecks:         request.TotalChecks,
		PassedChecks:        request.PassedChecks,
		FailedChecks:        request.FailedChecks,
		WarningCount:        request.WarningCount,
		Mismatches:          request.Mismatches,
		Warnings:            request.Warnings,
		Recommendations:     request.Recommendations,
		ExtractedParameters: request.ExtractedParameters,
		ValidatedBy:         &request.ValidatedBy,
		ValidationNotes:     request.ValidationNotes,
		CreatedAt:           time.Now(),
	}

	slog.Info("Created validation record",
		"validation_id", validation.ID,
		"base_policy_id", request.BasePolicyID,
		"validation_status", request.ValidationStatus)

	// Commit temporary draft policy data if present
	if len(policyKeys) > 0 && validation.ValidationStatus == models.ValidationPassed {
		slog.Info("policies data are in temp cache, begin to commit before further operations")
		result, err := s.CommitPolicies(ctx, &models.CommitPoliciesRequest{
			BasePolicyID:    basePolicy.ID.String(),
			DeleteFromRedis: true,
		})
		if err != nil {
			slog.Error("commit temp policy data failed", "error", err)
			return nil, fmt.Errorf("commit temp policy data failed: %w", err)
		}
		if result.TotalCommitted == 0 {
			slog.Error("commit temp policy data failed", "detail", result)
			return nil, fmt.Errorf("commit temp policy data failed: %v", result)
		}

		slog.Info("commit temp policy data successfully", "result", result)

		// Save validation record
		if err := s.basePolicyRepo.CreateBasePolicyDocumentValidation(validation); err != nil {
			slog.Error("Failed to create validation record",
				"base_policy_id", request.BasePolicyID,
				"validation_id", validation.ID,
				"error", err)
			return nil, fmt.Errorf("failed to create validation record: %w", err)
		}

		// Update policy status (without score - score is deprecated)
		if err := s.UpdateBasePolicyValidationStatus(ctx, request.BasePolicyID, request.ValidationStatus, nil); err != nil {
			slog.Error("Failed to update policy validation status",
				"base_policy_id", request.BasePolicyID,
				"validation_status", request.ValidationStatus,
				"error", err)
			return nil, fmt.Errorf("failed to update policy validation status: %w", err)
		}
	} else {
		// Save validation to Redis for non-passed or non-cached policies
		slog.Info("Saving validation to Redis for non-passed status",
			"base_policy_id", request.BasePolicyID,
			"validation_status", request.ValidationStatus,
			"validation_id", validation.ID)

		if err := s.basePolicyRepo.SaveValidationToRedis(ctx, validation); err != nil {
			slog.Error("Failed to save validation to Redis",
				"base_policy_id", request.BasePolicyID,
				"validation_id", validation.ID,
				"error", err)
			return nil, fmt.Errorf("failed to save validation to Redis: %w", err)
		}

		slog.Info("Successfully saved validation to Redis",
			"base_policy_id", request.BasePolicyID,
			"validation_id", validation.ID)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction",
			"base_policy_id", request.BasePolicyID,
			"validation_id", validation.ID,
			"error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Successfully completed policy validation",
		"base_policy_id", request.BasePolicyID,
		"validation_id", validation.ID,
		"validation_status", request.ValidationStatus,
		"total_checks", request.TotalChecks,
		"passed_checks", request.PassedChecks,
		"failed_checks", request.FailedChecks,
		"warning_count", request.WarningCount,
		"duration", time.Since(start))

	go func() {
		for {
			err := s.notievent.NotifyBasePolicyReviewed(context.Background(), basePolicy.InsuranceProviderID, *basePolicy.ProductCode)
			if err == nil {
				slog.Info("policy reviewed notification sent", "base_policy_id", basePolicy.ID)
				return
			}
			slog.Error("error sending policy registeration partner notification", "error", err)
			time.Sleep(10 * time.Second)
		}
	}()

	return validation, nil
}

func (s *BasePolicyService) AIPolicyValidationJob(params map[string]any) error {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("AIPolicyValidationJob: recovered from panic", "panic", r)
		}
	}()

	// Extract and validate parameters
	fileName, ok := params["fileName"].(string)
	if !ok || fileName == "" {
		return fmt.Errorf("invalid or missing fileName parameter")
	}

	basePolicyIDStr, ok := params["base_policy_id"].(string)
	if !ok || basePolicyIDStr == "" {
		return fmt.Errorf("invalid or missing base_policy_id parameter")
	}

	slog.Info("Starting AI policy validation job",
		"base_policy_id", basePolicyIDStr,
		"file_name", fileName)

	// Get policy data
	completePolicies, err := s.GetAllDraftPolicyWFilter(context.Background(), "", basePolicyIDStr, "")
	if err != nil {
		return fmt.Errorf("failed to get draft policy data: %w", err)
	}

	if len(completePolicies) == 0 {
		slog.Warn("No draft policies found for validation",
			"base_policy_id", basePolicyIDStr)
		return nil
	}

	completePolicy := completePolicies[0]

	// Skip if already validated
	if len(completePolicy.Validations) != 0 {
		slog.Info("Policy already has validations, skipping",
			"base_policy_id", basePolicyIDStr,
			"validation_count", len(completePolicy.Validations))
		return nil
	}

	// Parse base policy ID
	basePolicyID, err := uuid.Parse(basePolicyIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse base_policy_id: %w", err)
	}

	// Download document from MinIO
	obj, err := s.minioClient.GetFile(context.Background(), minio.Storage.PolicyDocuments, fileName)
	if err != nil {
		return fmt.Errorf("failed to get document from MinIO: %w", err)
	}
	defer obj.Close()

	templateData, err := io.ReadAll(obj)
	if err != nil {
		return fmt.Errorf("failed to read PDF data: %w", err)
	}

	slog.Info("Document retrieved from MinIO",
		"file_name", fileName,
		"size_bytes", len(templateData))

	// Prepare AI validation request
	inputJSONBytes, err := json.MarshalIndent(completePolicy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal policy data to JSON: %w", err)
	}
	finalPrompt := fmt.Sprintf(gemini.ValidationPromptTemplate, string(inputJSONBytes))

	aiRequestData := map[string]any{"pdf": templateData}

	// Call AI validation service with automatic failover
	slog.Info("Sending validation request to AI service with multi-client failover",
		"base_policy_id", basePolicyIDStr)

	resp, err := gemini.SendAIWithPDFAndRetry(context.Background(), finalPrompt, aiRequestData, s.geminiSelector)
	if err != nil {
		return fmt.Errorf("AI validation request failed: %w", err)
	}

	// Parse AI response into validation request structure
	var aiResponse models.BasePolicyDocumentValidation
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal AI response: %w", err)
	}

	err = json.Unmarshal(respBytes, &aiResponse)
	if err != nil {
		return fmt.Errorf("failed to unmarshal AI response: %w", err)
	}

	slog.Info("AI validation response parsed",
		"base_policy_id", basePolicyIDStr,
		"validation_status", aiResponse.ValidationStatus,
		"total_checks", aiResponse.TotalChecks,
		"passed_checks", aiResponse.PassedChecks,
		"failed_checks", aiResponse.FailedChecks)

	// Create validation request using the same structure as ValidatePolicy
	validationRequest := &models.ValidatePolicyRequest{
		BasePolicyID:     basePolicyID,
		ValidationStatus: aiResponse.ValidationStatus,
		TotalChecks:      aiResponse.TotalChecks,
		PassedChecks:     aiResponse.PassedChecks,
		FailedChecks:     aiResponse.FailedChecks,
		WarningCount:     aiResponse.WarningCount,
		Mismatches:       aiResponse.Mismatches,
		Warnings:         aiResponse.Warnings,
		Recommendations:  aiResponse.Recommendations,
		ValidatedBy:      "AI-System",
		ValidationNotes:  nil,
	}

	// Use existing ValidatePolicy function for consistency
	slog.Info("Saving validation using ValidatePolicy function",
		"base_policy_id", basePolicyIDStr)

	validation, err := s.ValidatePolicy(context.Background(), validationRequest)
	if err != nil {
		return fmt.Errorf("failed to save validation: %w", err)
	}

	slog.Info("AI policy validation job completed successfully",
		"base_policy_id", basePolicyIDStr,
		"validation_id", validation.ID,
		"validation_status", validation.ValidationStatus)

	return nil
}
