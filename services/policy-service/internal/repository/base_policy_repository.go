package repository

import (
	"database/sql"
	"fmt"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BasePolicyRepository struct {
	db *sqlx.DB
}

func NewBasePolicyRepository(db *sqlx.DB) *BasePolicyRepository {
	return &BasePolicyRepository{db: db}
}

func (r *BasePolicyRepository) CreateBasePolicy(policy *models.BasePolicy) error {
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	query := `
		INSERT INTO base_policy (
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		) VALUES (
			:id, :insurance_provider_id, :product_name, :product_code, :product_description,
			:crop_type, :coverage_currency, :coverage_duration_days, :fix_premium_amount,
			:is_per_hectare, :premium_base_rate, :fix_payout_amount, :is_payout_per_hectare,
			:over_threshold_multiplier, :payout_base_rate, :payout_cap, :enrollment_start_day,
			:enrollment_end_day, :auto_renewal, :renewal_discount_rate, :base_policy_invalid_date,
			:insurance_valid_from_day, :insurance_valid_to_day, :status, :template_document_url,
			:document_validation_status, :document_validation_score, :important_additional_information,
			:created_at, :updated_at, :created_by
		)`

	_, err := r.db.NamedExec(query, policy)
	if err != nil {
		return fmt.Errorf("failed to create base policy: %w", err)
	}

	return nil
}

func (r *BasePolicyRepository) GetBasePolicyByID(id uuid.UUID) (*models.BasePolicy, error) {
	var policy models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, fix_payout_amount, is_payout_per_hectare,
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
			return nil, fmt.Errorf("base policy not found")
		}
		return nil, fmt.Errorf("failed to get base policy: %w", err)
	}

	return &policy, nil
}

func (r *BasePolicyRepository) GetAllBasePolicies() ([]models.BasePolicy, error) {
	var policies []models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, important_additional_information,
			created_at, updated_at, created_by
		FROM base_policy
		ORDER BY created_at DESC`

	err := r.db.Select(&policies, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policies: %w", err)
	}

	return policies, nil
}

func (r *BasePolicyRepository) GetBasePoliciesByProvider(providerID string) ([]models.BasePolicy, error) {
	var policies []models.BasePolicy
	query := `
		SELECT 
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, fix_payout_amount, is_payout_per_hectare,
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
			is_per_hectare, premium_base_rate, fix_payout_amount, is_payout_per_hectare,
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
			is_per_hectare, premium_base_rate, fix_payout_amount, is_payout_per_hectare,
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
		return fmt.Errorf("failed to update base policy: %w", err)
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

func (r *BasePolicyRepository) CreateBasePolicyTriggerConditionsBatch(conditions []models.BasePolicyTriggerCondition) error {
	if len(conditions) == 0 {
		return nil
	}

	// Start transaction for batch operation
	tx, err := r.db.Beginx()
	if err != nil {
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
	for _, condition := range conditions {
		_, err := tx.NamedExec(query, condition)
		if err != nil {
			return fmt.Errorf("failed to insert condition %s: %w", condition.ID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch insert: %w", err)
	}

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
