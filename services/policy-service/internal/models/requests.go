package models

import (
	utils "agrisa_utils"
	"errors"
	"fmt"
	"net/url"
	"policy-service/internal/database/minio"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Helper functions for validation
func isValidDataSourceType(dataSource DataSourceType) bool {
	switch dataSource {
	case DataSourceWeather, DataSourceSatellite, DataSourceDerived:
		return true
	default:
		return false
	}
}

func isValidParameterType(paramType ParameterType) bool {
	switch paramType {
	case ParameterNumeric, ParameterBoolean, ParameterCategorical:
		return true
	default:
		return false
	}
}

func isValidURL(urlStr string) bool {
	if urlStr == "" {
		return true // Empty URLs are handled by omitempty validation
	}
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func trimAndValidateString(str string, fieldName string, minLen, maxLen int) error {
	trimmed := strings.TrimSpace(str)
	if len(trimmed) < minLen {
		return fmt.Errorf("%s must be at least %d characters", fieldName, minLen)
	}
	if len(trimmed) > maxLen {
		return fmt.Errorf("%s must be %d characters or less", fieldName, maxLen)
	}
	return nil
}

func isValidArchiveStatus(status string) bool {
	if status == "" {
		return true // Empty status is allowed for filtering
	}
	switch status {
	case "true", "false", "archived", "active":
		return true
	default:
		return false
	}
}

type CreateDataTierCategoryRequest struct {
	CategoryName           string  `json:"category_name" validate:"required,min=1,max=100"`
	CategoryDescription    *string `json:"category_description,omitempty" validate:"omitempty,max=500"`
	CategoryCostMultiplier float64 `json:"category_cost_multiplier" validate:"required,min=0.01,max=100"`
}

func (r CreateDataTierCategoryRequest) Validate() error {
	if err := trimAndValidateString(r.CategoryName, "category_name", 1, 100); err != nil {
		return err
	}

	if r.CategoryDescription != nil && len(strings.TrimSpace(*r.CategoryDescription)) > 500 {
		return errors.New("category_description must be 500 characters or less")
	}

	if r.CategoryCostMultiplier <= 0 {
		return errors.New("category_cost_multiplier must be greater than 0")
	}

	if r.CategoryCostMultiplier > 100 {
		return errors.New("category_cost_multiplier must be 100 or less")
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
		if err := trimAndValidateString(*r.CategoryName, "category_name", 1, 100); err != nil {
			return err
		}
	}

	if r.CategoryDescription != nil && len(strings.TrimSpace(*r.CategoryDescription)) > 500 {
		return errors.New("category_description must be 500 characters or less")
	}

	if r.CategoryCostMultiplier != nil {
		if *r.CategoryCostMultiplier <= 0 {
			return errors.New("category_cost_multiplier must be greater than 0")
		}
		if *r.CategoryCostMultiplier > 100 {
			return errors.New("category_cost_multiplier must be 100 or less")
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
		return errors.New("data_tier_category_id is required")
	}

	if r.TierLevel < 1 {
		return errors.New("tier_level must be at least 1")
	}

	if r.TierLevel > 100 {
		return errors.New("tier_level must be 100 or less")
	}

	if err := trimAndValidateString(r.TierName, "tier_name", 1, 100); err != nil {
		return err
	}

	if r.DataTierMultiplier <= 0 {
		return errors.New("data_tier_multiplier must be greater than 0")
	}

	if r.DataTierMultiplier > 100 {
		return errors.New("data_tier_multiplier must be 100 or less")
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
			return errors.New("tier_level must be at least 1")
		}
		if *r.TierLevel > 100 {
			return errors.New("tier_level must be 100 or less")
		}
	}

	if r.TierName != nil {
		if err := trimAndValidateString(*r.TierName, "tier_name", 1, 100); err != nil {
			return err
		}
	}

	if r.DataTierMultiplier != nil {
		if *r.DataTierMultiplier <= 0 {
			return errors.New("data_tier_multiplier must be greater than 0")
		}
		if *r.DataTierMultiplier > 100 {
			return errors.New("data_tier_multiplier must be 100 or less")
		}
	}

	return nil
}

type CompletePolicyCreationRequest struct {
	BasePolicy     *BasePolicy                   `json:"base_policy" validate:"required"`
	Trigger        *BasePolicyTrigger            `json:"trigger" validate:"required"`
	Conditions     []*BasePolicyTriggerCondition `json:"conditions" validate:"required,min=1,max=50"`
	IsArchive      bool                          `json:"is_archive"`
	PolicyDocument PolicyDocument                `json:"policy_document" validate:"required"`
}

type PolicyDocument struct {
	Name string `json:"name" validate:"required"`
	Data string `json:"data" validate:"required"` // This is base64
}

func (r CompletePolicyCreationRequest) Validate() error {
	if r.BasePolicy == nil {
		return errors.New("base_policy is required")
	}
	if r.Trigger == nil {
		return errors.New("trigger is required")
	}
	if len(r.Conditions) == 0 {
		return errors.New("at least one condition is required")
	}
	if len(r.Conditions) > 50 {
		return errors.New("cannot have more than 50 conditions")
	}
	if r.PolicyDocument.Name == "" {
		return errors.New("policy document name is required")
	}
	if r.PolicyDocument.Data == "" {
		return errors.New("document data is required")
	}
	return nil
}

// CompletePolicyCreationResponse represents the successful creation result
type CompletePolicyCreationResponse struct {
	BasePolicyID    uuid.UUID   `json:"base_policy_id"`
	TriggerID       uuid.UUID   `json:"trigger_id"`
	ConditionIDs    []uuid.UUID `json:"condition_ids"`
	TotalConditions int         `json:"total_conditions"`
	TotalDataCost   float64     `json:"total_data_cost"`
	FilePath        string      `json:"-"`
	CreatedAt       time.Time   `json:"created_at"`
}

// CompletePolicyData represents a complete policy with all related entities
type CompletePolicyData struct {
	BasePolicy  *BasePolicy                     `json:"base_policy"`
	Trigger     *BasePolicyTrigger              `json:"trigger,omitempty"`
	Conditions  []*BasePolicyTriggerCondition   `json:"conditions,omitempty"`
	Validations []*BasePolicyDocumentValidation `json:"validations,omitempty"`
}

// ValidatePolicyRequest represents the request for manual policy validation
type ValidatePolicyRequest struct {
	BasePolicyID     uuid.UUID        `json:"base_policy_id" validate:"required"`
	ValidationStatus ValidationStatus `json:"validation_status" validate:"required"`
	ValidatedBy      string           `json:"validated_by" validate:"required"`

	// Validation metrics (user-controlled)
	TotalChecks  int `json:"total_checks" validate:"min=0"`
	PassedChecks int `json:"passed_checks" validate:"min=0"`
	FailedChecks int `json:"failed_checks" validate:"min=0"`
	WarningCount int `json:"warning_count" validate:"min=0"`

	// Optional detailed validation data (JSONB)
	Mismatches          map[string]any `json:"mismatches,omitempty"`
	Warnings            map[string]any `json:"warnings,omitempty"`
	Recommendations     map[string]any `json:"recommendations,omitempty"`
	ExtractedParameters map[string]any `json:"extracted_parameters,omitempty"`

	// Optional metadata
	ValidationNotes *string `json:"validation_notes,omitempty"`
}

func (r ValidatePolicyRequest) Validate() error {
	if r.BasePolicyID == uuid.Nil {
		return errors.New("base_policy_id is required")
	}

	// Validate validation status
	validStatuses := []ValidationStatus{
		ValidationPending,
		ValidationPassed,
		ValidationFailed,
		ValidationWarning,
	}
	isValidStatus := slices.Contains(validStatuses, r.ValidationStatus)
	if !isValidStatus {
		return fmt.Errorf("validation_status must be one of: %s, %s, %s, %s",
			ValidationPending, ValidationPassed, ValidationFailed, ValidationWarning)
	}

	// Validate check counts consistency
	if r.PassedChecks+r.FailedChecks > r.TotalChecks {
		return errors.New("passed_checks + failed_checks cannot exceed total_checks")
	}

	if r.TotalChecks < 0 || r.PassedChecks < 0 || r.FailedChecks < 0 || r.WarningCount < 0 {
		return errors.New("all check counts (total_checks, passed_checks, failed_checks, warning_count) must be non-negative")
	}

	// Validate optional validation notes length
	if r.ValidationNotes != nil && len(strings.TrimSpace(*r.ValidationNotes)) > 1000 {
		return errors.New("validation_notes must be 1000 characters or less")
	}

	return nil
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
	BaseCost          int64          `json:"base_cost" validate:"min=0"`
	DataTierID        uuid.UUID      `json:"data_tier_id" validate:"required"`
	DataProvider      *string        `json:"data_provider,omitempty" validate:"omitempty,max=200"`
	APIEndpoint       *string        `json:"api_endpoint,omitempty" validate:"omitempty,max=500"`
}

func (r CreateDataSourceRequest) Validate() error {
	// Validate enum values
	if !isValidDataSourceType(r.DataSource) {
		return fmt.Errorf("invalid data_source: must be one of %s, %s, %s",
			DataSourceWeather, DataSourceSatellite, DataSourceDerived)
	}

	if !isValidParameterType(r.ParameterType) {
		return fmt.Errorf("invalid parameter_type: must be one of %s, %s, %s",
			ParameterNumeric, ParameterBoolean, ParameterCategorical)
	}

	// Validate required fields with trimming
	if err := trimAndValidateString(r.ParameterName, "parameter_name", 1, 100); err != nil {
		return err
	}

	if r.DataTierID == uuid.Nil {
		return errors.New("data_tier_id is required")
	}

	if r.BaseCost < 0 {
		return errors.New("base_cost cannot be negative")
	}

	// Validate value ranges
	if r.MinValue != nil && r.MaxValue != nil && *r.MinValue > *r.MaxValue {
		return errors.New("min_value cannot be greater than max_value")
	}

	if r.AccuracyRating != nil && (*r.AccuracyRating < 0 || *r.AccuracyRating > 1) {
		return errors.New("accuracy_rating must be between 0 and 100")
	}

	// Validate optional string fields
	if r.Unit != nil && len(strings.TrimSpace(*r.Unit)) > 50 {
		return errors.New("unit must be 50 characters or less")
	}

	if r.DisplayNameVi != nil && len(strings.TrimSpace(*r.DisplayNameVi)) > 200 {
		return errors.New("display_name_vi must be 200 characters or less")
	}

	if r.DescriptionVi != nil && len(strings.TrimSpace(*r.DescriptionVi)) > 1000 {
		return errors.New("description_vi must be 1000 characters or less")
	}

	if r.UpdateFrequency != nil && len(strings.TrimSpace(*r.UpdateFrequency)) > 100 {
		return errors.New("update_frequency must be 100 characters or less")
	}

	if r.SpatialResolution != nil && len(strings.TrimSpace(*r.SpatialResolution)) > 100 {
		return errors.New("spatial_resolution must be 100 characters or less")
	}

	if r.DataProvider != nil && len(strings.TrimSpace(*r.DataProvider)) > 200 {
		return errors.New("data_provider must be 200 characters or less")
	}

	// Validate URL format for API endpoint
	if r.APIEndpoint != nil {
		endpoint := strings.TrimSpace(*r.APIEndpoint)
		if len(endpoint) > 500 {
			return errors.New("api_endpoint must be 500 characters or less")
		}
		if endpoint != "" && !isValidURL(endpoint) {
			return errors.New("api_endpoint must be a valid URL")
		}
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
	BaseCost          *int64          `json:"base_cost,omitempty" validate:"omitempty,min=0"`
	DataTierID        *uuid.UUID      `json:"data_tier_id,omitempty"`
	DataProvider      *string         `json:"data_provider,omitempty" validate:"omitempty,max=200"`
	APIEndpoint       *string         `json:"api_endpoint,omitempty" validate:"omitempty,max=500"`
	IsActive          *bool           `json:"is_active,omitempty"`
}

func (r UpdateDataSourceRequest) Validate() error {
	// Validate enum values if provided
	if r.DataSource != nil && !isValidDataSourceType(*r.DataSource) {
		return fmt.Errorf("invalid data_source: must be one of %s, %s, %s",
			DataSourceWeather, DataSourceSatellite, DataSourceDerived)
	}

	if r.ParameterType != nil && !isValidParameterType(*r.ParameterType) {
		return fmt.Errorf("invalid parameter_type: must be one of %s, %s, %s",
			ParameterNumeric, ParameterBoolean, ParameterCategorical)
	}

	// Validate parameter name if provided
	if r.ParameterName != nil {
		if err := trimAndValidateString(*r.ParameterName, "parameter_name", 1, 100); err != nil {
			return err
		}
	}

	if r.BaseCost != nil && *r.BaseCost < 0 {
		return errors.New("base_cost cannot be negative")
	}

	if r.MinValue != nil && r.MaxValue != nil && *r.MinValue > *r.MaxValue {
		return errors.New("min_value cannot be greater than max_value")
	}

	if r.AccuracyRating != nil && (*r.AccuracyRating < 0 || *r.AccuracyRating > 100) {
		return errors.New("accuracy_rating must be between 0 and 100")
	}

	// Validate optional string fields with trimming
	if r.Unit != nil && len(strings.TrimSpace(*r.Unit)) > 50 {
		return errors.New("unit must be 50 characters or less")
	}

	if r.DisplayNameVi != nil && len(strings.TrimSpace(*r.DisplayNameVi)) > 200 {
		return errors.New("display_name_vi must be 200 characters or less")
	}

	if r.DescriptionVi != nil && len(strings.TrimSpace(*r.DescriptionVi)) > 1000 {
		return errors.New("description_vi must be 1000 characters or less")
	}

	if r.UpdateFrequency != nil && len(strings.TrimSpace(*r.UpdateFrequency)) > 100 {
		return errors.New("update_frequency must be 100 characters or less")
	}

	if r.SpatialResolution != nil && len(strings.TrimSpace(*r.SpatialResolution)) > 100 {
		return errors.New("spatial_resolution must be 100 characters or less")
	}

	if r.DataProvider != nil && len(strings.TrimSpace(*r.DataProvider)) > 200 {
		return errors.New("data_provider must be 200 characters or less")
	}

	// Validate URL format for API endpoint
	if r.APIEndpoint != nil {
		endpoint := strings.TrimSpace(*r.APIEndpoint)
		if len(endpoint) > 500 {
			return errors.New("api_endpoint must be 500 characters or less")
		}
		if endpoint != "" && !isValidURL(endpoint) {
			return errors.New("api_endpoint must be a valid URL")
		}
	}

	return nil
}

type CreateDataSourceBatchRequest struct {
	DataSources []CreateDataSourceRequest `json:"data_sources" validate:"required,min=1,max=100"`
}

func (r CreateDataSourceBatchRequest) Validate() error {
	if len(r.DataSources) == 0 {
		return errors.New("data_sources array is required and must contain at least one data source")
	}
	if len(r.DataSources) > 100 {
		return errors.New("cannot create more than 100 data sources at once")
	}
	for i, ds := range r.DataSources {
		if err := ds.Validate(); err != nil {
			return fmt.Errorf("validation failed for data source at index %d: %s", i, err.Error())
		}
	}
	return nil
}

type DataSourceFiltersRequest struct {
	TierID         *uuid.UUID      `json:"tier_id,omitempty"`
	DataSourceType *DataSourceType `json:"data_source_type,omitempty"`
	ParameterName  *string         `json:"parameter_name,omitempty"`
	ActiveOnly     bool            `json:"active_only"`
	MinCost        *int64          `json:"min_cost,omitempty" validate:"omitempty,min=0"`
	MaxCost        *int64          `json:"max_cost,omitempty" validate:"omitempty,min=0"`
	MinAccuracy    *float64        `json:"min_accuracy,omitempty" validate:"omitempty,min=0,max=100"`
}

func (r DataSourceFiltersRequest) Validate() error {
	// Validate data source type if provided
	if r.DataSourceType != nil && !isValidDataSourceType(*r.DataSourceType) {
		return fmt.Errorf("invalid data_source_type: must be one of %s, %s, %s",
			DataSourceWeather, DataSourceSatellite, DataSourceDerived)
	}

	// Validate parameter name if provided
	if r.ParameterName != nil {
		paramName := strings.TrimSpace(*r.ParameterName)
		if len(paramName) > 100 {
			return errors.New("parameter_name must be 100 characters or less")
		}
	}

	// Validate cost filters
	if r.MinCost != nil && *r.MinCost < 0 {
		return errors.New("min_cost cannot be negative")
	}
	if r.MaxCost != nil && *r.MaxCost < 0 {
		return errors.New("max_cost cannot be negative")
	}
	if r.MinCost != nil && r.MaxCost != nil && *r.MinCost > *r.MaxCost {
		return errors.New("min_cost cannot be greater than max_cost")
	}

	// Validate accuracy filter
	if r.MinAccuracy != nil && (*r.MinAccuracy < 0 || *r.MinAccuracy > 100) {
		return errors.New("min_accuracy must be between 0 and 100")
	}

	return nil
}

// ============================================================================
// COMMIT POLICIES REQUESTS
// ============================================================================

// CommitPoliciesRequest represents the request for committing policies from Redis to database
type CommitPoliciesRequest struct {
	// Filtering criteria for policies to commit
	ProviderID    string `json:"provider_id,omitempty"`
	BasePolicyID  string `json:"base_policy_id,omitempty"`
	ArchiveStatus string `json:"archive_status,omitempty"`

	// Operational options
	DeleteFromRedis bool `json:"delete_from_redis"`    // Clean up after commit
	ValidateOnly    bool `json:"validate_only"`        // Dry run mode
	BatchSize       int  `json:"batch_size,omitempty"` // Control batch processing (default: 10)
}

func (r CommitPoliciesRequest) Validate() error {
	// At least one search parameter is required
	if strings.TrimSpace(r.ProviderID) == "" && strings.TrimSpace(r.BasePolicyID) == "" && strings.TrimSpace(r.ArchiveStatus) == "" {
		return errors.New("at least one search parameter (provider_id, base_policy_id, or archive_status) is required")
	}

	// Validate archive status if provided
	if !isValidArchiveStatus(strings.TrimSpace(r.ArchiveStatus)) {
		return errors.New("archive_status must be one of: 'true', 'false', 'archived', 'active'")
	}

	// Validate provider ID format if provided
	if r.ProviderID != "" {
		providerID := strings.TrimSpace(r.ProviderID)
		if len(providerID) > 100 {
			return errors.New("provider_id must be 100 characters or less")
		}
	}

	// Validate base policy ID format if provided
	if r.BasePolicyID != "" {
		basePolicyID := strings.TrimSpace(r.BasePolicyID)
		if len(basePolicyID) > 100 {
			return errors.New("base_policy_id must be 100 characters or less")
		}
		// Try to parse as UUID if it looks like one
		if len(basePolicyID) == 36 {
			if _, err := uuid.Parse(basePolicyID); err != nil {
				return errors.New("base_policy_id must be a valid UUID format")
			}
		}
	}

	// Validate batch size
	if r.BatchSize < 0 {
		return errors.New("batch_size cannot be negative")
	}
	if r.BatchSize > 100 {
		return errors.New("batch_size cannot exceed 100")
	}

	return nil
}

// CommitPoliciesResponse represents the response after committing policies
type CommitPoliciesResponse struct {
	CommittedPolicies  []CommittedPolicyInfo `json:"committed_policies"`
	TotalPoliciesFound int                   `json:"total_policies_found"`
	TotalCommitted     int                   `json:"total_committed"`
	TotalFailed        int                   `json:"total_failed"`
	FailedPolicies     []FailedPolicyInfo    `json:"failed_policies,omitempty"`
	ProcessingDuration time.Duration         `json:"processing_duration"`
	OperationTimestamp time.Time             `json:"operation_timestamp"`
}

// CommittedPolicyInfo represents information about a successfully committed policy
type CommittedPolicyInfo struct {
	BasePolicyID   uuid.UUID `json:"base_policy_id"`
	TriggerID      uuid.UUID `json:"trigger_id"`
	ConditionCount int       `json:"condition_count"`
}

// FailedPolicyInfo represents information about a failed policy commit
type FailedPolicyInfo struct {
	BasePolicyID uuid.UUID `json:"base_policy_id"`
	ErrorMessage string    `json:"error_message"`
	FailureStage string    `json:"failure_stage"` // "discovery", "validation", "commit", "cleanup"
}

// ============================================================================
// COMPLETE POLICY DETAIL RESPONSE MODELS
// ============================================================================

// CompletePolicyDetailResponse - Main response wrapper for complete policy details
type CompletePolicyDetailResponse struct {
	BasePolicy BasePolicy              `json:"base_policy"`
	Triggers   []TriggerWithConditions `json:"triggers"`
	Document   *PolicyDocumentInfo     `json:"document,omitempty"`
	Metadata   PolicyDetailMetadata    `json:"metadata"`
}

// TriggerWithConditions - Trigger with nested conditions
type TriggerWithConditions struct {
	ID                   uuid.UUID                    `json:"id"`
	BasePolicyID         uuid.UUID                    `json:"base_policy_id"`
	LogicalOperator      LogicalOperator              `json:"logical_operator"`
	GrowthStage          *string                      `json:"growth_stage,omitempty"`
	MonitorInterval      int                          `json:"monitor_interval"`
	MonitorFrequencyUnit MonitorFrequency             `json:"monitor_frequency_unit"`
	BlackoutPeriods      utils.JSONMap                `json:"blackout_periods,omitempty"`
	CreatedAt            time.Time                    `json:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at"`
	Conditions           []BasePolicyTriggerCondition `json:"conditions"`
}

// PolicyDocumentInfo - Document metadata and access info from MinIO
type PolicyDocumentInfo struct {
	HasDocument     bool       `json:"has_document"`
	DocumentURL     *string    `json:"document_url,omitempty"`
	PresignedURL    *string    `json:"presigned_url,omitempty"`
	PresignedExpiry *time.Time `json:"presigned_url_expiry,omitempty"`
	BucketName      string     `json:"bucket_name,omitempty"`
	ObjectName      string     `json:"object_name,omitempty"`
	ContentType     string     `json:"content_type,omitempty"`
	FileSizeBytes   int64      `json:"file_size_bytes,omitempty"`
	Error           *string    `json:"error,omitempty"`
}

// PolicyDetailMetadata - Summary statistics for policy details
type PolicyDetailMetadata struct {
	TotalTriggers   int       `json:"total_triggers"`
	TotalConditions int       `json:"total_conditions"`
	TotalDataCost   float64   `json:"total_data_cost"`
	DataSourceCount int       `json:"data_source_count"`
	RetrievedAt     time.Time `json:"retrieved_at"`
}

// PolicyDetailFilterRequest - Query parameters for filtering policy details
type PolicyDetailFilterRequest struct {
	ID             *uuid.UUID       `query:"id"`
	ProviderID     string           `query:"provider_id"`
	CropType       string           `query:"crop_type"`
	Status         BasePolicyStatus `query:"status"`
	IncludePDF     bool             `query:"include_pdf"`
	PDFExpiryHours int              `query:"pdf_expiry_hours"`
}

// Validate validates the filter request
func (r *PolicyDetailFilterRequest) Validate() error {
	// At least one filter required
	if r.ID == nil && r.ProviderID == "" && r.CropType == "" && r.Status == "" {
		return fmt.Errorf("at least one filter parameter is required (id, provider_id, crop_type, or status)")
	}

	// Set defaults
	if r.PDFExpiryHours <= 0 {
		r.PDFExpiryHours = 24 // Default 24 hours
	}

	// Validate PDF expiry hours range (1-168 hours = 1 week max)
	if r.PDFExpiryHours > 168 {
		return fmt.Errorf("pdf_expiry_hours must be between 1 and 168 hours")
	}

	return nil
}

type RegisterAPolicyAPIRequest struct {
	RegisteredPolicy RegisteredPolicy `json:"registered_policy" validate:"required"`
	Farm             Farm             `json:"farm"`
	PolicyDocument   PolicyDocument   `json:"policy_document"`
}

type RegisterAPolicyRequest struct {
	RegisteredPolicy RegisteredPolicy
	Farm             Farm
	FarmID           string
	IsNewFarm        bool
}

type RegisterAPolicyResponse struct{}

type VerifyNationalIDRequest struct {
	NationalID string `json:"national_id"`
}

type VerifyNationalIDResponse struct {
	Success bool `json:"success"`
	Data    struct {
		IsValid bool `json:"is_valid"`
	} `json:"data"`
	Meta struct {
		Timestamp string `json:"timestamp"`
	} `json:"meta"`
}

// VerifyNationalIDErrorResponse represents the error response body from API A
type VerifyNationalIDErrorResponse struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type VerifyLandCertificateRequest struct {
	OwnerNationalID       string
	Token                 string
	LandCertificatePhotos []minio.FileUpload
}
