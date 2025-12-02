package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FarmMonitoringDataRepository struct {
	db *sqlx.DB
}

func NewFarmMonitoringDataRepository(db *sqlx.DB) *FarmMonitoringDataRepository {
	return &FarmMonitoringDataRepository{db: db}
}

// ============================================================================
// CREATE OPERATIONS
// ============================================================================

// Create creates a new farm monitoring data record
func (r *FarmMonitoringDataRepository) Create(ctx context.Context, data *models.FarmMonitoringData) error {
	if data.ID == uuid.Nil {
		data.ID = uuid.New()
	}
	data.CreatedAt = time.Now()

	slog.Info("Creating farm monitoring data",
		"id", data.ID,
		"farm_id", data.FarmID,
		"parameter_name", data.ParameterName,
		"measured_value", data.MeasuredValue)

	query := `
		INSERT INTO farm_monitoring_data (
			id, farm_id, base_policy_trigger_condition_id,
			parameter_name, measured_value, unit, measurement_timestamp,
			component_data, data_quality, confidence_score,
			measurement_source, distance_from_farm_meters, cloud_cover_percentage,
			created_at
		) VALUES (
			:id, :farm_id, :base_policy_trigger_condition_id,
			:parameter_name, :measured_value, :unit, :measurement_timestamp,
			:component_data, :data_quality, :confidence_score,
			:measurement_source, :distance_from_farm_meters, :cloud_cover_percentage,
			:created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, data)
	if err != nil {
		slog.Error("Failed to create farm monitoring data",
			"id", data.ID,
			"farm_id", data.FarmID,
			"error", err)
		return fmt.Errorf("failed to create farm monitoring data: %w", err)
	}

	slog.Info("Successfully created farm monitoring data", "id", data.ID)
	return nil
}

// CreateBatch creates multiple farm monitoring data records in a transaction
func (r *FarmMonitoringDataRepository) CreateBatch(ctx context.Context, dataList []models.FarmMonitoringData) error {
	if len(dataList) == 0 {
		return nil
	}

	slog.Info("Creating farm monitoring data batch", "count", len(dataList))

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	for i := range dataList {
		if dataList[i].ID == uuid.Nil {
			dataList[i].ID = uuid.New()
		}
		dataList[i].CreatedAt = now
	}

	query := `
		INSERT INTO farm_monitoring_data (
			id, farm_id, base_policy_trigger_condition_id,
			parameter_name, measured_value, unit, measurement_timestamp,
			component_data, data_quality, confidence_score,
			measurement_source, distance_from_farm_meters, cloud_cover_percentage,
			created_at
		) VALUES (
			:id, :farm_id, :base_policy_trigger_condition_id,
			:parameter_name, :measured_value, :unit, :measurement_timestamp,
			:component_data, :data_quality, :confidence_score,
			:measurement_source, :distance_from_farm_meters, :cloud_cover_percentage,
			:created_at
		)`

	for _, data := range dataList {
		_, err := tx.NamedExecContext(ctx, query, data)
		if err != nil {
			slog.Error("Failed to insert farm monitoring data in batch",
				"id", data.ID,
				"error", err)
			return fmt.Errorf("failed to insert farm monitoring data %s: %w", data.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit batch insert", "error", err)
		return fmt.Errorf("failed to commit batch insert: %w", err)
	}

	slog.Info("Successfully created farm monitoring data batch", "count", len(dataList))
	return nil
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

// GetByID retrieves a farm monitoring data record by ID
func (r *FarmMonitoringDataRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FarmMonitoringData, error) {
	slog.Debug("Retrieving farm monitoring data by ID", "id", id)

	var data models.FarmMonitoringData
	query := `
		SELECT
			id, farm_id, base_policy_trigger_condition_id,
			parameter_name, measured_value, unit, measurement_timestamp,
			component_data, data_quality, confidence_score,
			measurement_source, distance_from_farm_meters, cloud_cover_percentage,
			created_at
		FROM farm_monitoring_data
		WHERE id = $1`

	err := r.db.GetContext(ctx, &data, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Farm monitoring data not found", "id", id)
			return nil, fmt.Errorf("farm monitoring data not found")
		}
		slog.Error("Failed to get farm monitoring data", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data", "id", id)
	return &data, nil
}

// GetByFarmID retrieves all monitoring data for a specific farm
func (r *FarmMonitoringDataRepository) GetByFarmID(ctx context.Context, farmID uuid.UUID) ([]models.FarmMonitoringData, error) {
	slog.Debug("Retrieving farm monitoring data by farm ID", "farm_id", farmID)

	var dataList []models.FarmMonitoringData
	query := `
		SELECT
			id, farm_id, base_policy_trigger_condition_id,
			parameter_name, measured_value, unit, measurement_timestamp,
			component_data, data_quality, confidence_score,
			measurement_source, distance_from_farm_meters, cloud_cover_percentage,
			created_at
		FROM farm_monitoring_data
		WHERE farm_id = $1
		ORDER BY measurement_timestamp DESC`

	err := r.db.SelectContext(ctx, &dataList, query, farmID)
	if err != nil {
		slog.Error("Failed to get farm monitoring data by farm ID",
			"farm_id", farmID,
			"error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data by farm ID: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data",
		"farm_id", farmID,
		"count", len(dataList))
	return dataList, nil
}

// GetByConditionID retrieves all monitoring data for a specific trigger condition
func (r *FarmMonitoringDataRepository) GetByConditionID(ctx context.Context, conditionID uuid.UUID) ([]models.FarmMonitoringData, error) {
	slog.Debug("Retrieving farm monitoring data by condition ID", "condition_id", conditionID)

	var dataList []models.FarmMonitoringData
	query := `
		SELECT
			id, farm_id, base_policy_trigger_condition_id,
			parameter_name, measured_value, unit, measurement_timestamp,
			component_data, data_quality, confidence_score,
			measurement_source, distance_from_farm_meters, cloud_cover_percentage,
			created_at
		FROM farm_monitoring_data
		WHERE base_policy_trigger_condition_id = $1
		ORDER BY measurement_timestamp DESC`

	err := r.db.SelectContext(ctx, &dataList, query, conditionID)
	if err != nil {
		slog.Error("Failed to get farm monitoring data by condition ID",
			"condition_id", conditionID,
			"error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data by condition ID: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data",
		"condition_id", conditionID,
		"count", len(dataList))
	return dataList, nil
}

// ============================================================================
// TIME-RANGE QUERY OPERATIONS
// ============================================================================

// GetByTimeRange retrieves monitoring data within a time range for a specific farm
func (r *FarmMonitoringDataRepository) GetByTimeRange(
	ctx context.Context,
	farmID uuid.UUID,
	startTimestamp int64,
	endTimestamp int64,
) ([]models.FarmMonitoringData, error) {
	slog.Debug("Retrieving farm monitoring data by time range",
		"farm_id", farmID,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	var dataList []models.FarmMonitoringData
	query := `
		SELECT
			id, farm_id, base_policy_trigger_condition_id,
			parameter_name, measured_value, unit, measurement_timestamp,
			component_data, data_quality, confidence_score,
			measurement_source, distance_from_farm_meters, cloud_cover_percentage,
			created_at
		FROM farm_monitoring_data
		WHERE farm_id = $1
			AND measurement_timestamp >= $2
			AND measurement_timestamp <= $3
		ORDER BY measurement_timestamp ASC`

	err := r.db.SelectContext(ctx, &dataList, query, farmID, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get farm monitoring data by time range",
			"farm_id", farmID,
			"error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data by time range: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data by time range",
		"farm_id", farmID,
		"count", len(dataList))
	return dataList, nil
}

// GetByTimeRangeAndParameter retrieves monitoring data for a specific parameter within a time range
func (r *FarmMonitoringDataRepository) GetByTimeRangeAndParameter(
	ctx context.Context,
	farmID uuid.UUID,
	parameterName string,
	startTimestamp int64,
	endTimestamp int64,
) ([]models.FarmMonitoringData, error) {
	slog.Debug("Retrieving farm monitoring data by time range and parameter",
		"farm_id", farmID,
		"parameter_name", parameterName,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	var dataList []models.FarmMonitoringData
	query := `
		SELECT
			id, farm_id, base_policy_trigger_condition_id,
			parameter_name, measured_value, unit, measurement_timestamp,
			component_data, data_quality, confidence_score,
			measurement_source, distance_from_farm_meters, cloud_cover_percentage,
			created_at
		FROM farm_monitoring_data
		WHERE farm_id = $1
			AND parameter_name = $2
			AND measurement_timestamp >= $3
			AND measurement_timestamp <= $4
		ORDER BY measurement_timestamp ASC`

	err := r.db.SelectContext(ctx, &dataList, query, farmID, parameterName, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get farm monitoring data by time range and parameter",
			"farm_id", farmID,
			"parameter_name", parameterName,
			"error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data by time range and parameter: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data by time range and parameter",
		"farm_id", farmID,
		"parameter_name", parameterName,
		"count", len(dataList))
	return dataList, nil
}

// CheckDataExistsInTimeRange checks if monitoring data exists for a farm within a time range
func (r *FarmMonitoringDataRepository) CheckDataExistsInTimeRange(
	ctx context.Context,
	farmID uuid.UUID,
	startTimestamp int64,
	endTimestamp int64,
) (bool, error) {
	slog.Debug("Checking if farm monitoring data exists in time range",
		"farm_id", farmID,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	var count int
	query := `
		SELECT COUNT(*)
		FROM farm_monitoring_data
		WHERE farm_id = $1
			AND measurement_timestamp >= $2
			AND measurement_timestamp <= $3`

	err := r.db.GetContext(ctx, &count, query, farmID, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to check data existence in time range",
			"farm_id", farmID,
			"error", err)
		return false, fmt.Errorf("failed to check data existence: %w", err)
	}

	exists := count > 0
	slog.Debug("Data existence check completed",
		"farm_id", farmID,
		"exists", exists,
		"count", count)
	return exists, nil
}

// ============================================================================
// UPDATE OPERATIONS
// ============================================================================

// Update updates a farm monitoring data record
func (r *FarmMonitoringDataRepository) Update(ctx context.Context, data *models.FarmMonitoringData) error {
	slog.Info("Updating farm monitoring data", "id", data.ID)

	query := `
		UPDATE farm_monitoring_data SET
			farm_id = :farm_id,
			base_policy_trigger_condition_id = :base_policy_trigger_condition_id,
			parameter_name = :parameter_name,
			measured_value = :measured_value,
			unit = :unit,
			measurement_timestamp = :measurement_timestamp,
			component_data = :component_data,
			data_quality = :data_quality,
			confidence_score = :confidence_score,
			measurement_source = :measurement_source,
			distance_from_farm_meters = :distance_from_farm_meters,
			cloud_cover_percentage = :cloud_cover_percentage
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, data)
	if err != nil {
		slog.Error("Failed to update farm monitoring data", "id", data.ID, "error", err)
		return fmt.Errorf("failed to update farm monitoring data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("Farm monitoring data not found for update", "id", data.ID)
		return fmt.Errorf("farm monitoring data not found")
	}

	slog.Info("Successfully updated farm monitoring data", "id", data.ID)
	return nil
}

// ============================================================================
// DELETE OPERATIONS
// ============================================================================

// Delete deletes a farm monitoring data record by ID
func (r *FarmMonitoringDataRepository) Delete(ctx context.Context, id uuid.UUID) error {
	slog.Info("Deleting farm monitoring data", "id", id)

	query := `DELETE FROM farm_monitoring_data WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		slog.Error("Failed to delete farm monitoring data", "id", id, "error", err)
		return fmt.Errorf("failed to delete farm monitoring data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("Farm monitoring data not found for deletion", "id", id)
		return fmt.Errorf("farm monitoring data not found")
	}

	slog.Info("Successfully deleted farm monitoring data", "id", id)
	return nil
}

// DeleteByFarmID deletes all monitoring data for a specific farm
func (r *FarmMonitoringDataRepository) DeleteByFarmID(ctx context.Context, farmID uuid.UUID) error {
	slog.Info("Deleting all farm monitoring data for farm", "farm_id", farmID)

	query := `DELETE FROM farm_monitoring_data WHERE farm_id = $1`

	result, err := r.db.ExecContext(ctx, query, farmID)
	if err != nil {
		slog.Error("Failed to delete farm monitoring data by farm ID",
			"farm_id", farmID,
			"error", err)
		return fmt.Errorf("failed to delete farm monitoring data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Successfully deleted farm monitoring data",
		"farm_id", farmID,
		"rows_deleted", rowsAffected)
	return nil
}

// DeleteByTimeRange deletes monitoring data within a time range for a specific farm
func (r *FarmMonitoringDataRepository) DeleteByTimeRange(
	ctx context.Context,
	farmID uuid.UUID,
	startTimestamp int64,
	endTimestamp int64,
) error {
	slog.Info("Deleting farm monitoring data by time range",
		"farm_id", farmID,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	query := `
		DELETE FROM farm_monitoring_data
		WHERE farm_id = $1
			AND measurement_timestamp >= $2
			AND measurement_timestamp <= $3`

	result, err := r.db.ExecContext(ctx, query, farmID, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to delete farm monitoring data by time range",
			"farm_id", farmID,
			"error", err)
		return fmt.Errorf("failed to delete farm monitoring data by time range: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Successfully deleted farm monitoring data by time range",
		"farm_id", farmID,
		"rows_deleted", rowsAffected)
	return nil
}

// ============================================================================
// UTILITY OPERATIONS
// ============================================================================

// GetCount returns the total count of monitoring data records
func (r *FarmMonitoringDataRepository) GetCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM farm_monitoring_data`

	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		slog.Error("Failed to get farm monitoring data count", "error", err)
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return count, nil
}

// GetCountByFarmID returns the count of monitoring data records for a specific farm
func (r *FarmMonitoringDataRepository) GetCountByFarmID(ctx context.Context, farmID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM farm_monitoring_data WHERE farm_id = $1`

	err := r.db.GetContext(ctx, &count, query, farmID)
	if err != nil {
		slog.Error("Failed to get farm monitoring data count by farm ID",
			"farm_id", farmID,
			"error", err)
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return count, nil
}

// GetLatestTimestampByFarmID returns the latest measurement timestamp for a farm
// Returns 0 if no data exists
func (r *FarmMonitoringDataRepository) GetLatestTimestampByFarmID(ctx context.Context, farmID uuid.UUID) (int64, error) {
	var timestamp sql.NullInt64
	query := `SELECT MAX(measurement_timestamp) FROM farm_monitoring_data WHERE farm_id = $1`

	err := r.db.GetContext(ctx, &timestamp, query, farmID)
	if err != nil {
		slog.Error("Failed to get latest timestamp by farm ID",
			"farm_id", farmID,
			"error", err)
		return 0, fmt.Errorf("failed to get latest timestamp: %w", err)
	}

	if !timestamp.Valid {
		return 0, nil
	}

	return timestamp.Int64, nil
}

// GetLatestTimestampByFarmIDAndParameterName returns the latest measurement timestamp for a farm and specific parameter
// Returns 0 if no data exists
func (r *FarmMonitoringDataRepository) GetLatestTimestampByFarmIDAndParameterName(ctx context.Context, farmID uuid.UUID, parameterName string) (int64, error) {
	var timestamp sql.NullInt64
	query := `SELECT MAX(measurement_timestamp) FROM farm_monitoring_data WHERE farm_id = $1 AND parameter_name = $2`

	err := r.db.GetContext(ctx, &timestamp, query, farmID, parameterName)
	if err != nil {
		slog.Error("Failed to get latest timestamp by farm ID and parameter name",
			"farm_id", farmID,
			"parameter_name", parameterName,
			"error", err)
		return 0, fmt.Errorf("failed to get latest timestamp: %w", err)
	}

	if !timestamp.Valid {
		return 0, nil
	}

	return timestamp.Int64, nil
}

// GetAllWithPolicyStatus retrieves all monitoring data with associated policy status
func (r *FarmMonitoringDataRepository) GetAllWithPolicyStatus(ctx context.Context, startTimestamp, endTimestamp *int64) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Debug("Retrieving all farm monitoring data with policy status",
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	var dataList []models.FarmMonitoringDataWithPolicyStatus
	var args []interface{}
	argIndex := 1

	query := `
		SELECT DISTINCT ON (fmd.id)
			fmd.id,
			fmd.farm_id,
			fmd.base_policy_trigger_condition_id,
			fmd.parameter_name,
			fmd.measured_value,
			fmd.unit,
			fmd.measurement_timestamp,
			fmd.component_data,
			fmd.data_quality,
			fmd.confidence_score,
			fmd.measurement_source,
			fmd.distance_from_farm_meters,
			fmd.cloud_cover_percentage,
			fmd.created_at,
			rp.id as registered_policy_id,
			rp.status as policy_status,
			rp.policy_number
		FROM farm_monitoring_data fmd
		LEFT JOIN registered_policy rp ON fmd.farm_id = rp.farm_id
			AND rp.status IN ('active', 'pending_payment', 'pending_review')
			AND (rp.coverage_start_date = 0 OR fmd.measurement_timestamp >= rp.coverage_start_date)
			AND (rp.coverage_end_date = 0 OR fmd.measurement_timestamp <= rp.coverage_end_date)
		WHERE 1=1`

	if startTimestamp != nil {
		query += fmt.Sprintf(" AND fmd.measurement_timestamp >= $%d", argIndex)
		args = append(args, *startTimestamp)
		argIndex++
	}
	if endTimestamp != nil {
		query += fmt.Sprintf(" AND fmd.measurement_timestamp <= $%d", argIndex)
		args = append(args, *endTimestamp)
	}

	query += " ORDER BY fmd.id, rp.created_at DESC NULLS LAST, fmd.measurement_timestamp DESC"

	err := r.db.SelectContext(ctx, &dataList, query, args...)
	if err != nil {
		slog.Error("Failed to get all farm monitoring data with policy status", "error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data with policy status: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data with policy status", "count", len(dataList))
	return dataList, nil
}

// GetAllWithPolicyStatusByFarmID retrieves monitoring data for a specific farm with policy status
func (r *FarmMonitoringDataRepository) GetAllWithPolicyStatusByFarmID(ctx context.Context, farmID uuid.UUID, startTimestamp, endTimestamp *int64) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Debug("Retrieving farm monitoring data with policy status by farm ID",
		"farm_id", farmID,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	var dataList []models.FarmMonitoringDataWithPolicyStatus
	var args []interface{}
	args = append(args, farmID)
	argIndex := 2

	query := `
		SELECT DISTINCT ON (fmd.id)
			fmd.id,
			fmd.farm_id,
			fmd.base_policy_trigger_condition_id,
			fmd.parameter_name,
			fmd.measured_value,
			fmd.unit,
			fmd.measurement_timestamp,
			fmd.component_data,
			fmd.data_quality,
			fmd.confidence_score,
			fmd.measurement_source,
			fmd.distance_from_farm_meters,
			fmd.cloud_cover_percentage,
			fmd.created_at,
			rp.id as registered_policy_id,
			rp.status as policy_status,
			rp.policy_number
		FROM farm_monitoring_data fmd
		LEFT JOIN registered_policy rp ON fmd.farm_id = rp.farm_id
			AND rp.status IN ('active', 'pending_payment', 'pending_review')
			AND (rp.coverage_start_date = 0 OR fmd.measurement_timestamp >= rp.coverage_start_date)
			AND (rp.coverage_end_date = 0 OR fmd.measurement_timestamp <= rp.coverage_end_date)
		WHERE fmd.farm_id = $1`

	if startTimestamp != nil {
		query += fmt.Sprintf(" AND fmd.measurement_timestamp >= $%d", argIndex)
		args = append(args, *startTimestamp)
		argIndex++
	}
	if endTimestamp != nil {
		query += fmt.Sprintf(" AND fmd.measurement_timestamp <= $%d", argIndex)
		args = append(args, *endTimestamp)
	}

	query += " ORDER BY fmd.id, rp.created_at DESC NULLS LAST, fmd.measurement_timestamp DESC"

	err := r.db.SelectContext(ctx, &dataList, query, args...)
	if err != nil {
		slog.Error("Failed to get farm monitoring data with policy status by farm ID",
			"farm_id", farmID,
			"error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data with policy status: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data with policy status",
		"farm_id", farmID,
		"count", len(dataList))
	return dataList, nil
}

// GetByFarmIDAndParameterNameWithPolicyStatus retrieves monitoring data for a farm and parameter with policy status
func (r *FarmMonitoringDataRepository) GetByFarmIDAndParameterNameWithPolicyStatus(
	ctx context.Context,
	farmID uuid.UUID,
	parameterName models.DataSourceParameterName,
	startTimestamp, endTimestamp *int64,
) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Debug("Retrieving farm monitoring data by farm ID and parameter name with policy status",
		"farm_id", farmID,
		"parameter_name", parameterName,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	var dataList []models.FarmMonitoringDataWithPolicyStatus
	var args []interface{}
	args = append(args, farmID, parameterName)
	argIndex := 3

	query := `
		SELECT DISTINCT ON (fmd.id)
			fmd.id,
			fmd.farm_id,
			fmd.base_policy_trigger_condition_id,
			fmd.parameter_name,
			fmd.measured_value,
			fmd.unit,
			fmd.measurement_timestamp,
			fmd.component_data,
			fmd.data_quality,
			fmd.confidence_score,
			fmd.measurement_source,
			fmd.distance_from_farm_meters,
			fmd.cloud_cover_percentage,
			fmd.created_at,
			rp.id as registered_policy_id,
			rp.status as policy_status,
			rp.policy_number
		FROM farm_monitoring_data fmd
		LEFT JOIN registered_policy rp ON fmd.farm_id = rp.farm_id
			AND rp.status IN ('active', 'pending_payment', 'pending_review')
			AND (rp.coverage_start_date = 0 OR fmd.measurement_timestamp >= rp.coverage_start_date)
			AND (rp.coverage_end_date = 0 OR fmd.measurement_timestamp <= rp.coverage_end_date)
		WHERE fmd.farm_id = $1 AND fmd.parameter_name = $2`

	if startTimestamp != nil {
		query += fmt.Sprintf(" AND fmd.measurement_timestamp >= $%d", argIndex)
		args = append(args, *startTimestamp)
		argIndex++
	}
	if endTimestamp != nil {
		query += fmt.Sprintf(" AND fmd.measurement_timestamp <= $%d", argIndex)
		args = append(args, *endTimestamp)
	}

	query += " ORDER BY fmd.id, rp.created_at DESC NULLS LAST, fmd.measurement_timestamp DESC"

	err := r.db.SelectContext(ctx, &dataList, query, args...)
	if err != nil {
		slog.Error("Failed to get farm monitoring data by farm ID and parameter name",
			"farm_id", farmID,
			"parameter_name", parameterName,
			"error", err)
		return nil, fmt.Errorf("failed to get farm monitoring data: %w", err)
	}

	slog.Debug("Successfully retrieved farm monitoring data",
		"farm_id", farmID,
		"parameter_name", parameterName,
		"count", len(dataList))
	return dataList, nil
}
