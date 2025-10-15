package services

import (
	"profile-service/internal/models"
	"profile-service/internal/repository"
)

type InsurancePartnerService struct {
	repo repository.IInsurancePartnerRepository
}

type IInsurancePartnerService interface {
	GetInsurancePartnerByID(partnerID string) (*models.InsurancePartner, error)
	GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error)
}

func (s *InsurancePartnerService) GetInsurancePartnerByID(partnerID string) (*models.InsurancePartner, error) {
	return s.repo.GetInsurancePartnerByID(partnerID)
}

func NewInsurancePartnerService(repo repository.IInsurancePartnerRepository) IInsurancePartnerService {
	return &InsurancePartnerService{
		repo: repo,
	}
}


func (s *InsurancePartnerService) GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error) {
	return s.repo.GetPartnerReviews(partnerID, sortBy, sortDirection, limit, offset)
}
