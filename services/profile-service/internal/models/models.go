package models

import "time"

// InsurancePartner
type InsurancePartner struct {
	PartnerID             string     `json:"partner_id" db:"partner_id"`
	PartnerName           string     `json:"partner_name" db:"partner_name"`
	PartnerLogoURL        *string    `json:"partner_logo_url,omitempty" db:"partner_logo_url"`
	CoverPhotoURL         *string    `json:"cover_photo_url,omitempty" db:"cover_photo_url"`
	PartnerTagline        *string    `json:"partner_tagline,omitempty" db:"partner_tagline"`
	PartnerDescription    *string    `json:"partner_description,omitempty" db:"partner_description"`
	PartnerPhone          *string    `json:"partner_phone,omitempty" db:"partner_phone"`
	PartnerEmail          *string    `json:"partner_email,omitempty" db:"partner_email"`
	PartnerAddress        *string    `json:"partner_address,omitempty" db:"partner_address"`
	PartnerWebsite        *string    `json:"partner_website,omitempty" db:"partner_website"`
	PartnerRatingScore    *float64   `json:"partner_rating_score,omitempty" db:"partner_rating_score"`
	PartnerRatingCount    int        `json:"partner_rating_count" db:"partner_rating_count"`
	TrustMetricExperience *int       `json:"trust_metric_experience,omitempty" db:"trust_metric_experience"`
	TrustMetricClients    *int       `json:"trust_metric_clients,omitempty" db:"trust_metric_clients"`
	TrustMetricClaimRate  *int       `json:"trust_metric_claim_rate,omitempty" db:"trust_metric_claim_rate"`
	TotalPayouts          *string    `json:"total_payouts,omitempty" db:"total_payouts"`
	AveragePayoutTime     *string    `json:"average_payout_time,omitempty" db:"average_payout_time"`
	ConfirmationTimeline  *string    `json:"confirmation_timeline,omitempty" db:"confirmation_timeline"`
	Hotline               *string    `json:"hotline,omitempty" db:"hotline"`
	SupportHours          *string    `json:"support_hours,omitempty" db:"support_hours"`
	CoverageAreas         *string    `json:"coverage_areas,omitempty" db:"coverage_areas"`
	IsSuspended           bool       `json:"is_suspended" db:"is_suspended"`
	SuspendedAt           *time.Time `json:"suspended_at,omitempty" db:"suspended_at"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
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
