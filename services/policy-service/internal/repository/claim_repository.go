package repository

import (
	"context"
	"fmt"
	"policy-service/internal/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ClaimRepository struct {
	db *sqlx.DB
}

func NewClaimRepository(db *sqlx.DB) *ClaimRepository {
	return &ClaimRepository{db: db}
}

// GetByID retrieves a claim by its ID
func (r *ClaimRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Claim, error) {
	var claim models.Claim
	query := `
		SELECT id, claim_number, registered_policy_id, base_policy_id, farm_id,
		       base_policy_trigger_id, trigger_timestamp, over_threshold_value,
		       calculated_fix_payout, calculated_threshold_payout, claim_amount,
		       status, auto_generated, partner_review_timestamp, partner_decision,
		       partner_notes, reviewed_by, auto_approval_deadline, auto_approved,
		       evidence_summary, created_at, updated_at
		FROM claim
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &claim, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get claim by id: %w", err)
	}

	return &claim, nil
}

// GetAll retrieves all claims with optional filters
func (r *ClaimRepository) GetAll(ctx context.Context, filters map[string]interface{}) ([]models.Claim, error) {
	var claims []models.Claim
	query := `
		SELECT id, claim_number, registered_policy_id, base_policy_id, farm_id,
		       base_policy_trigger_id, trigger_timestamp, over_threshold_value,
		       calculated_fix_payout, calculated_threshold_payout, claim_amount,
		       status, auto_generated, partner_review_timestamp, partner_decision,
		       partner_notes, reviewed_by, auto_approval_deadline, auto_approved,
		       evidence_summary, created_at, updated_at
		FROM claim
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Add filters dynamically
	if registeredPolicyID, ok := filters["registered_policy_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND registered_policy_id = $%d", argCount)
		args = append(args, registeredPolicyID)
		argCount++
	}

	if farmID, ok := filters["farm_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND farm_id = $%d", argCount)
		args = append(args, farmID)
		argCount++
	}

	if status, ok := filters["status"].(models.ClaimStatus); ok {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	err := r.db.SelectContext(ctx, &claims, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	return claims, nil
}

// GetByRegisteredPolicyID retrieves claims by registered policy ID
func (r *ClaimRepository) GetByRegisteredPolicyID(ctx context.Context, policyID uuid.UUID) ([]models.Claim, error) {
	var claims []models.Claim
	query := `
		SELECT id, claim_number, registered_policy_id, base_policy_id, farm_id,
		       base_policy_trigger_id, trigger_timestamp, over_threshold_value,
		       calculated_fix_payout, calculated_threshold_payout, claim_amount,
		       status, auto_generated, partner_review_timestamp, partner_decision,
		       partner_notes, reviewed_by, auto_approval_deadline, auto_approved,
		       evidence_summary, created_at, updated_at
		FROM claim
		WHERE registered_policy_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &claims, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims by policy id: %w", err)
	}

	return claims, nil
}

// GetByFarmID retrieves claims by farm ID
func (r *ClaimRepository) GetByFarmID(ctx context.Context, farmID uuid.UUID) ([]models.Claim, error) {
	var claims []models.Claim
	query := `
		SELECT id, claim_number, registered_policy_id, base_policy_id, farm_id,
		       base_policy_trigger_id, trigger_timestamp, over_threshold_value,
		       calculated_fix_payout, calculated_threshold_payout, claim_amount,
		       status, auto_generated, partner_review_timestamp, partner_decision,
		       partner_notes, reviewed_by, auto_approval_deadline, auto_approved,
		       evidence_summary, created_at, updated_at
		FROM claim
		WHERE farm_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &claims, query, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims by farm id: %w", err)
	}

	return claims, nil
}

// Delete removes a claim by ID
func (r *ClaimRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM claim WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete claim: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("claim not found")
	}

	return nil
}

// Exists checks if a claim exists by ID
func (r *ClaimRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM claim WHERE id = $1)`

	err := r.db.GetContext(ctx, &exists, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to check claim existence: %w", err)
	}

	return exists, nil
}
