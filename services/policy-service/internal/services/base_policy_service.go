package services

import (
	utils "agrisa_utils"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"policy-service/internal/ai/gemini"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	minioSDK "github.com/minio/minio-go/v7"
)

type BasePolicyService struct {
	basePolicyRepo *repository.BasePolicyRepository
	dataSourceRepo *repository.DataSourceRepository
	dataTierRepo   *repository.DataTierRepository
	minioClient    *minio.MinioClient
	geminiClient   *gemini.GeminiClient
}

func NewBasePolicyService(basePolicyRepo *repository.BasePolicyRepository, dataSourceRepo *repository.DataSourceRepository, dataTierRepo *repository.DataTierRepository, minioClient *minio.MinioClient, geminiClient *gemini.GeminiClient) *BasePolicyService {
	return &BasePolicyService{
		basePolicyRepo: basePolicyRepo,
		dataSourceRepo: dataSourceRepo,
		dataTierRepo:   dataTierRepo,
		minioClient:    minioClient,
		geminiClient:   geminiClient,
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
		slog.Info("Validating trigger condition",
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
	if triggerGr.MonitorInterval <= 0 {
		return fmt.Errorf("monitor interval must be greater than 0")
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

// validateCompletePolicyForCommit validates a complete policy before database commit
func (s *BasePolicyService) validateCompletePolicyForCommit(policy *models.CompletePolicyData) error {
	if policy == nil {
		return fmt.Errorf("policy data is nil")
	}

	if policy.BasePolicy == nil {
		return fmt.Errorf("base policy is nil")
	}

	// Validate base policy
	if err := s.validateBasePolicy(policy.BasePolicy); err != nil {
		return fmt.Errorf("base policy validation failed: %w", err)
	}

	// Validate trigger if present
	if policy.Trigger != nil {
		if err := s.validateBasePolicyTrigger(policy.Trigger); err != nil {
			return fmt.Errorf("trigger validation failed: %w", err)
		}

		// Ensure trigger is linked to base policy
		if policy.Trigger.BasePolicyID != policy.BasePolicy.ID {
			return fmt.Errorf("trigger is not linked to base policy")
		}
	}

	// Validate conditions if present
	if policy.Conditions != nil {
		for i, condition := range policy.Conditions {
			if err := s.validateBasePolicyTriggerCondition(condition); err != nil {
				return fmt.Errorf("condition %d validation failed: %w", i+1, err)
			}

			// Ensure condition is linked to trigger
			if policy.Trigger != nil && condition.BasePolicyTriggerID != policy.Trigger.ID {
				return fmt.Errorf("condition %d is not linked to trigger", i+1)
			}
		}
	}

	return nil
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
	slog.Info("Validating data source",
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

	slog.Info("Data source validation successful",
		"condition_id", condition.ID,
		"data_source_id", condition.DataSourceID,
		"total_cost", totalCost,
		"duration", time.Since(start))
	return nil
}

// ============================================================================
// BUSINESS PROCESS
// ============================================================================

func (s *BasePolicyService) CreateCompletePolicy(ctx context.Context, request *models.CompletePolicyCreationRequest, expiration time.Duration) (*models.CompletePolicyCreationResponse, error) {
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
	// Add default value for entities
	request.BasePolicy.Status = models.BasePolicyDraft
	request.BasePolicy.DocumentValidationStatus = models.ValidationPending

	// Begin Redis transaction
	slog.Info("Starting Redis transaction for complete policy creation",
		"base_policy_id", basePolicyID,
		"trigger_id", triggerID,
		"provider_id", request.BasePolicy.InsuranceProviderID)

	tx := s.basePolicyRepo.BeginRedisTransaction()
	shouldCommit := false

	defer func() {
		if shouldCommit {
			slog.Info("Executing Redis transaction commit",
				"base_policy_id", basePolicyID,
				"provider_id", request.BasePolicy.InsuranceProviderID)
			_, err := tx.Exec(ctx)
			if err != nil {
				slog.Error("Redis transaction commit failed",
					"base_policy_id", basePolicyID,
					"provider_id", request.BasePolicy.InsuranceProviderID,
					"error", err)
			} else {
				slog.Info("Redis transaction committed successfully",
					"base_policy_id", basePolicyID,
					"provider_id", request.BasePolicy.InsuranceProviderID)
			}
		} else {
			slog.Info("Discarding Redis transaction due to error",
				"base_policy_id", basePolicyID,
				"provider_id", request.BasePolicy.InsuranceProviderID)
			tx.Discard()
		}
	}()

	if request.PolicyDocument.Data != "" && request.PolicyDocument.Name != "" {
		//// upload policy document to Minio
		//files := minio.FileUploadRequest{
		//	minio.FileUpload{
		//		FieldName: "template_document_url",
		//		FileName:  request.PolicyDocument.Name,
		//		Data:      request.PolicyDocument.Data,
		//	},
		//}

		// allowedExts := []string{}

		//processedInfo, err := s.minioClient.FileProcessing(files, ctx, allowedExts, 50)
		//if err != nil {
		//	slog.Error("File processing failed",
		//		"base_policy_id", basePolicyID,
		//		"error", err)
		//	return nil, fmt.Errorf("file processing failed: %w", err)
		//}
		//slog.Info("file upload information", "info", processedInfo)

		templatePath := request.PolicyDocument.Name + "-" + basePolicyID.String()
		request.BasePolicy.TemplateDocumentURL = &templatePath
	}

	// Serialize and store BasePolicy
	slog.Info("Serializing base policy",
		"base_policy_id", basePolicyID,
		"product_name", request.BasePolicy.ProductName,
		"crop_type", request.BasePolicy.CropType)

	basePolicyByte, err := utils.SerializeModel(request.BasePolicy)
	if err != nil {
		slog.Error("Base policy serialization failed",
			"base_policy_id", basePolicyID,
			"provider_id", request.BasePolicy.InsuranceProviderID,
			"error", err)
		return nil, fmt.Errorf("base policy serialization failed: %w", err)
	}

	basePolicyKey := fmt.Sprintf("%s--%s--BasePolicy--archive:%v", request.BasePolicy.InsuranceProviderID, basePolicyID, request.IsArchive)
	slog.Info("Storing base policy in Redis transaction",
		"base_policy_id", basePolicyID,
		"key", basePolicyKey,
		"data_size_bytes", len(basePolicyByte),
		"expiration", expiration)

	if err := s.basePolicyRepo.CreateTempBasePolicyModelsWTransaction(ctx, basePolicyByte, basePolicyKey, tx, expiration); err != nil {
		slog.Error("Base policy storage in transaction failed",
			"base_policy_id", basePolicyID,
			"key", basePolicyKey,
			"error", err)
		return nil, fmt.Errorf("base policy creation failed: %w", err)
	}

	// Serialize and store BasePolicyTrigger
	slog.Info("Serializing base policy trigger",
		"trigger_id", triggerID,
		"base_policy_id", basePolicyID,
		"logical_operator", request.Trigger.LogicalOperator,
		"monitor_frequency", request.Trigger.MonitorInterval)

	basePolicyTriggerByte, err := utils.SerializeModel(request.Trigger)
	if err != nil {
		slog.Error("Base policy trigger serialization failed",
			"trigger_id", triggerID,
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("base policy trigger serialization failed: %w", err)
	}

	triggerKey := fmt.Sprintf("%s--%s--BasePolicyTrigger--%s--archive:%v", request.BasePolicy.InsuranceProviderID, triggerID, basePolicyID, request.IsArchive)
	slog.Info("Storing base policy trigger in Redis transaction",
		"trigger_id", triggerID,
		"key", triggerKey,
		"data_size_bytes", len(basePolicyTriggerByte),
		"expiration", expiration)

	if err := s.basePolicyRepo.CreateTempBasePolicyModelsWTransaction(ctx, basePolicyTriggerByte, triggerKey, tx, expiration); err != nil {
		slog.Error("Base policy trigger storage in transaction failed",
			"trigger_id", triggerID,
			"key", triggerKey,
			"error", err)
		return nil, fmt.Errorf("base policy trigger creation failed: %w", err)
	}

	// Serialize and store each condition in transaction
	slog.Info("Creating conditions in transaction", "condition_count", len(request.Conditions))
	for i, condition := range request.Conditions {
		conditionByte, err := utils.SerializeModel(condition)
		if err != nil {
			slog.Error("Failed to serialize condition",
				"condition_id", condition.ID,
				"condition_index", i+1,
				"error", err)
			return nil, fmt.Errorf("condition %d serialization failed: %w", i+1, err)
		}

		conditionKey := fmt.Sprintf("%s--%s--BasePolicyTriggerCondition--%d--%s--archive:%v", request.BasePolicy.InsuranceProviderID, condition.ID, i+1, basePolicyID, request.IsArchive)
		if err := s.basePolicyRepo.CreateTempBasePolicyModelsWTransaction(ctx, conditionByte, conditionKey, tx, expiration); err != nil {
			slog.Error("Failed to store condition in transaction",
				"condition_id", condition.ID,
				"condition_index", i+1,
				"error", err)
			return nil, fmt.Errorf("condition %d storage failed: %w", i+1, err)
		}

		slog.Info("Condition stored in transaction",
			"condition_id", condition.ID,
			"condition_index", i+1,
			"key", conditionKey)
	}

	// Calculate total cost
	slog.Info("Calculating total cost", "base_policy_id", basePolicyID)
	totalCost := s.CalculateBasePolicyTotalCost(request.Conditions)

	// Create and store response metadata in transaction
	response := &models.CompletePolicyCreationResponse{
		BasePolicyID:    basePolicyID,
		TriggerID:       triggerID,
		ConditionIDs:    conditionIDs,
		TotalConditions: len(request.Conditions),
		TotalDataCost:   totalCost,
		FilePath:        *request.BasePolicy.TemplateDocumentURL,
		CreatedAt:       time.Now(),
	}

	responseByte, err := utils.SerializeModel(response)
	if err != nil {
		slog.Error("Failed to serialize response metadata",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("response metadata serialization failed: %w", err)
	}

	responseKey := fmt.Sprintf("%s--%s--CompletePolicyResponse", request.BasePolicy.InsuranceProviderID, basePolicyID)
	if err := s.basePolicyRepo.CreateTempBasePolicyModelsWTransaction(ctx, responseByte, responseKey, tx, expiration); err != nil {
		slog.Error("Failed to store response metadata in transaction",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("response metadata storage failed: %w", err)
	}

	slog.Info("Response metadata stored in transaction",
		"base_policy_id", basePolicyID,
		"key", responseKey)

	// Mark transaction for commit
	shouldCommit = true

	slog.Info("Successfully created complete policy",
		"base_policy_id", basePolicyID,
		"trigger_id", triggerID,
		"total_conditions", len(request.Conditions),
		"total_cost", totalCost,
		"duration", time.Since(start))

	return response, nil
}

func (s *BasePolicyService) CalculateBasePolicyTotalCost(datas []*models.BasePolicyTriggerCondition) float64 {
	total_cost := 0.0
	for _, data := range datas {
		total_cost += data.CalculatedCost
	}
	return total_cost
}

func (s *BasePolicyService) GetAllDraftPolicyWFilter(ctx context.Context, providerID, basePolicyID, archiveStatus string) ([]*models.CompletePolicyData, error) {
	slog.Info("Getting draft policies from provider",
		"provider_id", providerID,
		"base_policy_id", basePolicyID,
		"archive_status", archiveStatus)
	start := time.Now()

	// Validate input parameters
	if providerID == "" && basePolicyID == "" && archiveStatus == "" {
		return nil, fmt.Errorf("at least one search parameter is required")
	}

	// Build flexible pattern with wildcards
	provider := providerID
	if provider == "" {
		provider = "*"
	}

	policy := basePolicyID
	if policy == "" {
		policy = "*"
	}

	archive := archiveStatus
	if archive == "" {
		archive = "*"
	}

	// Build Redis key pattern for specific policy
	policyPattern := fmt.Sprintf("%s--%s--BasePolicy--archive:%s", provider, policy, archive)
	slog.Info("Pattern DEBUG", "pattern", policyPattern)
	policyKeys, err := s.basePolicyRepo.FindKeysByPattern(ctx, policyPattern, "")
	if err != nil {
		slog.Error("Failed to find policy keys",
			"provider_id", providerID,
			"base_policy_id", basePolicyID,
			"archive_status", archiveStatus,
			"pattern", policyPattern,
			"error", err)
		return nil, fmt.Errorf("error getting policy %s from provider %s with archive status %s: %w", basePolicyID, providerID, archiveStatus, err)
	}
	slog.Info("Key founds", "keys", policyKeys)

	// Check if policy was found
	if len(policyKeys) == 0 {
		slog.Info("No policy found with given parameters",
			"provider_id", providerID,
			"base_policy_id", basePolicyID,
			"archive_status", archiveStatus)
		return []*models.CompletePolicyData{}, nil
	}

	var completePolicies []*models.CompletePolicyData

	for _, key := range policyKeys {
		// Get base policy
		basePolicyByte, err := s.basePolicyRepo.GetTempBasePolicyModels(ctx, key)
		if err != nil {
			slog.Info("Failed to get base policy data", "key", key, "error", err)
			continue
		}

		var basePolicy models.BasePolicy
		if err := utils.DeserializeModel(basePolicyByte, &basePolicy); err != nil {
			slog.Info("Failed to deserialize base policy", "key", key, "error", err)
			continue
		}

		// Filter for draft policies only
		if basePolicy.Status != models.BasePolicyDraft {
			continue
		}

		completePolicy := &models.CompletePolicyData{
			BasePolicy: &basePolicy,
		}

		// Get trigger for this policy
		triggerPattern := fmt.Sprintf("%s--*--BasePolicyTrigger--%s--archive:%s", provider, basePolicy.ID, archive)
		triggerKeys, err := s.basePolicyRepo.FindKeysByPattern(ctx, triggerPattern, "")
		if err == nil && len(triggerKeys) > 0 {
			triggerByte, err := s.basePolicyRepo.GetTempBasePolicyModels(ctx, triggerKeys[0])
			if err == nil {
				var trigger models.BasePolicyTrigger
				if err := utils.DeserializeModel(triggerByte, &trigger); err == nil {
					completePolicy.Trigger = &trigger
				}
			}
		}

		// Get conditions for this policy
		conditionPattern := fmt.Sprintf("%s--*--BasePolicyTriggerCondition--*--%s--archive:%s", provider, basePolicy.ID, archive)
		conditionKeys, err := s.basePolicyRepo.FindKeysByPattern(ctx, conditionPattern, "")
		if err == nil && len(conditionKeys) > 0 {
			var conditions []*models.BasePolicyTriggerCondition
			for _, condKey := range conditionKeys {
				conditionByte, err := s.basePolicyRepo.GetTempBasePolicyModels(ctx, condKey)
				if err != nil {
					continue
				}
				var condition models.BasePolicyTriggerCondition
				if err := utils.DeserializeModel(conditionByte, &condition); err == nil {
					conditions = append(conditions, &condition)
				}
			}
			completePolicy.Conditions = conditions
		}

		// Get validations for this policy
		validations, err := s.basePolicyRepo.GetValidationsFromRedis(ctx, basePolicy.ID)
		if err != nil {
			slog.Warn("Failed to get validations from Redis",
				"base_policy_id", basePolicy.ID,
				"error", err)
			// Non-critical: continue without validations
		} else if len(validations) > 0 {
			completePolicy.Validations = validations
			slog.Info("Retrieved validations for policy",
				"base_policy_id", basePolicy.ID,
				"validation_count", len(validations))
		}

		completePolicies = append(completePolicies, completePolicy)
	}

	slog.Info("Successfully retrieved policy data",
		"provider_id", providerID,
		"base_policy_id", basePolicyID,
		"archive_status", archiveStatus,
		"policy_count", len(completePolicies),
		"duration", time.Since(start))

	return completePolicies, nil
}

// UpdateBasePolicyValidationStatus updates the document validation status of a base policy
func (s *BasePolicyService) UpdateBasePolicyValidationStatus(ctx context.Context, basePolicyID uuid.UUID, validationStatus models.ValidationStatus, validationScore *float64) error {
	slog.Info("Updating base policy document validation status",
		"base_policy_id", basePolicyID,
		"validation_status", validationStatus)
	start := time.Now()

	// Validate the validation status
	if !s.isValidValidationStatus(validationStatus) {
		slog.Error("Invalid validation status provided",
			"base_policy_id", basePolicyID,
			"validation_status", validationStatus)
		return fmt.Errorf("invalid validation status: %s", validationStatus)
	}

	// Get the existing base policy
	basePolicy, err := s.basePolicyRepo.GetBasePolicyByID(basePolicyID)
	if err != nil {
		slog.Error("Failed to retrieve base policy for validation status update",
			"base_policy_id", basePolicyID,
			"error", err)
		return fmt.Errorf("failed to get base policy: %w", err)
	}

	// Update the validation fields
	oldStatus := basePolicy.DocumentValidationStatus

	basePolicy.DocumentValidationStatus = validationStatus
	basePolicy.UpdatedAt = time.Now()

	if validationStatus == models.ValidationPassed {
		basePolicy.Status = models.BasePolicyActive
	} else {
		basePolicy.Status = models.BasePolicyArchived
	}

	// Update in database
	if err := s.basePolicyRepo.UpdateBasePolicy(basePolicy); err != nil {
		slog.Error("Failed to update base policy validation status in database",
			"base_policy_id", basePolicyID,
			"validation_status", validationStatus,
			"error", err)
		return fmt.Errorf("failed to update base policy: %w", err)
	}

	slog.Info("Successfully updated base policy document validation status",
		"base_policy_id", basePolicyID,
		"old_status", oldStatus,
		"new_status", validationStatus,
		"duration", time.Since(start))

	return nil
}

// CommitPolicies transfers temporary policy data from Redis to PostgreSQL database
func (s *BasePolicyService) CommitPolicies(ctx context.Context, request *models.CommitPoliciesRequest) (*models.CommitPoliciesResponse, error) {
	slog.Info("Starting policy commit operation",
		"provider_id", request.ProviderID,
		"base_policy_id", request.BasePolicyID,
		"archive_status", request.ArchiveStatus,
		"validate_only", request.ValidateOnly,
		"delete_from_redis", request.DeleteFromRedis,
		"batch_size", request.BatchSize)
	start := time.Now()

	// Validate request
	if err := request.Validate(); err != nil {
		slog.Error("Request validation failed", "error", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Set default batch size
	batchSize := request.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}

	response := &models.CommitPoliciesResponse{
		CommittedPolicies:  make([]models.CommittedPolicyInfo, 0),
		FailedPolicies:     make([]models.FailedPolicyInfo, 0),
		OperationTimestamp: time.Now(),
	}

	// Phase 1: Discovery - Find policies from Redis
	slog.Info("Phase 1: Discovering policies from Redis")
	completePolicies, err := s.GetAllDraftPolicyWFilter(ctx, request.ProviderID, request.BasePolicyID, request.ArchiveStatus)
	if err != nil {
		slog.Error("Failed to discover policies from Redis", "error", err)
		return nil, fmt.Errorf("failed to discover policies: %w", err)
	}

	response.TotalPoliciesFound = len(completePolicies)
	slog.Info("Policy discovery completed", "policies_found", response.TotalPoliciesFound)

	if response.TotalPoliciesFound == 0 {
		slog.Info("No policies found to commit")
		response.ProcessingDuration = time.Since(start)
		return response, nil
	}

	// Phase 2: Validation (if validate_only mode or before commit)
	slog.Info("Phase 2: Validating policies", "policy_count", len(completePolicies))
	validPolicies := make([]*models.CompletePolicyData, 0)

	for _, policy := range completePolicies {
		if err := s.validateCompletePolicyForCommit(policy); err != nil {
			slog.Error("Policy validation failed",
				"base_policy_id", policy.BasePolicy.ID,
				"error", err)
			response.FailedPolicies = append(response.FailedPolicies, models.FailedPolicyInfo{
				BasePolicyID: policy.BasePolicy.ID,
				ErrorMessage: err.Error(),
				FailureStage: "validation",
			})
			response.TotalFailed++
			continue
		}
		validPolicies = append(validPolicies, policy)
	}

	slog.Info("Policy validation completed",
		"valid_policies", len(validPolicies),
		"failed_policies", response.TotalFailed)

	// If validate_only mode, return without committing
	if request.ValidateOnly {
		slog.Info("Validation-only mode completed")
		response.ProcessingDuration = time.Since(start)
		return response, nil
	}

	// Phase 3: Database Transaction Processing
	slog.Info("Phase 3: Starting database transaction processing")

	// Process policies in batches
	for i := 0; i < len(validPolicies); i += batchSize {
		end := min(i+batchSize, len(validPolicies))

		batch := validPolicies[i:end]
		slog.Info("Processing batch",
			"batch_number", (i/batchSize)+1,
			"batch_size", len(batch),
			"start_index", i,
			"end_index", end)

		// Begin database transaction for this batch
		tx, err := s.basePolicyRepo.BeginTransaction()
		if err != nil {
			slog.Error("Failed to begin database transaction", "error", err)
			// Mark all policies in this batch as failed
			for _, policy := range batch {
				response.FailedPolicies = append(response.FailedPolicies, models.FailedPolicyInfo{
					BasePolicyID: policy.BasePolicy.ID,
					ErrorMessage: fmt.Sprintf("failed to begin transaction: %v", err),
					FailureStage: "commit",
				})
				response.TotalFailed++
			}
			continue
		}

		// Process each policy in the batch
		batchSuccess := true
		for _, policy := range batch {
			if err := s.commitSinglePolicyInTransaction(ctx, tx, policy); err != nil {
				slog.Error("Failed to commit policy in transaction",
					"base_policy_id", policy.BasePolicy.ID,
					"error", err)
				response.FailedPolicies = append(response.FailedPolicies, models.FailedPolicyInfo{
					BasePolicyID: policy.BasePolicy.ID,
					ErrorMessage: err.Error(),
					FailureStage: "commit",
				})
				response.TotalFailed++
				batchSuccess = false
				break // Exit batch on first failure
			}
		}

		// Commit or rollback transaction
		if batchSuccess {
			if err := tx.Commit(); err != nil {
				slog.Error("Failed to commit transaction", "error", err)
				// Mark all policies in this batch as failed
				for _, policy := range batch {
					response.FailedPolicies = append(response.FailedPolicies, models.FailedPolicyInfo{
						BasePolicyID: policy.BasePolicy.ID,
						ErrorMessage: fmt.Sprintf("transaction commit failed: %v", err),
						FailureStage: "commit",
					})
					response.TotalFailed++
				}
			} else {
				// Mark all policies in this batch as successfully committed
				for _, policy := range batch {
					conditionCount := 0
					if policy.Conditions != nil {
						conditionCount = len(policy.Conditions)
					}

					triggerID := uuid.Nil
					if policy.Trigger != nil {
						triggerID = policy.Trigger.ID
					}

					response.CommittedPolicies = append(response.CommittedPolicies, models.CommittedPolicyInfo{
						BasePolicyID:   policy.BasePolicy.ID,
						TriggerID:      triggerID,
						ConditionCount: conditionCount,
					})
					response.TotalCommitted++
				}

				slog.Info("Batch committed successfully",
					"batch_size", len(batch),
					"batch_number", (i/batchSize)+1)
			}
		} else {
			tx.Rollback()
			slog.Warn("Batch rolled back due to failure",
				"batch_size", len(batch),
				"batch_number", (i/batchSize)+1)
		}
	}

	// Phase 4: Cleanup (Optional Redis cleanup)
	if request.DeleteFromRedis && response.TotalCommitted > 0 {
		slog.Info("Phase 4: Cleaning up Redis data")
		if err := s.cleanupCommittedPoliciesFromRedis(ctx, response.CommittedPolicies); err != nil {
			slog.Warn("Failed to cleanup Redis data", "error", err)
			// Not a critical failure, just log and continue
		} else {
			slog.Info("Redis cleanup completed successfully",
				"cleaned_policies", response.TotalCommitted)
		}
	}

	response.ProcessingDuration = time.Since(start)

	slog.Info("Policy commit operation completed",
		"total_found", response.TotalPoliciesFound,
		"total_committed", response.TotalCommitted,
		"total_failed", response.TotalFailed,
		"duration", response.ProcessingDuration)

	return response, nil
}

// commitSinglePolicyInTransaction commits a single policy within an existing transaction
func (s *BasePolicyService) commitSinglePolicyInTransaction(ctx context.Context, tx *sqlx.Tx, policy *models.CompletePolicyData) error {
	slog.Info("Committing single policy",
		"base_policy_id", policy.BasePolicy.ID,
		"product_name", policy.BasePolicy.ProductName)

	// 1. Insert BasePolicy
	if err := s.basePolicyRepo.CreateBasePolicyTx(tx, policy.BasePolicy); err != nil {
		return fmt.Errorf("failed to insert base policy: %w", err)
	}

	// 2. Insert BasePolicyTrigger if present
	if policy.Trigger != nil {
		if err := s.basePolicyRepo.CreateBasePolicyTriggerTx(tx, policy.Trigger); err != nil {
			return fmt.Errorf("failed to insert base policy trigger: %w", err)
		}
	}

	// 3. Insert BasePolicyTriggerConditions if present
	if len(policy.Conditions) > 0 {
		if err := s.basePolicyRepo.CreateBasePolicyTriggerConditionsBatchTx(tx, policy.Conditions); err != nil {
			return fmt.Errorf("failed to insert base policy trigger conditions: %w", err)
		}
	}

	// 4. Insert BasePolicyDocumentValidations if present
	if len(policy.Validations) > 0 {
		slog.Info("Committing validations to database",
			"base_policy_id", policy.BasePolicy.ID,
			"validation_count", len(policy.Validations))

		for _, validation := range policy.Validations {
			if err := s.basePolicyRepo.CreateBasePolicyDocumentValidationTx(tx, validation); err != nil {
				return fmt.Errorf("failed to insert validation %s: %w",
					validation.ID, err)
			}
		}

		slog.Info("Successfully committed validations",
			"base_policy_id", policy.BasePolicy.ID,
			"validation_count", len(policy.Validations))
	}

	slog.Info("Policy committed successfully",
		"base_policy_id", policy.BasePolicy.ID,
		"trigger_present", policy.Trigger != nil,
		"condition_count", len(policy.Conditions),
		"validation_count", len(policy.Validations))

	return nil
}

// cleanupCommittedPoliciesFromRedis removes successfully committed policies from Redis
func (s *BasePolicyService) cleanupCommittedPoliciesFromRedis(ctx context.Context, committedPolicies []models.CommittedPolicyInfo) error {
	slog.Info("Starting Redis cleanup", "policy_count", len(committedPolicies))

	for _, policy := range committedPolicies {
		// Find and delete all Redis keys for this policy
		patterns := []string{
			fmt.Sprintf("*--%s--BasePolicy--*", policy.BasePolicyID),
			fmt.Sprintf("*--%s--BasePolicyTrigger--*", policy.TriggerID),
			fmt.Sprintf("*--*--BasePolicyTriggerCondition--*--%s--*", policy.BasePolicyID),
			fmt.Sprintf("*--%s--CompletePolicyResponse", policy.BasePolicyID),
			fmt.Sprintf("%s--BasePolicyDocumentValidation--*", policy.BasePolicyID),
		}

		for _, pattern := range patterns {
			keys, err := s.basePolicyRepo.FindKeysByPattern(ctx, pattern, "")
			if err != nil {
				slog.Warn("Failed to find keys for cleanup",
					"pattern", pattern,
					"base_policy_id", policy.BasePolicyID,
					"error", err)
				continue
			}

			for _, key := range keys {
				if err := s.basePolicyRepo.DeleteTempBasePolicyModel(ctx, key); err != nil {
					slog.Warn("Failed to delete Redis key",
						"key", key,
						"base_policy_id", policy.BasePolicyID,
						"error", err)
				}
			}
		}

		slog.Info("Policy cleanup completed",
			"base_policy_id", policy.BasePolicyID,
			"patterns_processed", len(patterns))
	}

	return nil
}

func (s *BasePolicyService) GetActivePolicies(ctx context.Context) ([]models.BasePolicy, error) {
	return s.basePolicyRepo.GetBasePoliciesByStatus(models.BasePolicyActive)
}

func (s *BasePolicyService) GetAllPolicyCreationResponse(ctx context.Context) (any, error) {
	keyParttern := "*--*--CompletePolicyResponse"
	keys, err := s.basePolicyRepo.FindKeysByPattern(ctx, keyParttern, "")
	if err != nil {
		return nil, err
	}
	res := []map[string]any{}
	for _, key := range keys {
		completeResponseData, err := s.basePolicyRepo.GetTempBasePolicyModels(ctx, key)
		if err != nil {
			slog.Error("complete response data retrive failed", "error", err)
			continue
		}
		jsonFormat := make(map[string]any)
		err = json.Unmarshal(completeResponseData, &jsonFormat)
		if err != nil {
			slog.Error("complete response data retrive failed", "error", err)
			continue
		}
		res = append(res, jsonFormat)
	}
	return res, nil
}

// ============================================================================
// COMPLETE POLICY DETAIL SERVICE METHODS
// ============================================================================

// GetCompletePolicyDetail retrieves complete policy details with document
func (s *BasePolicyService) GetCompletePolicyDetail(
	ctx context.Context,
	filter models.PolicyDetailFilterRequest,
) (*models.CompletePolicyDetailResponse, error) {
	slog.Info("Getting complete policy detail",
		"id", filter.ID,
		"provider_id", filter.ProviderID,
		"crop_type", filter.CropType,
		"status", filter.Status)

	start := time.Now()

	// Step 1: Get base policy and triggers
	basePolicy, triggers, err := s.basePolicyRepo.GetCompletePolicyByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Step 2: Calculate metadata
	totalConditions := 0
	for _, trigger := range triggers {
		totalConditions += len(trigger.Conditions)
	}

	totalDataCost, err := s.basePolicyRepo.CalculateTotalBasePolicyDataCost(basePolicy.ID)
	if err != nil {
		slog.Warn("Failed to calculate total data cost",
			"policy_id", basePolicy.ID,
			"error", err)
		totalDataCost = 0
	}

	dataSourceCount, err := s.basePolicyRepo.GetBasePolicyDataSourceCount(basePolicy.ID)
	if err != nil {
		slog.Warn("Failed to get data source count",
			"policy_id", basePolicy.ID,
			"error", err)
		dataSourceCount = 0
	}

	metadata := models.PolicyDetailMetadata{
		TotalTriggers:   len(triggers),
		TotalConditions: totalConditions,
		TotalDataCost:   totalDataCost,
		DataSourceCount: dataSourceCount,
		RetrievedAt:     time.Now(),
	}

	// Step 3: Get document info from MinIO
	var documentInfo *models.PolicyDocumentInfo
	if filter.IncludePDF {
		documentInfo = s.getDocumentInfo(ctx, basePolicy, filter.PDFExpiryHours)
	}

	response := &models.CompletePolicyDetailResponse{
		BasePolicy: *basePolicy,
		Triggers:   triggers,
		Document:   documentInfo,
		Metadata:   metadata,
	}

	slog.Info("Successfully retrieved complete policy detail",
		"policy_id", basePolicy.ID,
		"triggers", metadata.TotalTriggers,
		"conditions", metadata.TotalConditions,
		"duration", time.Since(start))

	return response, nil
}

// getDocumentInfo retrieves document metadata and presigned URL from MinIO
func (s *BasePolicyService) getDocumentInfo(
	ctx context.Context,
	policy *models.BasePolicy,
	expiryHours int,
) *models.PolicyDocumentInfo {
	docInfo := &models.PolicyDocumentInfo{
		HasDocument: false,
	}

	// Check if template document URL exists
	if policy.TemplateDocumentURL == nil || *policy.TemplateDocumentURL == "" {
		slog.Info("No template document URL found for policy",
			"policy_id", policy.ID)
		return docInfo
	}

	docInfo.HasDocument = true
	docInfo.DocumentURL = policy.TemplateDocumentURL
	docInfo.BucketName = minio.Storage.PolicyDocuments

	// Extract object name from URL
	// Assuming URL format: "policy-documents/provider_id/policy_id/document.pdf"
	// or just the object path without bucket name
	objectName := s.extractObjectNameFromURL(*policy.TemplateDocumentURL)
	docInfo.ObjectName = objectName

	slog.Info("Processing document info",
		"policy_id", policy.ID,
		"bucket", docInfo.BucketName,
		"object", objectName)

	// Check if MinIO client is available
	if s.minioClient == nil {
		errMsg := "MinIO client not initialized"
		docInfo.Error = &errMsg
		slog.Error("MinIO client is nil",
			"policy_id", policy.ID)
		return docInfo
	}

	// Check if file exists in MinIO
	exists, err := s.minioClient.FileExists(ctx, docInfo.BucketName, objectName)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to check file existence: %v", err)
		docInfo.Error = &errMsg
		slog.Error("MinIO file check failed",
			"bucket", docInfo.BucketName,
			"object", objectName,
			"error", err)
		return docInfo
	}

	if !exists {
		errMsg := "Document file not found in storage"
		docInfo.Error = &errMsg
		slog.Warn("Document file not found",
			"bucket", docInfo.BucketName,
			"object", objectName)
		return docInfo
	}

	// Get file metadata
	minioClient := s.minioClient.GetClient()
	objInfo, err := minioClient.StatObject(ctx, docInfo.BucketName, objectName, minioSDK.StatObjectOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get file metadata: %v", err)
		docInfo.Error = &errMsg
		slog.Error("MinIO stat object failed",
			"bucket", docInfo.BucketName,
			"object", objectName,
			"error", err)
		return docInfo
	}

	docInfo.ContentType = objInfo.ContentType
	docInfo.FileSizeBytes = objInfo.Size

	// Generate presigned URL
	expiry := time.Duration(expiryHours) * time.Hour
	presignedURL, err := s.minioClient.GetPresignedURL(ctx, docInfo.BucketName, objectName, expiry)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to generate presigned URL: %v", err)
		docInfo.Error = &errMsg
		slog.Error("MinIO presigned URL generation failed",
			"bucket", docInfo.BucketName,
			"object", objectName,
			"error", err)
		return docInfo
	}

	docInfo.PresignedURL = &presignedURL
	expiryTime := time.Now().Add(expiry)
	docInfo.PresignedExpiry = &expiryTime

	slog.Info("Successfully generated document info",
		"policy_id", policy.ID,
		"object", objectName,
		"size_bytes", docInfo.FileSizeBytes,
		"expiry_hours", expiryHours)

	return docInfo
}

// extractObjectNameFromURL extracts object name from MinIO URL
func (s *BasePolicyService) extractObjectNameFromURL(url string) string {
	// Remove any leading/trailing whitespace
	url = strings.TrimSpace(url)

	// If URL contains bucket name prefix, remove it
	// Example: "policy-documents/provider_001/policy_123/document.pdf"
	// We want: "provider_001/policy_123/document.pdf"

	// Check if URL starts with bucket name
	bucketPrefix := minio.Storage.PolicyDocuments + "/"
	if strings.HasPrefix(url, bucketPrefix) {
		return strings.TrimPrefix(url, bucketPrefix)
	}

	// If no bucket prefix, return as is
	return url
}
