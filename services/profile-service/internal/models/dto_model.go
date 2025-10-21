package models

import "time"

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
}
