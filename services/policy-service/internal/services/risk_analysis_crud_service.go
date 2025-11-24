package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
)

type RiskAnalysisCRUDService struct {
	registeredPolicyRepo *repository.RegisteredPolicyRepository
}

func NewRiskAnalysisCRUDService(registeredPolicyRepo *repository.RegisteredPolicyRepository) *RiskAnalysisCRUDService {
	return &RiskAnalysisCRUDService{
		registeredPolicyRepo: registeredPolicyRepo,
	}
}

// GetByPolicyIDOwn retrieves all risk analyses for a farmer's own policy with ownership verification
func (s *RiskAnalysisCRUDService) GetByPolicyIDOwn(ctx context.Context, userID string, policyID uuid.UUID) ([]models.RegisteredPolicyRiskAnalysis, error) {
	slog.Info("Getting risk analyses by policy ID (own)", "user_id", userID, "policy_id", policyID)

	// Verify policy exists and user owns it
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("Policy not found", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if policy.FarmerID != userID {
		slog.Warn("User does not own policy",
			"user_id", userID,
			"policy_id", policyID,
			"policy_farmer_id", policy.FarmerID)
		return nil, fmt.Errorf("user does not own this policy")
	}

	analyses, err := s.registeredPolicyRepo.GetRiskAnalysesByPolicyID(policyID)
	if err != nil {
		slog.Error("Failed to get risk analyses by policy ID", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("failed to get risk analyses: %w", err)
	}

	slog.Info("Successfully retrieved risk analyses (own)",
		"user_id", userID,
		"policy_id", policyID,
		"count", len(analyses))
	return analyses, nil
}

// GetLatestByPolicyIDOwn retrieves the most recent risk analysis for a farmer's own policy
func (s *RiskAnalysisCRUDService) GetLatestByPolicyIDOwn(ctx context.Context, userID string, policyID uuid.UUID) (*models.RegisteredPolicyRiskAnalysis, error) {
	slog.Info("Getting latest risk analysis by policy ID (own)", "user_id", userID, "policy_id", policyID)

	// Verify policy exists and user owns it
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("Policy not found", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if policy.FarmerID != userID {
		slog.Warn("User does not own policy",
			"user_id", userID,
			"policy_id", policyID,
			"policy_farmer_id", policy.FarmerID)
		return nil, fmt.Errorf("user does not own this policy")
	}

	analysis, err := s.registeredPolicyRepo.GetLatestRiskAnalysis(policyID)
	if err != nil {
		slog.Error("Failed to get latest risk analysis", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("failed to get latest risk analysis: %w", err)
	}

	return analysis, nil
}

// GetByID retrieves a specific risk analysis by ID
func (s *RiskAnalysisCRUDService) GetByID(ctx context.Context, id uuid.UUID) (*models.RegisteredPolicyRiskAnalysis, error) {
	slog.Info("Getting risk analysis by ID", "id", id)

	analysis, err := s.registeredPolicyRepo.GetRiskAnalysisByID(id)
	if err != nil {
		slog.Error("Failed to get risk analysis", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get risk analysis: %w", err)
	}

	return analysis, nil
}

// GetByPolicyID retrieves all risk analyses for a specific policy
func (s *RiskAnalysisCRUDService) GetByPolicyID(ctx context.Context, policyID uuid.UUID) ([]models.RegisteredPolicyRiskAnalysis, error) {
	slog.Info("Getting risk analyses by policy ID", "policy_id", policyID)

	// Verify policy exists
	_, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("Policy not found", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	analyses, err := s.registeredPolicyRepo.GetRiskAnalysesByPolicyID(policyID)
	if err != nil {
		slog.Error("Failed to get risk analyses by policy ID", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("failed to get risk analyses: %w", err)
	}

	slog.Info("Successfully retrieved risk analyses",
		"policy_id", policyID,
		"count", len(analyses))
	return analyses, nil
}

// GetLatestByPolicyID retrieves the most recent risk analysis for a policy
func (s *RiskAnalysisCRUDService) GetLatestByPolicyID(ctx context.Context, policyID uuid.UUID) (*models.RegisteredPolicyRiskAnalysis, error) {
	slog.Info("Getting latest risk analysis by policy ID", "policy_id", policyID)

	// Verify policy exists
	_, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("Policy not found", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	analysis, err := s.registeredPolicyRepo.GetLatestRiskAnalysis(policyID)
	if err != nil {
		slog.Error("Failed to get latest risk analysis", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("failed to get latest risk analysis: %w", err)
	}

	return analysis, nil
}

// GetAll retrieves all risk analyses with pagination
func (s *RiskAnalysisCRUDService) GetAll(ctx context.Context, limit, offset int) ([]models.RegisteredPolicyRiskAnalysis, error) {
	slog.Info("Getting all risk analyses", "limit", limit, "offset", offset)

	// Validate pagination parameters
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	analyses, err := s.registeredPolicyRepo.GetAllRiskAnalyses(limit, offset)
	if err != nil {
		slog.Error("Failed to get all risk analyses", "error", err)
		return nil, fmt.Errorf("failed to get risk analyses: %w", err)
	}

	slog.Info("Successfully retrieved risk analyses", "count", len(analyses))
	return analyses, nil
}

// Delete removes a risk analysis by ID
func (s *RiskAnalysisCRUDService) Delete(ctx context.Context, id uuid.UUID, deletedBy string) error {
	slog.Info("Deleting risk analysis", "id", id, "deleted_by", deletedBy)

	// Verify the risk analysis exists
	analysis, err := s.registeredPolicyRepo.GetRiskAnalysisByID(id)
	if err != nil {
		slog.Error("Risk analysis not found for deletion", "id", id, "error", err)
		return fmt.Errorf("risk analysis not found: %w", err)
	}

	// Log the deletion with context
	slog.Info("Risk analysis found, proceeding with deletion",
		"id", id,
		"registered_policy_id", analysis.RegisteredPolicyID,
		"analysis_type", analysis.AnalysisType,
		"deleted_by", deletedBy)

	err = s.registeredPolicyRepo.DeleteRiskAnalysis(id)
	if err != nil {
		slog.Error("Failed to delete risk analysis", "id", id, "error", err)
		return fmt.Errorf("failed to delete risk analysis: %w", err)
	}

	slog.Info("Successfully deleted risk analysis",
		"id", id,
		"registered_policy_id", analysis.RegisteredPolicyID,
		"deleted_by", deletedBy)
	return nil
}

// Create creates a new risk analysis record
func (s *RiskAnalysisCRUDService) Create(ctx context.Context, analysis *models.RegisteredPolicyRiskAnalysis) error {
	slog.Info("Creating risk analysis",
		"registered_policy_id", analysis.RegisteredPolicyID,
		"analysis_type", analysis.AnalysisType)

	// Verify policy exists
	_, err := s.registeredPolicyRepo.GetByID(analysis.RegisteredPolicyID)
	if err != nil {
		slog.Error("Policy not found for risk analysis creation",
			"policy_id", analysis.RegisteredPolicyID,
			"error", err)
		return fmt.Errorf("policy not found: %w", err)
	}

	err = s.registeredPolicyRepo.CreateRiskAnalysis(analysis)
	if err != nil {
		slog.Error("Failed to create risk analysis",
			"registered_policy_id", analysis.RegisteredPolicyID,
			"error", err)
		return fmt.Errorf("failed to create risk analysis: %w", err)
	}

	slog.Info("Successfully created risk analysis",
		"id", analysis.ID,
		"registered_policy_id", analysis.RegisteredPolicyID)
	return nil
}
