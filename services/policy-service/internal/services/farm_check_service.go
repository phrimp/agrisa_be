package services

import (
	"context"
	"fmt"
	"log"
)

func (s *FarmService) CheckFarmOwner(ownerID string, farmID string) (bool, error) {
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
