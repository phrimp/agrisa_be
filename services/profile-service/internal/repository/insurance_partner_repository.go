package repository

import (
	"fmt"
	"log"
	"profile-service/internal/models"
	"strings"
	"utils"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type IInsurancePartnerRepository interface {
	GetInsurancePartnerByID(partnerID string) (*models.InsurancePartner, error)
	GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error)
	CreateInsurancePartner(req models.CreateInsurancePartnerRequest, createdByID, createdByName string) error
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
		return nil, fmt.Errorf("invalid. sortBy and sortDirection must have the same number of elements")
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

func (r *InsurancePartnerRepository) CreateInsurancePartner(req models.CreateInsurancePartnerRequest, createdByID, createdByName string) error {
	log.Printf("CreateInsurancePartner called with createdByID: %s, createdByName: %s", createdByID, createdByName)
	
	query := `
		INSERT INTO insurance_partners (
			legal_company_name,
			partner_trading_name,
			partner_display_name,
			company_type,
			incorporation_date,
			tax_identification_number,
			business_registration_number,
			partner_tagline,
			partner_description,
			partner_phone,
			partner_official_email,
			head_office_address,
			province_code,
			province_name,
			ward_code,
			ward_name,
			postal_code,
			fax_number,
			customer_service_hotline,
			insurance_license_number,
			license_issue_date,
			license_expiry_date,
			authorized_insurance_lines,
			operating_provinces,
			year_established,
			partner_website,
			trust_metric_experience,
			trust_metric_clients,
			trust_metric_claim_rate,
			total_payouts,
			average_payout_time,
			confirmation_timeline,
			hotline,
			support_hours,
			coverage_areas,
			last_updated_by_id,
			last_updated_by_name,
			status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38
		)
	`

	err := utils.ExecWithCheck(
		r.db,
		query,
		utils.ExecInsert,
		req.LegalCompanyName,
		req.PartnerTradingName,
		req.PartnerDisplayName,
		req.CompanyType,
		req.IncorporationDate,
		req.TaxIdentificationNumber,
		req.BusinessRegistrationNumber,
		req.PartnerTagline,
		req.PartnerDescription,
		req.PartnerPhone,
		req.PartnerOfficialEmail,
		req.HeadOfficeAddress,
		req.ProvinceCode,
		req.ProvinceName,
		req.WardCode,
		req.WardName,
		req.PostalCode,
		req.FaxNumber,
		req.CustomerServiceHotline,
		req.InsuranceLicenseNumber,
		req.LicenseIssueDate,
		req.LicenseExpiryDate,
		pq.Array(req.AuthorizedInsuranceLines),
		pq.Array(req.OperatingProvinces),
		req.YearEstablished,
		req.PartnerWebsite,
		req.TrustMetricExperience,
		req.TrustMetricClients,
		req.TrustMetricClaimRate,
		req.TotalPayouts,
		req.AveragePayoutTime,
		req.ConfirmationTimeline,
		req.Hotline,
		req.SupportHours,
		req.CoverageAreas,
		createdByID,
		createdByName,
		"pending",
	)

	if err != nil {
		log.Printf("Error creating insurance partner: %s", err.Error())
		return fmt.Errorf("failed to create insurance partner: %w", err)
	}

	return nil
}
