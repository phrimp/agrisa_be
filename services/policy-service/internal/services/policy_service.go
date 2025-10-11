package services

import "policy-service/internal/repository"

type PolicyService struct {
	policyRepository *repository.PolicyRepository
}
