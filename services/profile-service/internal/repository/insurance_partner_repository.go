package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"profile-service/internal/models"
	"strings"
	"utils"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type IInsurancePartnerRepository interface {
	GetInsurancePartnerByID(partnerID string) (*models.InsurancePartner, error)
	GetPartnerReviews(partnerID string, sortBy string, sortDirection string, limit int, offset int) ([]models.PartnerReview, error)
	CreateInsurancePartner(req models.CreateInsurancePartnerRequest, createdByID, createdByName string) error
	GetPublicProfile(partnerID string) (*models.PublicPartnerProfile, error)
	GetPrivateProfile(partnerID string) (*models.PrivatePartnerProfile, error)
	UpdateInsurancePartner(query string, args ...any) error
	GetAllPublicProfiles() ([]models.PublicPartnerProfile, error)
	SearchDeletionRequestsByRequesterName(ctx context.Context, searchTerm string) ([]models.PartnerDeletionRequest, error)
	CreateDeletionRequest(ctx context.Context, req *models.PartnerDeletionRequest) (*models.PartnerDeletionRequest, error)
	GetDeletionRequestsByRequesterID(ctx context.Context, requesterID string) ([]models.DeletionRequestResponse, error)
	ProcessRequestReview(request models.ProcessRequestReviewDTO) error
	GetDeletionRequestsByRequestID(requestID uuid.UUID) (*models.DeletionRequestResponse, error)
	UpdateStatusPartnerProfile(partnerID uuid.UUID, status string, updatedByID string, updatedByName string) error
	GetLatestDeletionRequestByRequesterID(requestedBy string) (*models.PartnerDeletionRequest, error)
	GetAllDeletionRequests(ctx context.Context) ([]models.DeletionRequestResponse, error)
	GetDeletionRequestsByPartnerID(ctx context.Context, partnerID, status string) ([]models.DeletionRequestResponse, error)
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
		log.Printf("Error getting insurance partner by ID %s: %s", partnerID, err.Error())
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
		"active",
	)

	if err != nil {
		log.Printf("Error creating insurance partner: %s", err.Error())
		return fmt.Errorf("failed to create insurance partner: %w", err)
	}

	return nil
}

func (r *InsurancePartnerRepository) GetPublicProfile(partnerID string) (*models.PublicPartnerProfile, error) {
	var profile models.PublicPartnerProfile
	query := `
		SELECT 
			-- A. Brand Identity Information
			ip.partner_id,
			COALESCE(ip.partner_display_name, '') AS partner_display_name,
			COALESCE(ip.partner_logo_url, '') AS partner_logo_url,
			COALESCE(ip.cover_photo_url, '') AS cover_photo_url,
			COALESCE(ip.partner_tagline, '') AS partner_tagline,
			COALESCE(ip.partner_description, '') AS partner_description,
			
			-- B. Public Contact Information
			COALESCE(ip.partner_phone, '') AS partner_phone,
			COALESCE(ip.partner_official_email, '') AS partner_official_email,
			COALESCE(ip.customer_service_hotline, '') AS customer_service_hotline,
			COALESCE(ip.hotline, '') AS hotline,
			COALESCE(ip.support_hours, '') AS support_hours,
			COALESCE(ip.partner_website, '') AS partner_website,
			COALESCE(ip.fax_number, '') AS fax_number,

			-- C. Location Information
			COALESCE(ip.head_office_address, '') AS head_office_address,
			COALESCE(ip.province_name, '') AS province_name,
			COALESCE(ip.ward_name, '') AS ward_name,
			
			-- D. Trust Metrics and Ratings
			COALESCE(ip.partner_rating_score, 0.0) AS partner_rating_score,
			COALESCE(ip.partner_rating_count, 0) AS partner_rating_count,
			COALESCE(ip.trust_metric_experience, 0) AS trust_metric_experience,
			COALESCE(ip.trust_metric_clients, 0) AS trust_metric_clients,
			COALESCE(ip.trust_metric_claim_rate, 0) AS trust_metric_claim_rate,
			COALESCE(ip.total_payouts, '') AS total_payouts,
			
			-- E. Product and Service Information
			COALESCE(ip.average_payout_time, '') AS average_payout_time,
			COALESCE(ip.confirmation_timeline, '') AS confirmation_timeline,
			COALESCE(ip.coverage_areas, '') AS coverage_areas,
			COALESCE(ip.year_established, 0) AS year_established
		FROM insurance_partners ip
		WHERE ip.partner_id = $1
		  AND ip.status = 'active'
	`

	// Execute query
	err := r.db.Get(&profile, query, partnerID)
	if err != nil {
		log.Printf("Error getting public profile for partnerID %s: %s", partnerID, err.Error())
		return nil, fmt.Errorf("failed to get public profile: %w", err)
	}

	return &profile, nil
}

func (r *InsurancePartnerRepository) GetAllPublicProfiles() ([]models.PublicPartnerProfile, error) {
	query := `
		SELECT 
			-- A. Brand Identity Information
			ip.partner_id,
			COALESCE(ip.partner_display_name, '') AS partner_display_name,
			COALESCE(ip.partner_logo_url, '') AS partner_logo_url,
			COALESCE(ip.cover_photo_url, '') AS cover_photo_url,
			COALESCE(ip.partner_tagline, '') AS partner_tagline,
			COALESCE(ip.partner_description, '') AS partner_description,
			
			-- B. Public Contact Information
			COALESCE(ip.partner_phone, '') AS partner_phone,
			COALESCE(ip.partner_official_email, '') AS partner_official_email,
			COALESCE(ip.customer_service_hotline, '') AS customer_service_hotline,
			COALESCE(ip.hotline, '') AS hotline,
			COALESCE(ip.support_hours, '') AS support_hours,
			COALESCE(ip.partner_website, '') AS partner_website,
			COALESCE(ip.fax_number, '') AS fax_number,

			-- C. Location Information
			COALESCE(ip.head_office_address, '') AS head_office_address,
			COALESCE(ip.province_name, '') AS province_name,
			COALESCE(ip.ward_name, '') AS ward_name,
			
			-- D. Trust Metrics and Ratings
			COALESCE(ip.partner_rating_score, 0.0) AS partner_rating_score,
			COALESCE(ip.partner_rating_count, 0) AS partner_rating_count,
			COALESCE(ip.trust_metric_experience, 0) AS trust_metric_experience,
			COALESCE(ip.trust_metric_clients, 0) AS trust_metric_clients,
			COALESCE(ip.trust_metric_claim_rate, 0) AS trust_metric_claim_rate,
			COALESCE(ip.total_payouts, '') AS total_payouts,
			
			-- E. Product and Service Information
			COALESCE(ip.average_payout_time, '') AS average_payout_time,
			COALESCE(ip.confirmation_timeline, '') AS confirmation_timeline,
			COALESCE(ip.coverage_areas, '') AS coverage_areas,
			COALESCE(ip.year_established, 0) AS year_established
		FROM insurance_partners ip
		WHERE ip.status = 'active'
		ORDER BY ip.partner_rating_score DESC, ip.partner_rating_count DESC
	`

	var profiles []models.PublicPartnerProfile
	err := r.db.Select(&profiles, query)
	if err != nil {
		slog.Error("Error getting all public profiles: ", "error", err)
		return nil, fmt.Errorf("internal server error")
	}

	if profiles == nil {
		profiles = []models.PublicPartnerProfile{}
	}

	return profiles, nil
}

// GetPrivateProfile - Lấy TOÀN BỘ thông tin của Insurance Partner (PUBLIC + PRIVATE)
func (r *InsurancePartnerRepository) GetPrivateProfile(partnerID string) (*models.PrivatePartnerProfile, error) {
	var profile models.PrivatePartnerProfile
	log.Printf("GetPrivateProfile called with partnerID: %s", partnerID)
	// Build query
	query := `
		SELECT 
			-- ========== PUBLIC INFORMATION ==========
			-- A. Brand Identification Information
			ip.partner_id,
			COALESCE(ip.partner_display_name, '') AS partner_display_name,
			COALESCE(ip.partner_logo_url, '') AS partner_logo_url,
			COALESCE(ip.cover_photo_url, '') AS cover_photo_url,
			COALESCE(ip.partner_tagline, '') AS partner_tagline,
			COALESCE(ip.partner_description, '') AS partner_description,
			
			-- B. Public Contact Information
			COALESCE(ip.partner_phone, '') AS partner_phone,
			COALESCE(ip.partner_official_email, '') AS partner_official_email,
			COALESCE(ip.customer_service_hotline, '') AS customer_service_hotline,
			COALESCE(ip.hotline, '') AS hotline,
			COALESCE(ip.support_hours, '') AS support_hours,
			COALESCE(ip.partner_website, '') AS partner_website,
			
			-- C. Head Office Address (PUBLIC)
			COALESCE(ip.head_office_address, '') AS head_office_address,
			COALESCE(ip.province_name, '') AS province_name,
			COALESCE(ip.ward_name, '') AS ward_name,
			
			-- D. Trust Metrics and Ratings
			COALESCE(ip.partner_rating_score, 0.0) AS partner_rating_score,
			COALESCE(ip.partner_rating_count, 0) AS partner_rating_count,
			COALESCE(ip.trust_metric_experience, 0) AS trust_metric_experience,
			COALESCE(ip.trust_metric_clients, 0) AS trust_metric_clients,
			COALESCE(ip.trust_metric_claim_rate, 0) AS trust_metric_claim_rate,
			COALESCE(ip.total_payouts, '') AS total_payouts,
			
			-- E. Product Information and Scope of Operations
			COALESCE(ip.average_payout_time, '') AS average_payout_time,
			COALESCE(ip.confirmation_timeline, '') AS confirmation_timeline,
			COALESCE(ip.coverage_areas, '') AS coverage_areas,
			COALESCE(ip.year_established, 0) AS year_established,
			
			-- ========== PRIVATE INFORMATION ==========
			-- A. Legal and Document Information
			ip.legal_company_name,
			COALESCE(ip.partner_trading_name, '') AS partner_trading_name,
			COALESCE(ip.company_type, '') AS company_type,
			COALESCE(ip.incorporation_date, '1970-01-01'::date) AS incorporation_date,
			ip.tax_identification_number,
			ip.business_registration_number,
			COALESCE(ip.insurance_license_number, '') AS insurance_license_number,
			ip.license_issue_date,
			ip.license_expiry_date,
			COALESCE(ip.authorized_insurance_lines, ARRAY[]::TEXT[]) AS authorized_insurance_lines,
			COALESCE(ip.operating_provinces, ARRAY[]::TEXT[]) AS operating_provinces,
			COALESCE(ip.legal_document_urls, ARRAY[]::TEXT[]) AS legal_document_urls,
			
			-- B. Administrative and Technical Information
			COALESCE(ip.province_code, '') AS province_code,
			COALESCE(ip.ward_code, '') AS ward_code,
			COALESCE(ip.postal_code, '') AS postal_code,
			COALESCE(ip.fax_number, '') AS fax_number,
			
			-- C. Status and Management Information
			ip.status,
			ip.created_at,
			ip.updated_at,
			last_updated_by_id,
			COALESCE(ip.last_updated_by_name, '') AS last_updated_by_name
		FROM insurance_partners ip
		WHERE ip.partner_id = $1
	`

	// Execute query (không filter theo status vì partner cần xem cả khi bị suspended)
	err := r.db.Get(&profile, query, partnerID)
	if err != nil {
		log.Printf("Error getting private profile for partnerID %s: %s", partnerID, err.Error())
		return nil, fmt.Errorf("failed to get private profile: %w", err)
	}

	return &profile, nil
}

func (r *InsurancePartnerRepository) UpdateInsurancePartner(query string, args ...any) error {
	if err := utils.ExecWithCheck(r.db, query, utils.ExecUpdate, args...); err != nil {
		return fmt.Errorf("failed to update insurance partner: %w", err)
	}
	return nil
}

func (r *InsurancePartnerRepository) UpdateStatusPartnerProfile(partnerID uuid.UUID, status string, updatedByID string, updatedByName string) error {
	query := `
	update insurance_partners
		set status = $1, updated_at = NOW(), last_updated_by_id = $2, last_updated_by_name = $3, cancellable_until = NOW() + INTERVAL '7 days'
		where partner_id = $4
	`
	return utils.ExecWithCheck(
		r.db,
		query,
		utils.ExecUpdate,
		status,
		updatedByID,
		updatedByName,
		partnerID,
	)
}

// ======= PARTNER DELETION REQUESTS =======

func (r *InsurancePartnerRepository) CreateDeletionRequest(
	ctx context.Context,
	req *models.PartnerDeletionRequest,
) (*models.PartnerDeletionRequest, error) {
	query := `
        INSERT INTO partner_deletion_requests (
            partner_id,
            requested_by,
            requested_by_name,
            detailed_explanation,
            requested_at,
            cancellable_until
        ) VALUES (
            :partner_id,
            :requested_by,
            :requested_by_name,
            :detailed_explanation,
            :requested_at,
            :cancellable_until
        )
        RETURNING 
            request_id,
            partner_id,
            requested_by,
            requested_by_name,
            detailed_explanation,
            status,
            requested_at,
            cancellable_until,
            reviewed_by_id,
            reviewed_by_name,
            reviewed_at,
            review_note,
            updated_at
    `
	rows, err := r.db.NamedQueryContext(ctx, query, req)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result models.PartnerDeletionRequest
	if rows.Next() {
		if err := rows.StructScan(&result); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

func (r *InsurancePartnerRepository) GetDeletionRequestsByRequestID(requestID uuid.UUID) (*models.DeletionRequestResponse, error) {
	query := `
		SELECT 
			pdr.request_id,
			ip.partner_display_name,
			pdr.partner_id,
			pdr.requested_by,
			pdr.requested_by_name,
			pdr.detailed_explanation,
			pdr.status,
			pdr.requested_at,
			pdr.cancellable_until,
			pdr.reviewed_by_id,
			pdr.reviewed_by_name,
			pdr.reviewed_at,
			pdr.review_note,
			pdr.updated_at
		FROM partner_deletion_requests pdr
		INNER JOIN insurance_partners ip ON pdr.partner_id = ip.partner_id
		WHERE pdr.request_id = $1
		ORDER BY pdr.requested_at DESC
	`

	var request models.DeletionRequestResponse
	err := r.db.Get(&request, query, requestID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			slog.Error("not_found: deletion request not found", "request_id", requestID)
			return nil, fmt.Errorf("not_found: deletion request not found")
		}
		slog.Error("Error retrieving deletion request", "error", err, "request_id", requestID)
		return nil, fmt.Errorf("Error retrieving deletion request")
	}

	return &request, nil
}

func (r *InsurancePartnerRepository) GetAllDeletionRequests(
	ctx context.Context,
) ([]models.DeletionRequestResponse, error) {
	query := `
		SELECT 
			pdr.request_id,
			ip.partner_display_name,
			pdr.partner_id,
			pdr.requested_by,
			pdr.requested_by_name,
			pdr.detailed_explanation,
			pdr.status,
			pdr.requested_at,
			pdr.cancellable_until,
			pdr.reviewed_by_id,
			pdr.reviewed_by_name,
			pdr.reviewed_at,
			pdr.review_note,
			pdr.updated_at
		FROM partner_deletion_requests pdr
		INNER JOIN insurance_partners ip ON pdr.partner_id = ip.partner_id
		WHERE pdr.requested_by = $1
		ORDER BY pdr.requested_at DESC
	`

	var requests []models.DeletionRequestResponse
	err := r.db.SelectContext(ctx, &requests, query)
	if err != nil {
		log.Printf("Error retrieving all deletion requests: %s", err.Error())
		return nil, err
	}

	return requests, nil
}

func (r *InsurancePartnerRepository) SearchDeletionRequestsByRequesterName(
	ctx context.Context,
	searchTerm string,
) ([]models.PartnerDeletionRequest, error) {
	query := `
        SELECT 
            request_id,
            partner_id,
            requested_by,
            requested_by_name,
            detailed_explanation,
            status,
            requested_at,
            cancellable_until,
            created_at,
            updated_at
        FROM partner_deletion_requests
        WHERE requested_by_name ILIKE $1
        ORDER BY requested_at DESC
    `

	var requests []models.PartnerDeletionRequest
	searchPattern := "%" + searchTerm + "%"
	err := r.db.SelectContext(ctx, &requests, query, searchPattern)
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *InsurancePartnerRepository) GetDeletionRequestsByRequesterID(
	ctx context.Context,
	requesterID string,
) ([]models.DeletionRequestResponse, error) {
	query := `
		select 
pdr.request_id, 
ip.partner_display_name, 
pdr.partner_id, 
pdr.requested_by, 
pdr.requested_by_name, 
pdr.detailed_explanation, 
pdr.status, 
pdr.requested_at, 
pdr.cancellable_until, 
pdr.reviewed_by_id, 
pdr.reviewed_by_name, 
pdr.reviewed_at, 
pdr.review_note, 
pdr.updated_at from partner_deletion_requests pdr
inner join insurance_partners ip on pdr.partner_id = ip.partner_id 
where pdr.requested_by = $1
	`

	var requests []models.DeletionRequestResponse
	err := r.db.SelectContext(ctx, &requests, query, requesterID)
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *InsurancePartnerRepository) ProcessRequestReview(request models.ProcessRequestReviewDTO) error {
	query := `
		UPDATE partner_deletion_requests 
		SET 
			status = $1,
			reviewed_by_id = $2,
			reviewed_by_name = $3,
			reviewed_at = NOW(),
			review_note = $4,
			updated_at = NOW()
		WHERE request_id = $5
	`

	return utils.ExecWithCheck(
		r.db,
		query,
		utils.ExecUpdate,
		request.Status,
		request.ReviewedByID,
		request.ReviewedByName,
		request.ReviewNote,
		request.RequestID,
	)
}

func (r *InsurancePartnerRepository) GetLatestDeletionRequestByRequesterID(requestedBy string) (*models.PartnerDeletionRequest, error) {
	query := `
		SELECT *
		FROM partner_deletion_requests
		WHERE requested_by = $1
		ORDER BY requested_at DESC
		LIMIT 1
	`

	var request models.PartnerDeletionRequest
	err := r.db.Get(&request, query, requestedBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying partner deletion request: %w", err)
	}

	return &request, nil
}

func (r *InsurancePartnerRepository) GetDeletionRequestsByPartnerID(ctx context.Context, partnerID, status string) ([]models.DeletionRequestResponse, error) {
	var query string
	var requests []models.DeletionRequestResponse

	if status == "all" {
		query = `
			SELECT 
				pdr.request_id,
				ip.partner_display_name,
				pdr.partner_id,
				pdr.requested_by,
				pdr.requested_by_name,
				pdr.detailed_explanation,
				pdr.status,
				pdr.requested_at,
				pdr.cancellable_until,
				pdr.reviewed_by_id,
				pdr.reviewed_by_name,
				pdr.reviewed_at,
				pdr.review_note,
				pdr.updated_at
			FROM partner_deletion_requests pdr
			INNER JOIN insurance_partners ip ON pdr.partner_id = ip.partner_id
			WHERE pdr.partner_id = $1
			ORDER BY pdr.requested_at DESC
		`
		err := r.db.SelectContext(ctx, &requests, query, partnerID)
		if err != nil {
			slog.Error("Error retrieving deletion requests by partner ID", "error", err, "partnerID", partnerID)
			return nil, err
		}
	} else {
		query = `
			SELECT 
				pdr.request_id,
				ip.partner_display_name,
				pdr.partner_id,
				pdr.requested_by,
				pdr.requested_by_name,
				pdr.detailed_explanation,
				pdr.status,
				pdr.requested_at,
				pdr.cancellable_until,
				pdr.reviewed_by_id,
				pdr.reviewed_by_name,
				pdr.reviewed_at,
				pdr.review_note,
				pdr.updated_at
			FROM partner_deletion_requests pdr
			INNER JOIN insurance_partners ip ON pdr.partner_id = ip.partner_id
			WHERE pdr.partner_id = $1 AND pdr.status = $2
			ORDER BY pdr.requested_at DESC
		`
		err := r.db.SelectContext(ctx, &requests, query, partnerID, status)
		if err != nil {
			slog.Error("Error retrieving deletion requests by partner ID and status", "error", err, "partnerID", partnerID, "status", status)
			return nil, err
		}
	}

	return requests, nil
}
