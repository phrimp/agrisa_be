package models

import (
	utils "agrisa_utils"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MONITORING DATA (TIME-SERIES)
// ============================================================================

type FarmMonitoringData struct {
	ID                     uuid.UUID               `json:"id" db:"id"`
	FarmID                 uuid.UUID               `json:"farm_id" db:"farm_id"`
	DataSourceID           uuid.UUID               `json:"data_source_id" db:"data_source_id"`
	ParameterName          DataSourceParameterName `json:"parameter_name" db:"parameter_name"`
	MeasuredValue          float64                 `json:"measured_value" db:"measured_value"`
	Unit                   *string                 `json:"unit,omitempty" db:"unit"`
	MeasurementTimestamp   int64                   `json:"measurement_timestamp" db:"measurement_timestamp"`
	ComponentData          utils.JSONMap           `json:"component_data" db:"component_data"`
	DataQuality            DataQuality             `json:"data_quality" db:"data_quality"`
	ConfidenceScore        *float64                `json:"confidence_score,omitempty" db:"confidence_score"`
	MeasurementSource      *string                 `json:"measurement_source,omitempty" db:"measurement_source"`
	DistanceFromFarmMeters *float64                `json:"distance_from_farm_meters,omitempty" db:"distance_from_farm_meters"`
	CloudCoverPercentage   *float64                `json:"cloud_cover_percentage,omitempty" db:"cloud_cover_percentage"`
	CreatedAt              time.Time               `json:"created_at" db:"created_at"`
}

// FarmMonitoringDataWithPolicyStatus extends FarmMonitoringData with policy status info
type FarmMonitoringDataWithPolicyStatus struct {
	ID                     uuid.UUID               `json:"id" db:"id"`
	FarmID                 uuid.UUID               `json:"farm_id" db:"farm_id"`
	DataSourceID           uuid.UUID               `json:"data_source_id" db:"data_source_id"`
	ParameterName          DataSourceParameterName `json:"parameter_name" db:"parameter_name"`
	MeasuredValue          float64                 `json:"measured_value" db:"measured_value"`
	Unit                   *string                 `json:"unit,omitempty" db:"unit"`
	MeasurementTimestamp   int64                   `json:"measurement_timestamp" db:"measurement_timestamp"`
	ComponentData          utils.JSONMap           `json:"component_data" db:"component_data"`
	DataQuality            DataQuality             `json:"data_quality" db:"data_quality"`
	ConfidenceScore        *float64                `json:"confidence_score,omitempty" db:"confidence_score"`
	MeasurementSource      *string                 `json:"measurement_source,omitempty" db:"measurement_source"`
	DistanceFromFarmMeters *float64                `json:"distance_from_farm_meters,omitempty" db:"distance_from_farm_meters"`
	CloudCoverPercentage   *float64                `json:"cloud_cover_percentage,omitempty" db:"cloud_cover_percentage"`
	CreatedAt              time.Time               `json:"created_at" db:"created_at"`

	// Policy information
	RegisteredPolicyID *uuid.UUID    `json:"registered_policy_id,omitempty" db:"registered_policy_id"`
	PolicyStatus       *PolicyStatus `json:"policy_status,omitempty" db:"policy_status"`
	PolicyNumber       *string       `json:"policy_number,omitempty" db:"policy_number"`
}
