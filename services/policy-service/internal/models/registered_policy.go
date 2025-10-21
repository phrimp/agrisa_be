package models

import (
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
	FarmID                  uuid.UUID          `json:"farm_id" db:"farm_id"`
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
	Reason                  *string            `json:"reason,omitempty" db:"reason"`
	ReasonEvidence          any                `json:"reason_evidence" db:"reason_evidence"`
	SignedPolicyDocumentURL *string            `json:"signed_policy_document_url,omitempty" db:"signed_policy_document_url"`
	CreatedAt               time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at" db:"updated_at"`
	RegisteredBy            *string            `json:"registered_by,omitempty" db:"registered_by"`
}
