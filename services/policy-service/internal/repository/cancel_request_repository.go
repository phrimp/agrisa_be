package repository

import (
	"context"
	"fmt"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CancelRequestRepository struct {
	db *sqlx.DB
}

func NewCancelRequestRepository(db *sqlx.DB) *CancelRequestRepository {
	return &CancelRequestRepository{db: db}
}

func (r *CancelRequestRepository) GetAllRequestRepository() ([]models.CancelRequest, error) {
	var cancelRequests []models.CancelRequest
	query := `SELECT * FROM cancel_request ORDER BY created_at DESC`
	err := r.db.Select(&cancelRequests, query)
	if err != nil {
		return nil, err
	}
	return cancelRequests, nil
}

func (r *CancelRequestRepository) GetCancelRequestByID(id uuid.UUID) (*models.CancelRequest, error) {
	var cancelRequest models.CancelRequest
	query := `SELECT * FROM cancel_request WHERE id = $1`
	err := r.db.Get(&cancelRequest, query, id)
	if err != nil {
		return nil, err
	}
	return &cancelRequest, nil
}

func (r *CancelRequestRepository) GetCancelRequestByPolicyID(id uuid.UUID) ([]models.CancelRequest, error) {
	var cancelRequest []models.CancelRequest
	query := `SELECT * FROM cancel_request WHERE registered_policy_id = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.Get(&cancelRequest, query, id)
	if err != nil {
		return nil, err
	}
	return cancelRequest, nil
}

func (r *CancelRequestRepository) GetCancelRequestByCancelRequestType(requestType models.CancelRequestType) (*models.CancelRequest, error) {
	var cancelRequest models.CancelRequest
	query := `SELECT * FROM cancel_request WHERE cancel_request_type = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.Get(&cancelRequest, query, requestType)
	if err != nil {
		return nil, err
	}
	return &cancelRequest, nil
}

func (r *CancelRequestRepository) CreateNewCancelRequest(cancelRequest models.CancelRequest) error {
	if cancelRequest.ID == uuid.Nil {
		cancelRequest.ID = uuid.New()
	}

	cancelRequest.CreatedAt = time.Now()

	query := `
		INSERT INTO cancel_request (
			id, registered_policy_id, cancel_request_type, reason, evidence,
			status, requested_by, requested_at, compensate_amount,
			reviewed_by, reviewed_at, review_notes,
            paid, paid_at, during_notice_period
		) VALUES (
			:id, :registered_policy_id, :cancel_request_type, :reason, :evidence,
			:status, :requested_by, :requested_at, :compensate_amount,
			:reviewed_by, :reviewed_at, :review_notes,
            :paid, :paid_at, :during_notice_period
		)
	`
	_, err := r.db.NamedExec(query, cancelRequest)
	if err != nil {
		return err
	}
	return nil
}

func (r *CancelRequestRepository) CreateNewCancelRequestTx(tx *sqlx.Tx, cancelRequest models.CancelRequest) error {
	if cancelRequest.ID == uuid.Nil {
		cancelRequest.ID = uuid.New()
	}

	cancelRequest.CreatedAt = time.Now()
	cancelRequest.UpdatedAt = time.Now()

	query := `
		INSERT INTO cancel_request (
			id, registered_policy_id, cancel_request_type, reason, evidence,
			status, requested_by, requested_at, compensate_amount,
			reviewed_by, reviewed_at, review_notes,
            paid, paid_at, during_notice_period
		) VALUES (
			:id, :registered_policy_id, :cancel_request_type, :reason, :evidence,
			:status, :requested_by, :requested_at, :compensate_amount,
			:reviewed_by, :reviewed_at, :review_notes,
            :paid, :paid_at, :during_notice_period
		)
	`
	_, err := tx.NamedExec(query, cancelRequest)
	if err != nil {
		return err
	}
	return nil
}

func (r *CancelRequestRepository) UpdateCancelRequest(cancelRequest models.CancelRequest) error {
	query := `
		UPDATE cancel_request SET
			registered_policy_id = :registered_policy_id,
			cancel_request_type = :cancel_request_type,
			reason = :reason,
			evidence = :evidence,
			status = :status,
			requested_by = :requested_by,
			requested_at = :requested_at,
			compensate_amount = :compensate_amount,
			reviewed_by = :reviewed_by,
			reviewed_at = :reviewed_at,
			review_notes = :review_notes,
            paid = :paid,
            paid_at = :paid_at,
            during_notice_period = :during_notice_period
		WHERE id = :id
	`
	_, err := r.db.NamedExec(query, cancelRequest)
	if err != nil {
		return err
	}
	return nil
}

func (r *CancelRequestRepository) UpdateCancelRequestTx(tx *sqlx.Tx, cancelRequest models.CancelRequest) error {
	query := `
		UPDATE cancel_request SET
			registered_policy_id = :registered_policy_id,
			cancel_request_type = :cancel_request_type,
			reason = :reason,
			evidence = :evidence,
			status = :status,
			requested_by = :requested_by,
			requested_at = :requested_at,
			compensate_amount = :compensate_amount,
			reviewed_by = :reviewed_by,
			reviewed_at = :reviewed_at,
			review_notes = :review_notes,
            paid = :paid,
            paid_at = :paid_at,
            during_notice_period = :during_notice_period
		WHERE id = :id
	`
	_, err := tx.NamedExec(query, cancelRequest)
	if err != nil {
		return err
	}
	return nil
}

func (r *CancelRequestRepository) DeleteCancelRequestByID(id uuid.UUID) error {
	query := `DELETE FROM cancel_request WHERE id = $1`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *CancelRequestRepository) GetAllRequestsByFarmerID(ctx context.Context, farmerID string) ([]models.CancelRequest, error) {
	var requests []models.CancelRequest
	query := `SELECT cr.id, registered_policy_id, cancel_request_type, reason, evidence, cr.status, requested_by, requested_at, reviewed_by, reviewed_at, review_notes, compensate_amount, 
	paid, paid_at, during_notice_period, -- Added missing fields
	cr.created_at, cr.updated_at
FROM public.cancel_request cr
JOIN
    registered_policy rp ON cr.registered_policy_id = rp.id
WHERE
    rp.farmer_id = $1
ORDER BY
    cr.requested_at DESC;`

	err := r.db.SelectContext(ctx, &requests, query, farmerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel requests by farmer ID: %w", err)
	}

	return requests, nil
}

func (r *CancelRequestRepository) GetAllRequestsByProviderID(ctx context.Context, providerID string) ([]models.CancelRequest, error) {
	var requests []models.CancelRequest
	query := `SELECT cr.id, registered_policy_id, cancel_request_type, reason, evidence, cr.status, requested_by, requested_at, reviewed_by, reviewed_at, review_notes, compensate_amount, 
	paid, paid_at, during_notice_period, -- Added missing fields
	cr.created_at, cr.updated_at
FROM public.cancel_request cr
JOIN
    registered_policy rp ON cr.registered_policy_id = rp.id
WHERE
    rp.insurance_provider_id = $1
ORDER BY
    cr.requested_at DESC;`

	err := r.db.SelectContext(ctx, &requests, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel requests by provider ID: %w", err)
	}

	return requests, nil
}
