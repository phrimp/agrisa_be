package services

import (
	"fmt"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
)

type DataTierService struct {
	dataTierRepo *repository.DataTierRepository
}

func NewDataTierService(dataTierRepo *repository.DataTierRepository) *DataTierService {
	return &DataTierService{dataTierRepo: dataTierRepo}
}

func (s *DataTierService) CreateDataTierCategory(req models.CreateDataTierCategoryRequest) (*models.DataTierCategory, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	category := &models.DataTierCategory{
		CategoryName:           req.CategoryName,
		CategoryDescription:    req.CategoryDescription,
		CategoryCostMultiplier: req.CategoryCostMultiplier,
	}

	if err := s.dataTierRepo.CreateDataTierCategory(category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

func (s *DataTierService) GetDataTierCategoryByID(id uuid.UUID) (*models.DataTierCategory, error) {
	category, err := s.dataTierRepo.GetDataTierCategoryByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return category, nil
}

func (s *DataTierService) GetAllDataTierCategories() ([]models.DataTierCategory, error) {
	categories, err := s.dataTierRepo.GetAllDataTierCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return categories, nil
}

func (s *DataTierService) UpdateDataTierCategory(id uuid.UUID, req models.UpdateDataTierCategoryRequest) (*models.DataTierCategory, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	category, err := s.dataTierRepo.GetDataTierCategoryByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	if req.CategoryName != nil {
		category.CategoryName = *req.CategoryName
	}
	if req.CategoryDescription != nil {
		category.CategoryDescription = req.CategoryDescription
	}
	if req.CategoryCostMultiplier != nil {
		category.CategoryCostMultiplier = *req.CategoryCostMultiplier
	}

	if err := s.dataTierRepo.UpdateDataTierCategory(category); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return category, nil
}

func (s *DataTierService) DeleteDataTierCategory(id uuid.UUID) error {
	tiers, err := s.dataTierRepo.GetDataTiersByCategoryID(id)
	if err != nil {
		return fmt.Errorf("failed to check for existing tiers: %w", err)
	}

	if len(tiers) > 0 {
		return fmt.Errorf("cannot delete category: %d data tiers still exist in this category", len(tiers))
	}

	if err := s.dataTierRepo.DeleteDataTierCategory(id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}

func (s *DataTierService) CreateDataTier(req models.CreateDataTierRequest) (*models.DataTier, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	exists, err := s.dataTierRepo.CheckCategoryExists(req.DataTierCategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to check category existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("category with ID %s does not exist", req.DataTierCategoryID)
	}

	tierExists, err := s.dataTierRepo.CheckTierLevelExists(req.DataTierCategoryID, req.TierLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to check tier level existence: %w", err)
	}
	if tierExists {
		return nil, fmt.Errorf("tier level %d already exists in category %s", req.TierLevel, req.DataTierCategoryID)
	}

	tier := &models.DataTier{
		DataTierCategoryID: req.DataTierCategoryID,
		TierLevel:          req.TierLevel,
		TierName:           req.TierName,
		DataTierMultiplier: req.DataTierMultiplier,
	}

	if err := s.dataTierRepo.CreateDataTier(tier); err != nil {
		return nil, fmt.Errorf("failed to create tier: %w", err)
	}

	return tier, nil
}

func (s *DataTierService) GetDataTierByID(id uuid.UUID) (*models.DataTier, error) {
	tier, err := s.dataTierRepo.GetDataTierByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}

	return tier, nil
}

func (s *DataTierService) GetDataTiersByCategoryID(categoryID uuid.UUID) ([]models.DataTier, error) {
	exists, err := s.dataTierRepo.CheckCategoryExists(categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to check category existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("category with ID %s does not exist", categoryID)
	}

	tiers, err := s.dataTierRepo.GetDataTiersByCategoryID(categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tiers: %w", err)
	}

	return tiers, nil
}

func (s *DataTierService) GetAllDataTiers() ([]models.DataTier, error) {
	tiers, err := s.dataTierRepo.GetAllDataTiers()
	if err != nil {
		return nil, fmt.Errorf("failed to get tiers: %w", err)
	}

	return tiers, nil
}

func (s *DataTierService) UpdateDataTier(id uuid.UUID, req models.UpdateDataTierRequest) (*models.DataTier, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	tier, err := s.dataTierRepo.GetDataTierByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}

	if req.DataTierCategoryID != nil {
		exists, err := s.dataTierRepo.CheckCategoryExists(*req.DataTierCategoryID)
		if err != nil {
			return nil, fmt.Errorf("failed to check category existence: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("category with ID %s does not exist", *req.DataTierCategoryID)
		}
		tier.DataTierCategoryID = *req.DataTierCategoryID
	}

	if req.TierLevel != nil {
		if *req.TierLevel != tier.TierLevel {
			tierExists, err := s.dataTierRepo.CheckTierLevelExists(tier.DataTierCategoryID, *req.TierLevel)
			if err != nil {
				return nil, fmt.Errorf("failed to check tier level existence: %w", err)
			}
			if tierExists {
				return nil, fmt.Errorf("tier level %d already exists in category %s", *req.TierLevel, tier.DataTierCategoryID)
			}
		}
		tier.TierLevel = *req.TierLevel
	}

	if req.TierName != nil {
		tier.TierName = *req.TierName
	}
	if req.DataTierMultiplier != nil {
		tier.DataTierMultiplier = *req.DataTierMultiplier
	}

	if err := s.dataTierRepo.UpdateDataTier(tier); err != nil {
		return nil, fmt.Errorf("failed to update tier: %w", err)
	}

	return tier, nil
}

func (s *DataTierService) DeleteDataTier(id uuid.UUID) error {
	if err := s.dataTierRepo.DeleteDataTier(id); err != nil {
		return fmt.Errorf("failed to delete tier: %w", err)
	}

	return nil
}

func (s *DataTierService) GetDataTierWithCategory(tierID uuid.UUID) (*models.DataTier, *models.DataTierCategory, error) {
	tier, category, err := s.dataTierRepo.GetDataTierWithCategory(tierID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tier with category: %w", err)
	}

	return tier, category, nil
}

func (s *DataTierService) CalculateTotalMultiplier(tierID uuid.UUID) (float64, error) {
	tier, category, err := s.GetDataTierWithCategory(tierID)
	if err != nil {
		return 0, err
	}

	totalMultiplier := category.CategoryCostMultiplier * tier.DataTierMultiplier
	return totalMultiplier, nil
}
