package services

import (
	"fmt"
	"policy-service/internal/models"
	"policy-service/internal/repository"

	"github.com/google/uuid"
)

type BasePolicyService struct {
	basePolicyRepo *repository.BasePolicyRepository
}

func NewBasePolicyService(basePolicyRepo *repository.BasePolicyRepository) *BasePolicyService {
	return &BasePolicyService{basePolicyRepo: basePolicyRepo}
}

func (s *BasePolicyService) CreateBasePolicy(policy *models.BasePolicy) error {
	if err := s.validateBasePolicy(policy); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if err := s.basePolicyRepo.CreateBasePolicy(policy); err != nil {
		return fmt.Errorf("failed to create base policy: %w", err)
	}

	return nil
}

func (s *BasePolicyService) CreateDataSelectionGroup(basePolicyTrigger *models.BasePolicyTrigger) error {
	if err := s.validateBasePolicyTrigger(basePolicyTrigger); err != nil {
		return fmt.Errorf("validate error: %w", err)
	}
	if err := s.basePolicyRepo.CreateBasePolicyTrigger(basePolicyTrigger); err != nil {
		return fmt.Errorf("failed to create base policy trigger: %w", err)
	}
	return nil
}

func (s *BasePolicyService) DataSelection(dataSourceIDs []uuid.UUID) error {
	// TODO: Implement data selection logic
	return fmt.Errorf("not implemented")
}

func (s *BasePolicyService) CreateBasePolicyTriggerConditionsBatch(conditions []models.BasePolicyTriggerCondition) error {
	if len(conditions) == 0 {
		return fmt.Errorf("no conditions provided")
	}

	// Validate all conditions before batch insert
	for i, condition := range conditions {
		if err := s.validateBasePolicyTriggerCondition(&condition); err != nil {
			return fmt.Errorf("validation error for condition %d: %w", i, err)
		}
	}

	if err := s.basePolicyRepo.CreateBasePolicyTriggerConditionsBatch(conditions); err != nil {
		return fmt.Errorf("failed to create base policy trigger conditions batch: %w", err)
	}

	return nil
}

func (s *BasePolicyService) GetBasePolicyCount() (int, error) {
	count, err := s.basePolicyRepo.GetBasePolicyCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get base policy count: %w", err)
	}

	return count, nil
}

func (s *BasePolicyService) GetBasePolicyCountByStatus(status models.BasePolicyStatus) (int, error) {
	if !s.isValidBasePolicyStatus(status) {
		return 0, fmt.Errorf("invalid base policy status: %s", status)
	}

	count, err := s.basePolicyRepo.GetBasePolicyCountByStatus(status)
	if err != nil {
		return 0, fmt.Errorf("failed to get base policy count by status: %w", err)
	}

	return count, nil
}

// ============================================================================
// VALIDATION HELPERS
// ============================================================================

func (s *BasePolicyService) validateBasePolicy(policy *models.BasePolicy) error {
	if policy.InsuranceProviderID == "" {
		return fmt.Errorf("insurance provider ID is required")
	}
	if policy.ProductName == "" {
		return fmt.Errorf("product name is required")
	}
	if policy.CropType == "" {
		return fmt.Errorf("crop type is required")
	}
	if policy.CoverageCurrency == "" {
		return fmt.Errorf("coverage currency is required")
	}
	if policy.CoverageDurationDays <= 0 {
		return fmt.Errorf("coverage duration must be greater than 0")
	}
	if policy.FixPremiumAmount < 0 {
		return fmt.Errorf("fix premium amount cannot be negative")
	}
	if policy.PremiumBaseRate < 0 {
		return fmt.Errorf("premium base rate cannot be negative")
	}
	if policy.FixPayoutAmount < 0 {
		return fmt.Errorf("fix payout amount cannot be negative")
	}
	if policy.PayoutBaseRate < 0 {
		return fmt.Errorf("payout base rate cannot be negative")
	}
	if policy.OverThresholdMultiplier < 0 {
		return fmt.Errorf("over threshold multiplier cannot be negative")
	}
	if !s.isValidBasePolicyStatus(policy.Status) {
		return fmt.Errorf("invalid status: %s", policy.Status)
	}
	if !s.isValidValidationStatus(policy.DocumentValidationStatus) {
		return fmt.Errorf("invalid document validation status: %s", policy.DocumentValidationStatus)
	}

	return nil
}

func (s *BasePolicyService) validateBasePolicyTrigger(triggerGr *models.BasePolicyTrigger) error {
	if !s.isValidTriggerGroupLogicalOperator(triggerGr.LogicalOperator) {
		return fmt.Errorf("invalid operator: %s", triggerGr.LogicalOperator)
	}
	if triggerGr.MonitorFrequencyValue <= 0 {
		return fmt.Errorf("monitor frequency must be greater than 0")
	}
	if !s.isValidMonitorFrequencyUnit(triggerGr.MonitorFrequencyUnit) {
		return fmt.Errorf("invalid monitor frequency unit: %s", triggerGr.MonitorFrequencyUnit)
	}
	return nil
}

func (s *BasePolicyService) isValidMonitorFrequencyUnit(unit models.MonitorFrequency) bool {
	switch unit {
	case models.MonitorFrequencyDay, models.MonitorFrequencyHour, models.MonitorFrequencyMonth, models.MonitorFrequencyWeek, models.MonitorFrequencyYear:
		return true
	default:
		return false
	}
}

func (s *BasePolicyService) isValidTriggerGroupLogicalOperator(operator models.LogicalOperator) bool {
	switch operator {
	case models.LogicalAND, models.LogicalOR:
		return true
	default:
		return false
	}
}

func (s *BasePolicyService) isValidBasePolicyStatus(status models.BasePolicyStatus) bool {
	switch status {
	case models.BasePolicyDraft, models.BasePolicyActive, models.BasePolicyArchived:
		return true
	default:
		return false
	}
}

func (s *BasePolicyService) isValidValidationStatus(status models.ValidationStatus) bool {
	switch status {
	case models.ValidationPending, models.ValidationPassed, models.ValidationFailed, models.ValidationWarning:
		return true
	default:
		return false
	}
}

func (s *BasePolicyService) isValidThresholdOperator(operator models.ThresholdOperator) bool {
	switch operator {
	case models.ThresholdLT, models.ThresholdGT, models.ThresholdLTE, models.ThresholdGTE,
		models.ThresholdEQ, models.ThresholdNE, models.ThresholdChangeGT, models.ThresholdChangeLT:
		return true
	default:
		return false
	}
}

func (s *BasePolicyService) isValidAggregationFunction(function models.AggregationFunction) bool {
	switch function {
	case models.AggregationSum, models.AggregationAvg, models.AggregationMin,
		models.AggregationMax, models.AggregationChange:
		return true
	default:
		return false
	}
}

func (s *BasePolicyService) validateBasePolicyTriggerCondition(condition *models.BasePolicyTriggerCondition) error {
	if condition.BasePolicyTriggerID == uuid.Nil {
		return fmt.Errorf("base policy trigger ID is required")
	}
	if condition.DataSourceID == uuid.Nil {
		return fmt.Errorf("data source ID is required")
	}
	if !s.isValidThresholdOperator(condition.ThresholdOperator) {
		return fmt.Errorf("invalid threshold operator: %s", condition.ThresholdOperator)
	}
	if !s.isValidAggregationFunction(condition.AggregationFunction) {
		return fmt.Errorf("invalid aggregation function: %s", condition.AggregationFunction)
	}
	if condition.AggregationWindowDays <= 0 {
		return fmt.Errorf("aggregation window days must be greater than 0")
	}
	if condition.ValidationWindowDays <= 0 {
		return fmt.Errorf("validation window days must be greater than 0")
	}
	if condition.BaseCost < 0 {
		return fmt.Errorf("base cost cannot be negative")
	}
	if condition.CategoryMultiplier <= 0 {
		return fmt.Errorf("category multiplier must be greater than 0")
	}
	if condition.TierMultiplier <= 0 {
		return fmt.Errorf("tier multiplier must be greater than 0")
	}
	if condition.CalculatedCost < 0 {
		return fmt.Errorf("calculated cost cannot be negative")
	}
	return nil
}
