package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CreateDataTierCategoryRequest struct {
	CategoryName           string  `json:"category_name" validate:"required,min=1,max=100"`
	CategoryDescription    *string `json:"category_description,omitempty" validate:"omitempty,max=500"`
	CategoryCostMultiplier float64 `json:"category_cost_multiplier" validate:"required,min=0.01,max=100"`
}

func (r CreateDataTierCategoryRequest) Validate() error {
	if r.CategoryName == "" {
		return errors.New("category name is required")
	}
	if len(r.CategoryName) > 100 {
		return errors.New("category name must be 100 characters or less")
	}
	if r.CategoryDescription != nil && len(*r.CategoryDescription) > 500 {
		return errors.New("category description must be 500 characters or less")
	}
	if r.CategoryCostMultiplier <= 0 {
		return errors.New("category cost multiplier must be greater than 0")
	}
	if r.CategoryCostMultiplier > 100 {
		return errors.New("category cost multiplier must be 100 or less")
	}
	return nil
}

type UpdateDataTierCategoryRequest struct {
	CategoryName           *string  `json:"category_name,omitempty" validate:"omitempty,min=1,max=100"`
	CategoryDescription    *string  `json:"category_description,omitempty" validate:"omitempty,max=500"`
	CategoryCostMultiplier *float64 `json:"category_cost_multiplier,omitempty" validate:"omitempty,min=0.01,max=100"`
}

func (r UpdateDataTierCategoryRequest) Validate() error {
	if r.CategoryName != nil {
		if *r.CategoryName == "" {
			return errors.New("category name cannot be empty")
		}
		if len(*r.CategoryName) > 100 {
			return errors.New("category name must be 100 characters or less")
		}
	}
	if r.CategoryDescription != nil && len(*r.CategoryDescription) > 500 {
		return errors.New("category description must be 500 characters or less")
	}
	if r.CategoryCostMultiplier != nil {
		if *r.CategoryCostMultiplier <= 0 {
			return errors.New("category cost multiplier must be greater than 0")
		}
		if *r.CategoryCostMultiplier > 100 {
			return errors.New("category cost multiplier must be 100 or less")
		}
	}
	return nil
}

type CreateDataTierRequest struct {
	DataTierCategoryID uuid.UUID `json:"data_tier_category_id" validate:"required"`
	TierLevel          int       `json:"tier_level" validate:"required,min=1,max=100"`
	TierName           string    `json:"tier_name" validate:"required,min=1,max=100"`
	DataTierMultiplier float64   `json:"data_tier_multiplier" validate:"required,min=0.01,max=100"`
}

func (r CreateDataTierRequest) Validate() error {
	if r.DataTierCategoryID == uuid.Nil {
		return errors.New("data tier category ID is required")
	}
	if r.TierLevel < 1 {
		return errors.New("tier level must be at least 1")
	}
	if r.TierLevel > 100 {
		return errors.New("tier level must be 100 or less")
	}
	if r.TierName == "" {
		return errors.New("tier name is required")
	}
	if len(r.TierName) > 100 {
		return errors.New("tier name must be 100 characters or less")
	}
	if r.DataTierMultiplier <= 0 {
		return errors.New("data tier multiplier must be greater than 0")
	}
	if r.DataTierMultiplier > 100 {
		return errors.New("data tier multiplier must be 100 or less")
	}
	return nil
}

type UpdateDataTierRequest struct {
	DataTierCategoryID *uuid.UUID `json:"data_tier_category_id,omitempty"`
	TierLevel          *int       `json:"tier_level,omitempty" validate:"omitempty,min=1,max=100"`
	TierName           *string    `json:"tier_name,omitempty" validate:"omitempty,min=1,max=100"`
	DataTierMultiplier *float64   `json:"data_tier_multiplier,omitempty" validate:"omitempty,min=0.01,max=100"`
}

func (r UpdateDataTierRequest) Validate() error {
	if r.TierLevel != nil {
		if *r.TierLevel < 1 {
			return errors.New("tier level must be at least 1")
		}
		if *r.TierLevel > 100 {
			return errors.New("tier level must be 100 or less")
		}
	}
	if r.TierName != nil {
		if *r.TierName == "" {
			return errors.New("tier name cannot be empty")
		}
		if len(*r.TierName) > 100 {
			return errors.New("tier name must be 100 characters or less")
		}
	}
	if r.DataTierMultiplier != nil {
		if *r.DataTierMultiplier <= 0 {
			return errors.New("data tier multiplier must be greater than 0")
		}
		if *r.DataTierMultiplier > 100 {
			return errors.New("data tier multiplier must be 100 or less")
		}
	}
	return nil
}

type CompletePolicyCreationRequest struct {
	BasePolicy *BasePolicy                   `json:"base_policy"`
	Trigger    *BasePolicyTrigger            `json:"trigger"`
	Conditions []*BasePolicyTriggerCondition `json:"conditions"`
}

// CompletePolicyCreationResponse represents the successful creation result
type CompletePolicyCreationResponse struct {
	BasePolicyID    uuid.UUID   `json:"base_policy_id"`
	TriggerID       uuid.UUID   `json:"trigger_id"`
	ConditionIDs    []uuid.UUID `json:"condition_ids"`
	TotalConditions int         `json:"total_conditions"`
	TotalDataCost   float64     `json:"total_data_cost"`
	CreatedAt       time.Time   `json:"created_at"`
}

// ============================================================================
// DATA SOURCE REQUESTS
// ============================================================================

type CreateDataSourceRequest struct {
	DataSource        DataSourceType `json:"data_source" validate:"required"`
	ParameterName     string         `json:"parameter_name" validate:"required,min=1,max=100"`
	ParameterType     ParameterType  `json:"parameter_type" validate:"required"`
	Unit              *string        `json:"unit,omitempty" validate:"omitempty,max=50"`
	DisplayNameVi     *string        `json:"display_name_vi,omitempty" validate:"omitempty,max=200"`
	DescriptionVi     *string        `json:"description_vi,omitempty" validate:"omitempty,max=1000"`
	MinValue          *float64       `json:"min_value,omitempty"`
	MaxValue          *float64       `json:"max_value,omitempty"`
	UpdateFrequency   *string        `json:"update_frequency,omitempty" validate:"omitempty,max=100"`
	SpatialResolution *string        `json:"spatial_resolution,omitempty" validate:"omitempty,max=100"`
	AccuracyRating    *float64       `json:"accuracy_rating,omitempty" validate:"omitempty,min=0,max=100"`
	BaseCost          float64        `json:"base_cost" validate:"min=0"`
	DataTierID        uuid.UUID      `json:"data_tier_id" validate:"required"`
	DataProvider      *string        `json:"data_provider,omitempty" validate:"omitempty,max=200"`
	APIEndpoint       *string        `json:"api_endpoint,omitempty" validate:"omitempty,max=500"`
}

func (r CreateDataSourceRequest) Validate() error {
	if r.ParameterName == "" {
		return errors.New("parameter name is required")
	}
	if len(r.ParameterName) > 100 {
		return errors.New("parameter name must be 100 characters or less")
	}
	if r.DataTierID == uuid.Nil {
		return errors.New("data tier ID is required")
	}
	if r.BaseCost < 0 {
		return errors.New("base cost cannot be negative")
	}
	if r.MinValue != nil && r.MaxValue != nil && *r.MinValue > *r.MaxValue {
		return errors.New("min value cannot be greater than max value")
	}
	if r.AccuracyRating != nil && (*r.AccuracyRating < 0 || *r.AccuracyRating > 100) {
		return errors.New("accuracy rating must be between 0 and 100")
	}
	if r.Unit != nil && len(*r.Unit) > 50 {
		return errors.New("unit must be 50 characters or less")
	}
	if r.DisplayNameVi != nil && len(*r.DisplayNameVi) > 200 {
		return errors.New("display name must be 200 characters or less")
	}
	if r.DescriptionVi != nil && len(*r.DescriptionVi) > 1000 {
		return errors.New("description must be 1000 characters or less")
	}
	if r.UpdateFrequency != nil && len(*r.UpdateFrequency) > 100 {
		return errors.New("update frequency must be 100 characters or less")
	}
	if r.SpatialResolution != nil && len(*r.SpatialResolution) > 100 {
		return errors.New("spatial resolution must be 100 characters or less")
	}
	if r.DataProvider != nil && len(*r.DataProvider) > 200 {
		return errors.New("data provider must be 200 characters or less")
	}
	if r.APIEndpoint != nil && len(*r.APIEndpoint) > 500 {
		return errors.New("API endpoint must be 500 characters or less")
	}
	return nil
}

type UpdateDataSourceRequest struct {
	DataSource        *DataSourceType `json:"data_source,omitempty"`
	ParameterName     *string         `json:"parameter_name,omitempty" validate:"omitempty,min=1,max=100"`
	ParameterType     *ParameterType  `json:"parameter_type,omitempty"`
	Unit              *string         `json:"unit,omitempty" validate:"omitempty,max=50"`
	DisplayNameVi     *string         `json:"display_name_vi,omitempty" validate:"omitempty,max=200"`
	DescriptionVi     *string         `json:"description_vi,omitempty" validate:"omitempty,max=1000"`
	MinValue          *float64        `json:"min_value,omitempty"`
	MaxValue          *float64        `json:"max_value,omitempty"`
	UpdateFrequency   *string         `json:"update_frequency,omitempty" validate:"omitempty,max=100"`
	SpatialResolution *string         `json:"spatial_resolution,omitempty" validate:"omitempty,max=100"`
	AccuracyRating    *float64        `json:"accuracy_rating,omitempty" validate:"omitempty,min=0,max=100"`
	BaseCost          *float64        `json:"base_cost,omitempty" validate:"omitempty,min=0"`
	DataTierID        *uuid.UUID      `json:"data_tier_id,omitempty"`
	DataProvider      *string         `json:"data_provider,omitempty" validate:"omitempty,max=200"`
	APIEndpoint       *string         `json:"api_endpoint,omitempty" validate:"omitempty,max=500"`
	IsActive          *bool           `json:"is_active,omitempty"`
}

func (r UpdateDataSourceRequest) Validate() error {
	if r.ParameterName != nil {
		if *r.ParameterName == "" {
			return errors.New("parameter name cannot be empty")
		}
		if len(*r.ParameterName) > 100 {
			return errors.New("parameter name must be 100 characters or less")
		}
	}
	if r.BaseCost != nil && *r.BaseCost < 0 {
		return errors.New("base cost cannot be negative")
	}
	if r.MinValue != nil && r.MaxValue != nil && *r.MinValue > *r.MaxValue {
		return errors.New("min value cannot be greater than max value")
	}
	if r.AccuracyRating != nil && (*r.AccuracyRating < 0 || *r.AccuracyRating > 100) {
		return errors.New("accuracy rating must be between 0 and 100")
	}
	if r.Unit != nil && len(*r.Unit) > 50 {
		return errors.New("unit must be 50 characters or less")
	}
	if r.DisplayNameVi != nil && len(*r.DisplayNameVi) > 200 {
		return errors.New("display name must be 200 characters or less")
	}
	if r.DescriptionVi != nil && len(*r.DescriptionVi) > 1000 {
		return errors.New("description must be 1000 characters or less")
	}
	if r.UpdateFrequency != nil && len(*r.UpdateFrequency) > 100 {
		return errors.New("update frequency must be 100 characters or less")
	}
	if r.SpatialResolution != nil && len(*r.SpatialResolution) > 100 {
		return errors.New("spatial resolution must be 100 characters or less")
	}
	if r.DataProvider != nil && len(*r.DataProvider) > 200 {
		return errors.New("data provider must be 200 characters or less")
	}
	if r.APIEndpoint != nil && len(*r.APIEndpoint) > 500 {
		return errors.New("API endpoint must be 500 characters or less")
	}
	return nil
}

type CreateDataSourceBatchRequest struct {
	DataSources []CreateDataSourceRequest `json:"data_sources" validate:"required,min=1,max=100"`
}

func (r CreateDataSourceBatchRequest) Validate() error {
	if len(r.DataSources) == 0 {
		return errors.New("at least one data source is required")
	}
	if len(r.DataSources) > 100 {
		return errors.New("cannot create more than 100 data sources at once")
	}
	for i, ds := range r.DataSources {
		if err := ds.Validate(); err != nil {
			return errors.New("validation failed for data source at index " + string(rune(i)) + ": " + err.Error())
		}
	}
	return nil
}

type DataSourceFiltersRequest struct {
	TierID         *uuid.UUID       `json:"tier_id,omitempty"`
	DataSourceType *DataSourceType  `json:"data_source_type,omitempty"`
	ParameterName  *string          `json:"parameter_name,omitempty"`
	ActiveOnly     bool             `json:"active_only"`
	MinCost        *float64         `json:"min_cost,omitempty" validate:"omitempty,min=0"`
	MaxCost        *float64         `json:"max_cost,omitempty" validate:"omitempty,min=0"`
	MinAccuracy    *float64         `json:"min_accuracy,omitempty" validate:"omitempty,min=0,max=100"`
}

func (r DataSourceFiltersRequest) Validate() error {
	if r.MinCost != nil && *r.MinCost < 0 {
		return errors.New("minimum cost cannot be negative")
	}
	if r.MaxCost != nil && *r.MaxCost < 0 {
		return errors.New("maximum cost cannot be negative")
	}
	if r.MinCost != nil && r.MaxCost != nil && *r.MinCost > *r.MaxCost {
		return errors.New("minimum cost cannot be greater than maximum cost")
	}
	if r.MinAccuracy != nil && (*r.MinAccuracy < 0 || *r.MinAccuracy > 100) {
		return errors.New("minimum accuracy must be between 0 and 100")
	}
	return nil
}
