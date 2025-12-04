package repository

import (
	"context"
	"fmt"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PayoutRepository struct {
	db *sqlx.DB
}

func NewPayoutRepository(db *sqlx.DB) *PayoutRepository {
	return &PayoutRepository{db: db}
}

func (p *PayoutRepository) GetByID(ctx context.Context, payoutID uuid.UUID) (*models.Payout, error) {
	var payout models.Payout
	query := `
		SELECT id, claim_id, registered_policy_id, farm_id, farmer_id, payout_amount, currency, status, initiated_at,
		completed_at, farmer_confirmed, farmer_confirmation_timestamp, farmer_rating, farmer_feedback, created_at
		FROM payout
		WHERE id = $1
	`
	err := p.db.GetContext(ctx, &payout, query, payoutID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payout by id: %w", err)
	}

	return &payout, nil
}

func (p *PayoutRepository) GetByClaimID(ctx context.Context, claimID uuid.UUID) (*models.Payout, error) {
	var payout models.Payout
	query := `
		SELECT id, claim_id, registered_policy_id, farm_id, farmer_id, payout_amount, currency, status, initiated_at,
		completed_at, farmer_confirmed, farmer_confirmation_timestamp, farmer_rating, farmer_feedback, created_at
		FROM payout
		WHERE claim_id = $1
	`
	err := p.db.GetContext(ctx, &payout, query, claimID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payout by claim id: %w", err)
	}

	return &payout, nil
}

func (p *PayoutRepository) GetByRegisteredPolicyID(ctx context.Context, policyID uuid.UUID) ([]models.Payout, error) {
	var payouts []models.Payout
	query := `
		SELECT id, claim_id, registered_policy_id, farm_id, farmer_id, payout_amount, currency, status, initiated_at,
		completed_at, farmer_confirmed, farmer_confirmation_timestamp, farmer_rating, farmer_feedback, created_at
		FROM payout
		WHERE registered_policy_id = $1
		ORDER BY created_at DESC
	`
	err := p.db.SelectContext(ctx, &payouts, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts by registered policy id: %w", err)
	}

	return payouts, nil
}

func (p *PayoutRepository) GetByFarmID(ctx context.Context, farmID uuid.UUID) ([]models.Payout, error) {
	var payouts []models.Payout
	query := `
		SELECT id, claim_id, registered_policy_id, farm_id, farmer_id, payout_amount, currency, status, initiated_at,
		completed_at, farmer_confirmed, farmer_confirmation_timestamp, farmer_rating, farmer_feedback, created_at
		FROM payout
		WHERE farm_id = $1
		ORDER BY created_at DESC
	`
	err := p.db.SelectContext(ctx, &payouts, query, farmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts by farm id: %w", err)
	}

	return payouts, nil
}

func (p *PayoutRepository) GetByFarmerID(ctx context.Context, farmerID string) ([]models.Payout, error) {
	var payouts []models.Payout
	query := `
		SELECT id, claim_id, registered_policy_id, farm_id, farmer_id, payout_amount, currency, status, initiated_at,
		completed_at, farmer_confirmed, farmer_confirmation_timestamp, farmer_rating, farmer_feedback, created_at
		FROM payout
		WHERE farmer_id = $1
		ORDER BY created_at DESC
	`
	err := p.db.SelectContext(ctx, &payouts, query, farmerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payouts by farmer id: %w", err)
	}

	return payouts, nil
}

func (r *PayoutRepository) UpdatePayoutTx(tx *sqlx.Tx, payout *models.Payout) error {
	query := `
		UPDATE payout SET
			claim_id = :claim_id, 
			registered_policy_id = :registered_policy_id, 
			farm_id = :farm_id, 
			farmer_id = :farmer_id, 
			payout_amount = :payout_amount, 
			currency = :currency, 
			status = :status, 
			initiated_at = :initiated_at, 
			completed_at = :completed_at, 
			farmer_confirmed = :farmer_confirmed, 
			farmer_confirmation_timestamp = :farmer_confirmation_timestamp, 
			farmer_rating = :farmer_rating, 
			farmer_feedback = :farmer_feedback
		WHERE id = :id`

	_, err := tx.NamedExec(query, payout)
	if err != nil {
		return fmt.Errorf("failed to update payout in transaction: %w", err)
	}

	return nil
}

func (r *PayoutRepository) UpdatePayout(payout *models.Payout) error {
	query := `
		UPDATE payout SET
			claim_id = :claim_id, 
			registered_policy_id = :registered_policy_id, 
			farm_id = :farm_id, 
			farmer_id = :farmer_id, 
			payout_amount = :payout_amount, 
			currency = :currency, 
			status = :status, 
			initiated_at = :initiated_at, 
			completed_at = :completed_at, 
			farmer_confirmed = :farmer_confirmed, 
			farmer_confirmation_timestamp = :farmer_confirmation_timestamp, 
			farmer_rating = :farmer_rating, 
			farmer_feedback = :farmer_feedback
		WHERE id = :id`

	_, err := r.db.NamedExec(query, payout)
	if err != nil {
		return fmt.Errorf("failed to update payout in transaction: %w", err)
	}

	return nil
}

func (r *PayoutRepository) CreateTx(tx *sqlx.Tx, payout *models.Payout) error {
	if payout.ID == uuid.Nil {
		payout.ID = uuid.New()
	}
	if payout.CreatedAt.IsZero() {
		payout.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO payout (
			id, claim_id, registered_policy_id, farm_id, farmer_id,
			payout_amount, currency, status, initiated_at, completed_at,
			farmer_confirmed, farmer_confirmation_timestamp, farmer_rating, farmer_feedback,
			created_at
		) VALUES (
			:id, :claim_id, :registered_policy_id, :farm_id, :farmer_id,
			:payout_amount, :currency, :status, :initiated_at, :completed_at,
			:farmer_confirmed, :farmer_confirmation_timestamp, :farmer_rating, :farmer_feedback,
			:created_at
		)`

	_, err := tx.NamedExec(query, payout)
	if err != nil {
		return fmt.Errorf("failed to create payout in transaction: %w", err)
	}

	return nil
}
