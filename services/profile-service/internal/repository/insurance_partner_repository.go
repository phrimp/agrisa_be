package repository

import (
	"fmt"
	"profile-service/internal/models"
	"strings"

	"github.com/jmoiron/sqlx"
)

type IInsurancePartnerRepository interface {
	GetInsurancePartnerByID(partnerID string) (*models.InsurancePartner, error)
	GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error)
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

func (r *InsurancePartnerRepository) GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error) {
	var reviews []models.PartnerReview

	// Parse sortBy and sortDirection
	sortByFields := strings.Split(sortBy, ",")
	sortDirections := strings.Split(sortDirection, ",")

	// Validate that the number of fields and directions must be equal
	if len(sortByFields) != len(sortDirections) {
		return nil, fmt.Errorf("sortBy and sortDirection must have the same number of elements")
	}

	// Whitelist allowed fields to prevent SQL injection
	allowedFields := map[string]bool{
		"rating_stars":  true,
		"created_at":    true,
		"updated_at":    true,
		"reviewer_name": true,
	}

	// Build ORDER BY clause
	var orderByClauses []string
	for i, field := range sortByFields {
		field = strings.TrimSpace(field)
		direction := strings.TrimSpace(strings.ToUpper(sortDirections[i]))

		// Validate field name
		if !allowedFields[field] {
			return nil, fmt.Errorf("invalid sort field: %s", field)
		}

		// Validate direction
		if direction != "ASC" && direction != "DESC" {
			return nil, fmt.Errorf("invalid sort direction: %s", direction)
		}

		orderByClauses = append(orderByClauses, fmt.Sprintf("%s %s", field, direction))
	}

	orderByClause := strings.Join(orderByClauses, ", ")

	// Calculate actual offset (input offset is page number, starting from 1)
	actualOffset := (offset - 1) * limit
	if actualOffset < 0 {
		actualOffset = 0
	}

	// Build query
	query := fmt.Sprintf(`
		SELECT 
			review_id,
			partner_id,
			reviewer_id,
			reviewer_name,
			reviewer_avatar_url,
			rating_stars,
			review_content,
			created_at,
			updated_at
		FROM partner_reviews
		WHERE partner_id = $1
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, orderByClause)

	// Execute query
	err := r.db.Select(&reviews, query, partnerID, limit, actualOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to get partner reviews: %w", err)
	}

	return reviews, nil
}
