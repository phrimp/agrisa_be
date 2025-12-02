package repository

import (
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

func (r *CancelRequestRepository) GetCancelRequestByPolicyID(id uuid.UUID) (*models.CancelRequest, error) {
	var cancelRequest models.CancelRequest
	query := `SELECT * FROM cancel_request WHERE registered_policy_id = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.Get(&cancelRequest, query, id)
	if err != nil {
		return nil, err
	}
	return &cancelRequest, nil
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
			reviewed_by, reviewed_at, review_notes
		) VALUES (
			:id, :registered_policy_id, :cancel_request_type, :reason, :evidence,
			:status, :requested_by, :requested_at, :compensate_amount,
			:reviewed_by, :reviewed_at, :review_notes
		)
	`
	_, err := r.db.NamedExec(query, cancelRequest)
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
			review_notes = :review_notes
		WHERE id = :id
	`
	_, err := r.db.NamedExec(query, cancelRequest)
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
