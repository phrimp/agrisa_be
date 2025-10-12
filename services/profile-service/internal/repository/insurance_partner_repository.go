package repository

import (
	"profile-service/internal/models"

	"github.com/jmoiron/sqlx"
)

type IInsurancePartnerRepository interface {
	GetInsurancePartnerByID(partnerID string) (*models.InsurancePartner, error)
}
type InsurancePartnerRepository struct {
	db *sqlx.DB
}

func NewInsurancePartnerRepository(db *sqlx.DB) IInsurancePartnerRepository {
	return &InsurancePartnerRepository{
		db: db,
	}
}

func (r *InsurancePartnerRepository) GetInsurancePartnerByID(partnerID string) (*models.InsurancePartner, error) {
	var partner models.InsurancePartner
	query := `
	select * from insurance_partners ip
	WHERE partner_id=$1`
	err := r.db.Get(&partner, query, partnerID)
	if err != nil {
		return nil, err
	}
	return &partner, nil
}
