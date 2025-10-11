package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// MONITORING DATA (TIME-SERIES)
// ============================================================================

type FarmMonitoringData struct {
	ID                     uuid.UUID   `json:"id" db:"id"`
	FarmID                 uuid.UUID   `json:"farm_id" db:"farm_id"`
	DataSourceID           uuid.UUID   `json:"data_source_id" db:"data_source_id"`
	ParameterName          string      `json:"parameter_name" db:"parameter_name"`
	MeasuredValue          float64     `json:"measured_value" db:"measured_value"`
	Unit                   *string     `json:"unit,omitempty" db:"unit"`
	MeasurementTimestamp   int64       `json:"measurement_timestamp" db:"measurement_timestamp"`
	DataQuality            DataQuality `json:"data_quality" db:"data_quality"`
	ConfidenceScore        *float64    `json:"confidence_score,omitempty" db:"confidence_score"`
	MeasurementSource      *string     `json:"measurement_source,omitempty" db:"measurement_source"`
	DistanceFromFarmMeters *float64    `json:"distance_from_farm_meters,omitempty" db:"distance_from_farm_meters"`
	CloudCoverPercentage   *float64    `json:"cloud_cover_percentage,omitempty" db:"cloud_cover_percentage"`
	CreatedAt              time.Time   `json:"created_at" db:"created_at"`
}