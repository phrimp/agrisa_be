package services

import (
	"context"
	"fmt"
	"log"
	"policy-service/internal/config"
	"policy-service/internal/repository"
)

type FarmCheckService struct {
	farmRepository *repository.FarmRepository
	config         *config.PolicyServiceConfig
}

func NewFarmCheckService(farmRepo *repository.FarmRepository, cfg *config.PolicyServiceConfig) *FarmCheckService {
	return &FarmCheckService{farmRepository: farmRepo, config: cfg}
}

func (s *FarmCheckService) CheckFarmOwner(ownerID string, farmID string) (bool, error) {
	farm, err := s.farmRepository.GetFarmByID(context.Background(), farmID)
	if err != nil {
		return false, err
	}
	if farm.OwnerID != ownerID {
		log.Printf("Owner ID mismatch: expected %s, got %s", farm.OwnerID, ownerID)
		return false, fmt.Errorf("unauthorize: ower id mismatch")
	}
	return true, nil
}
