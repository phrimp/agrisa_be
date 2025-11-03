package services

import (
	utils "agrisa_utils"
	"fmt"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
)

type FarmService struct {
	farmRepository *repository.FarmRepository
}

func NewFarmService(farmRepo *repository.FarmRepository) *FarmService {
	return &FarmService{farmRepository: farmRepo}
}

func (s *FarmService) GetFarmByID(id string, userID string) (*models.Farm, error) {
	farmID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	farm, err := s.farmRepository.GetByID(farmID)
	if err != nil {
		return nil, err
	}
	if farm.OwnerID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	return farm, nil
}

func (s *FarmService) CreateFarm(farm *models.Farm, ownerID string) error {
	farm.OwnerID = ownerID
	farmcode := utils.GenerateRandomStringWithLength(10)
	farm.FarmCode = &farmcode
	return s.farmRepository.Create(farm)
}

func (s *FarmService) GetAllFarms() ([]models.Farm, error) {
	return s.farmRepository.GetAll()
}

func (s *FarmService) GetByOwnerID(ownerID string) ([]models.Farm, error) {
	return s.farmRepository.GetByOwnerID(ownerID)
}

func (s *FarmService) UpdateFarm(farm *models.Farm, updatedBy string) error {
	// check if farm exists
	_, err := s.farmRepository.GetByID(farm.ID)
	if err != nil {
		return err
	}

	// Validate required fields
	if farm.CropType == "" {
		return fmt.Errorf("badrequest: crop_type is required")
	}
	if farm.AreaSqm <= 0 {
		return fmt.Errorf("badrequest: area_sqm must be greater than 0")
	}

	if (updatedBy != "") && (farm.OwnerID != updatedBy) {
		return fmt.Errorf("unauthorized to update farm")
	}

	// check if farm_code has already existed
	if farm.FarmCode != nil {
		existingFarm, err := s.farmRepository.GetFarmByFarmCode(*farm.FarmCode)
		if err != nil {
			return err
		}

		if existingFarm != nil && existingFarm.ID != farm.ID {
			return fmt.Errorf("badrequest: farm_code already exists")
		}
	}

	// Validate expected_harvest_date is after or equal to planting_date if provided
	if farm.ExpectedHarvestDate != nil {
		if farm.PlantingDate == nil {
			return fmt.Errorf("badrequest: planting_date is required when expected_harvest_date is provided")
		}
		if *farm.ExpectedHarvestDate < *farm.PlantingDate {
			return fmt.Errorf("badrequest: expected_harvest_date must be greater than or equal to planting_date")
		}
	}

	return s.farmRepository.Update(farm)
}

func (s *FarmService) DeleteFarm(id string, deletedBy string) error {
	// check if farm exists
	farmID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid farm ID: %w", err)
	}

	existFarm, err := s.farmRepository.GetByID(farmID)
	if err != nil {
		return err
	}

	if existFarm == nil {
		return fmt.Errorf("not found: farm not found")
	}

	// check if user is authorized to delete farm
	if deletedBy != "" {
		farm, err := s.farmRepository.GetByID(farmID)
		if err != nil {
			return err
		}
		if farm.OwnerID != deletedBy {
			return fmt.Errorf("unauthorized to delete farm")
		}
	}

	return s.farmRepository.Delete(farmID)
}
