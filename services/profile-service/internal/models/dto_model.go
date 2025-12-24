package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// request
type CreateInsurancePartnerRequest struct {
	LegalCompanyName           string    `json:"legal_company_name"`
	PartnerTradingName         string    `json:"partner_trading_name"`
	PartnerDisplayName         string    `json:"partner_display_name"`
	CompanyType                string    `json:"company_type"`
	IncorporationDate          time.Time `json:"incorporation_date"`
	TaxIdentificationNumber    string    `json:"tax_identification_number"`
	BusinessRegistrationNumber string    `json:"business_registration_number"`
	PartnerTagline             string    `json:"partner_tagline"`
	PartnerDescription         string    `json:"partner_description"`
	PartnerPhone               string    `json:"partner_phone"`
	PartnerOfficialEmail       string    `json:"partner_official_email"`
	HeadOfficeAddress          string    `json:"head_office_address"`
	ProvinceCode               string    `json:"province_code"`
	ProvinceName               string    `json:"province_name"`
	WardCode                   string    `json:"ward_code"`
	WardName                   string    `json:"ward_name"`
	PostalCode                 string    `json:"postal_code"`
	FaxNumber                  string    `json:"fax_number"`
	CustomerServiceHotline     string    `json:"customer_service_hotline"`
	InsuranceLicenseNumber     string    `json:"insurance_license_number"`
	LicenseIssueDate           time.Time `json:"license_issue_date"`
	LicenseExpiryDate          time.Time `json:"license_expiry_date"`
	AuthorizedInsuranceLines   []string  `json:"authorized_insurance_lines"`
	OperatingProvinces         []string  `json:"operating_provinces"`
	YearEstablished            int       `json:"year_established"`
	PartnerWebsite             string    `json:"partner_website"`
	TrustMetricExperience      int       `json:"trust_metric_experience"`
	TrustMetricClients         int       `json:"trust_metric_clients"`
	TrustMetricClaimRate       int       `json:"trust_metric_claim_rate"`
	TotalPayouts               string    `json:"total_payouts"`
	AveragePayoutTime          string    `json:"average_payout_time"`
	ConfirmationTimeline       string    `json:"confirmation_timeline"`
	Hotline                    string    `json:"hotline"`
	SupportHours               string    `json:"support_hours"`
	CoverageAreas              string    `json:"coverage_areas"`
	AccountNumber              *string   `json:"account_number"`
	AccountName                *string   `json:"account_name"`
	BankCode                   *string   `json:"bank_code"`
}

// response
type PublicPartnerProfile struct {
	// A. Brand Identity Information
	PartnerID          uuid.UUID `db:"partner_id" json:"partner_id"`
	PartnerDisplayName string    `db:"partner_display_name" json:"partner_display_name"`
	PartnerLogoURL     string    `db:"partner_logo_url" json:"partner_logo_url"`
	CoverPhotoURL      string    `db:"cover_photo_url" json:"cover_photo_url"`
	PartnerTagline     string    `db:"partner_tagline" json:"partner_tagline"`
	PartnerDescription string    `db:"partner_description" json:"partner_description"`

	// B. Public Contact Information
	PartnerPhone           string `db:"partner_phone" json:"partner_phone"`
	PartnerOfficialEmail   string `db:"partner_official_email" json:"partner_official_email"`
	CustomerServiceHotline string `db:"customer_service_hotline" json:"customer_service_hotline"`
	Hotline                string `db:"hotline" json:"hotline"`
	SupportHours           string `db:"support_hours" json:"support_hours"`
	PartnerWebsite         string `db:"partner_website" json:"partner_website"`
	FaxNumber              string `db:"fax_number" json:"fax_number"`

	// C. Location Information
	HeadOfficeAddress string `db:"head_office_address" json:"head_office_address"`
	ProvinceName      string `db:"province_name" json:"province_name"`
	WardName          string `db:"ward_name" json:"ward_name"`

	// D. Trust Metrics and Ratings
	PartnerRatingScore    float64 `db:"partner_rating_score" json:"partner_rating_score"`
	PartnerRatingCount    int     `db:"partner_rating_count" json:"partner_rating_count"`
	TrustMetricExperience int     `db:"trust_metric_experience" json:"trust_metric_experience"`
	TrustMetricClients    int     `db:"trust_metric_clients" json:"trust_metric_clients"`
	TrustMetricClaimRate  int     `db:"trust_metric_claim_rate" json:"trust_metric_claim_rate"`
	TotalPayouts          string  `db:"total_payouts" json:"total_payouts"`

	// E. Product and Service Information
	AveragePayoutTime    string `db:"average_payout_time" json:"average_payout_time"`
	ConfirmationTimeline string `db:"confirmation_timeline" json:"confirmation_timeline"`
	CoverageAreas        string `db:"coverage_areas" json:"coverage_areas"`
	YearEstablished      int    `db:"year_established" json:"year_established"`
}

// PrivatePartnerProfile - detailed profile view for the insurance partner themselves
type PrivatePartnerProfile struct {
	// ========== PUBLIC INFORMATION (similar to PublicPartnerProfile) ==========
	// A. Brand Identification Information
	PartnerID          uuid.UUID `db:"partner_id" json:"partner_id"`
	PartnerDisplayName string    `db:"partner_display_name" json:"partner_display_name"`
	PartnerLogoURL     string    `db:"partner_logo_url" json:"partner_logo_url"`
	CoverPhotoURL      string    `db:"cover_photo_url" json:"cover_photo_url"`
	PartnerTagline     string    `db:"partner_tagline" json:"partner_tagline"`
	PartnerDescription string    `db:"partner_description" json:"partner_description"`

	// B. Public Contact Information
	PartnerPhone           string `db:"partner_phone" json:"partner_phone"`
	PartnerOfficialEmail   string `db:"partner_official_email" json:"partner_official_email"`
	CustomerServiceHotline string `db:"customer_service_hotline" json:"customer_service_hotline"`
	Hotline                string `db:"hotline" json:"hotline"`
	SupportHours           string `db:"support_hours" json:"support_hours"`
	PartnerWebsite         string `db:"partner_website" json:"partner_website"`

	// C. Head Office Address (PUBLIC)
	HeadOfficeAddress string `db:"head_office_address" json:"head_office_address"`
	ProvinceName      string `db:"province_name" json:"province_name"`
	WardName          string `db:"ward_name" json:"ward_name"`

	// D. Trust Metrics and Ratings
	PartnerRatingScore    float64 `db:"partner_rating_score" json:"partner_rating_score"`
	PartnerRatingCount    int     `db:"partner_rating_count" json:"partner_rating_count"`
	TrustMetricExperience int     `db:"trust_metric_experience" json:"trust_metric_experience"`
	TrustMetricClients    int     `db:"trust_metric_clients" json:"trust_metric_clients"`
	TrustMetricClaimRate  int     `db:"trust_metric_claim_rate" json:"trust_metric_claim_rate"`
	TotalPayouts          string  `db:"total_payouts" json:"total_payouts"`

	// E. Product Information and Scope of Operations
	AveragePayoutTime    string `db:"average_payout_time" json:"average_payout_time"`
	ConfirmationTimeline string `db:"confirmation_timeline" json:"confirmation_timeline"`
	CoverageAreas        string `db:"coverage_areas" json:"coverage_areas"`
	YearEstablished      int    `db:"year_established" json:"year_established"`

	// ========== PRIVATE INFORMATION (visible only to the partner) ==========
	// A. Legal and Document Information
	LegalCompanyName           string         `db:"legal_company_name" json:"legal_company_name"`
	PartnerTradingName         string         `db:"partner_trading_name" json:"partner_trading_name"`
	CompanyType                string         `db:"company_type" json:"company_type"`
	IncorporationDate          time.Time      `db:"incorporation_date" json:"incorporation_date"`
	TaxIdentificationNumber    string         `db:"tax_identification_number" json:"tax_identification_number"`
	BusinessRegistrationNumber string         `db:"business_registration_number" json:"business_registration_number"`
	InsuranceLicenseNumber     string         `db:"insurance_license_number" json:"insurance_license_number"`
	LicenseIssueDate           *time.Time     `db:"license_issue_date" json:"license_issue_date"`
	LicenseExpiryDate          *time.Time     `db:"license_expiry_date" json:"license_expiry_date"`
	AuthorizedInsuranceLines   pq.StringArray `db:"authorized_insurance_lines" json:"authorized_insurance_lines"`
	OperatingProvinces         pq.StringArray `db:"operating_provinces" json:"operating_provinces"`
	LegalDocumentURLs          pq.StringArray `db:"legal_document_urls" json:"legal_document_urls"`

	// B. Administrative and Technical Information
	ProvinceCode string `db:"province_code" json:"province_code"`
	DistrictCode string `db:"district_code" json:"district_code"`
	WardCode     string `db:"ward_code" json:"ward_code"`
	PostalCode   string `db:"postal_code" json:"postal_code"`
	FaxNumber    string `db:"fax_number" json:"fax_number"`

	// C. Status and Management Information
	Status            string     `db:"status" json:"status"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         *time.Time `db:"updated_at" json:"updated_at"`
	LastUpdatedByID   string     `db:"last_updated_by_id" json:"last_updated_by_id"`
	LastUpdatedByName string     `db:"last_updated_by_name" json:"last_updated_by_name"`

	// bank info
	AccountNumber *string `db:"account_number" json:"account_number"`
	AccountName   *string `db:"account_name" json:"account_name"`
	BankCode      *string `db:"bank_code" json:"bank_code"`
}

type CreateUserProfileRequest struct {
	// Identity
	UserID string `json:"user_id" db:"user_id"`
	RoleID string `json:"role_id" db:"role_id"`

	// Company Association
	PartnerID *uuid.UUID `json:"partner_id,omitempty" db:"partner_id"`

	// Basic Personal Information
	FullName    string  `json:"full_name" db:"full_name"`
	DisplayName *string `json:"display_name,omitempty" db:"display_name"`
	DateOfBirth *string `json:"date_of_birth,omitempty" db:"date_of_birth"`
	Gender      *string `json:"gender,omitempty" db:"gender"`
	Nationality string  `json:"nationality" db:"nationality"`

	// Contact Information
	PrimaryPhone   string  `json:"primary_phone" db:"primary_phone"`
	AlternatePhone *string `json:"alternate_phone,omitempty" db:"alternate_phone"`
	Email          *string `json:"email,omitempty" db:"email"`

	// Address Information
	PermanentAddress *string `json:"permanent_address,omitempty" db:"permanent_address"`
	CurrentAddress   *string `json:"current_address,omitempty" db:"current_address"`
	ProvinceCode     *string `json:"province_code,omitempty" db:"province_code"`
	ProvinceName     *string `json:"province_name,omitempty" db:"province_name"`
	DistrictCode     *string `json:"district_code,omitempty" db:"district_code"`
	DistrictName     *string `json:"district_name,omitempty" db:"district_name"`
	WardCode         *string `json:"ward_code,omitempty" db:"ward_code"`
	WardName         *string `json:"ward_name,omitempty" db:"ward_name"`
	PostalCode       *string `json:"postal_code,omitempty" db:"postal_code"`
}

type CreatePartnerDeletionRequest struct {
}

type UserBankInfo struct {
	UserID        string `json:"user_id" db:"user_id"`
	AccountNumber string `json:"account_number" db:"account_number"`
	AccountName   string `json:"account_name" db:"account_name"`
	BankCode      string `json:"bank_code" db:"bank_code"`
}

type ReviewDeletionRequest struct {
	RequestID  uuid.UUID             `db:"request_id" json:"request_id"`
	Status     DeletionRequestStatus `db:"status" json:"status"`
	ReviewNote string                `db:"review_note" json:"review_note"`
}

type ProcessRequestReviewDTO struct {
	RequestID      uuid.UUID             `json:"request_id"`
	ReviewedByID   string                `json:"reviewed_by_id"`
	Status         DeletionRequestStatus `json:"status"`
	ReviewedByName string                `json:"reviewed_by_name"`
	ReviewNote     string                `json:"review_note"`
}

type DeletionRequestResponse struct {
	RequestID           uuid.UUID             `db:"request_id" json:"request_id"`
	PartnerDisplayName  string                `db:"partner_display_name" json:"partner_display_name"`
	PartnerID           uuid.UUID             `db:"partner_id" json:"partner_id"`
	RequestedBy         string                `db:"requested_by" json:"requested_by"`
	RequestedByName     string                `db:"requested_by_name" json:"requested_by_name"`
	DetailedExplanation string                `db:"detailed_explanation" json:"detailed_explanation"`
	Status              DeletionRequestStatus `db:"status" json:"status"`
	RequestedAt         time.Time             `db:"requested_at" json:"requested_at"`
	CancellableUntil    *time.Time            `db:"cancellable_until" json:"cancellable_until"`
	ReviewedByID        *string               `db:"reviewed_by_id" json:"reviewed_by_id"`
	ReviewedByName      *string               `db:"reviewed_by_name" json:"reviewed_by_name"`
	ReviewedAt          *time.Time            `db:"reviewed_at" json:"reviewed_at"`
	ReviewNote          *string               `db:"review_note" json:"review_note"`
	UpdatedAt           *time.Time            `db:"updated_at" json:"updated_at"`
}
