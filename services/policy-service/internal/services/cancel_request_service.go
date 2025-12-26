package services

import (
	utils "agrisa_utils"
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
	claimRepo         *repository.ClaimRepository
}

func NewCancelRequestService(
	policyRepo *repository.RegisteredPolicyRepository,
	cancelRequestRepo *repository.CancelRequestRepository,
	notievent *event.NotificationHelper,
	redisClient *redis.Client,
	claimRepo *repository.ClaimRepository,
) *CancelRequestService {
	return &CancelRequestService{
		cancelRequestRepo: cancelRequestRepo,
		policyRepo:        policyRepo,
		notievent:         notievent,
		redisClient:       redisClient,
		claimRepo:         claimRepo,
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
	if req.CancelRequestType != models.CancelRequestTransferContract {
		claims, err := c.claimRepo.GetByRegisteredPolicyID(ctx, policy.ID)
		if err != nil {
			return nil, err
		}
		for _, claim := range claims {
			if claim.Status == models.ClaimPendingPartnerReview {
				return nil, fmt.Errorf("there are existing pending review claim")
			}
		}
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
		if policy.FarmerID != createdBy {
			slog.Error("cannot direct cancel others policy", "owner", policy.FarmerID, "requested by", createdBy)
			return nil, fmt.Errorf("cannot direct cancel others policy")
		}
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

	if request.CancelRequestType != models.CancelRequestTransferContract {
		err = c.policyRepo.UpdateTx(tx, policy)
		if err != nil {
			slog.Error("error updating policy status", "error", err, "policy_id", policyID)
			return nil, fmt.Errorf("error updating policy status error=%w", err)
		}
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

func (c *CancelRequestService) GetFarmerTransferContract(ctx context.Context, farmerID string, policyID uuid.UUID) ([]models.CancelRequest, error) {
	return c.cancelRequestRepo.GetLatestTransferRequestByFarmer(ctx, farmerID, policyID)
}

// confirm the decision, update the request, start the notice period, and flag the payment process
func (c *CancelRequestService) ReviewCancelRequest(ctx context.Context, review models.ReviewCancelRequestReq) (string, error) {
	now := time.Now()
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
	if now.Compare(request.CreatedAt.Add(1*time.Minute)) == -1 && request.CancelRequestType != models.CancelRequestTransferContract {
		return "", fmt.Errorf("cannot review newly created request")
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

	request.ReviewedBy = &review.ReviewedBy
	request.ReviewedAt = &now
	request.ReviewNotes = &review.ReviewNote

	if policy.Status != models.PolicyPendingCancel && request.CancelRequestType != models.CancelRequestTransferContract {
		return "", fmt.Errorf("policy status invalid for review: expected PendingCancel, got %s", policy.Status)
	}

	if policy.InsuranceProviderID != review.ReviewedBy && policy.FarmerID != review.ReviewedBy {
		return "", fmt.Errorf("reviewer cannot review cancel request of this policy")
	}

	if request.CancelRequestType == models.CancelRequestTransferContract {
		oldProvider := policy.InsuranceProviderID
		policy.InsuranceProviderID = request.Evidence["toProvider"].(string)
		successMessage := ""
		if review.Approved {
			request.Status = models.CancelRequestStatusApproved
			successMessage = "Contract transferred"
			err := c.policyRepo.UpdateTx(tx, policy)
			if err != nil {
				slog.Error("error updating policy", "error", err)
				return "", err
			}
		} else {
			// create real cancel request
			req, err := c.CreateCancelRequest(ctx, policy.ID, *request.ReviewedBy, models.CreateCancelRequestRequest{
				Reason:            fmt.Sprintf("Từ chối chuyển giao hợp đồng từ công ty %s qua công ty %s", oldProvider, policy.InsuranceProviderID),
				CancelRequestType: models.CancelRequestTypeOther,
				Evidence:          utils.JSONMap{"requestID": request.ID},
			})
			if err != nil {
				return "", fmt.Errorf("error creating new cancel request from refusing transfer request: %w", err)
			}
			request.Status = models.CancelRequestStatusDenied
			slog.Info("Cancel request created successfully", "request", req)
			successMessage = "New request cancel created"
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
		return successMessage, nil
	}

	if review.Approved {
		request.Status = models.CancelRequestStatusApproved
		request.DuringNoticePeriod = true
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

	if review.ReviewedBy != *request.ReviewedBy {
		return "", fmt.Errorf("you can not resolve this request")
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

	finalNote := "After Resolse: " + review.ReviewNote
	now := time.Now()
	request.ReviewedBy = &review.ReviewedBy
	request.ReviewedAt = &now
	request.ReviewNotes = &finalNote
	request.Status = review.FinalDecision

	if review.FinalDecision == models.CancelRequestStatusApproved {
		if policy.Status != models.PolicyDispute {
			return "", fmt.Errorf("policy is not in dispute state")
		}
		policy.Status = models.PolicyPendingCancel
		request.DuringNoticePeriod = true
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
		key := request.ID.String() + "--CancelRequest--NoticePeriod"
		c.redisClient.GetClient().Set(ctx, key, "", models.NoticePeriod)
	}
	return "Cancel Request Resolved", nil
}

func (s *CancelRequestService) GetCompensationAmount(ctx context.Context, requestID, policyID uuid.UUID) (float64, error) {
	request, err := s.cancelRequestRepo.GetCancelRequestByID(requestID)
	if err != nil {
		return 0, err
	}
	return s.policyRepo.GetCompensationAmount(policyID, request.RequestedBy, request.CancelRequestType)
}

func (s *CancelRequestService) RevokeRequest(ctx context.Context, requestID uuid.UUID, requestBy string) error {
	request, err := s.cancelRequestRepo.GetCancelRequestByID(requestID)
	if err != nil {
		return err
	}
	policy, err := s.policyRepo.GetByID(request.RegisteredPolicyID)
	if err != nil {
		return err
	}
	if requestBy != request.RequestedBy {
		return fmt.Errorf("you cannot revoke what is not yours")
	}
	if policy.Status != models.PolicyPendingCancel {
		return fmt.Errorf("invalid policy status")
	}
	if request.Status == models.CancelRequestPaymentFailed {
		return fmt.Errorf("invalid request status")
	}

	if request.Status == models.CancelRequestStatusApproved {
		//key := request.ID.String() + "--CancelRequest--NoticePeriod"
		//remainTime, err := s.redisClient.GetClient().TTL(ctx, key).Result()
		//if err != nil {
		//	slog.Error("remaining notice period time failed to retrive", "error", err)
		//	return fmt.Errorf("remaining notice period time failed to retrive: err=%w", err)
		//}
		//if models.NoticePeriod.Hours()-remainTime.Hours() > models.RevokeDeadline*24 {
		//	slog.Error("cannot cancel the request", "detail", fmt.Sprintf("%v day revoke deadline passed", models.RevokeDeadline))
		//	return fmt.Errorf("cannot cancel the request: err= %v day revoke deadline passed", models.RevokeDeadline)
		//}
		//err = s.redisClient.GetClient().Del(ctx, key).Err()
		//if err != nil {
		//	slog.Error("error remove notice period", "request", request.ID)
		//	return fmt.Errorf("error remove notice period for request %s : err=%w", requestID, err)
		//}
		return fmt.Errorf("approved request cannot be revoked")
	}

	request.Status = models.CancelRequestStatusCancelled
	policy.Status = models.PolicyActive

	tx, err := s.policyRepo.BeginTransaction()
	if err != nil {
		slog.Error("error beginning transaction", "error", err)
		return fmt.Errorf("error beginning transaction error=%w", err)
	}

	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()

	err = s.policyRepo.UpdateTx(tx, policy)
	if err != nil {
		slog.Error("error updating policy", "error", err)
		return err
	}

	err = s.cancelRequestRepo.UpdateCancelRequestTx(tx, *request)
	if err != nil {
		slog.Error("error updating cancel request", "error", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		slog.Error("error commiting transaction", "error", err)
		return fmt.Errorf("error commiting transaction=%w", err)
	}
	return nil
}

func (c *CancelRequestService) CreateTransferRequest(ctx context.Context, createdBy string, fromProvider, toProvider string) error {
	policies, err := c.policyRepo.GetByInsuranceProviderIDAndStatus(fromProvider, models.PolicyActive)
	if err != nil {
		return err
	}
	for _, policy := range policies {
		req, err := c.CreateCancelRequest(ctx, policy.ID, fromProvider, models.CreateCancelRequestRequest{
			Reason:            fmt.Sprintf("Chuyển giao hợp đồng từ công ty %s qua công ty %s", fromProvider, toProvider),
			CancelRequestType: models.CancelRequestTransferContract,
			Evidence:          utils.JSONMap{"toProvider": toProvider},
		})
		if err != nil {
			return fmt.Errorf("error creating transfer request for contract: %s error=%w", policy.ID, err)
		}
		slog.Info("Created transfer request policy succeed", "request", req)

		go func() {
			for {
				err := c.notievent.NotifyTransferPolicyRequest(ctx, policy.FarmerID, policy.PolicyNumber, toProvider)
				if err == nil {
					slog.Info("policy transfer request notification sent", "policy_id", policy.ID)
					return
				}
				slog.Error("error sending payout completed notification", "error", err)
				time.Sleep(10 * time.Second)
			}
		}()

	}

	return nil
}

func (c *CancelRequestService) RevokeAllTransferRequest(ctx context.Context, createdBy string, fromProvider string) error {
	requests, err := c.cancelRequestRepo.GetAllByProviderIDWithStatusAndType(ctx, fromProvider, models.CancelRequestStatusPendingReview, models.CancelRequestTransferContract)
	if err != nil {
		return err
	}
	requestIDs := []uuid.UUID{}
	for _, request := range requests {
		requestIDs = append(requestIDs, request.ID)
	}
	res, err := c.cancelRequestRepo.BulkUpdateStatusWhereProviderStatusAndType(ctx, requestIDs, fromProvider, models.CancelRequestStatusPendingReview, models.CancelRequestTransferContract, models.CancelRequestStatusCancelled)
	if err != nil {
		return err
	}
	slog.Info("revoke all transfer request", "count", res)
	return nil
}

func (c *CancelRequestService) CheckProfileCancelReady(ctx context.Context, providerID string) error {
	requests, err := c.cancelRequestRepo.GetAllRequestsByProviderIDWithStatusAndType(ctx, providerID)
	if err != nil {
		return err
	}
	if len(requests) > 0 {
		return fmt.Errorf("there are existing cancel request to resolve: %v", len(requests))
	}
	return nil
}
