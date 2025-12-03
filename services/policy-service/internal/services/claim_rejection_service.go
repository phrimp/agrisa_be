package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
)

type ClaimRejectionService struct {
	policyRepo         *repository.RegisteredPolicyRepository
	claimRepo          *repository.ClaimRepository
	claimRejectionRepo *repository.ClaimRejectionRepository
}

func NewClaimRejectionService(
	policyRepo *repository.RegisteredPolicyRepository,
	claimRepo *repository.ClaimRepository,
	claimRejectionRepo *repository.ClaimRejectionRepository,
) *ClaimRejectionService {
	return &ClaimRejectionService{
		policyRepo:         policyRepo,
		claimRepo:          claimRepo,
		claimRejectionRepo: claimRejectionRepo,
	}
}

func (c *ClaimRejectionService) CreateNewClaimRejection(ctx context.Context, claimRejection *models.ClaimRejection, claimID uuid.UUID) (*models.CreateNewClaimRejectionReponse, error) {
	// Get claim and validate
	claim, err := c.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		slog.Error("error retrieving existing claim", "claim_id", claimID, "error", err)
		return nil, fmt.Errorf("claim not found: %w", err)
	}
	if claim == nil {
		slog.Error("claim is nil", "claim_id", claimID)
		return nil, fmt.Errorf("claim not found")
	}

	// Validate claim status before starting transaction
	if claim.Status != models.ClaimPendingPartnerReview {
		slog.Error("invalid operation: invalid status", "claim_id", claimID, "status", claim.Status)
		return nil, fmt.Errorf("invalid operation: claim must be in PendingPartnerReview status, current status: %s", claim.Status)
	}

	// Begin transaction
	tx, err := c.claimRepo.BeginTransaction()
	if err != nil {
		slog.Error("error beginning transaction", "error", err)
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}

	// Ensure rollback on error
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("error rolling back transaction", "error", rbErr)
			}
			slog.Info("transaction rolled back due to error", "claim_id", claimID)
		}
	}()

	// Create claim rejection
	err = c.claimRejectionRepo.CreateNewClaimRejectionTX(tx, *claimRejection)
	if err != nil {
		slog.Error("error creating new claim rejection", "claim_id", claimID, "error", err)
		return nil, fmt.Errorf("error creating claim rejection: %w", err)
	}

	// Update claim status to rejected
	err = c.claimRepo.UpdateStatusTX(tx, ctx, claimID, models.ClaimRejected)
	if err != nil {
		slog.Error("error updating claim status to rejected", "claim_id", claimID, "error", err)
		return nil, fmt.Errorf("error updating claim status: %w", err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		slog.Error("error committing transaction", "claim_id", claimID, "error", err)
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	slog.Info("claim rejection created successfully", "claim_id", claimID, "rejection_id", claimRejection.ID)

	response := models.CreateNewClaimRejectionReponse{
		ClaimRejectionID: claimRejection.ID,
	}

	// TODO: Push notification to farmer
	return &response, nil
}

func (c *ClaimRejectionService) GetAll(ctx context.Context) ([]models.ClaimRejection, error) {
	return c.claimRejectionRepo.GetAllClaimRejection()
}

func (c *ClaimRejectionService) GetByID(ctx context.Context, claimRejectionID uuid.UUID) (*models.ClaimRejection, error) {
	return c.claimRejectionRepo.GetClaimrejectionByID(claimRejectionID)
}

func (c *ClaimRejectionService) GetByClaimID(ctx context.Context, claimID uuid.UUID) (*models.ClaimRejection, error) {
	return c.claimRejectionRepo.GetClaimrejectionByClaimID(claimID)
}

func (c *ClaimRejectionService) Update(ctx context.Context, claimRejection *models.ClaimRejection) error {
	// Verify claim rejection exists
	existing, err := c.claimRejectionRepo.GetClaimrejectionByID(claimRejection.ID)
	if err != nil {
		slog.Error("error retrieving claim rejection for update", "id", claimRejection.ID, "error", err)
		return fmt.Errorf("claim rejection not found: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("claim rejection not found")
	}

	err = c.claimRejectionRepo.UpdateClaimRejection(*claimRejection)
	if err != nil {
		slog.Error("error updating claim rejection", "id", claimRejection.ID, "error", err)
		return fmt.Errorf("error updating claim rejection: %w", err)
	}
	return nil
}

func (c *ClaimRejectionService) Delete(ctx context.Context, claimRejectionID uuid.UUID) error {
	// Verify claim rejection exists
	existing, err := c.claimRejectionRepo.GetClaimrejectionByID(claimRejectionID)
	if err != nil {
		slog.Error("error retrieving claim rejection for deletion", "id", claimRejectionID, "error", err)
		return fmt.Errorf("claim rejection not found: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("claim rejection not found")
	}

	err = c.claimRejectionRepo.DeleteClaimRejectionByID(claimRejectionID)
	if err != nil {
		slog.Error("error deleting claim rejection", "id", claimRejectionID, "error", err)
		return fmt.Errorf("error deleting claim rejection: %w", err)
	}
	return nil
}

// GetAllByProviderID retrieves all claim rejections for a partner's policies
func (c *ClaimRejectionService) GetAllByProviderID(ctx context.Context, providerID string) ([]models.ClaimRejection, error) {
	// Get all policies for this insurance provider
	policies, err := c.policyRepo.GetByInsuranceProviderID(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider's policies: %w", err)
	}

	if len(policies) == 0 {
		return []models.ClaimRejection{}, nil
	}

	// Fetch claim rejections for all provider's policies
	var allClaimRejections []models.ClaimRejection
	for _, policy := range policies {
		claims, err := c.claimRepo.GetByRegisteredPolicyID(ctx, policy.ID)
		if err != nil {
			slog.Warn("failed to get claims for policy", "policy_id", policy.ID, "error", err)
			continue
		}

		for _, claim := range claims {
			claimRejection, err := c.claimRejectionRepo.GetClaimrejectionByClaimID(claim.ID)
			if err != nil {
				// No rejection for this claim, skip
				continue
			}
			if claimRejection != nil {
				allClaimRejections = append(allClaimRejections, *claimRejection)
			}
		}
	}

	return allClaimRejections, nil
}

// GetByIDForPartner retrieves a claim rejection by ID with partner authorization
func (c *ClaimRejectionService) GetByIDForPartner(ctx context.Context, claimRejectionID uuid.UUID, providerID string) (*models.ClaimRejection, error) {
	claimRejection, err := c.claimRejectionRepo.GetClaimrejectionByID(claimRejectionID)
	if err != nil {
		return nil, fmt.Errorf("claim rejection not found: %w", err)
	}

	// Get the associated claim
	claim, err := c.claimRepo.GetByID(ctx, claimRejection.ClaimID)
	if err != nil {
		return nil, fmt.Errorf("failed to get associated claim: %w", err)
	}

	// Verify the claim's policy belongs to the partner
	policy, err := c.policyRepo.GetByID(claim.RegisteredPolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy.InsuranceProviderID != providerID {
		return nil, fmt.Errorf("unauthorized: claim rejection does not belong to this partner")
	}

	return claimRejection, nil
}

// GetByClaimIDForPartner retrieves a claim rejection by claim ID with partner authorization
func (c *ClaimRejectionService) GetByClaimIDForPartner(ctx context.Context, claimID uuid.UUID, providerID string) (*models.ClaimRejection, error) {
	// Verify the claim belongs to the partner
	claim, err := c.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	policy, err := c.policyRepo.GetByID(claim.RegisteredPolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy.InsuranceProviderID != providerID {
		return nil, fmt.Errorf("unauthorized: claim does not belong to this partner")
	}

	// Get the claim rejection
	claimRejection, err := c.claimRejectionRepo.GetClaimrejectionByClaimID(claimID)
	if err != nil {
		return nil, fmt.Errorf("claim rejection not found: %w", err)
	}

	return claimRejection, nil
}

// CreateClaimRejectionForPartner creates a claim rejection with partner authorization
func (c *ClaimRejectionService) CreateClaimRejectionForPartner(ctx context.Context, claimRejection *models.ClaimRejection, claimID uuid.UUID, providerID string) (*models.CreateNewClaimRejectionReponse, error) {
	// Verify the claim belongs to the partner
	claim, err := c.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		slog.Error("error retrieving claim for partner rejection", "claim_id", claimID, "error", err)
		return nil, fmt.Errorf("claim not found: %w", err)
	}
	if claim == nil {
		return nil, fmt.Errorf("claim not found")
	}

	// Verify ownership
	policy, err := c.policyRepo.GetByID(claim.RegisteredPolicyID)
	if err != nil {
		slog.Error("error retrieving policy for partner rejection", "policy_id", claim.RegisteredPolicyID, "error", err)
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy.InsuranceProviderID != providerID {
		slog.Warn("unauthorized partner attempting to reject claim", "claim_id", claimID, "partner_id", providerID, "policy_partner", policy.InsuranceProviderID)
		return nil, fmt.Errorf("unauthorized: claim does not belong to this partner")
	}

	// Use the standard creation method which handles status update
	return c.CreateNewClaimRejection(ctx, claimRejection, claimID)
}
