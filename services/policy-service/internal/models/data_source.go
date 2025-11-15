package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// CORE DATA SOURCE
// ============================================================================

type DataSource struct {
	ID                uuid.UUID               `json:"id" db:"id"`
	DataSource        DataSourceType          `json:"data_source" db:"data_source"`
	ParameterName     DataSourceParameterName `json:"parameter_name" db:"parameter_name"`
	ParameterType     ParameterType           `json:"parameter_type" db:"parameter_type"`
	Unit              *string                 `json:"unit,omitempty" db:"unit"`
	SupportComponent  bool                    `json:"support_component" db:"support_component"`
	DisplayNameVi     *string                 `json:"display_name_vi,omitempty" db:"display_name_vi"`
	DescriptionVi     *string                 `json:"description_vi,omitempty" db:"description_vi"`
	MinValue          *float64                `json:"min_value,omitempty" db:"min_value"`
	MaxValue          *float64                `json:"max_value,omitempty" db:"max_value"`
	UpdateFrequency   *string                 `json:"update_frequency,omitempty" db:"update_frequency"`
	SpatialResolution *string                 `json:"spatial_resolution,omitempty" db:"spatial_resolution"`
	AccuracyRating    *float64                `json:"accuracy_rating,omitempty" db:"accuracy_rating"`
	BaseCost          int64                   `json:"base_cost" db:"base_cost"`
	DataTierID        uuid.UUID               `json:"data_tier_id" db:"data_tier_id"`
	DataProvider      *string                 `json:"data_provider,omitempty" db:"data_provider"`
	APIEndpoint       *string                 `json:"api_endpoint,omitempty" db:"api_endpoint"`
	IsActive          bool                    `json:"is_active" db:"is_active"`
	CreatedAt         time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at" db:"updated_at"`
}
