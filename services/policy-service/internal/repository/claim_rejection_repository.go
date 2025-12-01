package repository

import (
	"policy-service/internal/models"

	"github.com/jmoiron/sqlx"
)

type ClaimRejectionRepository struct {
	db *sqlx.DB
}

func NewClaimRejectionRepository(db *sqlx.DB) *ClaimRejectionRepository {
	return &ClaimRejectionRepository{db: db}
}

func (cj ClaimRejectionRepository) GetAllClaimRejection() ([]models.ClaimRejection, error) {
	var claimRejections []models.ClaimRejection
	query := `SELECT * FROM claim_rejection ORDER BY created_at DESC`
	err := cj.db.Select(&claimRejections, query)
	if err != nil {
		return nil, err
	}
	return claimRejections, nil
}
