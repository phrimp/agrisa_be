package services

import (
	"context"
	"fmt"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
)

type PayoutService struct {
	payoutRepo *repository.PayoutRepository
	policyRepo *repository.RegisteredPolicyRepository
	farmRepo   *repository.FarmRepository
}

func NewPayoutService(
	payoutRepo *repository.PayoutRepository,
	policyRepo *repository.RegisteredPolicyRepository,
	farmRepo *repository.FarmRepository,
) *PayoutService {
	return &PayoutService{
		payoutRepo: payoutRepo,
		policyRepo: policyRepo,
		farmRepo:   farmRepo,
	}
}

// GetPayoutByID retrieves a payout by ID (no authorization - handled by route permissions)
func (s *PayoutService) GetPayoutByID(ctx context.Context, payoutID uuid.UUID) (*models.Payout, error) {
	payout, err := s.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, fmt.Errorf("payout not found: %w", err)
	}

	return payout, nil
}

// GetPayoutByClaimID retrieves a payout by claim ID (no authorization - handled by route permissions)
func (s *PayoutService) GetPayoutByClaimID(ctx context.Context, claimID uuid.UUID) (*models.Payout, error) {
	payout, err := s.payoutRepo.GetByClaimID(ctx, claimID)
	if err != nil {
		return nil, fmt.Errorf("payout not found for claim: %w", err)
	}

	return payout, nil
}

// GetPayoutsByRegisteredPolicyID retrieves all payouts for a registered policy (no authorization - handled by route permissions)
func (s *PayoutService) GetPayoutsByRegisteredPolicyID(ctx context.Context, policyID uuid.UUID) ([]models.Payout, error) {
	payouts, err := s.payoutRepo.GetByRegisteredPolicyID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts for policy: %w", err)
	}

	return payouts, nil
}

// GetPayoutsByFarmID retrieves all payouts for a farm (no authorization - handled by route permissions)
func (s *PayoutService) GetPayoutsByFarmID(ctx context.Context, farmID uuid.UUID) ([]models.Payout, error) {
	payouts, err := s.payoutRepo.GetByFarmID(ctx, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts for farm: %w", err)
	}

	return payouts, nil
}

// GetPayoutsByFarmerID retrieves all payouts for a farmer (no authorization - handled by route permissions)
func (s *PayoutService) GetPayoutsByFarmerID(ctx context.Context, farmerID string) ([]models.Payout, error) {
	payouts, err := s.payoutRepo.GetByFarmerID(ctx, farmerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts for farmer: %w", err)
	}

	return payouts, nil
}

// GetPayoutByIDForFarmer retrieves a payout by ID with farmer authorization
func (s *PayoutService) GetPayoutByIDForFarmer(ctx context.Context, payoutID uuid.UUID, farmerID string) (*models.Payout, error) {
	payout, err := s.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, fmt.Errorf("payout not found: %w", err)
	}

	// Verify the payout belongs to the farmer
	if payout.FarmerID != farmerID {
		return nil, fmt.Errorf("unauthorized: payout does not belong to this farmer")
	}

	return payout, nil
}

// GetPayoutByIDForPartner retrieves a payout by ID with partner authorization
func (s *PayoutService) GetPayoutByIDForPartner(ctx context.Context, payoutID uuid.UUID, providerID string) (*models.Payout, error) {
	payout, err := s.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, fmt.Errorf("payout not found: %w", err)
	}

	// Verify the payout's policy belongs to the partner
	policy, err := s.policyRepo.GetByID(payout.RegisteredPolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if policy.InsuranceProviderID != providerID {
		return nil, fmt.Errorf("unauthorized: payout does not belong to this partner")
	}

	return payout, nil
}

// GetPayoutsByRegisteredPolicyIDForFarmer retrieves payouts for a policy with farmer authorization
func (s *PayoutService) GetPayoutsByRegisteredPolicyIDForFarmer(ctx context.Context, policyID uuid.UUID, farmerID string) ([]models.Payout, error) {
	// Verify policy belongs to farmer
	policy, err := s.policyRepo.GetByID(policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if policy.FarmerID != farmerID {
		return nil, fmt.Errorf("unauthorized: policy does not belong to this farmer")
	}

	payouts, err := s.payoutRepo.GetByRegisteredPolicyID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts: %w", err)
	}

	return payouts, nil
}

// GetPayoutsByRegisteredPolicyIDForPartner retrieves payouts for a policy with partner authorization
func (s *PayoutService) GetPayoutsByRegisteredPolicyIDForPartner(ctx context.Context, policyID uuid.UUID, providerID string) ([]models.Payout, error) {
	// Verify policy belongs to partner
	policy, err := s.policyRepo.GetByID(policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if policy.InsuranceProviderID != providerID {
		return nil, fmt.Errorf("unauthorized: policy does not belong to this partner")
	}

	payouts, err := s.payoutRepo.GetByRegisteredPolicyID(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts: %w", err)
	}

	return payouts, nil
}

// GetPayoutsByFarmIDForFarmer retrieves payouts for a farm with farmer authorization
func (s *PayoutService) GetPayoutsByFarmIDForFarmer(ctx context.Context, farmID uuid.UUID, farmerID string) ([]models.Payout, error) {
	// Verify farm belongs to farmer
	farm, err := s.farmRepo.GetFarmByID(ctx, farmID.String())
	if err != nil {
		return nil, fmt.Errorf("farm not found: %w", err)
	}

	if farm.OwnerID != farmerID {
		return nil, fmt.Errorf("unauthorized: farm does not belong to this farmer")
	}

	payouts, err := s.payoutRepo.GetByFarmID(ctx, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts: %w", err)
	}

	return payouts, nil
}

// GetPayoutsByFarmIDForPartner retrieves payouts for a farm with partner authorization
func (s *PayoutService) GetPayoutsByFarmIDForPartner(ctx context.Context, farmID uuid.UUID, providerID string) ([]models.Payout, error) {
	// Get all payouts for the farm
	payouts, err := s.payoutRepo.GetByFarmID(ctx, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts: %w", err)
	}

	// Verify at least one payout's policy belongs to the partner
	// If no payouts exist, return empty list
	if len(payouts) == 0 {
		return payouts, nil
	}

	// Verify the first payout's policy belongs to the partner
	// (all payouts for a farm should belong to the same partner through their policies)
	policy, err := s.policyRepo.GetByID(payouts[0].RegisteredPolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if policy.InsuranceProviderID != providerID {
		return nil, fmt.Errorf("unauthorized: payouts do not belong to this partner")
	}

	return payouts, nil
}
