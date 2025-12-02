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

func (c *ClaimRejectionService) CreateNewClaimRejection(ctx context.Context, claimRejection *models.ClaimRejection, policyID uuid.UUID) (*models.CreateNewClaimRejectionReponse, error) {
	claim, err := c.claimRepo.GetByRegisteredPolicyID(ctx, policyID)
	if err != nil || len(claim) == 0 {
		slog.Error("error retrieving existing claims", "error", err, "claim length", len(claim))
		return nil, fmt.Errorf("error retrieving existing claims or claim not found: %w", err)
	}
	err = c.claimRejectionRepo.CreateNewClaimRejection(*claimRejection)
	if err != nil {
		slog.Error("error creating new claim rejection", "error", err)
		return nil, fmt.Errorf("error creating new claim rejection", "error", err)
	}
	response := models.CreateNewClaimRejectionReponse{
		ClaimRejectionID: claimRejection.ID,
	}
	// Push Noti to farmer id
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
