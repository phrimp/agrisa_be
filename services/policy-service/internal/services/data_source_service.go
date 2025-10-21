package services

import (
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
)

type DataSourceService struct {
	repo *repository.DataSourceRepository
}

func NewDataSourceService(repo *repository.DataSourceRepository) *DataSourceService {
	return &DataSourceService{
		repo: repo,
	}
}

// ============================================================================
// CREATE OPERATIONS
// ============================================================================

func (s *DataSourceService) CreateDataSource(dataSource *models.DataSource) error {
	slog.Info("DataSourceService: Creating data source",
		"parameter_name", dataSource.ParameterName,
		"data_source_type", dataSource.DataSource)

	// Validate input
	if err := s.validateDataSource(dataSource); err != nil {
		slog.Error("DataSourceService: Validation failed", "error", err)
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate ID if not provided
	if dataSource.ID == uuid.Nil {
		dataSource.ID = uuid.New()
	}

	// Set default values
	if dataSource.BaseCost == 0 {
		dataSource.BaseCost = 0.0
	}
	dataSource.IsActive = true

	return s.repo.CreateDataSource(dataSource)
}

func (s *DataSourceService) CreateDataSourcesBatch(dataSources []models.DataSource) error {
	slog.Info("DataSourceService: Creating data sources batch", "count", len(dataSources))

	if len(dataSources) == 0 {
		return fmt.Errorf("no data sources provided")
	}

	// Validate all data sources
	for i, dataSource := range dataSources {
		if err := s.validateDataSource(&dataSource); err != nil {
			return fmt.Errorf("validation failed for data source at index %d: %w", i, err)
		}

		// Generate ID if not provided
		if dataSources[i].ID == uuid.Nil {
			dataSources[i].ID = uuid.New()
		}

		// Set default values
		if dataSources[i].BaseCost == 0 {
			dataSources[i].BaseCost = 0.0
		}
		dataSources[i].IsActive = true
	}

	return s.repo.CreateDataSourcesBatch(dataSources)
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

func (s *DataSourceService) GetDataSourceByID(id uuid.UUID) (*models.DataSource, error) {
	slog.Debug("DataSourceService: Getting data source by ID", "id", id)

	if id == uuid.Nil {
		return nil, fmt.Errorf("invalid data source ID")
	}

	return s.repo.GetDataSourceByID(id)
}

func (s *DataSourceService) GetAllDataSources() ([]models.DataSource, error) {
	slog.Debug("DataSourceService: Getting all data sources")
	return s.repo.GetAllDataSources()
}

func (s *DataSourceService) GetActiveDataSources() ([]models.DataSource, error) {
	slog.Debug("DataSourceService: Getting active data sources")
	return s.repo.GetActiveDataSources()
}

func (s *DataSourceService) GetDataSourcesByType(dataSourceType models.DataSourceType) ([]models.DataSource, error) {
	slog.Debug("DataSourceService: Getting data sources by type", "type", dataSourceType)
	return s.repo.GetDataSourcesByType(dataSourceType)
}

func (s *DataSourceService) GetDataSourcesByTierID(tierID uuid.UUID) ([]models.DataSource, error) {
	slog.Debug("DataSourceService: Getting data sources by tier ID", "tier_id", tierID)

	if tierID == uuid.Nil {
		return nil, fmt.Errorf("invalid tier ID")
	}

	return s.repo.GetDataSourcesByTierID(tierID)
}

func (s *DataSourceService) GetDataSourcesByParameterName(parameterName string) ([]models.DataSource, error) {
	slog.Debug("DataSourceService: Getting data sources by parameter name", "parameter_name", parameterName)

	if parameterName == "" {
		return nil, fmt.Errorf("parameter name cannot be empty")
	}

	return s.repo.GetDataSourcesByParameterName(parameterName)
}

// ============================================================================
// UPDATE OPERATIONS
// ============================================================================

func (s *DataSourceService) UpdateDataSource(id uuid.UUID, dataSource *models.DataSource) error {
	slog.Info("DataSourceService: Updating data source", "id", id)

	if id == uuid.Nil {
		return fmt.Errorf("invalid data source ID")
	}

	// Validate input
	if err := s.validateDataSource(dataSource); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Ensure the ID matches
	dataSource.ID = id

	// Check if data source exists
	exists, err := s.repo.CheckDataSourceExists(id)
	if err != nil {
		return fmt.Errorf("failed to check data source existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("data source not found")
	}

	return s.repo.UpdateDataSource(dataSource)
}

func (s *DataSourceService) UpdateDataSourceStatus(id uuid.UUID, isActive bool) error {
	slog.Info("DataSourceService: Updating data source status", "id", id, "is_active", isActive)

	if id == uuid.Nil {
		return fmt.Errorf("invalid data source ID")
	}

	return s.repo.UpdateDataSourceStatus(id, isActive)
}

func (s *DataSourceService) ActivateDataSource(id uuid.UUID) error {
	slog.Info("DataSourceService: Activating data source", "id", id)
	return s.UpdateDataSourceStatus(id, true)
}

func (s *DataSourceService) DeactivateDataSource(id uuid.UUID) error {
	slog.Info("DataSourceService: Deactivating data source", "id", id)
	return s.UpdateDataSourceStatus(id, false)
}

// ============================================================================
// DELETE OPERATIONS
// ============================================================================

func (s *DataSourceService) DeleteDataSource(id uuid.UUID) error {
	slog.Info("DataSourceService: Deleting data source", "id", id)

	if id == uuid.Nil {
		return fmt.Errorf("invalid data source ID")
	}

	// Check if data source exists
	exists, err := s.repo.CheckDataSourceExists(id)
	if err != nil {
		return fmt.Errorf("failed to check data source existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("data source not found")
	}

	return s.repo.DeleteDataSource(id)
}

// ============================================================================
// UTILITY OPERATIONS
// ============================================================================

func (s *DataSourceService) CheckDataSourceExists(id uuid.UUID) (bool, error) {
	slog.Debug("DataSourceService: Checking data source existence", "id", id)

	if id == uuid.Nil {
		return false, fmt.Errorf("invalid data source ID")
	}

	return s.repo.CheckDataSourceExists(id)
}

func (s *DataSourceService) GetDataSourceCount() (int, error) {
	slog.Debug("DataSourceService: Getting data source count")
	return s.repo.GetDataSourceCount()
}

func (s *DataSourceService) GetActiveDataSourceCount() (int, error) {
	slog.Debug("DataSourceService: Getting active data source count")
	return s.repo.GetActiveDataSourceCount()
}

func (s *DataSourceService) GetDataSourceCountByType(dataSourceType models.DataSourceType) (int, error) {
	slog.Debug("DataSourceService: Getting data source count by type", "type", dataSourceType)
	return s.repo.GetDataSourceCountByType(dataSourceType)
}

func (s *DataSourceService) GetDataSourceCountByTier(tierID uuid.UUID) (int, error) {
	slog.Debug("DataSourceService: Getting data source count by tier", "tier_id", tierID)

	if tierID == uuid.Nil {
		return 0, fmt.Errorf("invalid tier ID")
	}

	return s.repo.GetDataSourceCountByTier(tierID)
}

// ============================================================================
// VALIDATION OPERATIONS
// ============================================================================

func (s *DataSourceService) validateDataSource(dataSource *models.DataSource) error {
	if dataSource == nil {
		return fmt.Errorf("data source cannot be nil")
	}

	if dataSource.ParameterName == "" {
		return fmt.Errorf("parameter name is required")
	}

	if dataSource.DataTierID == uuid.Nil {
		return fmt.Errorf("data tier ID is required")
	}

	if dataSource.BaseCost < 0 {
		return fmt.Errorf("base cost cannot be negative")
	}

	// Validate min/max values if provided
	if dataSource.MinValue != nil && dataSource.MaxValue != nil {
		if *dataSource.MinValue > *dataSource.MaxValue {
			return fmt.Errorf("min value cannot be greater than max value")
		}
	}

	// Validate accuracy rating if provided
	if dataSource.AccuracyRating != nil {
		if *dataSource.AccuracyRating < 0 || *dataSource.AccuracyRating > 100 {
			return fmt.Errorf("accuracy rating must be between 0 and 100")
		}
	}

	return nil
}

// ============================================================================
// BUSINESS LOGIC OPERATIONS
// ============================================================================

func (s *DataSourceService) GetDataSourcesWithFilters(filters DataSourceFilters) ([]models.DataSource, error) {
	slog.Debug("DataSourceService: Getting data sources with filters")

	var dataSources []models.DataSource
	var err error

	// Apply filters based on provided criteria
	if filters.TierID != nil && *filters.TierID != uuid.Nil {
		dataSources, err = s.repo.GetDataSourcesByTierID(*filters.TierID)
	} else if filters.DataSourceType != nil {
		dataSources, err = s.repo.GetDataSourcesByType(*filters.DataSourceType)
	} else if filters.ParameterName != nil && *filters.ParameterName != "" {
		dataSources, err = s.repo.GetDataSourcesByParameterName(*filters.ParameterName)
	} else if filters.ActiveOnly {
		dataSources, err = s.repo.GetActiveDataSources()
	} else {
		dataSources, err = s.repo.GetAllDataSources()
	}

	if err != nil {
		return nil, err
	}

	// Apply additional filtering in memory
	filteredSources := make([]models.DataSource, 0)
	for _, ds := range dataSources {
		if s.matchesFilters(ds, filters) {
			filteredSources = append(filteredSources, ds)
		}
	}

	return filteredSources, nil
}

func (s *DataSourceService) matchesFilters(ds models.DataSource, filters DataSourceFilters) bool {
	// Filter by active status
	if filters.ActiveOnly && !ds.IsActive {
		return false
	}

	// Filter by cost range
	if filters.MinCost != nil && ds.BaseCost < *filters.MinCost {
		return false
	}
	if filters.MaxCost != nil && ds.BaseCost > *filters.MaxCost {
		return false
	}

	// Filter by accuracy rating
	if filters.MinAccuracy != nil && ds.AccuracyRating != nil && *ds.AccuracyRating < *filters.MinAccuracy {
		return false
	}

	return true
}

// DataSourceFilters represents filtering criteria for data sources
type DataSourceFilters struct {
	TierID         *uuid.UUID              `json:"tier_id,omitempty"`
	DataSourceType *models.DataSourceType  `json:"data_source_type,omitempty"`
	ParameterName  *string                 `json:"parameter_name,omitempty"`
	ActiveOnly     bool                    `json:"active_only"`
	MinCost        *float64                `json:"min_cost,omitempty"`
	MaxCost        *float64                `json:"max_cost,omitempty"`
	MinAccuracy    *float64                `json:"min_accuracy,omitempty"`
}
