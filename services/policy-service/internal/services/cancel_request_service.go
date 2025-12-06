package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
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

func (c *CancelRequestService) CreateCancelRequest(ctx context.Context, policyID uuid.UUID, createdBy string, request models.CancelRequest) (*models.CreateCancelRequestResponse, error) {
	policy, err := c.policyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("error retriving policy", "error", err)
		return nil, fmt.Errorf("error retriving policy by id err=%w", "err")
	}
	allowedStatus := map[models.PolicyStatus]bool{
		models.PolicyActive:         true,
		models.PolicyPendingPayment: true,
		models.PolicyPendingReview:  true,
	}

	if createdBy != policy.FarmerID && createdBy != policy.InsuranceProviderID {
		return nil, fmt.Errorf("operation invalid")
	}

	if !allowedStatus[policy.Status] {
		return nil, fmt.Errorf("invalid policy status")
	}

	tx, err := c.policyRepo.BeginTransaction()
	if err != nil {
		slog.Error("error beginning transaction", "error", err)
		return nil, fmt.Errorf("error beginning transaction error=%w", err)
	}

	if policy.Status == models.PolicyPendingReview || policy.Status == models.PolicyPendingPayment {
		slog.Info("Policy has not activated change status to cancelled", "policy_id", policyID, "status", policy.Status)
		err = c.policyRepo.UpdateStatus(policyID, models.PolicyCancelled)
		if err != nil {
			slog.Error("error updating policy status", "error", err, "policy_id", policyID)
			return nil, fmt.Errorf("error updating policy status error=%w", err)
		}
		return &models.CreateCancelRequestResponse{}, nil
	}

	err = c.cancelRequestRepo.CreateNewCancelRequestTx(tx, request)
	if err != nil {
		tx.Rollback()
		slog.Error("error creating request cancel for policy", "policy", policy.ID, "error", err)
		return nil, fmt.Errorf("error creating request cancel for policy=%s error=%w", policy.ID, err)
	}

	err = c.policyRepo.UpdateStatus(policyID, models.PolicyPendingCancel)
	if err != nil {
		tx.Rollback()
		slog.Error("error updating policy status", "error", err, "policy_id", policyID)
		return nil, fmt.Errorf("error updating policy status error=%w", err)
	}

	return &models.CreateCancelRequestResponse{}, nil
}

func (c *CancelRequestService) GetAllFarmerCancelRequest(ctx context.Context, farmerID string) ([]models.CancelRequest, error) {
	return c.cancelRequestRepo.GetAllRequestsByFarmerID(ctx, farmerID)
}

func (c *CancelRequestService) GetByPolicyID(ctx context.Context, policyID uuid.UUID) ([]models.CancelRequest, error) {
	return c.cancelRequestRepo.GetCancelRequestByPolicyID(policyID)
}

func (c *CancelRequestService) GetAllProviderCancelRequests(ctx context.Context, providerID string) ([]models.CancelRequest, error) {
	return c.cancelRequestRepo.GetAllRequestsByProviderID(ctx, providerID)
}
