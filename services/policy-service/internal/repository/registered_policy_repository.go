package repository

import (
	utils "agrisa_utils"
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

func (r *RegisteredPolicyRepository) GetInsuranceProviderIDByID(id uuid.UUID) (string, error) {
	var insuranceID string
	query := `SELECT insurance_provider_id FROM public.registered_policy where id = $1;`
	err := r.db.Get(&insuranceID, query, id)
	if err != nil {
		slog.Error("failed to get insurance provider id by policy id", "policy id", id, "error", err)
		return "", fmt.Errorf("failed to get insurance provider id by policy id: %w", err)
	}
	return insuranceID, nil
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

func (r *RegisteredPolicyRepository) GetAllPoliciesAndStatus() (map[uuid.UUID]models.PolicyStatus, error) {
	query := `
  		SELECT id, status
  		FROM public.registered_policy 
  		WHERE status NOT IN ('rejected', 'cancelled', 'expired')
  	`

	var results []struct {
		ID     uuid.UUID           `db:"id"`
		Status models.PolicyStatus `db:"status"`
	}

	err := r.db.Select(&results, query)
	if err != nil {
		return nil, fmt.Errorf("error getting policy ids and status: %w", err)
	}

	queryResult := make(map[uuid.UUID]models.PolicyStatus, len(results))
	for _, row := range results {
		queryResult[row.ID] = row.Status
	}

	return queryResult, nil
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

// ============================================================================
// RISK ANALYSIS OPERATIONS
// ============================================================================

// CreateRiskAnalysis creates a new risk analysis record for a registered policy
func (r *RegisteredPolicyRepository) CreateRiskAnalysis(analysis *models.RegisteredPolicyRiskAnalysis) error {
	if analysis.ID == uuid.Nil {
		analysis.ID = uuid.New()
	}
	analysis.CreatedAt = time.Now()

	slog.Info("Creating risk analysis record",
		"id", analysis.ID,
		"registered_policy_id", analysis.RegisteredPolicyID,
		"analysis_status", analysis.AnalysisStatus,
		"analysis_type", analysis.AnalysisType)

	query := `
		INSERT INTO registered_policy_risk_analysis (
			id, registered_policy_id, analysis_status, analysis_type,
			analysis_source, analysis_timestamp, overall_risk_score,
			overall_risk_level, identified_risks, recommendations,
			raw_output, analysis_notes, created_at
		) VALUES (
			:id, :registered_policy_id, :analysis_status, :analysis_type,
			:analysis_source, :analysis_timestamp, :overall_risk_score,
			:overall_risk_level, :identified_risks, :recommendations,
			:raw_output, :analysis_notes, :created_at
		)`

	_, err := r.db.NamedExec(query, analysis)
	if err != nil {
		slog.Error("Failed to create risk analysis record",
			"id", analysis.ID,
			"registered_policy_id", analysis.RegisteredPolicyID,
			"error", err)
		return fmt.Errorf("failed to create risk analysis: %w", err)
	}

	slog.Info("Successfully created risk analysis record", "id", analysis.ID)
	return nil
}

func (r *RegisteredPolicyRepository) CreateRiskAnalysisTX(analysis *models.RegisteredPolicyRiskAnalysis, tx *sqlx.Tx) error {
	if analysis.ID == uuid.Nil {
		analysis.ID = uuid.New()
	}
	analysis.CreatedAt = time.Now()

	slog.Info("Creating risk analysis record",
		"id", analysis.ID,
		"registered_policy_id", analysis.RegisteredPolicyID,
		"analysis_status", analysis.AnalysisStatus,
		"analysis_type", analysis.AnalysisType)

	query := `
		INSERT INTO registered_policy_risk_analysis (
			id, registered_policy_id, analysis_status, analysis_type,
			analysis_source, analysis_timestamp, overall_risk_score,
			overall_risk_level, identified_risks, recommendations,
			raw_output, analysis_notes, created_at
		) VALUES (
			:id, :registered_policy_id, :analysis_status, :analysis_type,
			:analysis_source, :analysis_timestamp, :overall_risk_score,
			:overall_risk_level, :identified_risks, :recommendations,
			:raw_output, :analysis_notes, :created_at
		)`

	_, err := tx.ExecContext(context.Background(), query, analysis)
	if err != nil {
		slog.Error("Failed to create risk analysis record",
			"id", analysis.ID,
			"registered_policy_id", analysis.RegisteredPolicyID,
			"error", err)
		return fmt.Errorf("failed to create risk analysis: %w", err)
	}

	slog.Info("Successfully created risk analysis record", "id", analysis.ID)
	return nil
}

// GetRiskAnalysesByPolicyID retrieves all risk analyses for a specific registered policy
func (r *RegisteredPolicyRepository) GetRiskAnalysesByPolicyID(policyID uuid.UUID) ([]models.RegisteredPolicyRiskAnalysis, error) {
	slog.Debug("Retrieving risk analyses by policy ID", "registered_policy_id", policyID)

	var analyses []models.RegisteredPolicyRiskAnalysis
	query := `
		SELECT * FROM registered_policy_risk_analysis
		WHERE registered_policy_id = $1
		ORDER BY analysis_timestamp DESC`

	err := r.db.Select(&analyses, query, policyID)
	if err != nil {
		slog.Error("Failed to get risk analyses by policy ID",
			"registered_policy_id", policyID,
			"error", err)
		return nil, fmt.Errorf("failed to get risk analyses: %w", err)
	}

	slog.Debug("Successfully retrieved risk analyses",
		"registered_policy_id", policyID,
		"count", len(analyses))
	return analyses, nil
}

// GetLatestRiskAnalysis retrieves the most recent risk analysis for a policy
func (r *RegisteredPolicyRepository) GetLatestRiskAnalysis(policyID uuid.UUID) (*models.RegisteredPolicyRiskAnalysis, error) {
	slog.Debug("Retrieving latest risk analysis", "registered_policy_id", policyID)

	var analysis models.RegisteredPolicyRiskAnalysis
	query := `
		SELECT * FROM registered_policy_risk_analysis
		WHERE registered_policy_id = $1
		ORDER BY analysis_timestamp DESC
		LIMIT 1`

	err := r.db.Get(&analysis, query, policyID)
	if err != nil {
		slog.Error("Failed to get latest risk analysis",
			"registered_policy_id", policyID,
			"error", err)
		return nil, fmt.Errorf("failed to get latest risk analysis: %w", err)
	}

	return &analysis, nil
}

// UpdateUnderwritingStatus updates the underwriting status of a registered policy
func (r *RegisteredPolicyRepository) UpdateUnderwritingStatus(policyID uuid.UUID, status models.UnderwritingStatus) error {
	slog.Info("Updating underwriting status",
		"registered_policy_id", policyID,
		"new_status", status)

	query := `
		UPDATE registered_policy
		SET underwriting_status = $1, updated_at = $2
		WHERE id = $3`

	result, err := r.db.Exec(query, status, time.Now(), policyID)
	if err != nil {
		slog.Error("Failed to update underwriting status",
			"registered_policy_id", policyID,
			"error", err)
		return fmt.Errorf("failed to update underwriting status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("Registered policy not found for underwriting status update",
			"registered_policy_id", policyID)
		return fmt.Errorf("registered policy not found")
	}

	slog.Info("Successfully updated underwriting status",
		"registered_policy_id", policyID,
		"new_status", status)
	return nil
}

// GetRiskAnalysisByID retrieves a specific risk analysis by ID
func (r *RegisteredPolicyRepository) GetRiskAnalysisByID(id uuid.UUID) (*models.RegisteredPolicyRiskAnalysis, error) {
	slog.Debug("Retrieving risk analysis by ID", "id", id)

	var analysis models.RegisteredPolicyRiskAnalysis
	query := `SELECT * FROM registered_policy_risk_analysis WHERE id = $1`

	err := r.db.Get(&analysis, query, id)
	if err != nil {
		slog.Error("Failed to get risk analysis by ID", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get risk analysis: %w", err)
	}

	return &analysis, nil
}

// DeleteRiskAnalysis deletes a risk analysis record
func (r *RegisteredPolicyRepository) DeleteRiskAnalysis(id uuid.UUID) error {
	slog.Info("Deleting risk analysis", "id", id)

	query := `DELETE FROM registered_policy_risk_analysis WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		slog.Error("Failed to delete risk analysis", "id", id, "error", err)
		return fmt.Errorf("failed to delete risk analysis: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("Risk analysis not found for deletion", "id", id)
		return fmt.Errorf("risk analysis not found")
	}

	slog.Info("Successfully deleted risk analysis", "id", id)
	return nil
}

// GetAllRiskAnalyses retrieves all risk analyses with pagination
func (r *RegisteredPolicyRepository) GetAllRiskAnalyses(limit, offset int) ([]models.RegisteredPolicyRiskAnalysis, error) {
	slog.Debug("Retrieving all risk analyses", "limit", limit, "offset", offset)

	var analyses []models.RegisteredPolicyRiskAnalysis
	query := `
		SELECT * FROM registered_policy_risk_analysis
		ORDER BY analysis_timestamp DESC
		LIMIT $1 OFFSET $2`

	err := r.db.Select(&analyses, query, limit, offset)
	if err != nil {
		slog.Error("Failed to get all risk analyses", "error", err)
		return nil, fmt.Errorf("failed to get risk analyses: %w", err)
	}

	slog.Debug("Successfully retrieved risk analyses", "count", len(analyses))
	return analyses, nil
}

// GetWithFilters retrieves registered policies based on filter criteria
func (r *RegisteredPolicyRepository) GetWithFilters(filter models.RegisteredPolicyFilterRequest) ([]models.RegisteredPolicy, error) {
	slog.Info("Querying registered policies with filters", "filter", filter)

	query := `SELECT * FROM registered_policy WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.PolicyID != nil {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, *filter.PolicyID)
		argIndex++
	}
	if filter.PolicyNumber != "" {
		query += fmt.Sprintf(" AND policy_number = $%d", argIndex)
		args = append(args, filter.PolicyNumber)
		argIndex++
	}
	if filter.FarmerID != "" {
		query += fmt.Sprintf(" AND farmer_id = $%d", argIndex)
		args = append(args, filter.FarmerID)
		argIndex++
	}
	if filter.BasePolicyID != nil {
		query += fmt.Sprintf(" AND base_policy_id = $%d", argIndex)
		args = append(args, *filter.BasePolicyID)
		argIndex++
	}
	if filter.FarmID != nil {
		query += fmt.Sprintf(" AND farm_id = $%d", argIndex)
		args = append(args, *filter.FarmID)
		argIndex++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}
	if filter.UnderwritingStatus != nil {
		query += fmt.Sprintf(" AND underwriting_status = $%d", argIndex)
		args = append(args, *filter.UnderwritingStatus)
		argIndex++
	}
	if filter.InsuranceProviderID != "" {
		query += fmt.Sprintf(" AND insurance_provider_id = $%d", argIndex)
		args = append(args, filter.InsuranceProviderID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	var policies []models.RegisteredPolicy
	err := r.db.Select(&policies, query, args...)
	if err != nil {
		slog.Error("Failed to query registered policies with filters", "error", err)
		return nil, fmt.Errorf("failed to get registered policies with filters: %w", err)
	}

	slog.Info("Successfully retrieved registered policies", "count", len(policies))
	return policies, nil
}

// GetByInsuranceProviderID retrieves all policies for a specific insurance provider
func (r *RegisteredPolicyRepository) GetByInsuranceProviderID(providerID string) ([]models.RegisteredPolicy, error) {
	var policies []models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE insurance_provider_id = $1 ORDER BY created_at DESC`
	err := r.db.Select(&policies, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by provider ID: %w", err)
	}
	return policies, nil
}

// GetPolicyStats retrieves aggregated statistics for policies
func (r *RegisteredPolicyRepository) GetPolicyStats(providerID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Base query with optional provider filter
	whereClause := ""
	args := []interface{}{}
	if providerID != "" {
		whereClause = " WHERE insurance_provider_id = $1"
		args = append(args, providerID)
	}

	// Total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM registered_policy` + whereClause
	err := r.db.Get(&totalCount, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total_count"] = totalCount

	// Count by status
	statusCounts := make(map[string]int)
	statusQuery := `SELECT status, COUNT(*) as count FROM registered_policy` + whereClause + ` GROUP BY status`
	rows, err := r.db.Queryx(statusQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		statusCounts[status] = count
	}
	stats["by_status"] = statusCounts

	// Count by underwriting status
	underwritingCounts := make(map[string]int)
	uwQuery := `SELECT underwriting_status, COUNT(*) as count FROM registered_policy` + whereClause + ` GROUP BY underwriting_status`
	rows2, err := r.db.Queryx(uwQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get underwriting status counts: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var status string
		var count int
		if err := rows2.Scan(&status, &count); err != nil {
			continue
		}
		underwritingCounts[status] = count
	}
	stats["by_underwriting_status"] = underwritingCounts

	// Total coverage amount
	var totalCoverage float64
	coverageQuery := `SELECT COALESCE(SUM(coverage_amount), 0) FROM registered_policy` + whereClause
	err = r.db.Get(&totalCoverage, coverageQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get total coverage: %w", err)
	}
	stats["total_coverage_amount"] = totalCoverage

	// Total premium collected
	var totalPremium float64
	premiumQuery := `SELECT COALESCE(SUM(total_farmer_premium), 0) FROM registered_policy` + whereClause
	err = r.db.Get(&totalPremium, premiumQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get total premium: %w", err)
	}
	stats["total_premium_collected"] = totalPremium

	return stats, nil
}

// UpdateStatus updates the status of a registered policy
func (r *RegisteredPolicyRepository) UpdateStatus(policyID uuid.UUID, status models.PolicyStatus) error {
	query := `UPDATE registered_policy SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.db.Exec(query, status, policyID)
	if err != nil {
		return fmt.Errorf("failed to update policy status: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("policy not found")
	}
	return nil
}

// count active registered policies by farmer_id
func (r *RegisteredPolicyRepository) CountActivePoliciesByFarmerID(farmerID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM registered_policy WHERE farmer_id = $1 AND status = 'active'`
	err := r.db.Get(&count, query, farmerID)
	if err != nil {
		return 0, fmt.Errorf("failed to count active policies: %w", err)
	}
	return count, nil
}

// ============================================================================
// UNDERWRITING OPERATIONS
// ============================================================================

// CreateUnderwriting creates a new underwriting record for a registered policy
func (r *RegisteredPolicyRepository) CreateUnderwriting(underwriting *models.RegisteredPolicyUnderwriting) error {
	if underwriting.ID == uuid.Nil {
		underwriting.ID = uuid.New()
	}
	underwriting.CreatedAt = time.Now()

	slog.Info("Creating underwriting record",
		"id", underwriting.ID,
		"registered_policy_id", underwriting.RegisteredPolicyID,
		"underwriting_status", underwriting.UnderwritingStatus)

	query := `
		INSERT INTO registered_policy_underwriting (
			id, registered_policy_id, validation_timestamp, underwriting_status,
			recommendations, reason, reason_evidence, validated_by, validation_notes, created_at
		) VALUES (
			:id, :registered_policy_id, :validation_timestamp, :underwriting_status,
			:recommendations, :reason, :reason_evidence, :validated_by, :validation_notes, :created_at
		)`

	_, err := r.db.NamedExec(query, underwriting)
	if err != nil {
		slog.Error("Failed to create underwriting record",
			"id", underwriting.ID,
			"registered_policy_id", underwriting.RegisteredPolicyID,
			"error", err)
		return fmt.Errorf("failed to create underwriting: %w", err)
	}

	slog.Info("Successfully created underwriting record", "id", underwriting.ID)
	return nil
}

// GetUnderwritingsByPolicyID retrieves all underwriting records for a specific registered policy
func (r *RegisteredPolicyRepository) GetUnderwritingsByPolicyID(policyID uuid.UUID) ([]models.RegisteredPolicyUnderwriting, error) {
	slog.Debug("Retrieving underwritings by policy ID", "registered_policy_id", policyID)

	var underwritings []models.RegisteredPolicyUnderwriting
	query := `
		SELECT * FROM registered_policy_underwriting
		WHERE registered_policy_id = $1
		ORDER BY validation_timestamp DESC`

	err := r.db.Select(&underwritings, query, policyID)
	if err != nil {
		slog.Error("Failed to get underwritings by policy ID",
			"registered_policy_id", policyID,
			"error", err)
		return nil, fmt.Errorf("failed to get underwritings: %w", err)
	}

	slog.Debug("Successfully retrieved underwritings",
		"registered_policy_id", policyID,
		"count", len(underwritings))
	return underwritings, nil
}

func (r *RegisteredPolicyRepository) GetUnderwritingsByPolicyIDAndFarmerID(policyID uuid.UUID, farmerID string) ([]models.RegisteredPolicyUnderwriting, error) {
	slog.Debug("Retrieving underwritings by policy ID and farmer ID", "registered_policy_id", policyID, "farmer_id", farmerID)

	var underwritings []models.RegisteredPolicyUnderwriting
	query := `
    SELECT rpu.* 
    FROM registered_policy_underwriting rpu
    JOIN registered_policy rp ON rpu.registered_policy_id = rp.id 
    WHERE rpu.registered_policy_id = $1 
      AND rp.farmer_id = $2 
    ORDER BY rpu.validation_timestamp DESC`

	err := r.db.Select(&underwritings, query, policyID, farmerID)
	if err != nil {
		slog.Error("Failed to get underwritings by policy ID and farmer ID",
			"registered_policy_id", policyID,
			"farmer_id", farmerID,
			"error", err)
		return nil, fmt.Errorf("failed to get underwritings: %w", err)
	}

	slog.Debug("Successfully retrieved underwritings",
		"registered_policy_id", policyID,
		"farmer_id", farmerID,
		"count", len(underwritings))
	return underwritings, nil
}

func (r *RegisteredPolicyRepository) GetAllUnderwriting() ([]models.RegisteredPolicyUnderwriting, error) {
	slog.Debug("Retrieving all underwritings")

	var underwritings []models.RegisteredPolicyUnderwriting
	query := `
		SELECT * FROM registered_policy_underwriting
		ORDER BY validation_timestamp DESC`

	err := r.db.Select(&underwritings, query)
	if err != nil {
		slog.Error("Failed to get all underwritings",
			"error", err)
		return nil, fmt.Errorf("failed to get underwritings: %w", err)
	}

	slog.Debug("Successfully retrieved underwritings",
		"count", len(underwritings))
	return underwritings, nil
}

// GetLatestUnderwriting retrieves the most recent underwriting for a policy
func (r *RegisteredPolicyRepository) GetLatestUnderwriting(policyID uuid.UUID) (*models.RegisteredPolicyUnderwriting, error) {
	slog.Debug("Retrieving latest underwriting", "registered_policy_id", policyID)

	var underwriting models.RegisteredPolicyUnderwriting
	query := `
		SELECT * FROM registered_policy_underwriting
		WHERE registered_policy_id = $1
		ORDER BY validation_timestamp DESC
		LIMIT 1`

	err := r.db.Get(&underwriting, query, policyID)
	if err != nil {
		slog.Error("Failed to get latest underwriting",
			"registered_policy_id", policyID,
			"error", err)
		return nil, fmt.Errorf("failed to get latest underwriting: %w", err)
	}

	return &underwriting, nil
}

// CreateClaim creates a new claim record
func (r *RegisteredPolicyRepository) CreateClaim(claim *models.Claim) error {
	slog.Debug("Creating claim", "claim_id", claim.ID, "policy_id", claim.RegisteredPolicyID)

	if claim.ID == uuid.Nil {
		claim.ID = uuid.New()
	}
	claim.CreatedAt = time.Now()
	claim.UpdatedAt = time.Now()

	query := `
		INSERT INTO claim (
			id, claim_number, registered_policy_id, base_policy_id, farm_id,
			base_policy_trigger_id, trigger_timestamp, over_threshold_value,
			calculated_fix_payout, calculated_threshold_payout, claim_amount,
			status, auto_generated, partner_review_timestamp, partner_decision,
			partner_notes, reviewed_by, auto_approval_deadline, auto_approved,
			evidence_summary, created_at, updated_at
		) VALUES (
			:id, :claim_number, :registered_policy_id, :base_policy_id, :farm_id,
			:base_policy_trigger_id, :trigger_timestamp, :over_threshold_value,
			:calculated_fix_payout, :calculated_threshold_payout, :claim_amount,
			:status, :auto_generated, :partner_review_timestamp, :partner_decision,
			:partner_notes, :reviewed_by, :auto_approval_deadline, :auto_approved,
			:evidence_summary, :created_at, :updated_at
		)`

	_, err := r.db.NamedExec(query, claim)
	if err != nil {
		slog.Error("Failed to create claim", "claim_id", claim.ID, "error", err)
		return fmt.Errorf("failed to create claim: %w", err)
	}

	slog.Info("Successfully created claim", "claim_id", claim.ID, "claim_number", claim.ClaimNumber)
	return nil
}

// GetClaimsByPolicyID retrieves all claims for a registered policy
func (r *RegisteredPolicyRepository) GetClaimsByPolicyID(policyID uuid.UUID) ([]models.Claim, error) {
	slog.Debug("Retrieving claims by policy ID", "registered_policy_id", policyID)

	var claims []models.Claim
	query := `
		SELECT * FROM claim
		WHERE registered_policy_id = $1
		ORDER BY created_at DESC`

	err := r.db.Select(&claims, query, policyID)
	if err != nil {
		slog.Error("Failed to get claims by policy ID",
			"registered_policy_id", policyID,
			"error", err)
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// GetClaimByID retrieves a claim by its ID
func (r *RegisteredPolicyRepository) GetClaimByID(claimID uuid.UUID) (*models.Claim, error) {
	slog.Debug("Retrieving claim by ID", "claim_id", claimID)

	var claim models.Claim
	query := `SELECT * FROM claim WHERE id = $1`

	err := r.db.Get(&claim, query, claimID)
	if err != nil {
		slog.Error("Failed to get claim by ID", "claim_id", claimID, "error", err)
		return nil, fmt.Errorf("failed to get claim: %w", err)
	}

	return &claim, nil
}

// GetRecentClaimByPolicyAndTrigger checks if a claim was recently generated for the same policy and trigger
func (r *RegisteredPolicyRepository) GetRecentClaimByPolicyAndTrigger(
	policyID uuid.UUID,
	triggerID uuid.UUID,
	withinSeconds int64,
) (*models.Claim, error) {
	slog.Debug("Checking for recent claim",
		"policy_id", policyID,
		"trigger_id", triggerID,
		"within_seconds", withinSeconds)

	var claim models.Claim
	cutoffTime := time.Now().Unix() - withinSeconds

	query := `
		SELECT * FROM claim
		WHERE registered_policy_id = $1
		AND base_policy_trigger_id = $2
		AND trigger_timestamp > $3
		ORDER BY trigger_timestamp DESC
		LIMIT 1`

	err := r.db.Get(&claim, query, policyID, triggerID, cutoffTime)
	if err != nil {
		// No recent claim found is not an error
		return nil, nil
	}

	return &claim, nil
}

func (r *RegisteredPolicyRepository) GetMonthlyDataCostByProvider(
	providerID string,
	year int,
	month int,
	direction string,
	status, underwritingStatus string,
	orderBy string,
) ([]models.BasePolicyDataCost, error) {
	var costs []models.BasePolicyDataCost

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	startTimestamp := startDate.Unix()
	endTimestamp := endDate.Unix()

	if direction != "ASC" && direction != "DESC" {
		direction = "DESC"
	}

	query := `
		SELECT
			rp.base_policy_id,
			bp.product_name,
			COUNT(rp.id) as active_policy_count,
			COALESCE(SUM(rp.total_data_cost), 0) as sum_total_data_cost
		FROM registered_policy rp 
		INNER JOIN base_policy bp ON bp.id = rp.base_policy_id 
		WHERE 
			rp.insurance_provider_id = $1
			AND rp.status = $2
			AND rp.underwriting_status = $3
			AND rp.coverage_start_date >= $4
			AND rp.coverage_start_date < $5
		GROUP BY rp.base_policy_id, bp.product_name
		ORDER BY ` + orderBy + ` ` + direction

	err := r.db.Select(&costs, query,
		providerID,
		status,
		underwritingStatus,
		startTimestamp,
		endTimestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly data cost: %w", err)
	}

	return costs, nil
}

func (r *RegisteredPolicyRepository) GetTotalFilterStatusProviders(status []string, underwritingStatus []string) (int64, error) {
	query := `
		SELECT COUNT(DISTINCT insurance_provider_id) 
		FROM registered_policy 
		WHERE status = any($1) AND underwriting_status = any($2)
	`

	var count int64
	err := r.db.GetContext(context.Background(), &count, query, status, underwritingStatus)
	if err != nil {
		slog.Error("Failed to count active approved providers", "error", err)
		return 0, fmt.Errorf("failed to count active approved providers: %w", err)
	}
	return count, nil
}

func (r *RegisteredPolicyRepository) GetTotalFilterStatusPolicies(status []string, underwritingStatus []string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM registered_policy
		WHERE status = any($1) AND underwriting_status = any($2)
	`
	var count int64
	err := r.db.GetContext(context.Background(), &count, query, status, underwritingStatus)
	if err != nil {
		slog.Error("Failed to count active approved policies", "error", err)
		return 0, fmt.Errorf("failed to count active approved policies: %w", err)
	}
	return count, nil
}

// ============================================================================
// POLICY EXPIRATION METHODS (Phase 1.4)
// ============================================================================

// GetByBasePolicyID retrieves all registered policies for a base policy
func (r *RegisteredPolicyRepository) GetByBasePolicyID(ctx context.Context, basePolicyID uuid.UUID) ([]models.RegisteredPolicy, error) {
	var policies []models.RegisteredPolicy
	query := `SELECT * FROM registered_policy WHERE base_policy_id = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &policies, query, basePolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policies by base_policy_id: %w", err)
	}

	return policies, nil
}

// ResetPaymentFields resets payment-related fields to default values
func (r *RegisteredPolicyRepository) ResetPaymentFields(ctx context.Context, policyID uuid.UUID) error {
	query := `
		UPDATE registered_policy SET
			premium_paid_by_farmer = false,
			premium_paid_at = NULL,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, policyID)
	if err != nil {
		return fmt.Errorf("failed to reset payment fields: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	slog.Info("Reset payment fields", "policy_id", policyID)
	return nil
}

// ResetPaymentFieldsBatch resets payment fields for multiple policies
func (r *RegisteredPolicyRepository) ResetPaymentFieldsBatch(ctx context.Context, policyIDs []uuid.UUID) error {
	if len(policyIDs) == 0 {
		return nil
	}

	query := `
		UPDATE registered_policy SET
			premium_paid_by_farmer = false,
			premium_paid_at = NULL,
			updated_at = NOW()
		WHERE id = ANY($1)`

	// Convert UUIDs to strings for PostgreSQL array
	policyIDStrs := make([]string, len(policyIDs))
	for i, id := range policyIDs {
		policyIDStrs[i] = id.String()
	}

	result, err := r.db.ExecContext(ctx, query, policyIDStrs)
	if err != nil {
		return fmt.Errorf("failed to batch reset payment fields: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Batch reset payment fields",
		"policy_count", len(policyIDs),
		"rows_affected", rowsAffected)

	return nil
}

// UpdateStatusBatch updates status for multiple policies atomically
func (r *RegisteredPolicyRepository) UpdateStatusBatch(ctx context.Context, policyIDs []uuid.UUID, status models.PolicyStatus) error {
	if len(policyIDs) == 0 {
		return nil
	}

	query := `
		UPDATE registered_policy SET
			status = $1,
			updated_at = NOW()
		WHERE id = ANY($2)`

	// Convert UUIDs to strings for PostgreSQL array
	policyIDStrs := make([]string, len(policyIDs))
	for i, id := range policyIDs {
		policyIDStrs[i] = id.String()
	}

	result, err := r.db.ExecContext(ctx, query, status, policyIDStrs)
	if err != nil {
		return fmt.Errorf("failed to batch update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Batch updated policy status",
		"policy_count", len(policyIDs),
		"new_status", status,
		"rows_affected", rowsAffected)

	return nil
}

// UpdateStatusAndResetPaymentBatch updates status and resets payment fields atomically
func (r *RegisteredPolicyRepository) UpdateStatusAndResetPaymentBatch(
	ctx context.Context,
	policyIDs []uuid.UUID,
	status models.PolicyStatus,
) error {
	if len(policyIDs) == 0 {
		return nil
	}

	query := `
		UPDATE registered_policy SET
			status = $1,
			premium_paid_by_farmer = false,
			premium_paid_at = NULL,
			updated_at = NOW()
		WHERE id = ANY($2)`

	// Convert UUIDs to strings for PostgreSQL array
	policyIDStrs := make([]string, len(policyIDs))
	for i, id := range policyIDs {
		policyIDStrs[i] = id.String()
	}

	result, err := r.db.ExecContext(ctx, query, status, policyIDStrs)
	if err != nil {
		return fmt.Errorf("failed to batch update status and reset payment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Batch updated policy status and reset payment fields",
		"policy_count", len(policyIDs),
		"new_status", status,
		"rows_affected", rowsAffected)

	return nil
}

func (r *RegisteredPolicyRepository) GetTotalMonthlyRevenue(year int, month int, status []string, underwritingStatus []string) (float64, error) {
	query := `
	WITH month_range AS (
		SELECT
			EXTRACT(EPOCH FROM DATE_TRUNC('month', make_date($1, $2, 1)))::INT AS month_start,
			EXTRACT(EPOCH FROM (DATE_TRUNC('month', make_date($1, $2, 1)) + INTERVAL '1 month - 1 second'))::INT AS month_end
	)
	SELECT COALESCE(SUM(rp.total_data_cost), 0) AS total_cost
	FROM registered_policy rp
	CROSS JOIN month_range mr
	WHERE rp.status = ANY($3)
		AND rp.underwriting_status = ANY($4)
		AND rp.coverage_start_date IS NOT NULL
		AND rp.coverage_start_date > 0
		AND rp.coverage_end_date IS NOT NULL
		AND rp.coverage_end_date > 0
		AND rp.coverage_start_date <= mr.month_end
		AND rp.coverage_end_date >= mr.month_start;
	`

	var totalCost float64

	err := r.db.GetContext(
		context.Background(),
		&totalCost,
		query,
		year,
		month,
		status,
		underwritingStatus,
	)
	if err != nil {
		slog.Error("Failed to calculate total monthly revenue", "error", err)
		return 0, err
	}

	return totalCost, nil
}

func (r *RegisteredPolicyRepository) GetMonthlyTotalRegisteredPolicyByStatus(
	year int,
	month int,
	statuses []string,
	underwritingStatuses []string,
) (int64, error) {
	query := `
		WITH month_range AS (
			SELECT
				EXTRACT(EPOCH FROM DATE_TRUNC('month', make_date($1, $2, 1)))::INT AS month_start,
				EXTRACT(EPOCH FROM (DATE_TRUNC('month', make_date($1, $2, 1)) + INTERVAL '1 month - 1 second'))::INT AS month_end
		)
		SELECT COALESCE(COUNT(rp.id), 0) AS total_cost
		FROM registered_policy rp
		CROSS JOIN month_range mr
		WHERE rp.status = ANY($3)
			AND rp.underwriting_status = ANY($4)
			AND rp.coverage_start_date IS NOT NULL
			AND rp.coverage_start_date > 0
			AND rp.coverage_end_date IS NOT NULL
			AND rp.coverage_end_date > 0
			AND rp.coverage_start_date <= mr.month_end
			AND rp.coverage_end_date >= mr.month_start
	`

	var result int64
	err := r.db.GetContext(
		context.Background(),
		&result,
		query,
		year,
		month,
		pq.Array(statuses),
		pq.Array(underwritingStatuses),
	)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (r *RegisteredPolicyRepository) GetTotalProvidersByMonth(year int, month int, status []string, underwritingStatus []string) (int64, error) {
	query := `
	WITH month_range AS (
		SELECT
			EXTRACT(EPOCH FROM DATE_TRUNC('month', make_date($1, $2, 1)))::INT AS month_start,
			EXTRACT(EPOCH FROM (DATE_TRUNC('month', make_date($1, $2, 1)) + INTERVAL '1 month - 1 second'))::INT AS month_end
	)
	SELECT COALESCE(count(DISTINCT insurance_provider_id), 0) AS total_active_provider
	FROM registered_policy rp
	CROSS JOIN month_range mr
	WHERE rp.status = ANY($3)
		AND rp.underwriting_status = ANY($4)
		AND rp.coverage_start_date IS NOT NULL
		AND rp.coverage_start_date > 0
		AND rp.coverage_end_date IS NOT NULL
		AND rp.coverage_end_date > 0
		AND rp.coverage_start_date <= mr.month_end
		AND rp.coverage_end_date >= mr.month_start;
	`

	var count int64
	err := r.db.GetContext(
		context.Background(),
		&count,
		query,
		year,
		month,
		status,
		underwritingStatus,
	)
	if err != nil {
		slog.Error("Failed to count total providers by month", "error", err)
		return 0, err
	}

	return count, nil
}

func (r *RegisteredPolicyRepository) GetByBasePolicyIDAndFarmID(basePolicyID, farmID uuid.UUID) (*models.RegisteredPolicy, error) {
	var result models.RegisteredPolicy
	query := `SELECT * FROM public.registered_policy where base_policy_id = $1 and farm_id = $2;`
	err := r.db.Get(&result, query, basePolicyID, farmID)
	if err != nil {
		return nil, fmt.Errorf("error getting registered_policy by base_policy_id and farm_id: %w", err)
	}
	return &result, nil
}

func (r *RegisteredPolicyRepository) GetSumOfTotalPremiumAmountByProviderWithStatusActive(providerID string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(total_farmer_premium), 0) as total_premium
		FROM public.registered_policy 
		WHERE status = 'active' 
  	AND insurance_provider_id = $1 ;
	`
	var totalAmount int64
	err := r.db.GetContext(context.Background(), &totalAmount, query, providerID)
	if err != nil {
		slog.Error("falied to count total premium amount by status active", "provider", providerID, "error", err)
		return 0, err
	}
	return totalAmount, nil
}

func (r *RegisteredPolicyRepository) UpdateStatusByProviderAndStatus(providerID string, updatedStatus, byStatus models.PolicyStatus) error {
	query := `
		UPDATE registered_policy
		SET status = $1 
		WHERE insurance_provider_id = $2 
		AND status = $3;
	`

	_, err := r.db.Exec(query, updatedStatus, providerID, byStatus)
	if err != nil {
		return fmt.Errorf("failed to update registered policy: %w", err)
	}

	return nil
}
