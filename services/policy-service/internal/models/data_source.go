package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

// ============================================================================
// CORE DATA SOURCE
// ============================================================================

type DataTierCategory struct {
	ID                     uuid.UUID `json:"id" db:"id"`
	CategoryName           string    `json:"category_name" db:"category_name"`
	CategoryDescription    *string   `json:"category_description,omitempty" db:"category_description"`
	CategoryCostMultiplier float64   `json:"category_cost_multiplier" db:"category_cost_multiplier"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

type DataTier struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	DataTierCategoryID uuid.UUID `json:"data_tier_category_id" db:"data_tier_category_id"`
	TierLevel          int       `json:"tier_level" db:"tier_level"`
	TierName           string    `json:"tier_name" db:"tier_name"`
	DataTierMultiplier float64   `json:"data_tier_multiplier" db:"data_tier_multiplier"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

type DataSource struct {
	ID                  uuid.UUID      `json:"id" db:"id"`
	DataSource          DataSourceType `json:"data_source" db:"data_source"`
	ParameterName       string         `json:"parameter_name" db:"parameter_name"`
	ParameterType       ParameterType  `json:"parameter_type" db:"parameter_type"`
	Unit                *string        `json:"unit,omitempty" db:"unit"`
	DisplayNameVi       *string        `json:"display_name_vi,omitempty" db:"display_name_vi"`
	DescriptionVi       *string        `json:"description_vi,omitempty" db:"description_vi"`
	MinValue            *float64       `json:"min_value,omitempty" db:"min_value"`
	MaxValue            *float64       `json:"max_value,omitempty" db:"max_value"`
	UpdateFrequency     *string        `json:"update_frequency,omitempty" db:"update_frequency"`
	SpatialResolution   *string        `json:"spatial_resolution,omitempty" db:"spatial_resolution"`
	AccuracyRating      *float64       `json:"accuracy_rating,omitempty" db:"accuracy_rating"`
	BaseCost            float64        `json:"base_cost" db:"base_cost"`
	DataTierID          uuid.UUID      `json:"data_tier_id" db:"data_tier_id"`
	DataComplexityScore float64        `json:"data_complexity_score" db:"data_complexity_score"`
	DataProvider        *string        `json:"data_provider,omitempty" db:"data_provider"`
	APIEndpoint         *string        `json:"api_endpoint,omitempty" db:"api_endpoint"`
	IsActive            bool           `json:"is_active" db:"is_active"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// FARM MANAGEMENT
// ============================================================================

type Farm struct {
	ID                      uuid.UUID     `json:"id" db:"id"`
	OwnerID                 string        `json:"owner_id" db:"owner_id"`
	FarmName                *string       `json:"farm_name,omitempty" db:"farm_name"`
	FarmCode                *string       `json:"farm_code,omitempty" db:"farm_code"`
	Boundary                *geom.Polygon `json:"boundary,omitempty" db:"boundary"`
	CenterLocation          *geom.Point   `json:"center_location,omitempty" db:"center_location"`
	AreaSqm                 float64       `json:"area_sqm" db:"area_sqm"`
	Province                *string       `json:"province,omitempty" db:"province"`
	District                *string       `json:"district,omitempty" db:"district"`
	Commune                 *string       `json:"commune,omitempty" db:"commune"`
	Address                 *string       `json:"address,omitempty" db:"address"`
	CropType                string        `json:"crop_type" db:"crop_type"`
	PlantingDate            *int64        `json:"planting_date,omitempty" db:"planting_date"`
	ExpectedHarvestDate     *int64        `json:"expected_harvest_date,omitempty" db:"expected_harvest_date"`
	CropTypeVerified        bool          `json:"crop_type_verified" db:"crop_type_verified"`
	CropTypeVerifiedAt      *int64        `json:"crop_type_verified_at,omitempty" db:"crop_type_verified_at"`
	CropTypeVerifiedBy      *string       `json:"crop_type_verified_by,omitempty" db:"crop_type_verified_by"`
	CropTypeConfidence      *float64      `json:"crop_type_confidence,omitempty" db:"crop_type_confidence"`
	LandCertificateNumber   *string       `json:"land_certificate_number,omitempty" db:"land_certificate_number"`
	LandCertificateURL      *string       `json:"land_certificate_url,omitempty" db:"land_certificate_url"`
	LandOwnershipVerified   bool          `json:"land_ownership_verified" db:"land_ownership_verified"`
	LandOwnershipVerifiedAt *int64        `json:"land_ownership_verified_at,omitempty" db:"land_ownership_verified_at"`
	HasIrrigation           bool          `json:"has_irrigation" db:"has_irrigation"`
	IrrigationType          *string       `json:"irrigation_type,omitempty" db:"irrigation_type"`
	SoilType                *string       `json:"soil_type,omitempty" db:"soil_type"`
	Status                  FarmStatus    `json:"status" db:"status"`
	CreatedAt               time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at" db:"updated_at"`
}

type FarmPhoto struct {
	ID        uuid.UUID `json:"id" db:"id"`
	FarmID    uuid.UUID `json:"farm_id" db:"farm_id"`
	PhotoURL  string    `json:"photo_url" db:"photo_url"`
	PhotoType PhotoType `json:"photo_type" db:"photo_type"`
	TakenAt   *int64    `json:"taken_at,omitempty" db:"taken_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

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

// ============================================================================
// REGISTERED POLICY (ACTUAL POLICY INSTANCES)
// ============================================================================

type RegisteredPolicy struct {
	ID                      uuid.UUID          `json:"id" db:"id"`
	PolicyNumber            string             `json:"policy_number" db:"policy_number"`
	BasePolicyID            uuid.UUID          `json:"base_policy_id" db:"base_policy_id"`
	InsuranceProviderID     string             `json:"insurance_provider_id" db:"insurance_provider_id"`
	FarmID                  uuid.UUID          `json:"farm_id" db:"farm_id"`
	FarmerID                string             `json:"farmer_id" db:"farmer_id"`
	CoverageAmount          float64            `json:"coverage_amount" db:"coverage_amount"`
	CoverageStartDate       int64              `json:"coverage_start_date" db:"coverage_start_date"`
	CoverageEndDate         int64              `json:"coverage_end_date" db:"coverage_end_date"`
	PlantingDate            int64              `json:"planting_date" db:"planting_date"`
	AreaMultiplier          float64            `json:"area_multiplier" db:"area_multiplier"`
	TotalFarmerPremium      float64            `json:"total_farmer_premium" db:"total_farmer_premium"`
	PremiumPaidByFarmer     bool               `json:"premium_paid_by_farmer" db:"premium_paid_by_farmer"`
	PremiumPaidAt           *int64             `json:"premium_paid_at,omitempty" db:"premium_paid_at"`
	MonthlyDataCost         float64            `json:"monthly_data_cost" db:"monthly_data_cost"`
	TotalDataCost           float64            `json:"total_data_cost" db:"total_data_cost"`
	Status                  PolicyStatus       `json:"status" db:"status"`
	UnderwritingStatus      UnderwritingStatus `json:"underwriting_status" db:"underwriting_status"`
	RejectionReason         *string            `json:"rejection_reason,omitempty" db:"rejection_reason"`
	SignedPolicyDocumentURL *string            `json:"signed_policy_document_url,omitempty" db:"signed_policy_document_url"`
	CreatedAt               time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at" db:"updated_at"`
	RegisteredBy            *string            `json:"registered_by,omitempty" db:"registered_by"`
}

// ============================================================================
// CLAIMS & PAYOUTS
// ============================================================================

type Claim struct {
	ID                     uuid.UUID   `json:"id" db:"id"`
	ClaimNumber            string      `json:"claim_number" db:"claim_number"`
	RegisteredPolicyID     uuid.UUID   `json:"registered_policy_id" db:"registered_policy_id"`
	BasePolicyID           uuid.UUID   `json:"base_policy_id" db:"base_policy_id"`
	FarmID                 uuid.UUID   `json:"farm_id" db:"farm_id"`
	BasePolicyTriggerID    uuid.UUID   `json:"base_policy_trigger_id" db:"base_policy_trigger_id"`
	TriggerTimestamp       int64       `json:"trigger_timestamp" db:"trigger_timestamp"`
	ClaimAmount            float64     `json:"claim_amount" db:"claim_amount"`
	Status                 ClaimStatus `json:"status" db:"status"`
	AutoGenerated          bool        `json:"auto_generated" db:"auto_generated"`
	PartnerReviewTimestamp *int64      `json:"partner_review_timestamp,omitempty" db:"partner_review_timestamp"`
	PartnerDecision        *string     `json:"partner_decision,omitempty" db:"partner_decision"`
	PartnerNotes           *string     `json:"partner_notes,omitempty" db:"partner_notes"`
	ReviewedBy             *string     `json:"reviewed_by,omitempty" db:"reviewed_by"`
	AutoApprovalDeadline   *int64      `json:"auto_approval_deadline,omitempty" db:"auto_approval_deadline"`
	AutoApproved           bool        `json:"auto_approved" db:"auto_approved"`
	EvidenceSummary        interface{} `json:"evidence_summary,omitempty" db:"evidence_summary"` // JSONB
	CreatedAt              time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at" db:"updated_at"`
}

type Payout struct {
	ID                          uuid.UUID    `json:"id" db:"id"`
	ClaimID                     uuid.UUID    `json:"claim_id" db:"claim_id"`
	RegisteredPolicyID          uuid.UUID    `json:"registered_policy_id" db:"registered_policy_id"`
	FarmID                      uuid.UUID    `json:"farm_id" db:"farm_id"`
	FarmerID                    string       `json:"farmer_id" db:"farmer_id"`
	PayoutAmount                float64      `json:"payout_amount" db:"payout_amount"`
	Currency                    string       `json:"currency" db:"currency"`
	Status                      PayoutStatus `json:"status" db:"status"`
	InitiatedAt                 *int64       `json:"initiated_at,omitempty" db:"initiated_at"`
	CompletedAt                 *int64       `json:"completed_at,omitempty" db:"completed_at"`
	FarmerConfirmed             bool         `json:"farmer_confirmed" db:"farmer_confirmed"`
	FarmerConfirmationTimestamp *int64       `json:"farmer_confirmation_timestamp,omitempty" db:"farmer_confirmation_timestamp"`
	FarmerRating                *int         `json:"farmer_rating,omitempty" db:"farmer_rating"`
	FarmerFeedback              *string      `json:"farmer_feedback,omitempty" db:"farmer_feedback"`
	CreatedAt                   time.Time    `json:"created_at" db:"created_at"`
}

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

// ============================================================================
// BILLING & INVOICING
// ============================================================================

type PartnerInvoice struct {
	ID                     uuid.UUID     `json:"id" db:"id"`
	InsuranceProviderID    string        `json:"insurance_provider_id" db:"insurance_provider_id"`
	InvoiceMonth           int64         `json:"invoice_month" db:"invoice_month"`
	InvoiceNumber          string        `json:"invoice_number" db:"invoice_number"`
	ActivePoliciesCount    int           `json:"active_policies_count" db:"active_policies_count"`
	TotalDataComplexityFee float64       `json:"total_data_complexity_fee" db:"total_data_complexity_fee"`
	Subtotal               float64       `json:"subtotal" db:"subtotal"`
	Tax                    float64       `json:"tax" db:"tax"`
	TotalDue               float64       `json:"total_due" db:"total_due"`
	PaymentStatus          PaymentStatus `json:"payment_status" db:"payment_status"`
	DueDate                int64         `json:"due_date" db:"due_date"`
	PaidDate               *int64        `json:"paid_date,omitempty" db:"paid_date"`
	CreatedAt              time.Time     `json:"created_at" db:"created_at"`
}

type InvoiceLineItem struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	InvoiceID          uuid.UUID  `json:"invoice_id" db:"invoice_id"`
	ItemType           string     `json:"item_type" db:"item_type"`
	BasePolicyID       *uuid.UUID `json:"base_policy_id,omitempty" db:"base_policy_id"`
	RegisteredPolicyID *uuid.UUID `json:"registered_policy_id,omitempty" db:"registered_policy_id"`
	Description        *string    `json:"description,omitempty" db:"description"`
	Quantity           int        `json:"quantity" db:"quantity"`
	UnitCost           float64    `json:"unit_cost" db:"unit_cost"`
	TotalCost          float64    `json:"total_cost" db:"total_cost"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
}

// ============================================================================
// ANALYTICS & LOGGING
// ============================================================================

type TriggerEvaluationLog struct {
	ID                   uuid.UUID   `json:"id" db:"id"`
	RegisteredPolicyID   uuid.UUID   `json:"registered_policy_id" db:"registered_policy_id"`
	BasePolicyID         uuid.UUID   `json:"base_policy_id" db:"base_policy_id"`
	FarmID               uuid.UUID   `json:"farm_id" db:"farm_id"`
	BasePolicyTriggerID  uuid.UUID   `json:"base_policy_trigger_id" db:"base_policy_trigger_id"`
	EvaluationTimestamp  int64       `json:"evaluation_timestamp" db:"evaluation_timestamp"`
	EvaluationResult     bool        `json:"evaluation_result" db:"evaluation_result"`
	ConditionsEvaluated  int         `json:"conditions_evaluated" db:"conditions_evaluated"`
	ConditionsMet        int         `json:"conditions_met" db:"conditions_met"`
	ConditionDetails     interface{} `json:"condition_details,omitempty" db:"condition_details"` // JSONB
	ClaimGenerated       bool        `json:"claim_generated" db:"claim_generated"`
	ClaimID              *uuid.UUID  `json:"claim_id,omitempty" db:"claim_id"`
	EvaluationDurationMs *int        `json:"evaluation_duration_ms,omitempty" db:"evaluation_duration_ms"`
	DataSourcesQueried   *int        `json:"data_sources_queried,omitempty" db:"data_sources_queried"`
	CreatedAt            time.Time   `json:"created_at" db:"created_at"`
}
