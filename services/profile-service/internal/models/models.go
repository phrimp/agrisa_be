package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const NoticePeriod = 1

// InsurancePartner
type InsurancePartner struct {
	PartnerID                  uuid.UUID      `db:"partner_id"`
	LegalCompanyName           string         `db:"legal_company_name"`
	PartnerTradingName         string         `db:"partner_trading_name"`
	PartnerDisplayName         string         `db:"partner_display_name"`
	PartnerLogoURL             string         `db:"partner_logo_url"`
	CoverPhotoURL              string         `db:"cover_photo_url"`
	CompanyType                string         `db:"company_type"`
	IncorporationDate          *time.Time     `db:"incorporation_date"`
	TaxIdentificationNumber    string         `db:"tax_identification_number"`
	BusinessRegistrationNumber string         `db:"business_registration_number"`
	PartnerTagline             string         `db:"partner_tagline"`
	PartnerDescription         string         `db:"partner_description"`
	PartnerPhone               string         `db:"partner_phone"`
	PartnerOfficialEmail       string         `db:"partner_official_email"`
	HeadOfficeAddress          string         `db:"head_office_address"`
	ProvinceCode               string         `db:"province_code"`
	ProvinceName               string         `db:"province_name"`
	WardCode                   string         `db:"ward_code"`
	WardName                   string         `db:"ward_name"`
	PostalCode                 string         `db:"postal_code"`
	FaxNumber                  string         `db:"fax_number"`
	CustomerServiceHotline     string         `db:"customer_service_hotline"`
	InsuranceLicenseNumber     string         `db:"insurance_license_number"`
	LicenseIssueDate           *time.Time     `db:"license_issue_date"`
	LicenseExpiryDate          *time.Time     `db:"license_expiry_date"`
	AuthorizedInsuranceLines   pq.StringArray `db:"authorized_insurance_lines"`
	OperatingProvinces         pq.StringArray `db:"operating_provinces"`
	YearEstablished            int            `db:"year_established"`
	PartnerWebsite             string         `db:"partner_website"`
	PartnerRatingScore         float32        `db:"partner_rating_score"`
	PartnerRatingCount         int            `db:"partner_rating_count"`
	TrustMetricExperience      int            `db:"trust_metric_experience"`
	TrustMetricClients         int            `db:"trust_metric_clients"`
	TrustMetricClaimRate       int            `db:"trust_metric_claim_rate"`
	TotalPayouts               string         `db:"total_payouts"`
	AveragePayoutTime          string         `db:"average_payout_time"`
	ConfirmationTimeline       string         `db:"confirmation_timeline"`
	Hotline                    string         `db:"hotline"`
	SupportHours               string         `db:"support_hours"`
	CoverageAreas              string         `db:"coverage_areas"`
	Status                     string         `db:"status"`
	CreatedAt                  time.Time      `db:"created_at"`
	UpdatedAt                  time.Time      `db:"updated_at"`
	LastUpdatedByID            *string        `db:"last_updated_by_id"`
	LastUpdatedByName          *string        `db:"last_updated_by_name"`
	LegalDocumentURLs          pq.StringArray `db:"legal_document_urls"`
	// bank info
	AccountNumber *string `db:"account_number"`
	AccountName   *string `db:"account_name"`
	BankCode      *string `db:"bank_code"`
}

type Product struct {
	ProductID            string    `json:"product_id" db:"product_id"`
	PartnerID            string    `json:"partner_id" db:"partner_id"`
	ProductName          string    `json:"product_name" db:"product_name"`
	ProductIcon          *string   `json:"product_icon,omitempty" db:"product_icon"`
	ProductDescription   *string   `json:"product_description,omitempty" db:"product_description"`
	ProductSupportedCrop *string   `json:"product_supported_crop,omitempty" db:"product_supported_crop"` // 'lúa nước' hoặc 'cà phê'
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

type PartnerReview struct {
	ReviewID          string    `json:"review_id" db:"review_id"`
	PartnerID         string    `json:"partner_id" db:"partner_id"`
	ReviewerID        string    `json:"reviewer_id" db:"reviewer_id"`
	ReviewerName      string    `json:"reviewer_name" db:"reviewer_name"`
	ReviewerAvatarURL *string   `json:"reviewer_avatar_url,omitempty" db:"reviewer_avatar_url"`
	RatingStars       int       `json:"rating_stars" db:"rating_stars"` // 1-5
	ReviewContent     *string   `json:"review_content,omitempty" db:"review_content"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type UserProfile struct {
	ProfileID         uuid.UUID  `json:"profile_id" db:"profile_id"`
	UserID            string     `json:"user_id" db:"user_id"`
	RoleID            string     `json:"role_id" db:"role_id"`
	PartnerID         *uuid.UUID `json:"partner_id" db:"partner_id"`
	FullName          string     `json:"full_name" db:"full_name"`
	DisplayName       string     `json:"display_name" db:"display_name"`
	DateOfBirth       *time.Time `json:"date_of_birth" db:"date_of_birth"`
	Gender            string     `json:"gender" db:"gender"`
	Nationality       string     `json:"nationality" db:"nationality"`
	PrimaryPhone      string     `json:"primary_phone" db:"primary_phone"`
	AlternatePhone    string     `json:"alternate_phone" db:"alternate_phone"`
	Email             string     `json:"email" db:"email"`
	PermanentAddress  string     `json:"permanent_address" db:"permanent_address"`
	CurrentAddress    string     `json:"current_address" db:"current_address"`
	ProvinceCode      string     `json:"province_code" db:"province_code"`
	ProvinceName      string     `json:"province_name" db:"province_name"`
	DistrictCode      string     `json:"district_code" db:"district_code"`
	DistrictName      string     `json:"district_name" db:"district_name"`
	WardCode          string     `json:"ward_code" db:"ward_code"`
	WardName          string     `json:"ward_name" db:"ward_name"`
	PostalCode        string     `json:"postal_code" db:"postal_code"`
	AccountNumber     *string    `json:"account_number,omitempty" db:"account_number"`
	AccountName       *string    `json:"account_name,omitempty" db:"account_name"`
	BankCode          *string    `json:"bank_code,omitempty" db:"bank_code"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	LastUpdatedBy     string     `json:"last_updated_by" db:"last_updated_by"`
	LastUpdatedByName string     `json:"last_updated_by_name" db:"last_updated_by_name"`
}

type PartnerDeletionRequest struct {
	RequestID string     `json:"request_id" db:"request_id"`
	PartnerID *uuid.UUID `json:"partner_id" db:"partner_id"`

	// Requester info
	RequestedBy         string `json:"requested_by" db:"requested_by"`
	RequestedByName     string `json:"requested_by_name" db:"requested_by_name"`
	DetailedExplanation string `json:"detailed_explanation" db:"detailed_explanation"`

	// Status and timeline
	Status           DeletionRequestStatus `json:"status" db:"status"`
	RequestedAt      time.Time             `json:"requested_at" db:"requested_at"`
	CancellableUntil *time.Time            `json:"cancellable_until" db:"cancellable_until"`

	// Reviewer info
	ReviewedByID   *string    `json:"reviewed_by_id,omitempty" db:"reviewed_by_id"`
	ReviewedByName *string    `json:"reviewed_by_name,omitempty" db:"reviewed_by_name"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewNote     *string    `json:"review_note,omitempty" db:"review_note"`

	// Metadata
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	TransferPartnerID *uuid.UUID `json:"transfer_partner_id,omitempty" db:"transfer_partner_id"`
}
