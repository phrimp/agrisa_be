package repository

import (
	utils "agrisa_utils"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RegisteredPolicyRepository struct {
	db *sqlx.DB
}

func NewRegisteredPolicyRepository(db *sqlx.DB) *RegisteredPolicyRepository {
	return &RegisteredPolicyRepository{db: db}
}

func (r *RegisteredPolicyRepository) Create(policy *models.RegisteredPolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	query := `
		INSERT INTO registered_policy (
			id, policy_number, base_policy_id, insurance_provider_id, farm_id, farmer_id,
			coverage_amount, coverage_start_date, coverage_end_date, planting_date,
			area_multiplier, total_farmer_premium, premium_paid_by_farmer, premium_paid_at,
			data_complexity_score, monthly_data_cost, total_data_cost,
			status, underwriting_status, signed_policy_document_url,
			created_at, updated_at, registered_by
		) VALUES (
			:id, :policy_number, :base_policy_id, :insurance_provider_id, :farm_id, :farmer_id,
			:coverage_amount, :coverage_start_date, :coverage_end_date, :planting_date,
			:area_multiplier, :total_farmer_premium, :premium_paid_by_farmer, :premium_paid_at,
			:data_complexity_score, :monthly_data_cost, :total_data_cost,
			:status, :underwriting_status, :signed_policy_document_url,
			:created_at, :updated_at, :registered_by
		)`

	_, err := r.db.NamedExec(query, policy)
	if err != nil {
		return fmt.Errorf("failed to create registered policy: %w", err)
	}

	return nil
}

func (r *RegisteredPolicyRepository) GetByID(id uuid.UUID) (*models.RegisteredPolicy, error) {
	var policy models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE id = $1`

	err := r.db.Get(&policy, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policy: %w", err)
	}

	return &policy, nil
}

func (r *RegisteredPolicyRepository) GetByPolicyNumber(policyNumber string) (*models.RegisteredPolicy, error) {
	var policy models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE policy_number = $1`

	err := r.db.Get(&policy, query, policyNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policy: %w", err)
	}

	return &policy, nil
}

func (r *RegisteredPolicyRepository) GetAll() ([]models.RegisteredPolicy, error) {
	var policies []models.RegisteredPolicy
	query := `SELECT * FROM registered_policy ORDER BY created_at DESC`

	err := r.db.Select(&policies, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policies: %w", err)
	}

	return policies, nil
}

func (r *RegisteredPolicyRepository) GetByFarmerID(farmerID string) ([]models.RegisteredPolicy, error) {
	var policies []models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE farmer_id = $1 ORDER BY created_at DESC`

	err := r.db.Select(&policies, query, farmerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policies by farmer: %w", err)
	}

	return policies, nil
}

func (r *RegisteredPolicyRepository) GetByFarmID(farmID uuid.UUID) ([]models.RegisteredPolicy, error) {
	var policies []models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE farm_id = $1 ORDER BY created_at DESC`

	err := r.db.Select(&policies, query, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policies by farm: %w", err)
	}

	return policies, nil
}

func (r *RegisteredPolicyRepository) Update(policy *models.RegisteredPolicy) error {
	policy.UpdatedAt = time.Now()

	query := `
		UPDATE registered_policy SET
			policy_number = :policy_number, base_policy_id = :base_policy_id,
			insurance_provider_id = :insurance_provider_id, farm_id = :farm_id, farmer_id = :farmer_id,
			coverage_amount = :coverage_amount, coverage_start_date = :coverage_start_date,
			coverage_end_date = :coverage_end_date, planting_date = :planting_date,
			area_multiplier = :area_multiplier, total_farmer_premium = :total_farmer_premium,
			premium_paid_by_farmer = :premium_paid_by_farmer, premium_paid_at = :premium_paid_at,
			data_complexity_score = :data_complexity_score, monthly_data_cost = :monthly_data_cost,
			total_data_cost = :total_data_cost, status = :status, underwriting_status = :underwriting_status,
			signed_policy_document_url = :signed_policy_document_url, updated_at = :updated_at,
			registered_by = :registered_by
		WHERE id = :id`

	_, err := r.db.NamedExec(query, policy)
	if err != nil {
		return fmt.Errorf("failed to update registered policy: %w", err)
	}

	return nil
}

func (r *RegisteredPolicyRepository) Delete(id uuid.UUID) error {
	query := `DELETE FROM registered_policy WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete registered policy: %w", err)
	}

	return nil
}

// GetByIDWithFarm retrieves a registered policy with farm details using FastAssembleWithPrefix
func (r *RegisteredPolicyRepository) GetByIDWithFarm(id uuid.UUID) (*models.RegisteredPolicyWFarm, error) {
	query := `
		SELECT
			rp.id, rp.policy_number, rp.base_policy_id, rp.insurance_provider_id,
			rp.farmer_id, rp.coverage_amount, rp.coverage_start_date, rp.coverage_end_date,
			rp.planting_date, rp.area_multiplier, rp.total_farmer_premium,
			rp.premium_paid_by_farmer, rp.premium_paid_at, rp.data_complexity_score,
			rp.monthly_data_cost, rp.total_data_cost, rp.status, rp.underwriting_status,
			rp.signed_policy_document_url, rp.created_at, rp.updated_at, rp.registered_by,
			f.id as farm_id,
			f.owner_id as farm_owner_id,
			f.farm_name as farm_farm_name,
			f.farm_code as farm_farm_code,
			f.boundary as farm_boundary,
			f.center_location as farm_center_location,
			f.area_sqm as farm_area_sqm,
			f.province as farm_province,
			f.district as farm_district,
			f.commune as farm_commune,
			f.address as farm_address,
			f.crop_type as farm_crop_type,
			f.planting_date as farm_planting_date,
			f.expected_harvest_date as farm_expected_harvest_date,
			f.crop_type_verified as farm_crop_type_verified,
			f.crop_type_verified_at as farm_crop_type_verified_at,
			f.crop_type_verified_by as farm_crop_type_verified_by,
			f.crop_type_confidence as farm_crop_type_confidence,
			f.land_certificate_number as farm_land_certificate_number,
			f.land_certificate_url as farm_land_certificate_url,
			f.land_ownership_verified as farm_land_ownership_verified,
			f.land_ownership_verified_at as farm_land_ownership_verified_at,
			f.has_irrigation as farm_has_irrigation,
			f.irrigation_type as farm_irrigation_type,
			f.soil_type as farm_soil_type,
			f.status as farm_status,
			f.created_at as farm_created_at,
			f.updated_at as farm_updated_at
		FROM registered_policy rp
		JOIN farm f ON rp.farm_id = f.id
		WHERE rp.id = $1`

	var queryResult map[string]any
	err := r.db.Get(&queryResult, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policy with farm: %w", err)
	}

	var result models.RegisteredPolicyWFarm
	err = utils.FastAssembleWithPrefix(&result, queryResult, "farm_")
	if err != nil {
		return nil, fmt.Errorf("failed to assemble registered policy with farm: %w", err)
	}

	return &result, nil
}

// GetAllWithFarm retrieves all registered policies with farm details
func (r *RegisteredPolicyRepository) GetAllWithFarm() ([]models.RegisteredPolicyWFarm, error) {
	query := `
		SELECT
			rp.id, rp.policy_number, rp.base_policy_id, rp.insurance_provider_id,
			rp.farmer_id, rp.coverage_amount, rp.coverage_start_date, rp.coverage_end_date,
			rp.planting_date, rp.area_multiplier, rp.total_farmer_premium,
			rp.premium_paid_by_farmer, rp.premium_paid_at, rp.data_complexity_score,
			rp.monthly_data_cost, rp.total_data_cost, rp.status, rp.underwriting_status,
			rp.signed_policy_document_url, rp.created_at, rp.updated_at, rp.registered_by,
			f.id as farm_id,
			f.owner_id as farm_owner_id,
			f.farm_name as farm_farm_name,
			f.farm_code as farm_farm_code,
			f.boundary as farm_boundary,
			f.center_location as farm_center_location,
			f.area_sqm as farm_area_sqm,
			f.province as farm_province,
			f.district as farm_district,
			f.commune as farm_commune,
			f.address as farm_address,
			f.crop_type as farm_crop_type,
			f.planting_date as farm_planting_date,
			f.expected_harvest_date as farm_expected_harvest_date,
			f.crop_type_verified as farm_crop_type_verified,
			f.crop_type_verified_at as farm_crop_type_verified_at,
			f.crop_type_verified_by as farm_crop_type_verified_by,
			f.crop_type_confidence as farm_crop_type_confidence,
			f.land_certificate_number as farm_land_certificate_number,
			f.land_certificate_url as farm_land_certificate_url,
			f.land_ownership_verified as farm_land_ownership_verified,
			f.land_ownership_verified_at as farm_land_ownership_verified_at,
			f.has_irrigation as farm_has_irrigation,
			f.irrigation_type as farm_irrigation_type,
			f.soil_type as farm_soil_type,
			f.status as farm_status,
			f.created_at as farm_created_at,
			f.updated_at as farm_updated_at
		FROM registered_policy rp
		JOIN farm f ON rp.farm_id = f.id
		ORDER BY rp.created_at DESC`

	var queryResults []map[string]any
	err := r.db.Select(&queryResults, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policies with farm: %w", err)
	}

	results := make([]models.RegisteredPolicyWFarm, len(queryResults))
	for i, queryResult := range queryResults {
		err = utils.FastAssembleWithPrefix(&results[i], queryResult, "farm_")
		if err != nil {
			return nil, fmt.Errorf("failed to assemble registered policy %d with farm: %w", i, err)
		}
	}

	return results, nil
}

// GetByFarmerIDWithFarm retrieves registered policies by farmer ID with farm details
func (r *RegisteredPolicyRepository) GetByFarmerIDWithFarm(farmerID string) ([]models.RegisteredPolicyWFarm, error) {
	query := `
		SELECT
			rp.id, rp.policy_number, rp.base_policy_id, rp.insurance_provider_id,
			rp.farmer_id, rp.coverage_amount, rp.coverage_start_date, rp.coverage_end_date,
			rp.planting_date, rp.area_multiplier, rp.total_farmer_premium,
			rp.premium_paid_by_farmer, rp.premium_paid_at, rp.data_complexity_score,
			rp.monthly_data_cost, rp.total_data_cost, rp.status, rp.underwriting_status,
			rp.signed_policy_document_url, rp.created_at, rp.updated_at, rp.registered_by,
			f.id as farm_id,
			f.owner_id as farm_owner_id,
			f.farm_name as farm_farm_name,
			f.farm_code as farm_farm_code,
			f.boundary as farm_boundary,
			f.center_location as farm_center_location,
			f.area_sqm as farm_area_sqm,
			f.province as farm_province,
			f.district as farm_district,
			f.commune as farm_commune,
			f.address as farm_address,
			f.crop_type as farm_crop_type,
			f.planting_date as farm_planting_date,
			f.expected_harvest_date as farm_expected_harvest_date,
			f.crop_type_verified as farm_crop_type_verified,
			f.crop_type_verified_at as farm_crop_type_verified_at,
			f.crop_type_verified_by as farm_crop_type_verified_by,
			f.crop_type_confidence as farm_crop_type_confidence,
			f.land_certificate_number as farm_land_certificate_number,
			f.land_certificate_url as farm_land_certificate_url,
			f.land_ownership_verified as farm_land_ownership_verified,
			f.land_ownership_verified_at as farm_land_ownership_verified_at,
			f.has_irrigation as farm_has_irrigation,
			f.irrigation_type as farm_irrigation_type,
			f.soil_type as farm_soil_type,
			f.status as farm_status,
			f.created_at as farm_created_at,
			f.updated_at as farm_updated_at
		FROM registered_policy rp
		JOIN farm f ON rp.farm_id = f.id
		WHERE rp.farmer_id = $1
		ORDER BY rp.created_at DESC`

	var queryResults []map[string]any
	err := r.db.Select(&queryResults, query, farmerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policies with farm by farmer: %w", err)
	}

	results := make([]models.RegisteredPolicyWFarm, len(queryResults))
	for i, queryResult := range queryResults {
		err = utils.FastAssembleWithPrefix(&results[i], queryResult, "farm_")
		if err != nil {
			return nil, fmt.Errorf("failed to assemble registered policy %d with farm: %w", i, err)
		}
	}

	return results, nil
}

// ============================================================================
// TRANSACTION SUPPORT
// ============================================================================

// BeginTransaction starts a new database transaction
func (r *RegisteredPolicyRepository) BeginTransaction() (*sqlx.Tx, error) {
	slog.Info("Beginning database transaction for registered policy")
	tx, err := r.db.Beginx()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// CreateTx creates a registered policy within a transaction
func (r *RegisteredPolicyRepository) CreateTx(tx *sqlx.Tx, policy *models.RegisteredPolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	query := `
		INSERT INTO registered_policy (
			id, policy_number, base_policy_id, insurance_provider_id, farm_id, farmer_id,
			coverage_amount, coverage_start_date, coverage_end_date, planting_date,
			area_multiplier, total_farmer_premium, premium_paid_by_farmer, premium_paid_at,
			data_complexity_score, monthly_data_cost, total_data_cost,
			status, underwriting_status, signed_policy_document_url,
			created_at, updated_at, registered_by
		) VALUES (
			:id, :policy_number, :base_policy_id, :insurance_provider_id, :farm_id, :farmer_id,
			:coverage_amount, :coverage_start_date, :coverage_end_date, :planting_date,
			:area_multiplier, :total_farmer_premium, :premium_paid_by_farmer, :premium_paid_at,
			:data_complexity_score, :monthly_data_cost, :total_data_cost,
			:status, :underwriting_status, :signed_policy_document_url,
			:created_at, :updated_at, :registered_by
		)`

	_, err := tx.NamedExec(query, policy)
	if err != nil {
		return fmt.Errorf("failed to create registered policy in transaction: %w", err)
	}

	return nil
}

// UpdateTx updates a registered policy within a transaction
func (r *RegisteredPolicyRepository) UpdateTx(tx *sqlx.Tx, policy *models.RegisteredPolicy) error {
	policy.UpdatedAt = time.Now()

	query := `
		UPDATE registered_policy SET
			policy_number = :policy_number, base_policy_id = :base_policy_id,
			insurance_provider_id = :insurance_provider_id, farm_id = :farm_id, farmer_id = :farmer_id,
			coverage_amount = :coverage_amount, coverage_start_date = :coverage_start_date,
			coverage_end_date = :coverage_end_date, planting_date = :planting_date,
			area_multiplier = :area_multiplier, total_farmer_premium = :total_farmer_premium,
			premium_paid_by_farmer = :premium_paid_by_farmer, premium_paid_at = :premium_paid_at,
			data_complexity_score = :data_complexity_score, monthly_data_cost = :monthly_data_cost,
			total_data_cost = :total_data_cost, status = :status, underwriting_status = :underwriting_status,
			signed_policy_document_url = :signed_policy_document_url, updated_at = :updated_at,
			registered_by = :registered_by
		WHERE id = :id`

	_, err := tx.NamedExec(query, policy)
	if err != nil {
		return fmt.Errorf("failed to update registered policy in transaction: %w", err)
	}

	return nil
}

// DeleteTx deletes a registered policy within a transaction
func (r *RegisteredPolicyRepository) DeleteTx(tx *sqlx.Tx, id uuid.UUID) error {
	query := `DELETE FROM registered_policy WHERE id = $1`

	_, err := tx.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete registered policy in transaction: %w", err)
	}

	return nil
}

// GetByIDTx retrieves a registered policy by ID within a transaction
func (r *RegisteredPolicyRepository) GetByIDTx(tx *sqlx.Tx, id uuid.UUID) (*models.RegisteredPolicy, error) {
	var policy models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE id = $1`

	err := tx.Get(&policy, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policy in transaction: %w", err)
	}

	return &policy, nil
}

// GetByFarmIDTx retrieves registered policies by farm ID within a transaction
func (r *RegisteredPolicyRepository) GetByFarmIDTx(tx *sqlx.Tx, farmID uuid.UUID) ([]models.RegisteredPolicy, error) {
	var policies []models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE farm_id = $1 ORDER BY created_at DESC`

	err := tx.Select(&policies, query, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policies by farm in transaction: %w", err)
	}

	return policies, nil
}
