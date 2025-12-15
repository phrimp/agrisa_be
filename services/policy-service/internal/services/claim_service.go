package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/event"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

type ClaimService struct {
	claimRepo  *repository.ClaimRepository
	policyRepo *repository.RegisteredPolicyRepository
	farmRepo   *repository.FarmRepository
	payoutRepo *repository.PayoutRepository
	notievent  *event.NotificationHelper
}

func NewClaimService(
	claimRepo *repository.ClaimRepository,
	policyRepo *repository.RegisteredPolicyRepository,
	farmRepo *repository.FarmRepository,
	payoutRepo *repository.PayoutRepository,
	notievent *event.NotificationHelper,
) *ClaimService {
	return &ClaimService{
		claimRepo:  claimRepo,
		policyRepo: policyRepo,
		farmRepo:   farmRepo,
		payoutRepo: payoutRepo,
		notievent:  notievent,
	}
}

// GetClaimByID retrieves a claim by ID (no authorization - handled by route permissions)
func (s *ClaimService) GetClaimByID(ctx context.Context, claimID uuid.UUID) (*models.Claim, error) {
	claim, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	return claim, nil
}

// GetClaimsByFarmerID retrieves all claims for a farmer's farms
func (s *ClaimService) GetClaimsByFarmerID(ctx context.Context, farmerID string) ([]models.Claim, error) {
	// Get all farms owned by this farmer
	farms, err := s.farmRepo.GetByOwnerID(ctx, farmerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get farmer's farms: %w", err)
	}

	if len(farms) == 0 {
		return []models.Claim{}, nil
	}

	// Fetch claims for all farmer's farms
	var allClaims []models.Claim
	for _, farm := range farms {
		claims, err := s.claimRepo.GetByFarmID(ctx, farm.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get claims for farm %s: %w", farm.ID, err)
		}
		allClaims = append(allClaims, claims...)
	}
	return allClaims, nil
}

// GetClaimsByProviderID retrieves all claims for a partner's policies
func (s *ClaimService) GetClaimsByProviderID(ctx context.Context, providerID string) ([]models.Claim, error) {
	// Get all policies for this insurance provider
	policies, err := s.policyRepo.GetByInsuranceProviderID(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider's policies: %w", err)
	}

	if len(policies) == 0 {
		return []models.Claim{}, nil
	}

	// Fetch claims for all provider's policies
	var allClaims []models.Claim
	for _, policy := range policies {
		claims, err := s.claimRepo.GetByRegisteredPolicyID(ctx, policy.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get claims for policy %s: %w", policy.ID, err)
		}
		allClaims = append(allClaims, claims...)
	}
	return allClaims, nil
}

// GetAllClaims retrieves all claims with optional filters (admin only)
func (s *ClaimService) GetAllClaims(ctx context.Context, filters map[string]any) ([]models.Claim, error) {
	claims, err := s.claimRepo.GetAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// GetClaimsByPolicyIDForFarmer retrieves claims for a specific policy (farmer authorization)
func (s *ClaimService) GetClaimsByPolicyIDForFarmer(ctx context.Context, policyID uuid.UUID, farmerID string) ([]models.Claim, error) {
	// Verify policy belongs to farmer
	policy, err := s.policyRepo.GetByID(policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if policy.FarmerID != farmerID {
		return nil, fmt.Errorf("unauthorized: policy does not belong to this farmer")
	}

	claims, err := s.claimRepo.GetByRegisteredPolicyID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// GetClaimsByPolicyIDForPartner retrieves claims for a specific policy (partner authorization)
func (s *ClaimService) GetClaimsByPolicyIDForPartner(ctx context.Context, policyID uuid.UUID, providerID string) ([]models.Claim, error) {
	// Verify policy belongs to partner
	policy, err := s.policyRepo.GetByID(policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if policy.InsuranceProviderID != providerID {
		return nil, fmt.Errorf("unauthorized: policy does not belong to this partner")
	}

	claims, err := s.claimRepo.GetByRegisteredPolicyID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// GetClaimsByPolicyID retrieves claims for a specific policy (admin only - no authorization)
func (s *ClaimService) GetClaimsByPolicyID(ctx context.Context, policyID uuid.UUID) ([]models.Claim, error) {
	claims, err := s.claimRepo.GetByRegisteredPolicyID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// GetClaimsByFarmIDForFarmer retrieves claims for a specific farm (farmer authorization)
func (s *ClaimService) GetClaimsByFarmIDForFarmer(ctx context.Context, farmID uuid.UUID, farmerID string) ([]models.Claim, error) {
	// Verify farm belongs to farmer
	farm, err := s.farmRepo.GetFarmByID(ctx, farmID.String())
	if err != nil {
		return nil, fmt.Errorf("farm not found: %w", err)
	}

	if farm.OwnerID != farmerID {
		return nil, fmt.Errorf("unauthorized: farm does not belong to this farmer")
	}

	claims, err := s.claimRepo.GetByFarmID(ctx, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// GetClaimsByFarmID retrieves claims for a specific farm (admin only - no authorization)
func (s *ClaimService) GetClaimsByFarmID(ctx context.Context, farmID uuid.UUID) ([]models.Claim, error) {
	claims, err := s.claimRepo.GetByFarmID(ctx, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// DeleteClaim removes a claim by ID (admin only - no role check)
func (s *ClaimService) DeleteClaim(ctx context.Context, claimID uuid.UUID) error {
	// Check if claim exists
	exists, err := s.claimRepo.Exists(ctx, claimID)
	if err != nil {
		return fmt.Errorf("failed to check claim existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("claim not found")
	}

	// Perform deletion
	err = s.claimRepo.Delete(ctx, claimID)
	if err != nil {
		return fmt.Errorf("failed to delete claim: %w", err)
	}

	return nil
}

// GetClaimByIDForFarmer retrieves a claim by ID with farmer authorization
func (s *ClaimService) GetClaimByIDForFarmer(ctx context.Context, claimID uuid.UUID, farmerID string) (*models.Claim, error) {
	claim, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	// Verify the claim's farm belongs to the farmer
	farm, err := s.farmRepo.GetFarmByID(ctx, claim.FarmID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get farm: %w", err)
	}
	if farm.OwnerID != farmerID {
		return nil, fmt.Errorf("unauthorized: claim does not belong to this farmer")
	}

	return claim, nil
}

// GetClaimByIDForPartner retrieves a claim by ID with partner authorization
func (s *ClaimService) GetClaimByIDForPartner(ctx context.Context, claimID uuid.UUID, providerID string) (*models.Claim, error) {
	claim, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	// Verify the claim's policy belongs to the partner
	policy, err := s.policyRepo.GetByID(claim.RegisteredPolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy.InsuranceProviderID != providerID {
		return nil, fmt.Errorf("unauthorized: claim does not belong to this partner")
	}

	return claim, nil
}

func (s *ClaimService) ValidateClaim(ctx context.Context, claimID uuid.UUID, request models.ValidateClaimRequest, partnerID string) (*models.ValidateClaimResponse, error) {
	claim, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("claim not found: %w", err)
	}

	// Verify the claim's policy belongs to the partner
	policy, err := s.policyRepo.GetByID(claim.RegisteredPolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if policy.InsuranceProviderID != partnerID {
		return nil, fmt.Errorf("unauthorized: claim does not belong to this partner")
	}

	if request.Status != models.ClaimApproved && request.Status != models.ClaimRejected {
		return nil, fmt.Errorf("invalid claim status=%v", request.Status)
	}
	tx, err := s.claimRepo.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}

	claim.Status = request.Status
	claim.PartnerDecision = &request.PartnerDecision
	claim.PartnerNotes = &request.PartnerNotes
	claim.ReviewedBy = &request.ReviewedBy
	err = s.claimRepo.UpdateTx(tx, claim)
	if err != nil {
		tx.Rollback()
		slog.Error("error updating claim", "error", err)
		return nil, fmt.Errorf("error updating claim: %w", err)
	}

	now := time.Now().Unix()
	payout := models.Payout{
		ClaimID:            claim.ID,
		RegisteredPolicyID: policy.ID,
		FarmID:             policy.FarmID,
		FarmerID:           policy.FarmerID,
		PayoutAmount:       claim.ClaimAmount,
		Currency:           "VND",
		Status:             models.PayoutProcessing,
		InitiatedAt:        &now,
	}
	err = s.payoutRepo.CreateTx(tx, &payout)
	if err != nil {
		tx.Rollback()
		slog.Error("error creating payout", "error", err)
		return nil, fmt.Errorf("error creating payout: %w", err)
	}

	res := models.ValidateClaimResponse{
		ClaimID:  claim.ID,
		PayoutID: payout.ID,
	}
	if err := tx.Commit(); err != nil {
		slog.Error("error commiting transaction", "error", err)
		return nil, fmt.Errorf("error commiting transaction: %w", err)
	}

	slog.Info("claim have been validated", "claim_id", claimID, "validated detail", request)

	if claim.Status == models.ClaimApproved {
		go func() {
			for {
				err := s.notievent.NotifyClaimApproved(context.Background(), policy.FarmerID, policy.PolicyNumber, payout.PayoutAmount)
				if err == nil {
					slog.Info("claim approved notification sent", "claim_id", claimID)
					return
				}
				slog.Error("error sending claim approved notificatio", "error", err)
				time.Sleep(10 * time.Second)
			}
		}()
	} else {
		go func() {
			for {
				err := s.notievent.NotifyClaimRejected(context.Background(), policy.FarmerID, policy.PolicyNumber, *claim.PartnerDecision)
				if err == nil {
					slog.Info("claim rejected notification sent", "policy id", policy.ID)
					return
				}
				slog.Error("error sending claim rejected notification", "error", err)
				time.Sleep(10 * time.Second)
			}
		}()
	}

	return &res, nil
}
