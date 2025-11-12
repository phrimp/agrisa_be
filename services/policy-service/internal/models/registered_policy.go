package models

import (
	utils "agrisa_utils"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// REGISTERED POLICY (ACTUAL POLICY INSTANCES)
// ============================================================================

type RegisteredPolicy struct {
	ID                      uuid.UUID          `json:"id" db:"id"`
	PolicyNumber            string             `json:"policy_number" db:"policy_number"`
	BasePolicyID            uuid.UUID          `json:"base_policy_id" db:"base_policy_id"`
	InsuranceProviderID     string             `json:"insurance_provider_id" db:"insurance_provider_id"`
	FarmID                  uuid.UUID          `json:"farm_id,omitempty" db:"farm_id"`
	FarmerID                string             `json:"farmer_id" db:"farmer_id"`
	CoverageAmount          float64            `json:"coverage_amount" db:"coverage_amount"`
	CoverageStartDate       int64              `json:"coverage_start_date,omitempty" db:"coverage_start_date"`
	CoverageEndDate         int64              `json:"coverage_end_date,omitempty" db:"coverage_end_date"`
	PlantingDate            int64              `json:"planting_date" db:"planting_date"`
	AreaMultiplier          float64            `json:"area_multiplier" db:"area_multiplier"`
	TotalFarmerPremium      float64            `json:"total_farmer_premium" db:"total_farmer_premium"`
	PremiumPaidByFarmer     bool               `json:"premium_paid_by_farmer,omitempty" db:"premium_paid_by_farmer"`
	PremiumPaidAt           *int64             `json:"premium_paid_at,omitempty" db:"premium_paid_at"`
	DataComplexityScore     int                `json:"data_complexity_score,omitempty" db:"data_complexity_score"`
	MonthlyDataCost         float64            `json:"monthly_data_cost,omitempty" db:"monthly_data_cost"`
	TotalDataCost           float64            `json:"total_data_cost,omitempty" db:"total_data_cost"`
	Status                  PolicyStatus       `json:"status,omitempty" db:"status"`
	UnderwritingStatus      UnderwritingStatus `json:"underwriting_status,omitempty" db:"underwriting_status"`
	SignedPolicyDocumentURL *string            `json:"signed_policy_document_url,omitempty" db:"signed_policy_document_url"`
	CreatedAt               time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at" db:"updated_at"`
	RegisteredBy            *string            `json:"registered_by,omitempty" db:"registered_by"`
}

type RegisteredPolicyWFarm struct {
	ID                      uuid.UUID          `json:"id" db:"id"`
	PolicyNumber            string             `json:"policy_number" db:"policy_number"`
	BasePolicyID            uuid.UUID          `json:"base_policy_id" db:"base_policy_id"`
	InsuranceProviderID     string             `json:"insurance_provider_id" db:"insurance_provider_id"`
	Farm                    Farm               `json:"farm" `
	FarmerID                string             `json:"farmer_id" db:"farmer_id"`
	CoverageAmount          float64            `json:"coverage_amount" db:"coverage_amount"`
	CoverageStartDate       int64              `json:"coverage_start_date" db:"coverage_start_date"`
	CoverageEndDate         int64              `json:"coverage_end_date" db:"coverage_end_date"`
	PlantingDate            int64              `json:"planting_date" db:"planting_date"`
	AreaMultiplier          float64            `json:"area_multiplier" db:"area_multiplier"`
	TotalFarmerPremium      float64            `json:"total_farmer_premium" db:"total_farmer_premium"`
	PremiumPaidByFarmer     bool               `json:"premium_paid_by_farmer" db:"premium_paid_by_farmer"`
	PremiumPaidAt           *int64             `json:"premium_paid_at,omitempty" db:"premium_paid_at"`
	DataComplexityScore     int                `json:"data_complexity_score" db:"data_complexity_score"`
	MonthlyDataCost         float64            `json:"monthly_data_cost" db:"monthly_data_cost"`
	TotalDataCost           float64            `json:"total_data_cost" db:"total_data_cost"`
	Status                  PolicyStatus       `json:"status" db:"status"`
	UnderwritingStatus      UnderwritingStatus `json:"underwriting_status" db:"underwriting_status"`
	SignedPolicyDocumentURL *string            `json:"signed_policy_document_url,omitempty" db:"signed_policy_document_url"`
	CreatedAt               time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at" db:"updated_at"`
	RegisteredBy            *string            `json:"registered_by,omitempty" db:"registered_by"`
}

type RegisteredPolicyUnderwriting struct {
	ID                  uuid.UUID          `json:"id" db:"id"`
	RegisteredPolicyID  uuid.UUID          `json:"registered_policy_id" db:"registered_policy_id"`
	ValidationTimestamp int64              `json:"validation_timestamp" db:"validation_timestamp"`
	UnderwritingStatus  UnderwritingStatus `json:"underwriting_status" db:"underwriting_status"`
	Recommendations     utils.JSONMap      `json:"recommendations,omitempty" db:"recommendations"`
	Reason              *string            `json:"reason,omitempty" db:"reason"`
	ReasonEvidence      utils.JSONMap      `json:"reason_evidence,omitempty" db:"reason_evidence"`
	ValidatedBy         *string            `json:"validated_by,omitempty" db:"validated_by"`
	ValidationNotes     *string            `json:"validation_notes,omitempty" db:"validation_notes"`
	CreatedAt           time.Time          `json:"created_at" db:"created_at"`
}
type CancelRequest struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	RegisteredPolicyID uuid.UUID `json:"registered_policy_id" db:"registered_policy_id"`

	// Request details
	CancelRequestType CancelRequestType `json:"cancel_request_type" db:"cancel_request_type"`
	Reason            string            `json:"reason" db:"reason"`
	Evidence          utils.JSONMap     `json:"evidence,omitempty" db:"evidence"`

	// Request status and processing
	Status      CancelRequestStatus `json:"status" db:"status"`
	RequestedBy string              `json:"requested_by" db:"requested_by"`
	RequestedAt time.Time           `json:"requested_at" db:"requested_at"`

	CompensateAmount int `json:"compensate_amount" db:"compensate_amount"`

	// Processing details
	ReviewedBy  *string    `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewNotes *string    `json:"review_notes,omitempty" db:"review_notes"`

	// Audit trail
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CancelRequestWithPolicy includes the registered policy details
type CancelRequestWithPolicy struct {
	CancelRequest
	RegisteredPolicy RegisteredPolicy `json:"registered_policy"`
}

// IsApproved checks if the cancel request has been approved
func (cr *CancelRequest) IsApproved() bool {
	return cr.Status == CancelRequestStatusApproved
}

// IsPending checks if the cancel request is pending review
func (cr *CancelRequest) IsPending() bool {
	return cr.ReviewedBy == nil && cr.ReviewedAt == nil
}

// CanBeReviewed checks if the request can still be reviewed (not yet reviewed)
func (cr *CancelRequest) CanBeReviewed() bool {
	return cr.IsPending()
}

// SetApproved marks the request as approved with reviewer details
func (cr *CancelRequest) SetApproved(reviewedBy string, reviewNotes *string) {
	now := time.Now()
	cr.Status = CancelRequestStatusApproved
	cr.ReviewedBy = &reviewedBy
	cr.ReviewedAt = &now
	cr.ReviewNotes = reviewNotes
	cr.UpdatedAt = now
}

// SetDenied marks the request as denied with reviewer details
func (cr *CancelRequest) SetDenied(reviewedBy string, reviewNotes *string) {
	now := time.Now()
	cr.Status = CancelRequestStatusDenied
	cr.ReviewedBy = &reviewedBy
	cr.ReviewedAt = &now
	cr.ReviewNotes = reviewNotes
	cr.UpdatedAt = now
}

// SetLitigation marks the request as in litigation
func (cr *CancelRequest) SetLitigation(reviewedBy string, reviewNotes *string) {
	now := time.Now()
	cr.Status = CancelRequestStatusLitigation
	cr.ReviewedBy = &reviewedBy
	cr.ReviewedAt = &now
	cr.ReviewNotes = reviewNotes
	cr.UpdatedAt = now
}
