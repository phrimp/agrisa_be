package services

import (
	utils "agrisa_utils"
	"context"
	"fmt"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FarmService struct {
	farmRepository *repository.FarmRepository
}

func NewFarmService(farmRepo *repository.FarmRepository) *FarmService {
	return &FarmService{farmRepository: farmRepo}
}

func (s *FarmService) GetFarmByOwnerID(ctx context.Context, userID string) ([]models.Farm, error) {
	farms, err := s.farmRepository.GetByOwnerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return farms, nil
}

func (s *FarmService) CreateFarm(farm *models.Farm, ownerID string) error {
	farm.OwnerID = ownerID
	farmcode := utils.GenerateRandomStringWithLength(10)
	farm.FarmCode = &farmcode
	// // Check if farmer has already owned a farm
	// existingFarm, err := s.farmRepository.GetByOwnerID(context.Background(), ownerID)
	// if err != nil && strings.Contains(err.Error(), "no rows in result set") {
	// 	// no existing farm, proceed to create
	// } else if existingFarm != nil {
	// 	return fmt.Errorf("badrequest: farmer has already owned a farm")
	// }

	return s.farmRepository.Create(farm)
}

func (s *FarmService) CreateFarmTx(farm *models.Farm, ownerID string, tx *sqlx.Tx) error {
	farm.OwnerID = ownerID
	farmcode := utils.GenerateRandomStringWithLength(10)
	farm.FarmCode = &farmcode
	// // Check if farmer has already owned a farm
	// existingFarm, err := s.farmRepository.GetByOwnerID(context.Background(), ownerID)
	// if err != nil && strings.Contains(err.Error(), "no rows in result set") {
	// 	// no existing farm, proceed to create
	// } else if existingFarm != nil {
	// 	return fmt.Errorf("badrequest: farmer has already owned a farm")
	// }

	return s.farmRepository.CreateTx(tx, farm)
}

func (s *FarmService) GetAllFarms(ctx context.Context) ([]models.Farm, error) {
	return s.farmRepository.GetAll(ctx)
}

func (s *FarmService) GetByFarmID(ctx context.Context, farmID string) (*models.Farm, error) {
	return s.farmRepository.GetFarmByID(ctx, farmID)
}

func (s *FarmService) UpdateFarm(ctx context.Context, farm *models.Farm, updatedBy string, farmID string) error {
	// check if farm exists
	_, err := s.farmRepository.GetFarmByID(ctx, farmID)
	if err != nil {
		return err
	}

	farm.ID, err = uuid.Parse(farmID)

	// Validate required fields
	if farm.CropType == "" {
		return fmt.Errorf("badrequest: crop_type is required")
	}
	if farm.AreaSqm <= 0 {
		return fmt.Errorf("badrequest: area_sqm must be greater than 0")
	}
	isDuplicateFarmCode := false
	// check if farm_code has already existed
	if farm.FarmCode != nil {
		existingFarm, err := s.farmRepository.GetFarmByFarmCode(*farm.FarmCode)
		if err != nil && strings.Contains(err.Error(), "no rows in result set") {
			isDuplicateFarmCode = false
		}

		if existingFarm != nil && existingFarm.ID != farm.ID {
			isDuplicateFarmCode = true
		}
	}

	if isDuplicateFarmCode {
		return fmt.Errorf("badrequest: farm_code already exists")
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

func (s *FarmService) DeleteFarm(ctx context.Context, id string, deletedBy string) error {
	// check if farm exists
	farmID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid farm ID: %w", err)
	}

	existFarm, err := s.farmRepository.GetFarmByID(ctx, id)
	if err != nil {
		return err
	}

	if existFarm == nil {
		return fmt.Errorf("not found: farm not found")
	}

	// check if user is authorized to delete farm
	if deletedBy != "" {
		farm, err := s.farmRepository.GetFarmByID(ctx, id)
		if err != nil {
			return err
		}
		if farm.OwnerID != deletedBy {
			return fmt.Errorf("unauthorized to delete farm")
		}
	}

	return s.farmRepository.Delete(farmID)
}

func (s *FarmService) GetFarmPhotoJob(params map[string]any) error {
	// call farm photo api
	// save to db
	return nil
}
