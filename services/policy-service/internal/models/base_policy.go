package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// BASE POLICY (TEMPLATE/PRODUCT)
// ============================================================================

type BasePolicy struct {
	ID                             uuid.UUID        `json:"id" db:"id"`
	InsuranceProviderID            string           `json:"insurance_provider_id" db:"insurance_provider_id"`
	EssentialAdditionalInformation interface{}      `json:"essential_additional_infomation" db:"essential_additional_infomation"`
	ProductName                    string           `json:"product_name" db:"product_name"`
	ProductCode                    *string          `json:"product_code,omitempty" db:"product_code"`
	ProductDescription             *string          `json:"product_description,omitempty" db:"product_description"`
	CropType                       string           `json:"crop_type" db:"crop_type"`
	CoverageCurrency               string           `json:"coverage_currency" db:"coverage_currency"`
	CoverageDurationDays           int              `json:"coverage_duration_days" db:"coverage_duration_days"`
	CoverageStartDayRule           *string          `json:"coverage_start_day_rule,omitempty" db:"coverage_start_day_rule"`
	PremiumBaseRate                float64          `json:"premium_base_rate" db:"premium_base_rate"`
	DataTierID                     uuid.UUID        `json:"data_tier_id" db:"data_tier_id"`
	DataComplexityScore            int              `json:"data_complexity_score" db:"data_complexity_score"`
	MonthlyDataCost                float64          `json:"monthly_data_cost" db:"monthly_data_cost"`
	Status                         BasePolicyStatus `json:"status" db:"status"`
	TemplateDocumentURL            *string          `json:"template_document_url,omitempty" db:"template_document_url"`
	DocumentValidationStatus       ValidationStatus `json:"document_validation_status" db:"document_validation_status"`
	DocumentValidationScore        *float64         `json:"document_validation_score,omitempty" db:"document_validation_score"`
	CreatedAt                      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt                      time.Time        `json:"updated_at" db:"updated_at"`
	CreatedBy                      *string          `json:"created_by,omitempty" db:"created_by"`
}

type BasePolicyTrigger struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	BasePolicyID     uuid.UUID       `json:"base_policy_id" db:"base_policy_id"`
	LogicalOperator  LogicalOperator `json:"logical_operator" db:"logical_operator"`
	PayoutPercentage float64         `json:"payout_percentage" db:"payout_percentage"`
	ValidFromDay     *int            `json:"valid_from_day,omitempty" db:"valid_from_day"`
	ValidToDay       *int            `json:"valid_to_day,omitempty" db:"valid_to_day"`
	GrowthStage      *string         `json:"growth_stage,omitempty" db:"growth_stage"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

type BasePolicyTriggerCondition struct {
	ID                    uuid.UUID            `json:"id" db:"id"`
	BasePolicyTriggerID   uuid.UUID            `json:"base_policy_trigger_id" db:"base_policy_trigger_id"`
	DataSourceID          uuid.UUID            `json:"data_source_id" db:"data_source_id"`
	ThresholdOperator     ThresholdOperator    `json:"threshold_operator" db:"threshold_operator"`
	ThresholdValue        float64              `json:"threshold_value" db:"threshold_value"`
	AggregationFunction   AggregationFunction  `json:"aggregation_function" db:"aggregation_function"`
	AggregationWindowDays int                  `json:"aggregation_window_days" db:"aggregation_window_days"`
	ConsecutiveRequired   bool                 `json:"consecutive_required" db:"consecutive_required"`
	BaselineWindowDays    *int                 `json:"baseline_window_days,omitempty" db:"baseline_window_days"`
	BaselineFunction      *AggregationFunction `json:"baseline_function,omitempty" db:"baseline_function"`
	ValidationWindowDays  int                  `json:"validation_window_days" db:"validation_window_days"`
	ConditionOrder        int                  `json:"condition_order" db:"condition_order"`
	CreatedAt             time.Time            `json:"created_at" db:"created_at"`
}

type BasePolicyDataUsage struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	BasePolicyID       uuid.UUID `json:"base_policy_id" db:"base_policy_id"`
	DataSourceID       uuid.UUID `json:"data_source_id" db:"data_source_id"`
	BaseCost           float64   `json:"base_cost" db:"base_cost"`
	CategoryMultiplier float64   `json:"category_multiplier" db:"category_multiplier"`
	TierMultiplier     float64   `json:"tier_multiplier" db:"tier_multiplier"`
	CalculatedCost     float64   `json:"calculated_cost" db:"calculated_cost"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

type BasePolicyDocumentValidation struct {
	ID                  uuid.UUID        `json:"id" db:"id"`
	BasePolicyID        uuid.UUID        `json:"base_policy_id" db:"base_policy_id"`
	ValidationTimestamp int64            `json:"validation_timestamp" db:"validation_timestamp"`
	ValidationStatus    ValidationStatus `json:"validation_status" db:"validation_status"`
	OverallScore        *float64         `json:"overall_score,omitempty" db:"overall_score"`
	TotalChecks         int              `json:"total_checks" db:"total_checks"`
	PassedChecks        int              `json:"passed_checks" db:"passed_checks"`
	FailedChecks        int              `json:"failed_checks" db:"failed_checks"`
	WarningCount        int              `json:"warning_count" db:"warning_count"`
	Mismatches          interface{}      `json:"mismatches,omitempty" db:"mismatches"`                     // JSONB
	Warnings            interface{}      `json:"warnings,omitempty" db:"warnings"`                         // JSONB
	Recommendations     interface{}      `json:"recommendations,omitempty" db:"recommendations"`           // JSONB
	ExtractedParameters interface{}      `json:"extracted_parameters,omitempty" db:"extracted_parameters"` // JSONB
	ValidatedBy         *string          `json:"validated_by,omitempty" db:"validated_by"`
	ValidationNotes     *string          `json:"validation_notes,omitempty" db:"validation_notes"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
}