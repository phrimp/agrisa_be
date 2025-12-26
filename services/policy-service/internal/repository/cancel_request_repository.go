package repository

import (
	"context"
	"fmt"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	err := r.db.Select(&cancelRequest, query, id)
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
            paid, paid_at, during_notice_period, created_at, updated_at
		) VALUES (
			:id, :registered_policy_id, :cancel_request_type, :reason, :evidence,
			:status, :requested_by, :requested_at, :compensate_amount,
			:reviewed_by, :reviewed_at, :review_notes,
			:paid, :paid_at, :during_notice_period, :created_at, updated_at
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
			updated_at= :updated_at,
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
	query := `
    SELECT 
        cr.id, registered_policy_id, cancel_request_type, reason, evidence, cr.status, 
        requested_by, requested_at, reviewed_by, reviewed_at, review_notes, compensate_amount, 
        paid, paid_at, during_notice_period,
        cr.created_at, cr.updated_at
    FROM cancel_request cr 
    JOIN registered_policy rp ON cr.registered_policy_id = rp.id
    WHERE rp.farmer_id = $1 
    AND NOT ((cr.requested_by != rp.farmer_id AND cr.created_at > NOW() - INTERVAL '2 minute') OR cancel_request_type = 'transfer_contract')
    ORDER BY cr.requested_at DESC
`

	err := r.db.SelectContext(ctx, &requests, query, farmerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel requests by farmer ID: %w", err)
	}

	return requests, nil
}

func (r *CancelRequestRepository) GetLatestTransferRequestByFarmer(ctx context.Context, farmerID string, policyID uuid.UUID) ([]models.CancelRequest, error) {
	var requests []models.CancelRequest
	query := `
    SELECT 
        cr.id, registered_policy_id, cancel_request_type, reason, evidence, cr.status, 
        requested_by, requested_at, reviewed_by, reviewed_at, review_notes, compensate_amount, 
        paid, paid_at, during_notice_period,
        cr.created_at, cr.updated_at
    FROM cancel_request cr 
    JOIN registered_policy rp ON cr.registered_policy_id = rp.id
    WHERE rp.farmer_id = $1 AND cancel_request_type = 'transfer_contract' AND rp.id = $2
		ORDER BY cr.created_at DESC
		LIMIT 1
`

	err := r.db.SelectContext(ctx, &requests, query, farmerID, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel requests by farmer ID: %w", err)
	}

	return requests, nil
}

func (r *CancelRequestRepository) GetLatestTransferRequestByProvider(ctx context.Context, providerID string) ([]models.CancelRequest, error) {
	var requests []models.CancelRequest
	query := `
    SELECT 
        cr.id, registered_policy_id, cancel_request_type, reason, evidence, cr.status, 
        requested_by, requested_at, reviewed_by, reviewed_at, review_notes, compensate_amount, 
        paid, paid_at, during_notice_period,
        cr.created_at, cr.updated_at
    FROM cancel_request cr 
    JOIN registered_policy rp ON cr.registered_policy_id = rp.id
    WHERE rp.insurance_provider_id = $1 AND cancel_request_type = 'transfer_contract'
		ORDER BY cr.created_at DESC
		LIMIT 1
`

	err := r.db.SelectContext(ctx, &requests, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel requests by farmer ID: %w", err)
	}

	return requests, nil
}

func (r *CancelRequestRepository) GetAllRequestsByProviderID(ctx context.Context, providerID string) ([]models.CancelRequest, error) {
	var requests []models.CancelRequest
	query := `
    SELECT 
        cr.id, registered_policy_id, cancel_request_type, reason, evidence, cr.status, 
        requested_by, requested_at, reviewed_by, reviewed_at, review_notes, compensate_amount, 
        paid, paid_at, during_notice_period,
        cr.created_at, cr.updated_at
    FROM cancel_request cr 
    JOIN registered_policy rp ON cr.registered_policy_id = rp.id
    WHERE rp.insurance_provider_id = $1 
    AND NOT ((cr.requested_by != rp.insurance_provider_id AND cr.created_at > NOW() - INTERVAL '2 minute') OR cancel_request_type = 'transfer_contract')
    ORDER BY cr.requested_at DESC
`

	err := r.db.SelectContext(ctx, &requests, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel requests by provider ID: %w", err)
	}

	return requests, nil
}

func (r *CancelRequestRepository) GetAllByProviderIDWithStatusAndType(ctx context.Context, providerID string, status models.CancelRequestStatus, requestType models.CancelRequestType) ([]models.CancelRequest, error) {
	var requests []models.CancelRequest
	query := `
    SELECT 
        cr.id, registered_policy_id, cancel_request_type, reason, evidence, cr.status, 
        requested_by, requested_at, reviewed_by, reviewed_at, review_notes, compensate_amount, 
        paid, paid_at, during_notice_period,
        cr.created_at, cr.updated_at
    FROM cancel_request cr 
    JOIN registered_policy rp ON cr.registered_policy_id = rp.id
    WHERE rp.insurance_provider_id = $1 AND cancel_request_type = $2 AND cr.status = $3
    ORDER BY cr.requested_at DESC
`

	err := r.db.SelectContext(ctx, &requests, query, providerID, requestType, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list cancel requests by provider ID: %w", err)
	}

	return requests, nil
}

// BulkUpdateStatusWhereProviderStatusAndType updates status for multiple cancel requests with provider, status, and type conditions
// Only updates cancel requests that match insurance_provider_id AND current status AND cancel_request_type
func (r *CancelRequestRepository) BulkUpdateStatusWhereProviderStatusAndType(
	ctx context.Context,
	requestIDs []uuid.UUID,
	providerID string,
	currentStatus models.CancelRequestStatus,
	requestType models.CancelRequestType,
	newStatus models.CancelRequestStatus,
) (int64, error) {
	if len(requestIDs) == 0 {
		return 0, nil
	}
	now := time.Now()

	fmt.Printf("Starting bulk status update for cancel requests with provider, status, and type WHERE conditions\n"+
		"request_count=%d, provider_id=%s, current_status=%s, request_type=%s, new_status=%s\n",
		len(requestIDs), providerID, currentStatus, requestType, newStatus)
	start := time.Now()

	// Convert UUIDs to strings for PostgreSQL array
	requestIDStrs := make([]string, len(requestIDs))
	for i, id := range requestIDs {
		requestIDStrs[i] = id.String()
	}

	query := `
		UPDATE cancel_request cr
		SET status = $1, updated_at = $6
		FROM registered_policy rp
		WHERE cr.id = ANY($2)
		  AND cr.registered_policy_id = rp.id
		  AND rp.insurance_provider_id = $3
		  AND cr.status = $4
		  AND cr.cancel_request_type = $5`

	result, err := r.db.ExecContext(ctx, query, newStatus, pq.Array(requestIDStrs), providerID, currentStatus, requestType, now)
	if err != nil {
		fmt.Printf("Failed to execute bulk status update for cancel requests: error=%v\n", err)
		return 0, fmt.Errorf("failed to bulk update cancel request status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected != int64(len(requestIDs)) {
		fmt.Printf("Bulk status update affected different number of rows than requested\n"+
			"requested=%d, affected=%d, provider_id=%s, current_status=%s, request_type=%s\n"+
			"reason=some requests may not match provider_id, current status, or request type\n",
			len(requestIDs), rowsAffected, providerID, currentStatus, requestType)
	}

	fmt.Printf("Bulk status update for cancel requests completed: request_count=%d, rows_affected=%d, provider_id=%s, duration=%v\n",
		len(requestIDs), rowsAffected, providerID, time.Since(start))

	return rowsAffected, nil
}

// BulkUpdateStatusWhereProviderStatusAndTypeTx updates status for multiple cancel requests in transaction
func (r *CancelRequestRepository) BulkUpdateStatusWhereProviderStatusAndTypeTx(
	tx *sqlx.Tx,
	requestIDs []uuid.UUID,
	providerID string,
	currentStatus models.CancelRequestStatus,
	requestType models.CancelRequestType,
	newStatus models.CancelRequestStatus,
) (int64, error) {
	if len(requestIDs) == 0 {
		return 0, nil
	}

	fmt.Printf("Starting bulk status update for cancel requests in transaction with provider, status, and type WHERE conditions\n"+
		"request_count=%d, provider_id=%s, current_status=%s, request_type=%s, new_status=%s\n",
		len(requestIDs), providerID, currentStatus, requestType, newStatus)
	start := time.Now()

	// Convert UUIDs to strings for PostgreSQL array
	requestIDStrs := make([]string, len(requestIDs))
	for i, id := range requestIDs {
		requestIDStrs[i] = id.String()
	}
	now := time.Now()

	query := `
		UPDATE cancel_request cr
		SET status = $1, updated_at = $6
		FROM registered_policy rp
		WHERE cr.id = ANY($2)
		  AND cr.registered_policy_id = rp.id
		  AND rp.insurance_provider_id = $3
		  AND cr.status = $4
		  AND cr.cancel_request_type = $5`

	result, err := tx.Exec(query, newStatus, pq.Array(requestIDStrs), providerID, currentStatus, requestType, now)
	if err != nil {
		fmt.Printf("Failed to execute bulk status update for cancel requests in transaction: error=%v\n", err)
		return 0, fmt.Errorf("failed to bulk update cancel request status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected != int64(len(requestIDs)) {
		fmt.Printf("Bulk status update in transaction affected different number of rows than requested\n"+
			"requested=%d, affected=%d, provider_id=%s, current_status=%s, request_type=%s\n"+
			"reason=some requests may not match provider_id, current status, or request type\n",
			len(requestIDs), rowsAffected, providerID, currentStatus, requestType)
	}

	fmt.Printf("Bulk status update for cancel requests in transaction completed: request_count=%d, rows_affected=%d, provider_id=%s, duration=%v\n",
		len(requestIDs), rowsAffected, providerID, time.Since(start))

	return rowsAffected, nil
}
