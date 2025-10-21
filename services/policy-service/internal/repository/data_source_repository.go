package repository

import (
	"database/sql"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DataSourceRepository struct {
	db *sqlx.DB
}

func NewDataSourceRepository(db *sqlx.DB) *DataSourceRepository {
	return &DataSourceRepository{db: db}
}

// ============================================================================
// CREATE OPERATIONS
// ============================================================================

func (r *DataSourceRepository) CreateDataSource(dataSource *models.DataSource) error {
	slog.Info("Creating data source",
		"data_source_id", dataSource.ID,
		"parameter_name", dataSource.ParameterName,
		"data_source_type", dataSource.DataSource,
		"data_tier_id", dataSource.DataTierID)
	start := time.Now()

	dataSource.CreatedAt = time.Now()
	dataSource.UpdatedAt = time.Now()

	query := `
		INSERT INTO data_source (
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		) VALUES (
			:id, :data_source, :parameter_name, :parameter_type, :unit,
			:display_name_vi, :description_vi, :min_value, :max_value,
			:update_frequency, :spatial_resolution, :accuracy_rating, :base_cost,
			:data_tier_id, :data_provider, :api_endpoint, :is_active,
			:created_at, :updated_at
		)`

	_, err := r.db.NamedExec(query, dataSource)
	if err != nil {
		slog.Error("Failed to create data source",
			"data_source_id", dataSource.ID,
			"parameter_name", dataSource.ParameterName,
			"error", err)
		return fmt.Errorf("failed to create data source: %w", err)
	}

	slog.Info("Successfully created data source",
		"data_source_id", dataSource.ID,
		"parameter_name", dataSource.ParameterName,
		"duration", time.Since(start))
	return nil
}

func (r *DataSourceRepository) CreateDataSourcesBatch(dataSources []models.DataSource) error {
	if len(dataSources) == 0 {
		return nil
	}

	// Start transaction for batch operation
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare data sources with timestamps
	now := time.Now()
	for i := range dataSources {
		dataSources[i].CreatedAt = now
		dataSources[i].UpdatedAt = now
	}

	query := `
		INSERT INTO data_source (
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		) VALUES (
			:id, :data_source, :parameter_name, :parameter_type, :unit,
			:display_name_vi, :description_vi, :min_value, :max_value,
			:update_frequency, :spatial_resolution, :accuracy_rating, :base_cost,
			:data_tier_id, :data_provider, :api_endpoint, :is_active,
			:created_at, :updated_at
		)`

	// Execute batch insert
	for _, dataSource := range dataSources {
		_, err := tx.NamedExec(query, dataSource)
		if err != nil {
			return fmt.Errorf("failed to insert data source %s: %w", dataSource.ID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch insert: %w", err)
	}

	return nil
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

func (r *DataSourceRepository) GetDataSourceByID(id uuid.UUID) (*models.DataSource, error) {
	slog.Debug("Retrieving data source by ID", "data_source_id", id)
	start := time.Now()

	var dataSource models.DataSource
	query := `
		SELECT 
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		FROM data_source
		WHERE id = $1`

	err := r.db.Get(&dataSource, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Data source not found", "data_source_id", id)
			return nil, fmt.Errorf("data source not found")
		}
		slog.Error("Failed to get data source",
			"data_source_id", id,
			"error", err)
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}

	slog.Debug("Successfully retrieved data source",
		"data_source_id", id,
		"parameter_name", dataSource.ParameterName,
		"data_source_type", dataSource.DataSource,
		"duration", time.Since(start))
	return &dataSource, nil
}

func (r *DataSourceRepository) GetAllDataSources() ([]models.DataSource, error) {
	var dataSources []models.DataSource
	query := `
		SELECT 
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		FROM data_source
		ORDER BY parameter_name`

	err := r.db.Select(&dataSources, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all data sources: %w", err)
	}

	return dataSources, nil
}

func (r *DataSourceRepository) GetActiveDataSources() ([]models.DataSource, error) {
	var dataSources []models.DataSource
	query := `
		SELECT 
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		FROM data_source
		WHERE is_active = true
		ORDER BY parameter_name`

	err := r.db.Select(&dataSources, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active data sources: %w", err)
	}

	return dataSources, nil
}

func (r *DataSourceRepository) GetDataSourcesByType(dataSourceType models.DataSourceType) ([]models.DataSource, error) {
	var dataSources []models.DataSource
	query := `
		SELECT 
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		FROM data_source
		WHERE data_source = $1
		ORDER BY parameter_name`

	err := r.db.Select(&dataSources, query, dataSourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources by type: %w", err)
	}

	return dataSources, nil
}

func (r *DataSourceRepository) GetDataSourcesByTierID(tierID uuid.UUID) ([]models.DataSource, error) {
	var dataSources []models.DataSource
	query := `
		SELECT 
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		FROM data_source
		WHERE data_tier_id = $1
		ORDER BY parameter_name`

	err := r.db.Select(&dataSources, query, tierID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources by tier ID: %w", err)
	}

	return dataSources, nil
}

func (r *DataSourceRepository) GetDataSourcesByParameterName(parameterName string) ([]models.DataSource, error) {
	var dataSources []models.DataSource
	query := `
		SELECT 
			id, data_source, parameter_name, parameter_type, unit,
			display_name_vi, description_vi, min_value, max_value,
			update_frequency, spatial_resolution, accuracy_rating, base_cost,
			data_tier_id, data_provider, api_endpoint, is_active,
			created_at, updated_at
		FROM data_source
		WHERE parameter_name ILIKE $1
		ORDER BY parameter_name`

	err := r.db.Select(&dataSources, query, "%"+parameterName+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources by parameter name: %w", err)
	}

	return dataSources, nil
}

// ============================================================================
// UPDATE OPERATIONS
// ============================================================================

func (r *DataSourceRepository) UpdateDataSource(dataSource *models.DataSource) error {
	dataSource.UpdatedAt = time.Now()

	query := `
		UPDATE data_source SET
			data_source = :data_source,
			parameter_name = :parameter_name,
			parameter_type = :parameter_type,
			unit = :unit,
			display_name_vi = :display_name_vi,
			description_vi = :description_vi,
			min_value = :min_value,
			max_value = :max_value,
			update_frequency = :update_frequency,
			spatial_resolution = :spatial_resolution,
			accuracy_rating = :accuracy_rating,
			base_cost = :base_cost,
			data_tier_id = :data_tier_id,
			data_provider = :data_provider,
			api_endpoint = :api_endpoint,
			is_active = :is_active,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExec(query, dataSource)
	if err != nil {
		return fmt.Errorf("failed to update data source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("data source not found")
	}

	return nil
}

func (r *DataSourceRepository) UpdateDataSourceStatus(id uuid.UUID, isActive bool) error {
	query := `
		UPDATE data_source SET
			is_active = $2,
			updated_at = $3
		WHERE id = $1`

	result, err := r.db.Exec(query, id, isActive, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update data source status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("data source not found")
	}

	return nil
}

// ============================================================================
// DELETE OPERATIONS
// ============================================================================

func (r *DataSourceRepository) DeleteDataSource(id uuid.UUID) error {
	query := `DELETE FROM data_source WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("data source not found")
	}

	return nil
}

// ============================================================================
// UTILITY OPERATIONS
// ============================================================================

func (r *DataSourceRepository) CheckDataSourceExists(id uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM data_source WHERE id = $1`

	err := r.db.Get(&count, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to check data source existence: %w", err)
	}

	return count > 0, nil
}

func (r *DataSourceRepository) GetDataSourceCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM data_source`

	err := r.db.Get(&count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get data source count: %w", err)
	}

	return count, nil
}

func (r *DataSourceRepository) GetActiveDataSourceCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM data_source WHERE is_active = true`

	err := r.db.Get(&count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get active data source count: %w", err)
	}

	return count, nil
}

func (r *DataSourceRepository) GetDataSourceCountByType(dataSourceType models.DataSourceType) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM data_source WHERE data_source = $1`

	err := r.db.Get(&count, query, dataSourceType)
	if err != nil {
		return 0, fmt.Errorf("failed to get data source count by type: %w", err)
	}

	return count, nil
}

func (r *DataSourceRepository) GetDataSourceCountByTier(tierID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM data_source WHERE data_tier_id = $1`

	err := r.db.Get(&count, query, tierID)
	if err != nil {
		return 0, fmt.Errorf("failed to get data source count by tier: %w", err)
	}

	return count, nil
}
