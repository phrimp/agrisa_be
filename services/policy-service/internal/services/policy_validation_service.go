package services

import (
	utils "agrisa_utils"
	"context"
	"fmt"
	"log/slog"
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
		slog.Info("commit temp policy data successfully", "result", result)
	}

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

	return validation, nil
}
