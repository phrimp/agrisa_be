package services

import "policy-service/internal/repository"

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
