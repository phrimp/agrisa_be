package services

import (
	"context"
	"policy-service/internal/models"
	"policy-service/internal/repository"
)

type CancelRequestService struct {
	policyRepo        *repository.RegisteredPolicyRepository
	cancelRequestRepo *repository.CancelRequestRepository
}

func NewCancelRequestService(
	policyRepo *repository.RegisteredPolicyRepository,
	cancelRequestRepo *repository.CancelRequestRepository,
) *CancelRequestService {
	return &CancelRequestService{
		cancelRequestRepo: cancelRequestRepo,
		policyRepo:        policyRepo,
	}
}

func (c *CancelRequestService) CreateCancelRequest(ctx context.Context, payout models.Payout) (*models.CreateCancelRequestResponse, error) {
	return nil, nil
}
