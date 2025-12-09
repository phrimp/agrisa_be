package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/database/redis"
	"policy-service/internal/event"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

type CancelRequestService struct {
	policyRepo        *repository.RegisteredPolicyRepository
	cancelRequestRepo *repository.CancelRequestRepository
	notievent         *event.NotificationHelper
	redisClient       *redis.Client
}

func NewCancelRequestService(
	policyRepo *repository.RegisteredPolicyRepository,
	cancelRequestRepo *repository.CancelRequestRepository,
	notievent *event.NotificationHelper,
	redisClient *redis.Client,
) *CancelRequestService {
	return &CancelRequestService{
		cancelRequestRepo: cancelRequestRepo,
		policyRepo:        policyRepo,
		notievent:         notievent,
		redisClient:       redisClient,
	}
}

func (c *CancelRequestService) CreateCancelRequest(ctx context.Context, policyID uuid.UUID, createdBy string, req models.CreateCancelRequestRequest) (*models.CreateCancelRequestResponse, error) {
	policy, err := c.policyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("error retriving policy", "error", err)
		return nil, fmt.Errorf("error retriving policy by id err=%w", err)
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

	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()

	compensationAmount, err := c.policyRepo.GetCompensationAmount(policy.ID, createdBy, req.CancelRequestType)
	if err != nil {
		return nil, fmt.Errorf("error calculating compensation amount: %w", err)
	}
	request := models.CancelRequest{
		RegisteredPolicyID: policy.ID,
		CancelRequestType:  req.CancelRequestType,
		Reason:             req.Reason,
		Evidence:           req.Evidence,
		CompensateAmount:   int(compensationAmount),
		RequestedBy:        createdBy,
		RequestedAt:        time.Now(),
	}

	if policy.Status == models.PolicyPendingReview || policy.Status == models.PolicyPendingPayment {
		slog.Info("Policy has not activated change status to cancelled", "policy_id", policyID, "status", policy.Status)
		policy.Status = models.PolicyCancelled
		request.Status = models.CancelRequestStatusApproved
	} else {
		policy.Status = models.PolicyPendingCancel
		request.Status = models.CancelRequestStatusPendingReview
	}

	err = c.cancelRequestRepo.CreateNewCancelRequestTx(tx, request)
	if err != nil {
		slog.Error("error creating request cancel for policy", "policy", policy.ID, "error", err)
		return nil, fmt.Errorf("error creating request cancel for policy=%s error=%w", policy.ID, err)
	}

	err = c.policyRepo.UpdateTx(tx, policy)
	if err != nil {
		slog.Error("error updating policy status", "error", err, "policy_id", policyID)
		return nil, fmt.Errorf("error updating policy status error=%w", err)
	}

	if err := tx.Commit(); err != nil {
		slog.Error("error commiting transaction", "error", err)
		return nil, fmt.Errorf("error commiting transaction=%w", err)
	}

	go func() {
		for {
			err := c.notievent.NotifyPolicyCancelRequestCreated(context.Background(), request.RequestedBy, policy.PolicyNumber)
			if err == nil {
				slog.Info("policy cancel request created notification sent", "policy id", policyID)
				return
			}
			slog.Error("error sending policy cancel request created notification", "error", err)
			time.Sleep(10 * time.Second)
		}
	}()

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

// confirm the decision, update the request, start the notice period, and flag the payment process
func (c *CancelRequestService) ReviewCancelRequest(ctx context.Context, review models.ReviewCancelRequestReq) (string, error) {
	request, err := c.cancelRequestRepo.GetCancelRequestByID(review.RequestID)
	if err != nil {
		slog.Error("error retriving cancel request", "error", err)
		return "", err
	}
	if request.Status != models.CancelRequestStatusPendingReview {
		return "", fmt.Errorf("cancel request status invalid")
	}
	if request.RequestedBy == review.ReviewedBy {
		return "", fmt.Errorf("cannot review your own request")
	}

	tx, err := c.policyRepo.BeginTransaction()
	if err != nil {
		slog.Error("error beginning transaction", "error", err)
		return "", fmt.Errorf("error beginning transaction error=%w", err)
	}

	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()

	policy, err := c.policyRepo.GetByID(request.RegisteredPolicyID)
	if err != nil {
		slog.Error("error retriving policy", "error", err)
		return "", fmt.Errorf("error retriving policy by id err=%w", err)

	}

	now := time.Now()
	request.ReviewedBy = &review.ReviewedBy
	request.ReviewedAt = &now
	request.ReviewNotes = &review.ReviewNote

	if policy.Status != models.PolicyPendingCancel {
		return "", fmt.Errorf("policy status invalid for review: expected PendingCancel, got %s", policy.Status)
	}

	if policy.InsuranceProviderID != review.ReviewedBy && policy.FarmerID != review.ReviewedBy {
		return "", fmt.Errorf("reviewer cannot review cancel request of this policy")
	}

	if review.Approved {
		request.Status = models.CancelRequestStatusApproved
	} else {
		policy.Status = models.PolicyDispute
		request.Status = models.CancelRequestStatusLitigation
		err := c.policyRepo.UpdateTx(tx, policy)
		if err != nil {
			slog.Error("error updating policy", "error", err)
			return "", err
		}
	}

	err = c.cancelRequestRepo.UpdateCancelRequestTx(tx, *request)
	if err != nil {
		slog.Error("error updating cancel request", "error", err)
		return "", err
	}

	if err := tx.Commit(); err != nil {
		slog.Error("error commiting transaction", "error", err)
		return "", fmt.Errorf("error commiting transaction=%w", err)
	}

	if review.Approved {
		key := request.ID.String() + "--CancelRequest--NoticePeriod"
		c.redisClient.GetClient().Set(ctx, key, "", models.NoticePeriod)
	}

	return "Cancel Request Reviewed", nil
}

func (c *CancelRequestService) ResolveConflict(ctx context.Context, review models.ResolveConflictCancelRequestReq) (string, error) {
	request, err := c.cancelRequestRepo.GetCancelRequestByID(review.RequestID)
	if err != nil {
		return "", err
	}
	if request.Status != models.CancelRequestStatusLitigation {
		return "", fmt.Errorf("cancel request status invalid")
	}

	if review.FinalDecision != models.CancelRequestStatusApproved && review.FinalDecision != models.CancelRequestStatusDenied {
		return "", fmt.Errorf("final decision status invalid")
	}

	tx, err := c.policyRepo.BeginTransaction()
	if err != nil {
		slog.Error("error beginning transaction", "error", err)
		return "", fmt.Errorf("error beginning transaction error=%w", err)
	}

	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()

	policy, err := c.policyRepo.GetByID(request.RegisteredPolicyID)
	if err != nil {
		slog.Error("error retriving policy", "error", err)
		return "", fmt.Errorf("error retriving policy by id err=%w", err)
	}

	now := time.Now()
	request.ReviewedBy = &review.ReviewedBy
	request.ReviewedAt = &now
	request.ReviewNotes = &review.ReviewNote
	request.Status = review.FinalDecision

	if review.FinalDecision == models.CancelRequestStatusApproved {
		if policy.Status != models.PolicyDispute {
			return "", fmt.Errorf("policy is not in dispute state")
		}
		policy.Status = models.PolicyPendingCancel
	} else {
		policy.Status = models.PolicyActive
	}

	err = c.policyRepo.UpdateTx(tx, policy)
	if err != nil {
		slog.Error("error updating policy", "error", err)
		return "", err
	}

	err = c.cancelRequestRepo.UpdateCancelRequestTx(tx, *request)
	if err != nil {
		slog.Error("error updating cancel request", "error", err)
		return "", err
	}

	if err := tx.Commit(); err != nil {
		slog.Error("error commiting transaction", "error", err)
		return "", fmt.Errorf("error commiting transaction=%w", err)
	}
	if request.Status == models.CancelRequestStatusApproved {
		// start notice period
	}
	return "Cancel Request Reviewed", nil
}

func (s *CancelRequestService) GetCompensationAmount(ctx context.Context, requestID, policyID uuid.UUID) (float64, error) {
	request, err := s.cancelRequestRepo.GetCancelRequestByID(requestID)
	if err != nil {
		return 0, err
	}
	return s.policyRepo.GetCompensationAmount(policyID, request.RequestedBy, request.CancelRequestType)
}

func (s *CancelRequestService) RevokeRequest(ctx context.Context, requestID uuid.UUID) error {
	request, err := s.cancelRequestRepo.GetCancelRequestByID(requestID)
	if err != nil {
		return err
	}
	if request.Status != models.CancelRequestStatusPendingReview {
		return fmt.Errorf("invalid request status")
	}
	policy, err := s.policyRepo.GetByID(request.RegisteredPolicyID)
	if err != nil {
		return err
	}
	if policy.Status != models.PolicyPendingCancel {
		return fmt.Errorf("invalid policy status")
	}
	request.Status = models.CancelRequestStatusCancelled

	return nil
}
