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
	ProductName                    string           `json:"product_name" db:"product_name"`
	ProductCode                    *string          `json:"product_code,omitempty" db:"product_code"`
	ProductDescription             *string          `json:"product_description,omitempty" db:"product_description"`
	CropType                       string           `json:"crop_type" db:"crop_type"`
	CoverageCurrency               string           `json:"coverage_currency" db:"coverage_currency"`
	CoverageDurationDays           int              `json:"coverage_duration_days" db:"coverage_duration_days"`
	FixPremiumAmount               int              `json:"fix_premium_amount" db:"fix_premium_amount"`
	IsPerHectare                   bool             `json:"is_per_hectare" db:"is_per_hectare"`
	PremiumBaseRate                float64          `json:"premium_base_rate" db:"premium_base_rate"`
	FixPayoutAmount                int              `json:"fix_payout_amount" db:"fix_payout_amount"`
	IsPayoutPerHectare             bool             `json:"is_payout_per_hectare" db:"is_payout_per_hectare"`
	OverThresholdMultiplier        float64          `json:"over_threshold_multiplier" db:"over_threshold_multiplier"`
	PayoutBaseRate                 float64          `json:"payout_base_rate" db:"payout_base_rate"`
	DataComplexityScore            int              `json:"data_complexity_score" db:"data_complexity_score"`
	MonthlyDataCost                float64          `json:"monthly_data_cost" db:"monthly_data_cost"`
	Status                         BasePolicyStatus `json:"status" db:"status"`
	TemplateDocumentURL            *string          `json:"template_document_url,omitempty" db:"template_document_url"`
	DocumentValidationStatus       ValidationStatus `json:"document_validation_status" db:"document_validation_status"`
	DocumentValidationScore        *float64         `json:"document_validation_score,omitempty" db:"document_validation_score"`
	ImportantAdditionalInformation interface{}      `json:"important_additional_information,omitempty" db:"important_additional_information"`
	CreatedAt                      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt                      time.Time        `json:"updated_at" db:"updated_at"`
	CreatedBy                      *string          `json:"created_by,omitempty" db:"created_by"`
}

type BasePolicyTrigger struct {
	ID                    uuid.UUID        `json:"id" db:"id"`
	BasePolicyID          uuid.UUID        `json:"base_policy_id" db:"base_policy_id"`
	LogicalOperator       LogicalOperator  `json:"logical_operator" db:"logical_operator"`
	ValidFromDay          *int             `json:"valid_from_day,omitempty" db:"valid_from_day"`
	ValidToDay            *int             `json:"valid_to_day,omitempty" db:"valid_to_day"`
	GrowthStage           *string          `json:"growth_stage,omitempty" db:"growth_stage"`
	MonitorFrequencyValue int              `json:"monitor_frequency_value" db:"monitor_frequency_value"`
	MonitorFrequencyUnit  MonitorFrequency `json:"monitor_frequency_unit" db:"monitor_frequency_unit"`
	CreatedAt             time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at" db:"updated_at"`
}

type BasePolicyTriggerCondition struct {
	ID                    uuid.UUID            `json:"id" db:"id"`
	BasePolicyTriggerID   uuid.UUID            `json:"base_policy_trigger_id" db:"base_policy_trigger_id"`
	DataSourceID          uuid.UUID            `json:"data_source_id" db:"data_source_id"`
	ThresholdOperator     ThresholdOperator    `json:"threshold_operator" db:"threshold_operator"`
	ThresholdValue        float64              `json:"threshold_value" db:"threshold_value"`
	EarlyWarningThreshold *float64             `json:"early_warning_threshold,omitempty" db:"early_warning_threshold"`
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

