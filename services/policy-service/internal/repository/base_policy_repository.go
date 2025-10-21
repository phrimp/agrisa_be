package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type BasePolicyRepository struct {
	db          *sqlx.DB
	redisClient *redis.Client
}

func NewBasePolicyRepository(db *sqlx.DB, redisClient *redis.Client) *BasePolicyRepository {
	return &BasePolicyRepository{
		db:          db,
		redisClient: redisClient,
	}
}

func (r *BasePolicyRepository) CreateTempBasePolicyModels(ctx context.Context, model []byte, key string, expiration time.Duration) error {
	err := r.redisClient.Set(ctx, key, model, expiration).Err()
	return err
}

func (r *BasePolicyRepository) GetTempBasePolicyModels(ctx context.Context, key string) ([]byte, error) {
	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *BasePolicyRepository) DeleteTempBasePolicyModel(ctx context.Context, key string) error {
	err := r.redisClient.Del(ctx, key).Err()
	return err
}

func (r *BasePolicyRepository) CreateTempBasePolicyModelsWTransaction(ctx context.Context, model []byte, key string, tx redis.Pipeliner, expiration time.Duration) error {
	err := tx.Set(ctx, key, model, expiration).Err()
	return err
}

func (r *BasePolicyRepository) BeginRedisTransaction() redis.Pipeliner {
	return r.redisClient.TxPipeline()
}

func (r *BasePolicyRepository) FindKeysByPattern(ctx context.Context, pattern string) ([]string, error) {
	var keys []string

	iter := r.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	return keys, nil
}

func (r *BasePolicyRepository) CreateBasePolicy(policy *models.BasePolicy) error {
	slog.Info("Creating base policy",
		"policy_id", policy.ID,
		"provider_id", policy.InsuranceProviderID,
		"product_name", policy.ProductName,
		"crop_type", policy.CropType)

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	query := `
		INSERT INTO base_policy (
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		) VALUES (
			:id, :insurance_provider_id, :product_name, :product_code, :product_description,
			:crop_type, :coverage_currency, :coverage_duration_days, :fix_premium_amount,
			:is_per_hectare, :premium_base_rate, :max_premium_payment_prolong, :fix_payout_amount, :is_payout_per_hectare,
			:over_threshold_multiplier, :payout_base_rate, :payout_cap, :enrollment_start_day,
			:enrollment_end_day, :auto_renewal, :renewal_discount_rate, :base_policy_invalid_date,
			:insurance_valid_from_day, :insurance_valid_to_day, :status, :template_document_url,
			:document_validation_status, :document_validation_score, :important_additional_information,
			:created_at, :updated_at, :created_by
		)`

	_, err := r.db.NamedExec(query, policy)
	if err != nil {
		slog.Error("Failed to create base policy",
			"policy_id", policy.ID,
			"error", err)
		return fmt.Errorf("failed to create base policy: %w", err)
	}

	slog.Info("Successfully created base policy",
		"policy_id", policy.ID,
		"provider_id", policy.InsuranceProviderID,
		"duration", time.Since(policy.CreatedAt))
	return nil
}

func (r *BasePolicyRepository) GetBasePolicyByID(id uuid.UUID) (*models.BasePolicy, error) {
	slog.Info("Retrieving base policy by ID", "policy_id", id)
	start := time.Now()

	var policy models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		FROM base_policy
		WHERE id = $1`

	err := r.db.Get(&policy, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Base policy not found", "policy_id", id)
			return nil, fmt.Errorf("base policy not found")
		}
		slog.Error("Failed to get base policy",
			"policy_id", id,
			"error", err)
		return nil, fmt.Errorf("failed to get base policy: %w", err)
	}

	slog.Info("Successfully retrieved base policy",
		"policy_id", id,
		"provider_id", policy.InsuranceProviderID,
		"product_name", policy.ProductName,
		"duration", time.Since(start))
	return &policy, nil
}

func (r *BasePolicyRepository) GetAllBasePolicies() ([]models.BasePolicy, error) {
	slog.Info("Retrieving all base policies")
	start := time.Now()

	var policies []models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		FROM base_policy
		ORDER BY created_at DESC`

	err := r.db.Select(&policies, query)
	if err != nil {
		slog.Error("Failed to get all base policies", "error", err)
		return nil, fmt.Errorf("failed to get base policies: %w", err)
	}

	slog.Info("Successfully retrieved all base policies",
		"count", len(policies),
		"duration", time.Since(start))
	return policies, nil
}

func (r *BasePolicyRepository) GetBasePoliciesByProvider(providerID string) ([]models.BasePolicy, error) {
	var policies []models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		FROM base_policy
		WHERE insurance_provider_id = $1
		ORDER BY created_at DESC`

	err := r.db.Select(&policies, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policies by provider: %w", err)
	}

	return policies, nil
}

func (r *BasePolicyRepository) GetBasePoliciesByStatus(status models.BasePolicyStatus) ([]models.BasePolicy, error) {
	var policies []models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		FROM base_policy
		WHERE status = $1
		ORDER BY created_at DESC`

	err := r.db.Select(&policies, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policies by status: %w", err)
	}

	return policies, nil
}

func (r *BasePolicyRepository) GetBasePoliciesByCropType(cropType string) ([]models.BasePolicy, error) {
	var policies []models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		FROM base_policy
		WHERE crop_type = $1
		ORDER BY created_at DESC`

	err := r.db.Select(&policies, query, cropType)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policies by crop type: %w", err)
	}

	return policies, nil
}

func (r *BasePolicyRepository) UpdateBasePolicy(policy *models.BasePolicy) error {
	slog.Info("Updating base policy",
		"policy_id", policy.ID,
		"provider_id", policy.InsuranceProviderID,
		"product_name", policy.ProductName)
	start := time.Now()

	policy.UpdatedAt = time.Now()

	query := `
		UPDATE base_policy SET
			insurance_provider_id = :insurance_provider_id,
			product_name = :product_name,
			product_code = :product_code,
			product_description = :product_description,
			crop_type = :crop_type,
			coverage_currency = :coverage_currency,
			coverage_duration_days = :coverage_duration_days,
			fix_premium_amount = :fix_premium_amount,
			is_per_hectare = :is_per_hectare,
			premium_base_rate = :premium_base_rate,
			max_premium_payment_prolong = :max_premium_payment_prolong,
			fix_payout_amount = :fix_payout_amount,
			is_payout_per_hectare = :is_payout_per_hectare,
			over_threshold_multiplier = :over_threshold_multiplier,
			payout_base_rate = :payout_base_rate,
			payout_cap = :payout_cap,
			enrollment_start_day = :enrollment_start_day,
			enrollment_end_day = :enrollment_end_day,
			auto_renewal = :auto_renewal,
			renewal_discount_rate = :renewal_discount_rate,
			base_policy_invalid_date = :base_policy_invalid_date,
			insurance_valid_from_day = :insurance_valid_from_day,
			insurance_valid_to_day = :insurance_valid_to_day,
			status = :status,
			template_document_url = :template_document_url,
			document_validation_status = :document_validation_status,
			document_validation_score = :document_validation_score,
			important_additional_information = :important_additional_information,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExec(query, policy)
	if err != nil {
		slog.Error("Failed to update base policy",
			"policy_id", policy.ID,
			"error", err)
		return fmt.Errorf("failed to update base policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to get rows affected for update",
			"policy_id", policy.ID,
			"error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("Base policy not found for update", "policy_id", policy.ID)
		return fmt.Errorf("base policy not found")
	}

	slog.Info("Successfully updated base policy",
		"policy_id", policy.ID,
		"rows_affected", rowsAffected,
		"duration", time.Since(start))
	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicy(id uuid.UUID) error {
	query := `DELETE FROM base_policy WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete base policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("base policy not found")
	}

	return nil
}

func (r *BasePolicyRepository) CheckBasePolicyExists(id uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM base_policy WHERE id = $1`

	err := r.db.Get(&count, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to check base policy existence: %w", err)
	}

	return count > 0, nil
}

func (r *BasePolicyRepository) GetBasePolicyCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM base_policy`

	err := r.db.Get(&count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get base policy count: %w", err)
	}

	return count, nil
}

func (r *BasePolicyRepository) GetBasePolicyCountByStatus(status models.BasePolicyStatus) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM base_policy WHERE status = $1`

	err := r.db.Get(&count, query, status)
	if err != nil {
		return 0, fmt.Errorf("failed to get base policy count by status: %w", err)
	}

	return count, nil
}

// ============================================================================
// BASE POLICY TRIGGER CRUD OPERATIONS
// ============================================================================

func (r *BasePolicyRepository) CreateBasePolicyTrigger(trigger *models.BasePolicyTrigger) error {
	trigger.CreatedAt = time.Now()
	trigger.UpdatedAt = time.Now()

	query := `
		INSERT INTO base_policy_trigger (
			id, base_policy_id, logical_operator, growth_stage, 
			monitor_frequency_value, monitor_frequency_unit, blackout_periods,
			created_at, updated_at
		) VALUES (
			:id, :base_policy_id, :logical_operator, :growth_stage,
			:monitor_frequency_value, :monitor_frequency_unit, :blackout_periods,
			:created_at, :updated_at
		)`

	_, err := r.db.NamedExec(query, trigger)
	if err != nil {
		return fmt.Errorf("failed to create base policy trigger: %w", err)
	}

	return nil
}

func (r *BasePolicyRepository) GetBasePolicyTriggerByID(id uuid.UUID) (*models.BasePolicyTrigger, error) {
	var trigger models.BasePolicyTrigger
	query := `
		SELECT 
			id, base_policy_id, logical_operator, growth_stage,
			monitor_frequency_value, monitor_frequency_unit, blackout_periods,
			created_at, updated_at
		FROM base_policy_trigger
		WHERE id = $1`

	err := r.db.Get(&trigger, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("base policy trigger not found")
		}
		return nil, fmt.Errorf("failed to get base policy trigger: %w", err)
	}

	return &trigger, nil
}

func (r *BasePolicyRepository) GetBasePolicyTriggersByPolicyID(policyID uuid.UUID) ([]models.BasePolicyTrigger, error) {
	var triggers []models.BasePolicyTrigger
	query := `
		SELECT 
			id, base_policy_id, logical_operator, growth_stage,
			monitor_frequency_value, monitor_frequency_unit, blackout_periods,
			created_at, updated_at
		FROM base_policy_trigger
		WHERE base_policy_id = $1
		ORDER BY created_at`

	err := r.db.Select(&triggers, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policy triggers: %w", err)
	}

	return triggers, nil
}

func (r *BasePolicyRepository) UpdateBasePolicyTrigger(trigger *models.BasePolicyTrigger) error {
	trigger.UpdatedAt = time.Now()

	query := `
		UPDATE base_policy_trigger SET
			logical_operator = :logical_operator,
			growth_stage = :growth_stage,
			monitor_frequency_value = :monitor_frequency_value,
			monitor_frequency_unit = :monitor_frequency_unit,
			blackout_periods = :blackout_periods,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExec(query, trigger)
	if err != nil {
		return fmt.Errorf("failed to update base policy trigger: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("base policy trigger not found")
	}

	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicyTrigger(id uuid.UUID) error {
	query := `DELETE FROM base_policy_trigger WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete base policy trigger: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("base policy trigger not found")
	}

	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicyTriggersByPolicyID(policyID uuid.UUID) error {
	query := `DELETE FROM base_policy_trigger WHERE base_policy_id = $1`

	_, err := r.db.Exec(query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete base policy triggers by policy ID: %w", err)
	}

	return nil
}

func (r *BasePolicyRepository) CheckBasePolicyTriggerExists(id uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM base_policy_trigger WHERE id = $1`

	err := r.db.Get(&count, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to check base policy trigger existence: %w", err)
	}

	return count > 0, nil
}

// ============================================================================
// BASE POLICY TRIGGER CONDITION CRUD OPERATIONS
// ============================================================================

func (r *BasePolicyRepository) CreateBasePolicyTriggerCondition(condition *models.BasePolicyTriggerCondition) error {
	condition.CreatedAt = time.Now()

	query := `
		INSERT INTO base_policy_trigger_condition (
			id, base_policy_trigger_id, data_source_id, threshold_operator,
			threshold_value, early_warning_threshold, aggregation_function,
			aggregation_window_days, consecutive_required, baseline_window_days,
			baseline_function, validation_window_days, condition_order,
			base_cost, category_multiplier, tier_multiplier, calculated_cost, created_at
		) VALUES (
			:id, :base_policy_trigger_id, :data_source_id, :threshold_operator,
			:threshold_value, :early_warning_threshold, :aggregation_function,
			:aggregation_window_days, :consecutive_required, :baseline_window_days,
			:baseline_function, :validation_window_days, :condition_order,
			:base_cost, :category_multiplier, :tier_multiplier, :calculated_cost, :created_at
		)`

	_, err := r.db.NamedExec(query, condition)
	if err != nil {
		return fmt.Errorf("failed to create base policy trigger condition: %w", err)
	}

	return nil
}

func (r *BasePolicyRepository) CreateBasePolicyTriggerConditionsBatch(conditions []*models.BasePolicyTriggerCondition) error {
	slog.Info("Creating base policy trigger conditions batch",
		"condition_count", len(conditions))
	start := time.Now()

	if len(conditions) == 0 {
		slog.Warn("Empty conditions batch provided")
		return nil
	}

	// Start transaction for batch operation
	slog.Debug("Starting transaction for batch insert")
	tx, err := r.db.Beginx()
	if err != nil {
		slog.Error("Failed to begin transaction for batch insert", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare conditions with timestamps
	now := time.Now()
	for i := range conditions {
		conditions[i].CreatedAt = now
	}

	query := `
		INSERT INTO base_policy_trigger_condition (
			id, base_policy_trigger_id, data_source_id, threshold_operator,
			threshold_value, early_warning_threshold, aggregation_function,
			aggregation_window_days, consecutive_required, baseline_window_days,
			baseline_function, validation_window_days, condition_order,
			base_cost, category_multiplier, tier_multiplier, calculated_cost, created_at
		) VALUES (
			:id, :base_policy_trigger_id, :data_source_id, :threshold_operator,
			:threshold_value, :early_warning_threshold, :aggregation_function,
			:aggregation_window_days, :consecutive_required, :baseline_window_days,
			:baseline_function, :validation_window_days, :condition_order,
			:base_cost, :category_multiplier, :tier_multiplier, :calculated_cost, :created_at
		)`

	// Execute batch insert
	for i, condition := range conditions {
		slog.Debug("Inserting condition",
			"index", i+1,
			"condition_id", condition.ID,
			"data_source_id", condition.DataSourceID)
		_, err := tx.NamedExec(query, condition)
		if err != nil {
			slog.Error("Failed to insert condition",
				"condition_id", condition.ID,
				"index", i+1,
				"error", err)
			return fmt.Errorf("failed to insert condition %s: %w", condition.ID, err)
		}
	}

	// Commit transaction
	slog.Debug("Committing batch transaction")
	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit batch transaction", "error", err)
		return fmt.Errorf("failed to commit batch insert: %w", err)
	}

	slog.Info("Successfully created batch trigger conditions",
		"condition_count", len(conditions),
		"duration", time.Since(start))
	return nil
}

func (r *BasePolicyRepository) GetBasePolicyTriggerConditionByID(id uuid.UUID) (*models.BasePolicyTriggerCondition, error) {
	var condition models.BasePolicyTriggerCondition
	query := `
		SELECT 
			id, base_policy_trigger_id, data_source_id, threshold_operator,
			threshold_value, early_warning_threshold, aggregation_function,
			aggregation_window_days, consecutive_required, baseline_window_days,
			baseline_function, validation_window_days, condition_order,
			base_cost, category_multiplier, tier_multiplier, calculated_cost, created_at
		FROM base_policy_trigger_condition
		WHERE id = $1`

	err := r.db.Get(&condition, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("base policy trigger condition not found")
		}
		return nil, fmt.Errorf("failed to get base policy trigger condition: %w", err)
	}

	return &condition, nil
}

func (r *BasePolicyRepository) GetBasePolicyTriggerConditionsByTriggerID(triggerID uuid.UUID) ([]models.BasePolicyTriggerCondition, error) {
	var conditions []models.BasePolicyTriggerCondition
	query := `
		SELECT 
			id, base_policy_trigger_id, data_source_id, threshold_operator,
			threshold_value, early_warning_threshold, aggregation_function,
			aggregation_window_days, consecutive_required, baseline_window_days,
			baseline_function, validation_window_days, condition_order,
			base_cost, category_multiplier, tier_multiplier, calculated_cost, created_at
		FROM base_policy_trigger_condition
		WHERE base_policy_trigger_id = $1
		ORDER BY condition_order`

	err := r.db.Select(&conditions, query, triggerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policy trigger conditions: %w", err)
	}

	return conditions, nil
}

func (r *BasePolicyRepository) GetBasePolicyTriggerConditionsByPolicyID(policyID uuid.UUID) ([]models.BasePolicyTriggerCondition, error) {
	var conditions []models.BasePolicyTriggerCondition
	query := `
		SELECT 
			btc.id, btc.base_policy_trigger_id, btc.data_source_id, btc.threshold_operator,
			btc.threshold_value, btc.early_warning_threshold, btc.aggregation_function,
			btc.aggregation_window_days, btc.consecutive_required, btc.baseline_window_days,
			btc.baseline_function, btc.validation_window_days, btc.condition_order,
			btc.base_cost, btc.category_multiplier, btc.tier_multiplier, btc.calculated_cost, btc.created_at
		FROM base_policy_trigger_condition btc
		JOIN base_policy_trigger bt ON bt.id = btc.base_policy_trigger_id
		WHERE bt.base_policy_id = $1
		ORDER BY bt.created_at, btc.condition_order`

	err := r.db.Select(&conditions, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policy trigger conditions by policy ID: %w", err)
	}

	return conditions, nil
}

func (r *BasePolicyRepository) UpdateBasePolicyTriggerCondition(condition *models.BasePolicyTriggerCondition) error {
	query := `
		UPDATE base_policy_trigger_condition SET
			data_source_id = :data_source_id,
			threshold_operator = :threshold_operator,
			threshold_value = :threshold_value,
			early_warning_threshold = :early_warning_threshold,
			aggregation_function = :aggregation_function,
			aggregation_window_days = :aggregation_window_days,
			consecutive_required = :consecutive_required,
			baseline_window_days = :baseline_window_days,
			baseline_function = :baseline_function,
			validation_window_days = :validation_window_days,
			condition_order = :condition_order,
			base_cost = :base_cost,
			category_multiplier = :category_multiplier,
			tier_multiplier = :tier_multiplier,
			calculated_cost = :calculated_cost
		WHERE id = :id`

	result, err := r.db.NamedExec(query, condition)
	if err != nil {
		return fmt.Errorf("failed to update base policy trigger condition: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("base policy trigger condition not found")
	}

	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicyTriggerCondition(id uuid.UUID) error {
	query := `DELETE FROM base_policy_trigger_condition WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete base policy trigger condition: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("base policy trigger condition not found")
	}

	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicyTriggerConditionsByTriggerID(triggerID uuid.UUID) error {
	query := `DELETE FROM base_policy_trigger_condition WHERE base_policy_trigger_id = $1`

	_, err := r.db.Exec(query, triggerID)
	if err != nil {
		return fmt.Errorf("failed to delete base policy trigger conditions by trigger ID: %w", err)
	}

	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicyTriggerConditionsByPolicyID(policyID uuid.UUID) error {
	query := `
		DELETE FROM base_policy_trigger_condition 
		WHERE base_policy_trigger_id IN (
			SELECT id FROM base_policy_trigger WHERE base_policy_id = $1
		)`

	_, err := r.db.Exec(query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete base policy trigger conditions by policy ID: %w", err)
	}

	return nil
}

func (r *BasePolicyRepository) CheckBasePolicyTriggerConditionExists(id uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM base_policy_trigger_condition WHERE id = $1`

	err := r.db.Get(&count, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to check base policy trigger condition existence: %w", err)
	}

	return count > 0, nil
}

// ============================================================================
// TRANSACTION METHODS FOR COMPLETE POLICY CREATION
// ============================================================================

func (r *BasePolicyRepository) BeginTransaction() (*sqlx.Tx, error) {
	slog.Debug("Beginning database transaction")
	tx, err := r.db.Beginx()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (r *BasePolicyRepository) CreateBasePolicyTx(tx *sqlx.Tx, policy *models.BasePolicy) error {
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	query := `
		INSERT INTO base_policy (
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		) VALUES (
			:id, :insurance_provider_id, :product_name, :product_code, :product_description,
			:crop_type, :coverage_currency, :coverage_duration_days, :fix_premium_amount,
			:is_per_hectare, :premium_base_rate, :max_premium_payment_prolong, :fix_payout_amount, :is_payout_per_hectare,
			:over_threshold_multiplier, :payout_base_rate, :payout_cap, :enrollment_start_day,
			:enrollment_end_day, :auto_renewal, :renewal_discount_rate, :base_policy_invalid_date,
			:insurance_valid_from_day, :insurance_valid_to_day, :status, :template_document_url,
			:document_validation_status, :document_validation_score, :important_additional_information,
			:created_at, :updated_at, :created_by
		)`

	_, err := tx.NamedExec(query, policy)
	return err
}

func (r *BasePolicyRepository) CreateBasePolicyTriggerTx(tx *sqlx.Tx, trigger *models.BasePolicyTrigger) error {
	trigger.CreatedAt = time.Now()
	trigger.UpdatedAt = time.Now()

	query := `
		INSERT INTO base_policy_trigger (
			id, base_policy_id, logical_operator, growth_stage, 
			monitor_frequency_value, monitor_frequency_unit, blackout_periods,
			created_at, updated_at
		) VALUES (
			:id, :base_policy_id, :logical_operator, :growth_stage,
			:monitor_frequency_value, :monitor_frequency_unit, :blackout_periods,
			:created_at, :updated_at
		)`

	_, err := tx.NamedExec(query, trigger)
	return err
}

func (r *BasePolicyRepository) CreateBasePolicyTriggerConditionsBatchTx(tx *sqlx.Tx, conditions []*models.BasePolicyTriggerCondition) error {
	if len(conditions) == 0 {
		return nil
	}

	now := time.Now()
	for i := range conditions {
		conditions[i].CreatedAt = now
	}

	query := `
		INSERT INTO base_policy_trigger_condition (
			id, base_policy_trigger_id, data_source_id, threshold_operator,
			threshold_value, early_warning_threshold, aggregation_function,
			aggregation_window_days, consecutive_required, baseline_window_days,
			baseline_function, validation_window_days, condition_order,
			base_cost, category_multiplier, tier_multiplier, calculated_cost, created_at
		) VALUES (
			:id, :base_policy_trigger_id, :data_source_id, :threshold_operator,
			:threshold_value, :early_warning_threshold, :aggregation_function,
			:aggregation_window_days, :consecutive_required, :baseline_window_days,
			:baseline_function, :validation_window_days, :condition_order,
			:base_cost, :category_multiplier, :tier_multiplier, :calculated_cost, :created_at
		)`

	for _, condition := range conditions {
		if _, err := tx.NamedExec(query, condition); err != nil {
			return err
		}
	}
	return nil
}

func (r *BasePolicyRepository) CalculateTotalBasePolicyDataCostTx(tx *sqlx.Tx, policyID uuid.UUID) (float64, error) {
	var totalCost float64
	query := `
		SELECT COALESCE(SUM(btc.calculated_cost), 0) 
		FROM base_policy_trigger_condition btc
		JOIN base_policy_trigger bt ON bt.id = btc.base_policy_trigger_id
		WHERE bt.base_policy_id = $1`

	err := tx.Get(&totalCost, query, policyID)
	return totalCost, err
}

// ============================================================================
// DATA COST CALCULATION METHODS (using merged trigger conditions)
// ============================================================================

func (r *BasePolicyRepository) CalculateTotalBasePolicyDataCost(policyID uuid.UUID) (float64, error) {
	var totalCost float64
	query := `
		SELECT COALESCE(SUM(btc.calculated_cost), 0) 
		FROM base_policy_trigger_condition btc
		JOIN base_policy_trigger bt ON bt.id = btc.base_policy_trigger_id
		WHERE bt.base_policy_id = $1`

	err := r.db.Get(&totalCost, query, policyID)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total base policy data cost: %w", err)
	}

	return totalCost, nil
}

// ============================================================================
// BASE POLICY DOCUMENT VALIDATION CRUD OPERATIONS
// ============================================================================

func (r *BasePolicyRepository) CreateBasePolicyDocumentValidation(validation *models.BasePolicyDocumentValidation) error {
	slog.Info("Creating base policy document validation",
		"validation_id", validation.ID,
		"base_policy_id", validation.BasePolicyID,
		"validation_status", validation.ValidationStatus)

	validation.CreatedAt = time.Now()

	query := `
		INSERT INTO base_policy_document_validation (
			id, base_policy_id, validation_timestamp, validation_status, overall_score,
			total_checks, passed_checks, failed_checks, warning_count, mismatches,
			warnings, recommendations, extracted_parameters, validated_by,
			validation_notes, created_at
		) VALUES (
			:id, :base_policy_id, :validation_timestamp, :validation_status, :overall_score,
			:total_checks, :passed_checks, :failed_checks, :warning_count, :mismatches,
			:warnings, :recommendations, :extracted_parameters, :validated_by,
			:validation_notes, :created_at
		)`

	_, err := r.db.NamedExec(query, validation)
	if err != nil {
		slog.Error("Failed to create base policy document validation",
			"validation_id", validation.ID,
			"base_policy_id", validation.BasePolicyID,
			"error", err)
		return fmt.Errorf("failed to create base policy document validation: %w", err)
	}

	slog.Info("Successfully created base policy document validation",
		"validation_id", validation.ID,
		"base_policy_id", validation.BasePolicyID)
	return nil
}

func (r *BasePolicyRepository) GetBasePolicyDocumentValidationByID(id uuid.UUID) (*models.BasePolicyDocumentValidation, error) {
	slog.Debug("Retrieving base policy document validation by ID", "validation_id", id)

	var validation models.BasePolicyDocumentValidation
	query := `
		SELECT 
			id, base_policy_id, validation_timestamp, validation_status, overall_score,
			total_checks, passed_checks, failed_checks, warning_count, mismatches,
			warnings, recommendations, extracted_parameters, validated_by,
			validation_notes, created_at
		FROM base_policy_document_validation
		WHERE id = $1`

	err := r.db.Get(&validation, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Base policy document validation not found", "validation_id", id)
			return nil, fmt.Errorf("base policy document validation not found")
		}
		slog.Error("Failed to get base policy document validation",
			"validation_id", id,
			"error", err)
		return nil, fmt.Errorf("failed to get base policy document validation: %w", err)
	}

	slog.Debug("Successfully retrieved base policy document validation",
		"validation_id", id,
		"base_policy_id", validation.BasePolicyID)
	return &validation, nil
}

func (r *BasePolicyRepository) GetBasePolicyDocumentValidationsByPolicyID(basePolicyID uuid.UUID) ([]models.BasePolicyDocumentValidation, error) {
	slog.Debug("Retrieving base policy document validations by policy ID", "base_policy_id", basePolicyID)

	var validations []models.BasePolicyDocumentValidation
	query := `
		SELECT 
			id, base_policy_id, validation_timestamp, validation_status, overall_score,
			total_checks, passed_checks, failed_checks, warning_count, mismatches,
			warnings, recommendations, extracted_parameters, validated_by,
			validation_notes, created_at
		FROM base_policy_document_validation
		WHERE base_policy_id = $1
		ORDER BY validation_timestamp DESC`

	err := r.db.Select(&validations, query, basePolicyID)
	if err != nil {
		slog.Error("Failed to get base policy document validations",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("failed to get base policy document validations: %w", err)
	}

	slog.Debug("Successfully retrieved base policy document validations",
		"base_policy_id", basePolicyID,
		"count", len(validations))
	return validations, nil
}

func (r *BasePolicyRepository) GetLatestBasePolicyDocumentValidation(basePolicyID uuid.UUID) (*models.BasePolicyDocumentValidation, error) {
	slog.Debug("Retrieving latest base policy document validation", "base_policy_id", basePolicyID)

	var validation models.BasePolicyDocumentValidation
	query := `
		SELECT 
			id, base_policy_id, validation_timestamp, validation_status, overall_score,
			total_checks, passed_checks, failed_checks, warning_count, mismatches,
			warnings, recommendations, extracted_parameters, validated_by,
			validation_notes, created_at
		FROM base_policy_document_validation
		WHERE base_policy_id = $1
		ORDER BY validation_timestamp DESC
		LIMIT 1`

	err := r.db.Get(&validation, query, basePolicyID)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Debug("No document validation found for base policy", "base_policy_id", basePolicyID)
			return nil, fmt.Errorf("no document validation found for base policy")
		}
		slog.Error("Failed to get latest base policy document validation",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("failed to get latest base policy document validation: %w", err)
	}

	slog.Debug("Successfully retrieved latest base policy document validation",
		"validation_id", validation.ID,
		"base_policy_id", basePolicyID)
	return &validation, nil
}

func (r *BasePolicyRepository) UpdateBasePolicyDocumentValidation(validation *models.BasePolicyDocumentValidation) error {
	slog.Info("Updating base policy document validation",
		"validation_id", validation.ID,
		"base_policy_id", validation.BasePolicyID,
		"validation_status", validation.ValidationStatus)

	query := `
		UPDATE base_policy_document_validation SET
			validation_timestamp = :validation_timestamp,
			validation_status = :validation_status,
			overall_score = :overall_score,
			total_checks = :total_checks,
			passed_checks = :passed_checks,
			failed_checks = :failed_checks,
			warning_count = :warning_count,
			mismatches = :mismatches,
			warnings = :warnings,
			recommendations = :recommendations,
			extracted_parameters = :extracted_parameters,
			validated_by = :validated_by,
			validation_notes = :validation_notes
		WHERE id = :id`

	result, err := r.db.NamedExec(query, validation)
	if err != nil {
		slog.Error("Failed to update base policy document validation",
			"validation_id", validation.ID,
			"error", err)
		return fmt.Errorf("failed to update base policy document validation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to get rows affected for validation update",
			"validation_id", validation.ID,
			"error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("Base policy document validation not found for update", "validation_id", validation.ID)
		return fmt.Errorf("base policy document validation not found")
	}

	slog.Info("Successfully updated base policy document validation",
		"validation_id", validation.ID,
		"base_policy_id", validation.BasePolicyID,
		"rows_affected", rowsAffected)
	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicyDocumentValidation(id uuid.UUID) error {
	slog.Info("Deleting base policy document validation", "validation_id", id)

	query := `DELETE FROM base_policy_document_validation WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		slog.Error("Failed to delete base policy document validation",
			"validation_id", id,
			"error", err)
		return fmt.Errorf("failed to delete base policy document validation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to get rows affected for validation deletion",
			"validation_id", id,
			"error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("Base policy document validation not found for deletion", "validation_id", id)
		return fmt.Errorf("base policy document validation not found")
	}

	slog.Info("Successfully deleted base policy document validation",
		"validation_id", id,
		"rows_affected", rowsAffected)
	return nil
}

func (r *BasePolicyRepository) DeleteBasePolicyDocumentValidationsByPolicyID(basePolicyID uuid.UUID) error {
	slog.Info("Deleting base policy document validations by policy ID", "base_policy_id", basePolicyID)

	query := `DELETE FROM base_policy_document_validation WHERE base_policy_id = $1`

	result, err := r.db.Exec(query, basePolicyID)
	if err != nil {
		slog.Error("Failed to delete base policy document validations by policy ID",
			"base_policy_id", basePolicyID,
			"error", err)
		return fmt.Errorf("failed to delete base policy document validations: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to get rows affected for validations deletion",
			"base_policy_id", basePolicyID,
			"error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Successfully deleted base policy document validations",
		"base_policy_id", basePolicyID,
		"rows_affected", rowsAffected)
	return nil
}

func (r *BasePolicyRepository) GetBasePolicyDataSourceCount(policyID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(DISTINCT btc.data_source_id)
		FROM base_policy_trigger_condition btc
		JOIN base_policy_trigger bt ON bt.id = btc.base_policy_trigger_id
		WHERE bt.base_policy_id = $1`

	err := r.db.Get(&count, query, policyID)
	if err != nil {
		return 0, fmt.Errorf("failed to get base policy data source count: %w", err)
	}

	return count, nil
}
