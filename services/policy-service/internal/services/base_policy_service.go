package services

import (
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

type BasePolicyService struct {
	basePolicyRepo *repository.BasePolicyRepository
	dataSourceRepo *repository.DataSourceRepository
	dataTierRepo   *repository.DataTierRepository
}

func NewBasePolicyService(basePolicyRepo *repository.BasePolicyRepository, dataSourceRepo *repository.DataSourceRepository, dataTierRepo *repository.DataTierRepository) *BasePolicyService {
	return &BasePolicyService{
		basePolicyRepo: basePolicyRepo,
		dataSourceRepo: dataSourceRepo,
		dataTierRepo:   dataTierRepo,
	}
}

func (s *BasePolicyService) CreateBasePolicy(policy *models.BasePolicy) error {
	slog.Info("Creating base policy",
		"policy_id", policy.ID,
		"provider_id", policy.InsuranceProviderID,
		"product_name", policy.ProductName,
		"crop_type", policy.CropType)
	start := time.Now()

	if err := s.validateBasePolicy(policy); err != nil {
		slog.Error("Base policy validation failed",
			"policy_id", policy.ID,
			"error", err)
		return fmt.Errorf("validation error: %w", err)
	}

	if err := s.basePolicyRepo.CreateBasePolicy(policy); err != nil {
		slog.Error("Failed to create base policy in repository",
			"policy_id", policy.ID,
			"error", err)
		return fmt.Errorf("failed to create base policy: %w", err)
	}

	slog.Info("Successfully created base policy",
		"policy_id", policy.ID,
		"provider_id", policy.InsuranceProviderID,
		"duration", time.Since(start))
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

func (s *BasePolicyService) DataSelection(selectedTriggerConditions []*models.BasePolicyTriggerCondition) error {
	slog.Info("Processing data selection",
		"condition_count", len(selectedTriggerConditions))
	start := time.Now()

	for i, selectedTriggerCondition := range selectedTriggerConditions {
		slog.Debug("Validating trigger condition",
			"index", i+1,
			"condition_id", selectedTriggerCondition.ID,
			"data_source_id", selectedTriggerCondition.DataSourceID)
		if err := s.validateBasePolicyTriggerCondition(selectedTriggerCondition); err != nil {
			slog.Error("Failed to validate trigger condition",
				"condition_id", selectedTriggerCondition.ID,
				"index", i+1,
				"error", err)
			return fmt.Errorf("failed to validate selected Trigger Condition: %w", err)
		}
		err := s.validateDataSource(selectedTriggerCondition)
		if err != nil {
			slog.Error("Data source validation failed",
				"condition_id", selectedTriggerCondition.ID,
				"data_source_id", selectedTriggerCondition.DataSourceID,
				"error", err)
			return fmt.Errorf("validate data source failed: %s --- err: %w", selectedTriggerCondition.DataSourceID, err)
		}
	}
	if err := s.basePolicyRepo.CreateBasePolicyTriggerConditionsBatch(selectedTriggerConditions); err != nil {
		slog.Error("Failed to create batch trigger conditions",
			"condition_count", len(selectedTriggerConditions),
			"error", err)
		return fmt.Errorf("failed to create batch trigger condition: %w", err)
	}

	slog.Info("Successfully completed data selection",
		"condition_count", len(selectedTriggerConditions),
		"duration", time.Since(start))
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

func (s *BasePolicyService) validateDataSource(condition *models.BasePolicyTriggerCondition) error {
	slog.Debug("Validating data source",
		"condition_id", condition.ID,
		"data_source_id", condition.DataSourceID)
	start := time.Now()

	dataSource, err := s.dataSourceRepo.GetDataSourceByID(condition.DataSourceID)
	if err != nil {
		slog.Error("Data source retrieval failed",
			"data_source_id", condition.DataSourceID,
			"error", err)
		return fmt.Errorf("data source does not exist: %w", err)
	}
	if condition.BaseCost != dataSource.BaseCost {
		slog.Error("Data source base cost mismatch",
			"condition_id", condition.ID,
			"expected_cost", dataSource.BaseCost,
			"provided_cost", condition.BaseCost)
		return fmt.Errorf("data base cost mistmatch")
	}
	dataTier, err := s.dataTierRepo.GetDataTierByID(dataSource.DataTierID)
	if err != nil {
		return fmt.Errorf("data tier retrive error: %w", err)
	}
	if condition.TierMultiplier != dataTier.DataTierMultiplier {
		return fmt.Errorf("data tier multiplier mismatch")
	}
	dataCategory, err := s.dataTierRepo.GetDataTierCategoryByID(dataTier.DataTierCategoryID)
	if err != nil {
		return fmt.Errorf("data tier category retrive error: %w", err)
	}
	if condition.CategoryMultiplier != dataCategory.CategoryCostMultiplier {
		return fmt.Errorf("data tier category multiplier mismatch")
	}
	totalCost := dataSource.BaseCost * dataTier.DataTierMultiplier * dataCategory.CategoryCostMultiplier
	if condition.CalculatedCost != totalCost {
		slog.Error("Total cost calculation mismatch",
			"condition_id", condition.ID,
			"expected_cost", totalCost,
			"provided_cost", condition.CalculatedCost,
			"base_cost", dataSource.BaseCost,
			"tier_multiplier", dataTier.DataTierMultiplier,
			"category_multiplier", dataCategory.CategoryCostMultiplier)
		return fmt.Errorf("total cost mismatch")
	}

	slog.Debug("Data source validation successful",
		"condition_id", condition.ID,
		"data_source_id", condition.DataSourceID,
		"total_cost", totalCost,
		"duration", time.Since(start))
	return nil
}

// ============================================================================
// COMPLETE POLICY CREATION
// ============================================================================

func (s *BasePolicyService) CreateCompletePolicy(request *models.CompletePolicyCreationRequest) (*models.CompletePolicyCreationResponse, error) {
	slog.Info("Creating complete policy",
		"provider_id", request.BasePolicy.InsuranceProviderID,
		"product_name", request.BasePolicy.ProductName,
		"condition_count", len(request.Conditions))
	start := time.Now()

	// Generate IDs and establish relationships
	basePolicyID := uuid.New()
	triggerID := uuid.New()

	request.BasePolicy.ID = basePolicyID
	request.Trigger.ID = triggerID
	request.Trigger.BasePolicyID = basePolicyID

	conditionIDs := make([]uuid.UUID, len(request.Conditions))
	for i := range request.Conditions {
		conditionIDs[i] = uuid.New()
		request.Conditions[i].ID = conditionIDs[i]
		request.Conditions[i].BasePolicyTriggerID = triggerID
	}

	// Validate all entities
	if err := s.validateBasePolicy(request.BasePolicy); err != nil {
		slog.Error("Base policy validation failed",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("base policy validation: %w", err)
	}
	if err := s.validateBasePolicyTrigger(request.Trigger); err != nil {
		slog.Error("Trigger validation failed",
			"trigger_id", triggerID,
			"error", err)
		return nil, fmt.Errorf("trigger validation: %w", err)
	}
	for i, condition := range request.Conditions {
		if err := s.validateBasePolicyTriggerCondition(condition); err != nil {
			slog.Error("Condition validation failed",
				"condition_id", condition.ID,
				"condition_index", i+1,
				"error", err)
			return nil, fmt.Errorf("condition %d validation: %w", i+1, err)
		}
		if err := s.validateDataSource(condition); err != nil {
			slog.Error("Condition data source validation failed",
				"condition_id", condition.ID,
				"condition_index", i+1,
				"error", err)
			return nil, fmt.Errorf("condition %d data source validation: %w", i+1, err)
		}
	}

	// Begin transaction
	tx, err := s.basePolicyRepo.BeginTransaction()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			slog.Error("Rolling back transaction", "base_policy_id", basePolicyID, "error", err)
			tx.Rollback()
		}
	}()

	// Create entities in transaction
	slog.Debug("Creating base policy in transaction", "base_policy_id", basePolicyID)
	if err = s.basePolicyRepo.CreateBasePolicyTx(tx, request.BasePolicy); err != nil {
		slog.Error("Failed to create base policy in transaction",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("failed to create base policy: %w", err)
	}

	slog.Debug("Creating trigger in transaction", "trigger_id", triggerID)
	if err = s.basePolicyRepo.CreateBasePolicyTriggerTx(tx, request.Trigger); err != nil {
		slog.Error("Failed to create trigger in transaction",
			"trigger_id", triggerID,
			"error", err)
		return nil, fmt.Errorf("failed to create trigger: %w", err)
	}

	slog.Debug("Creating conditions batch in transaction", "condition_count", len(request.Conditions))
	if err = s.basePolicyRepo.CreateBasePolicyTriggerConditionsBatchTx(tx, request.Conditions); err != nil {
		slog.Error("Failed to create conditions in transaction",
			"condition_count", len(request.Conditions),
			"error", err)
		return nil, fmt.Errorf("failed to create conditions: %w", err)
	}

	// Calculate total cost
	slog.Debug("Calculating total cost", "base_policy_id", basePolicyID)
	totalCost, err := s.basePolicyRepo.CalculateTotalBasePolicyDataCostTx(tx, basePolicyID)
	if err != nil {
		slog.Error("Failed to calculate total cost",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("failed to calculate total cost: %w", err)
	}

	// Commit transaction
	slog.Debug("Committing transaction", "base_policy_id", basePolicyID)
	if err = tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Successfully created complete policy",
		"base_policy_id", basePolicyID,
		"trigger_id", triggerID,
		"total_conditions", len(request.Conditions),
		"total_cost", totalCost,
		"duration", time.Since(start))

	return &models.CompletePolicyCreationResponse{
		BasePolicyID:    basePolicyID,
		TriggerID:       triggerID,
		ConditionIDs:    conditionIDs,
		TotalConditions: len(request.Conditions),
		TotalDataCost:   totalCost,
		CreatedAt:       time.Now(),
	}, nil
}
