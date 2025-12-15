package repository

import (
	utils "agrisa_utils"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"sort"
	"strings"
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
	err := tx.Set(ctx, key, model, expiration+5*time.Minute).Err()
	if err != nil {
		return err
	}
	if strings.Contains(key, "--BasePolicy--archive:true") {
		err := tx.Set(ctx, key+"--COMMIT_EVENT", 1, expiration).Err()
		if err != nil {
			slog.Error("commit event key failed", "error", err)
		}
	}
	return err
}

func (r *BasePolicyRepository) BeginRedisTransaction() redis.Pipeliner {
	return r.redisClient.TxPipeline()
}

func (r *BasePolicyRepository) FindKeysByPattern(ctx context.Context, pattern, exclude string) ([]string, error) {
	var keys []string

	iter := r.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if exclude == "" || !strings.Contains(iter.Val(), exclude) {
			keys = append(keys, iter.Val())
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan keys: %w", err)
	}

	return keys, nil
}

func (r *BasePolicyRepository) CreateBasePolicy(policy *models.BasePolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}

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
			document_validation_status, document_validation_score, document_tags, important_additional_information,
			created_at, updated_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32
		)`

	_, err := r.db.Exec(query,
		policy.ID, policy.InsuranceProviderID, policy.ProductName, policy.ProductCode, policy.ProductDescription,
		policy.CropType, policy.CoverageCurrency, policy.CoverageDurationDays, policy.FixPremiumAmount,
		policy.IsPerHectare, policy.PremiumBaseRate, policy.MaxPremiumPaymentProlong, policy.FixPayoutAmount, policy.IsPayoutPerHectare,
		policy.OverThresholdMultiplier, policy.PayoutBaseRate, policy.PayoutCap, policy.EnrollmentStartDay,
		policy.EnrollmentEndDay, policy.AutoRenewal, policy.RenewalDiscountRate, policy.BasePolicyInvalidDate,
		policy.InsuranceValidFromDay, policy.InsuranceValidToDay, policy.Status, policy.TemplateDocumentURL,
		policy.DocumentValidationStatus, policy.DocumentValidationScore, policy.DocumentTags, policy.ImportantAdditionalInformation,
		policy.CreatedAt, policy.UpdatedAt, policy.CreatedBy)
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
			document_validation_status, document_validation_score, document_tags, important_additional_information,
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
			document_validation_status, document_validation_score, document_tags, important_additional_information,
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
			document_validation_status, document_validation_score, document_tags, important_additional_information,
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
			document_validation_status, document_validation_score, document_tags, important_additional_information,
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
			document_validation_status, document_validation_score, document_tags, important_additional_information,
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

	// Serialize JSONB field to []byte before database update
	var documentTagsBytes []byte
	var err error

	if policy.DocumentTags != nil {
		documentTagsBytes, err = utils.SerializeMapToBytes(policy.DocumentTags)
		if err != nil {
			return fmt.Errorf("failed to serialize document_tags: %w", err)
		}
	}

	query := `
		UPDATE base_policy SET
			insurance_provider_id = $1,
			product_name = $2,
			product_code = $3,
			product_description = $4,
			crop_type = $5,
			coverage_currency = $6,
			coverage_duration_days = $7,
			fix_premium_amount = $8,
			is_per_hectare = $9,
			premium_base_rate = $10,
			max_premium_payment_prolong = $11,
			fix_payout_amount = $12,
			is_payout_per_hectare = $13,
			over_threshold_multiplier = $14,
			payout_base_rate = $15,
			payout_cap = $16,
			enrollment_start_day = $17,
			enrollment_end_day = $18,
			auto_renewal = $19,
			renewal_discount_rate = $20,
			base_policy_invalid_date = $21,
			insurance_valid_from_day = $22,
			insurance_valid_to_day = $23,
			status = $24,
			template_document_url = $25,
			document_validation_status = $26,
			document_validation_score = $27,
			document_tags = $28,
			important_additional_information = $29,
			updated_at = $30
		WHERE id = $31`

	result, err := r.db.Exec(query,
		policy.InsuranceProviderID, policy.ProductName, policy.ProductCode, policy.ProductDescription,
		policy.CropType, policy.CoverageCurrency, policy.CoverageDurationDays, policy.FixPremiumAmount,
		policy.IsPerHectare, policy.PremiumBaseRate, policy.MaxPremiumPaymentProlong, policy.FixPayoutAmount,
		policy.IsPayoutPerHectare, policy.OverThresholdMultiplier, policy.PayoutBaseRate, policy.PayoutCap,
		policy.EnrollmentStartDay, policy.EnrollmentEndDay, policy.AutoRenewal, policy.RenewalDiscountRate,
		policy.BasePolicyInvalidDate, policy.InsuranceValidFromDay, policy.InsuranceValidToDay, policy.Status,
		policy.TemplateDocumentURL, policy.DocumentValidationStatus, policy.DocumentValidationScore,
		documentTagsBytes, policy.ImportantAdditionalInformation, policy.UpdatedAt, policy.ID)
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

func (r *BasePolicyRepository) UpdateBasePolicyTx(tx *sqlx.Tx, policy *models.BasePolicy) error {
	slog.Info("Updating base policy",
		"policy_id", policy.ID,
		"provider_id", policy.InsuranceProviderID,
		"product_name", policy.ProductName)
	start := time.Now()

	policy.UpdatedAt = time.Now()

	// Serialize JSONB field to []byte before database update
	var documentTagsBytes []byte
	var err error

	if policy.DocumentTags != nil {
		documentTagsBytes, err = utils.SerializeMapToBytes(policy.DocumentTags)
		if err != nil {
			return fmt.Errorf("failed to serialize document_tags: %w", err)
		}
	}

	query := `
		UPDATE base_policy SET
			insurance_provider_id = $1,
			product_name = $2,
			product_code = $3,
			product_description = $4,
			crop_type = $5,
			coverage_currency = $6,
			coverage_duration_days = $7,
			fix_premium_amount = $8,
			is_per_hectare = $9,
			premium_base_rate = $10,
			max_premium_payment_prolong = $11,
			fix_payout_amount = $12,
			is_payout_per_hectare = $13,
			over_threshold_multiplier = $14,
			payout_base_rate = $15,
			payout_cap = $16,
			enrollment_start_day = $17,
			enrollment_end_day = $18,
			auto_renewal = $19,
			renewal_discount_rate = $20,
			base_policy_invalid_date = $21,
			insurance_valid_from_day = $22,
			insurance_valid_to_day = $23,
			status = $24,
			template_document_url = $25,
			document_validation_status = $26,
			document_validation_score = $27,
			document_tags = $28,
			important_additional_information = $29,
			updated_at = $30
		WHERE id = $31`

	result, err := tx.Exec(query,
		policy.InsuranceProviderID, policy.ProductName, policy.ProductCode, policy.ProductDescription,
		policy.CropType, policy.CoverageCurrency, policy.CoverageDurationDays, policy.FixPremiumAmount,
		policy.IsPerHectare, policy.PremiumBaseRate, policy.MaxPremiumPaymentProlong, policy.FixPayoutAmount,
		policy.IsPayoutPerHectare, policy.OverThresholdMultiplier, policy.PayoutBaseRate, policy.PayoutCap,
		policy.EnrollmentStartDay, policy.EnrollmentEndDay, policy.AutoRenewal, policy.RenewalDiscountRate,
		policy.BasePolicyInvalidDate, policy.InsuranceValidFromDay, policy.InsuranceValidToDay, policy.Status,
		policy.TemplateDocumentURL, policy.DocumentValidationStatus, policy.DocumentValidationScore,
		documentTagsBytes, policy.ImportantAdditionalInformation, policy.UpdatedAt, policy.ID)
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
	if trigger.ID == uuid.Nil {
		trigger.ID = uuid.New()
	}

	trigger.CreatedAt = time.Now()
	trigger.UpdatedAt = time.Now()

	query := `
		INSERT INTO base_policy_trigger (
			id, base_policy_id, logical_operator, growth_stage,
			monitor_interval, monitor_frequency_unit, blackout_periods,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`
	_, err := r.db.Exec(query,
		trigger.ID, trigger.BasePolicyID, trigger.LogicalOperator, trigger.GrowthStage,
		trigger.MonitorInterval, trigger.MonitorFrequencyUnit, trigger.BlackoutPeriods,
		trigger.CreatedAt, trigger.UpdatedAt)
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
			monitor_interval, monitor_frequency_unit, blackout_periods,
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
			monitor_interval, monitor_frequency_unit, blackout_periods,
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

	// Serialize JSONB field to []byte before database update
	var blackoutPeriodsBytes []byte
	var err error

	if trigger.BlackoutPeriods != nil {
		blackoutPeriodsBytes, err = utils.SerializeMapToBytes(trigger.BlackoutPeriods)
		if err != nil {
			return fmt.Errorf("failed to serialize blackout_periods: %w", err)
		}
	}

	query := `
		UPDATE base_policy_trigger SET
			logical_operator = $1,
			growth_stage = $2,
			monitor_interval = $3,
			monitor_frequency_unit = $4,
			blackout_periods = $5,
			updated_at = $6
		WHERE id = $7`

	result, err := r.db.Exec(query,
		trigger.LogicalOperator, trigger.GrowthStage, trigger.MonitorInterval,
		trigger.MonitorFrequencyUnit, blackoutPeriodsBytes, trigger.UpdatedAt, trigger.ID)
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
	if condition.ID == uuid.Nil {
		condition.ID = uuid.New()
	}

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
	for _, condition := range conditions {
		if condition.ID == uuid.Nil {
			condition.ID = uuid.New()
		}
	}

	slog.Info("Creating base policy trigger conditions batch",
		"condition_count", len(conditions))
	start := time.Now()

	if len(conditions) == 0 {
		slog.Warn("Empty conditions batch provided")
		return nil
	}

	// Start transaction for batch operation
	slog.Info("Starting transaction for batch insert")
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
		slog.Info("Inserting condition",
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
	slog.Info("Committing batch transaction")
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
	slog.Info("Beginning database transaction")
	tx, err := r.db.Beginx()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (r *BasePolicyRepository) CreateBasePolicyTx(tx *sqlx.Tx, policy *models.BasePolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	// Serialize JSONB field to []byte before database insertion

	query := `
		INSERT INTO base_policy (
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong, fix_payout_amount, is_payout_per_hectare,
			over_threshold_multiplier, payout_base_rate, payout_cap, enrollment_start_day,
			enrollment_end_day, auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status, template_document_url,
			document_validation_status, document_validation_score, document_tags, important_additional_information,
			created_at, updated_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33
		)`

	_, err := tx.Exec(query,
		policy.ID, policy.InsuranceProviderID, policy.ProductName, policy.ProductCode, policy.ProductDescription,
		policy.CropType, policy.CoverageCurrency, policy.CoverageDurationDays, policy.FixPremiumAmount,
		policy.IsPerHectare, policy.PremiumBaseRate, policy.MaxPremiumPaymentProlong, policy.FixPayoutAmount, policy.IsPayoutPerHectare,
		policy.OverThresholdMultiplier, policy.PayoutBaseRate, policy.PayoutCap, policy.EnrollmentStartDay,
		policy.EnrollmentEndDay, policy.AutoRenewal, policy.RenewalDiscountRate, policy.BasePolicyInvalidDate,
		policy.InsuranceValidFromDay, policy.InsuranceValidToDay, policy.Status, policy.TemplateDocumentURL,
		policy.DocumentValidationStatus, policy.DocumentValidationScore, policy.DocumentTags, policy.ImportantAdditionalInformation,
		policy.CreatedAt, policy.UpdatedAt, policy.CreatedBy)
	return err
}

func (r *BasePolicyRepository) CreateBasePolicyTriggerTx(tx *sqlx.Tx, trigger *models.BasePolicyTrigger) error {
	if trigger.ID == uuid.Nil {
		trigger.ID = uuid.New()
	}

	trigger.CreatedAt = time.Now()
	trigger.UpdatedAt = time.Now()

	query := `
		INSERT INTO base_policy_trigger (
			id, base_policy_id, logical_operator, growth_stage,
			monitor_interval, monitor_frequency_unit, blackout_periods,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`
	_, err := tx.Exec(query,
		trigger.ID, trigger.BasePolicyID, trigger.LogicalOperator, trigger.GrowthStage,
		trigger.MonitorInterval, trigger.MonitorFrequencyUnit, trigger.BlackoutPeriods,
		trigger.CreatedAt, trigger.UpdatedAt)
	return err
}

func (r *BasePolicyRepository) CreateBasePolicyTriggerConditionsBatchTx(tx *sqlx.Tx, conditions []*models.BasePolicyTriggerCondition) error {
	if len(conditions) == 0 {
		return nil
	}

	for _, condition := range conditions {
		if condition.ID == uuid.Nil {
			condition.ID = uuid.New()
		}
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
	if validation.ID == uuid.Nil {
		validation.ID = uuid.New()
	}

	slog.Info("Creating base policy document validation",
		"validation_id", validation.ID,
		"base_policy_id", validation.BasePolicyID,
		"validation_status", validation.ValidationStatus)

	validation.CreatedAt = time.Now()

	// Serialize JSONB fields to []byte before database insertion
	var err error

	query := `
		INSERT INTO base_policy_document_validation (
			id, base_policy_id, validation_timestamp, validation_status, overall_score,
			total_checks, passed_checks, failed_checks, warning_count, mismatches,
			warnings, recommendations, extracted_parameters, validated_by,
			validation_notes, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err = r.db.Exec(query,
		validation.ID, validation.BasePolicyID, validation.ValidationTimestamp,
		validation.ValidationStatus, validation.OverallScore, validation.TotalChecks,
		validation.PassedChecks, validation.FailedChecks, validation.WarningCount,
		validation.Mismatches, validation.Warnings, validation.Recommendations,
		validation.ExtractedParameters, validation.ValidatedBy, validation.ValidationNotes,
		validation.CreatedAt)
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
	slog.Info("Retrieving base policy document validation by ID", "validation_id", id)

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

	slog.Info("Successfully retrieved base policy document validation",
		"validation_id", id,
		"base_policy_id", validation.BasePolicyID)
	return &validation, nil
}

func (r *BasePolicyRepository) GetBasePolicyDocumentValidationsByPolicyID(basePolicyID uuid.UUID) ([]models.BasePolicyDocumentValidation, error) {
	slog.Info("Retrieving base policy document validations by policy ID", "base_policy_id", basePolicyID)

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

	slog.Info("Successfully retrieved base policy document validations",
		"base_policy_id", basePolicyID,
		"count", len(validations))
	return validations, nil
}

func (r *BasePolicyRepository) GetLatestBasePolicyDocumentValidation(basePolicyID uuid.UUID) (*models.BasePolicyDocumentValidation, error) {
	slog.Info("Retrieving latest base policy document validation", "base_policy_id", basePolicyID)

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
			slog.Info("No document validation found for base policy", "base_policy_id", basePolicyID)
			return nil, fmt.Errorf("no document validation found for base policy")
		}
		slog.Error("Failed to get latest base policy document validation",
			"base_policy_id", basePolicyID,
			"error", err)
		return nil, fmt.Errorf("failed to get latest base policy document validation: %w", err)
	}

	slog.Info("Successfully retrieved latest base policy document validation",
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
			validation_timestamp = $1,
			validation_status = $2,
			overall_score = $3,
			total_checks = $4,
			passed_checks = $5,
			failed_checks = $6,
			warning_count = $7,
			mismatches = $8,
			warnings = $9,
			recommendations = $10,
			extracted_parameters = $11,
			validated_by = $12,
			validation_notes = $13
		WHERE id = $14`

	result, err := r.db.Exec(query,
		validation.ValidationTimestamp, validation.ValidationStatus, validation.OverallScore,
		validation.TotalChecks, validation.PassedChecks, validation.FailedChecks,
		validation.WarningCount, validation.Mismatches, validation.Warnings, validation.Recommendations,
		validation.ExtractedParameters, validation.ValidatedBy, validation.ValidationNotes,
		validation.ID)
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

func (r *BasePolicyRepository) GetTemplateDocumentURL(id uuid.UUID) (*string, error) {
	var url *string
	query := `SELECT template_document_url FROM base_policy WHERE id = $1`

	err := r.db.Get(&url, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("base policy not found")
		}
		return nil, fmt.Errorf("failed to get template document url: %w", err)
	}

	return url, nil
}

// ============================================================================
// COMPLETE POLICY DETAIL RETRIEVAL METHODS
// ============================================================================

// GetCompletePolicyByFilter retrieves complete policy details with filters
func (r *BasePolicyRepository) GetCompletePolicyByFilter(
	ctx context.Context,
	filter models.PolicyDetailFilterRequest,
) (*models.BasePolicy, []models.TriggerWithConditions, error) {
	slog.Info("Retrieving complete policy with filters",
		"id", filter.ID,
		"provider_id", filter.ProviderID,
		"crop_type", filter.CropType,
		"status", filter.Status)

	start := time.Now()

	// Step 1: Get base policy
	var policy models.BasePolicy
	query := `
		SELECT
			id, insurance_provider_id, product_name, product_code, product_description,
			crop_type, coverage_currency, coverage_duration_days, fix_premium_amount,
			is_per_hectare, premium_base_rate, max_premium_payment_prolong,
			fix_payout_amount, is_payout_per_hectare, over_threshold_multiplier,
			payout_base_rate, payout_cap, enrollment_start_day, enrollment_end_day,
			auto_renewal, renewal_discount_rate, base_policy_invalid_date,
			insurance_valid_from_day, insurance_valid_to_day, status,
			template_document_url, document_validation_status,
			document_validation_score, document_tags, important_additional_information,
			created_at, updated_at, created_by, cancel_premium_rate
		FROM base_policy
		WHERE 1=1`

	args := []any{}
	argPos := 1

	if filter.ID != nil {
		query += fmt.Sprintf(" AND id = $%d", argPos)
		args = append(args, *filter.ID)
		argPos++
	}

	if filter.ProviderID != "" {
		query += fmt.Sprintf(" AND insurance_provider_id = $%d", argPos)
		args = append(args, filter.ProviderID)
		argPos++
	}

	if filter.CropType != "" {
		query += fmt.Sprintf(" AND crop_type = $%d", argPos)
		args = append(args, filter.CropType)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	query += " LIMIT 1"

	err := r.db.Get(&policy, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Base policy not found with filters",
				"id", filter.ID,
				"provider_id", filter.ProviderID)
			return nil, nil, fmt.Errorf("base policy not found")
		}
		slog.Error("Failed to get base policy", "error", err)
		return nil, nil, fmt.Errorf("failed to get base policy: %w", err)
	}

	// Step 2: Get triggers with conditions
	triggers, err := r.GetTriggersWithConditionsByPolicyID(ctx, policy.ID)
	if err != nil {
		slog.Error("Failed to get triggers with conditions",
			"policy_id", policy.ID,
			"error", err)
		return nil, nil, fmt.Errorf("failed to get triggers: %w", err)
	}

	slog.Info("Successfully retrieved complete policy",
		"policy_id", policy.ID,
		"triggers", len(triggers),
		"duration", time.Since(start))

	return &policy, triggers, nil
}

// GetTriggersWithConditionsByPolicyID retrieves triggers with nested conditions
func (r *BasePolicyRepository) GetTriggersWithConditionsByPolicyID(
	ctx context.Context,
	policyID uuid.UUID,
) ([]models.TriggerWithConditions, error) {
	type TriggerConditionRow struct {
		// Trigger fields
		TriggerID            uuid.UUID               `db:"trigger_id"`
		BasePolicyID         uuid.UUID               `db:"base_policy_id"`
		LogicalOperator      models.LogicalOperator  `db:"logical_operator"`
		GrowthStage          *string                 `db:"growth_stage"`
		MonitorInterval      int                     `db:"monitor_interval"`
		MonitorFrequencyUnit models.MonitorFrequency `db:"monitor_frequency_unit"`
		BlackoutPeriods      utils.JSONMap           `db:"blackout_periods"`
		TriggerCreatedAt     time.Time               `db:"trigger_created_at"`
		TriggerUpdatedAt     time.Time               `db:"trigger_updated_at"`

		// Condition fields (nullable for triggers without conditions)
		ConditionID           *uuid.UUID                  `db:"condition_id"`
		ConditionTriggerID    *uuid.UUID                  `db:"condition_trigger_id"`
		DataSourceID          *uuid.UUID                  `db:"data_source_id"`
		ThresholdOperator     *models.ThresholdOperator   `db:"threshold_operator"`
		ThresholdValue        *float64                    `db:"threshold_value"`
		EarlyWarningThreshold *float64                    `db:"early_warning_threshold"`
		AggregationFunction   *models.AggregationFunction `db:"aggregation_function"`
		AggregationWindowDays *int                        `db:"aggregation_window_days"`
		ConsecutiveRequired   *bool                       `db:"consecutive_required"`
		IncludeComponent      *bool                       `db:"include_component"`
		BaselineWindowDays    *int                        `db:"baseline_window_days"`
		BaselineFunction      *models.AggregationFunction `db:"baseline_function"`
		ValidationWindowDays  *int                        `db:"validation_window_days"`
		ConditionOrder        *int                        `db:"condition_order"`
		BaseCost              *int64                      `db:"base_cost"`
		CategoryMultiplier    *float64                    `db:"category_multiplier"`
		TierMultiplier        *float64                    `db:"tier_multiplier"`
		CalculatedCost        *float64                    `db:"calculated_cost"`
		ConditionCreatedAt    *time.Time                  `db:"condition_created_at"`
	}

	query := `
		SELECT
			bt.id as trigger_id,
			bt.base_policy_id,
			bt.logical_operator,
			bt.growth_stage,
			bt.monitor_interval,
			bt.monitor_frequency_unit,
			bt.blackout_periods,
			bt.created_at as trigger_created_at,
			bt.updated_at as trigger_updated_at,

			btc.id as condition_id,
			btc.base_policy_trigger_id as condition_trigger_id,
			btc.data_source_id,
			btc.threshold_operator,
			btc.threshold_value,
			btc.early_warning_threshold,
			btc.aggregation_function,
			btc.aggregation_window_days,
			btc.consecutive_required,
			btc.include_component,
			btc.baseline_window_days,
			btc.baseline_function,
			btc.validation_window_days,
			btc.condition_order,
			btc.base_cost,
			btc.category_multiplier,
			btc.tier_multiplier,
			btc.calculated_cost,
			btc.created_at as condition_created_at
		FROM base_policy_trigger bt
		LEFT JOIN base_policy_trigger_condition btc ON bt.id = btc.base_policy_trigger_id
		WHERE bt.base_policy_id = $1
		ORDER BY bt.created_at, btc.condition_order`

	var rows []TriggerConditionRow
	err := r.db.Select(&rows, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers with conditions: %w", err)
	}

	// Group results by trigger
	triggerMap := make(map[uuid.UUID]*models.TriggerWithConditions)
	var triggerOrder []uuid.UUID

	for _, row := range rows {
		// Create trigger if not exists
		if _, exists := triggerMap[row.TriggerID]; !exists {
			triggerMap[row.TriggerID] = &models.TriggerWithConditions{
				ID:                   row.TriggerID,
				BasePolicyID:         row.BasePolicyID,
				LogicalOperator:      row.LogicalOperator,
				GrowthStage:          row.GrowthStage,
				MonitorInterval:      row.MonitorInterval,
				MonitorFrequencyUnit: row.MonitorFrequencyUnit,
				BlackoutPeriods:      row.BlackoutPeriods,
				CreatedAt:            row.TriggerCreatedAt,
				UpdatedAt:            row.TriggerUpdatedAt,
				Conditions:           []models.BasePolicyTriggerCondition{},
			}
			triggerOrder = append(triggerOrder, row.TriggerID)
		}

		// Add condition if exists
		if row.ConditionID != nil {
			condition := models.BasePolicyTriggerCondition{
				ID:                    *row.ConditionID,
				BasePolicyTriggerID:   *row.ConditionTriggerID,
				DataSourceID:          *row.DataSourceID,
				ThresholdOperator:     *row.ThresholdOperator,
				ThresholdValue:        *row.ThresholdValue,
				AggregationFunction:   *row.AggregationFunction,
				AggregationWindowDays: *row.AggregationWindowDays,
				ConsecutiveRequired:   *row.ConsecutiveRequired,
				IncludeComponent:      *row.IncludeComponent,
				ValidationWindowDays:  *row.ValidationWindowDays,
				ConditionOrder:        *row.ConditionOrder,
				BaseCost:              *row.BaseCost,
				CategoryMultiplier:    *row.CategoryMultiplier,
				TierMultiplier:        *row.TierMultiplier,
				CalculatedCost:        *row.CalculatedCost,
				CreatedAt:             *row.ConditionCreatedAt,
			}

			if row.EarlyWarningThreshold != nil {
				condition.EarlyWarningThreshold = row.EarlyWarningThreshold
			}
			if row.BaselineWindowDays != nil {
				condition.BaselineWindowDays = row.BaselineWindowDays
			}
			if row.BaselineFunction != nil {
				condition.BaselineFunction = row.BaselineFunction
			}

			triggerMap[row.TriggerID].Conditions = append(
				triggerMap[row.TriggerID].Conditions,
				condition,
			)
		}
	}

	// Convert map to slice maintaining order
	result := make([]models.TriggerWithConditions, 0, len(triggerOrder))
	for _, triggerID := range triggerOrder {
		result = append(result, *triggerMap[triggerID])
	}

	return result, nil
}

// SaveValidationToRedis saves a validation record to Redis using validation ID
func (r *BasePolicyRepository) SaveValidationToRedis(
	ctx context.Context,
	validation *models.BasePolicyDocumentValidation,
) error {
	// Build Redis key using validation ID (no index calculation needed)
	key := fmt.Sprintf("%s--BasePolicyDocumentValidation--%s",
		validation.BasePolicyID, validation.ID)

	slog.Info("Saving validation to Redis",
		"base_policy_id", validation.BasePolicyID,
		"validation_id", validation.ID,
		"key", key)

	// Serialize validation
	validationBytes, err := utils.SerializeModel(validation)
	if err != nil {
		slog.Error("Failed to serialize validation",
			"validation_id", validation.ID,
			"error", err)
		return fmt.Errorf("failed to serialize validation: %w", err)
	}

	// Save to Redis with TTL (24 hours)
	ttl := 24 * time.Hour
	if err := r.redisClient.Set(ctx, key, validationBytes, ttl).Err(); err != nil {
		slog.Error("Failed to save validation to Redis",
			"validation_id", validation.ID,
			"key", key,
			"error", err)
		return fmt.Errorf("failed to save validation to Redis: %w", err)
	}

	slog.Info("Successfully saved validation to Redis",
		"base_policy_id", validation.BasePolicyID,
		"validation_id", validation.ID,
		"key", key)

	return nil
}

// GetValidationsFromRedis retrieves all validation records for a policy
func (r *BasePolicyRepository) GetValidationsFromRedis(
	ctx context.Context,
	basePolicyID uuid.UUID,
) ([]*models.BasePolicyDocumentValidation, error) {
	pattern := fmt.Sprintf("%s--BasePolicyDocumentValidation--*", basePolicyID)

	slog.Debug("Finding validation keys",
		"base_policy_id", basePolicyID,
		"pattern", pattern)

	keys, err := r.FindKeysByPattern(ctx, pattern, "")
	if err != nil {
		slog.Error("Failed to find validation keys",
			"base_policy_id", basePolicyID,
			"pattern", pattern,
			"error", err)
		return nil, fmt.Errorf("failed to find validation keys: %w", err)
	}

	if len(keys) == 0 {
		slog.Debug("No validations found in Redis",
			"base_policy_id", basePolicyID)
		return []*models.BasePolicyDocumentValidation{}, nil
	}

	validations := make([]*models.BasePolicyDocumentValidation, 0, len(keys))

	for _, key := range keys {
		validationBytes, err := r.GetTempBasePolicyModels(ctx, key)
		if err != nil {
			slog.Warn("Failed to get validation data",
				"key", key,
				"error", err)
			continue
		}

		var validation models.BasePolicyDocumentValidation
		if err := utils.DeserializeModel(validationBytes, &validation); err != nil {
			slog.Warn("Failed to deserialize validation",
				"key", key,
				"error", err)
			continue
		}

		validations = append(validations, &validation)
	}

	// Sort by validation timestamp for chronological order
	sort.Slice(validations, func(i, j int) bool {
		return validations[i].ValidationTimestamp < validations[j].ValidationTimestamp
	})

	slog.Info("Retrieved validations from Redis",
		"base_policy_id", basePolicyID,
		"validation_count", len(validations))

	return validations, nil
}

// CreateBasePolicyDocumentValidationTx creates validation in a transaction
func (r *BasePolicyRepository) CreateBasePolicyDocumentValidationTx(
	tx *sqlx.Tx,
	validation *models.BasePolicyDocumentValidation,
) error {
	if validation.ID == uuid.Nil {
		validation.ID = uuid.New()
	}

	slog.Info("Creating validation in transaction",
		"validation_id", validation.ID,
		"base_policy_id", validation.BasePolicyID,
		"validation_status", validation.ValidationStatus)

	validation.CreatedAt = time.Now()

	// PostgreSQL pq driver automatically handles Go maps -> JSONB conversion
	// Pass raw map objects directly (do NOT serialize to []byte)
	query := `
		INSERT INTO base_policy_document_validation (
			id, base_policy_id, validation_timestamp, validation_status,
			overall_score, total_checks, passed_checks, failed_checks,
			warning_count, mismatches, warnings, recommendations,
			extracted_parameters, validated_by, validation_notes, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`

	_, err := tx.ExecContext(context.Background(),
		query,
		validation.ID,
		validation.BasePolicyID,
		validation.ValidationTimestamp,
		validation.ValidationStatus,
		validation.OverallScore,
		validation.TotalChecks,
		validation.PassedChecks,
		validation.FailedChecks,
		validation.WarningCount,
		validation.Mismatches,          // Raw map, not serialized
		validation.Warnings,            // Raw map, not serialized
		validation.Recommendations,     // Raw map, not serialized
		validation.ExtractedParameters, // Raw map, not serialized
		validation.ValidatedBy,
		validation.ValidationNotes,
		validation.CreatedAt,
	)
	if err != nil {
		slog.Error("Failed to insert validation in transaction",
			"validation_id", validation.ID,
			"base_policy_id", validation.BasePolicyID,
			"error", err)
		return fmt.Errorf("failed to insert validation: %w", err)
	}

	slog.Info("Validation created in transaction",
		"validation_id", validation.ID,
		"base_policy_id", validation.BasePolicyID)

	return nil
}
