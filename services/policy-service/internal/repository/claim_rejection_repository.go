package repository

import (
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ClaimRejectionRepository struct {
	db *sqlx.DB
}

func NewClaimRejectionRepository(db *sqlx.DB) *ClaimRejectionRepository {
	return &ClaimRejectionRepository{db: db}
}

func (cj *ClaimRejectionRepository) GetAllClaimRejection() ([]models.ClaimRejection, error) {
	var claimRejections []models.ClaimRejection
	query := `SELECT * FROM claim_rejection ORDER BY created_at DESC`
	err := cj.db.Select(&claimRejections, query)
	if err != nil {
		return nil, err
	}
	return claimRejections, nil
}

func (cj *ClaimRejectionRepository) GetClaimrejectionByID(id uuid.UUID) (*models.ClaimRejection, error) {
	var claimRejection models.ClaimRejection

	query := `SELECT * FROM claim_rejection WHERE id = $1 ORDER BY created_at DESC`
	err := cj.db.Get(&claimRejection, query, id)
	if err != nil {
		return nil, err
	}
	return &claimRejection, nil
}

func (cj *ClaimRejectionRepository) GetClaimrejectionByClaimID(claimID uuid.UUID) (*models.ClaimRejection, error) {
	var claimRejection models.ClaimRejection

	query := `SELECT * FROM claim_rejection WHERE claim_id = $1 ORDER BY created_at DESC`
	err := cj.db.Get(&claimRejection, query, claimID)
	if err != nil {
		return nil, err
	}
	return &claimRejection, nil
}

func (cj *ClaimRejectionRepository) CreateNewClaimRejection(claimRejection models.ClaimRejection) error {
	if claimRejection.ID == uuid.Nil {
		claimRejection.ID = uuid.New()
	}

	claimRejection.CreatedAt = time.Now()

	query := `
		INSERT INTO claim_rejection (
			id, claim_id, validation_timestamp, claim_rejection_type,
			reason, reason_evidence, validated_by, validation_notes, created_at
		) VALUES (
			:id, :claim_id, :validation_timestamp, :claim_rejection_type,
			:reason, :reason_evidence, :validated_by, :validation_notes, :created_at
		)
	`
	_, err := cj.db.NamedExec(query, claimRejection)
	if err != nil {
		return err
	}
	return nil
}

func (cj *ClaimRejectionRepository) UpdateClaimRejection(claimRejection models.ClaimRejection) error {
	query := `
		UPDATE claim_rejection SET
			claim_id = :claim_id,
			validation_timestamp = :validation_timestamp,
			claim_rejection_type = :claim_rejection_type,
			reason = :reason,
			reason_evidence = :reason_evidence,
			validated_by = :validated_by,
			validation_notes = :validation_notes
		WHERE id = :id
	`
	_, err := cj.db.NamedExec(query, claimRejection)
	if err != nil {
		return err
	}
	return nil
}

func (cj *ClaimRejectionRepository) DeleteClaimRejectionByID(claimRejectionID uuid.UUID) error {
	query := `DELETE FROM claim_rejection WHERE id = $1`
	_, err := cj.db.Exec(query, claimRejectionID)
	if err != nil {
		return err
	}
	return nil
}
