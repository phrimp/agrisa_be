package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// BILLING & INVOICING (DEPRECATED)
// ============================================================================

type PartnerInvoice struct {
	ID                     uuid.UUID     `json:"id" db:"id"`
	InsuranceProviderID    string        `json:"insurance_provider_id" db:"insurance_provider_id"`
	InvoiceMonth           int64         `json:"invoice_month" db:"invoice_month"`
	InvoiceNumber          string        `json:"invoice_number" db:"invoice_number"`
	ActivePoliciesCount    int           `json:"active_policies_count" db:"active_policies_count"`
	TotalDataComplexityFee float64       `json:"total_data_complexity_fee" db:"total_data_complexity_fee"`
	Subtotal               float64       `json:"subtotal" db:"subtotal"`
	Tax                    float64       `json:"tax" db:"tax"`
	TotalDue               float64       `json:"total_due" db:"total_due"`
	PaymentStatus          PaymentStatus `json:"payment_status" db:"payment_status"`
	DueDate                int64         `json:"due_date" db:"due_date"`
	PaidDate               *int64        `json:"paid_date,omitempty" db:"paid_date"`
	CreatedAt              time.Time     `json:"created_at" db:"created_at"`
}

type InvoiceLineItem struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	InvoiceID          uuid.UUID  `json:"invoice_id" db:"invoice_id"`
	ItemType           string     `json:"item_type" db:"item_type"`
	BasePolicyID       *uuid.UUID `json:"base_policy_id,omitempty" db:"base_policy_id"`
	RegisteredPolicyID *uuid.UUID `json:"registered_policy_id,omitempty" db:"registered_policy_id"`
	Description        *string    `json:"description,omitempty" db:"description"`
	Quantity           int        `json:"quantity" db:"quantity"`
	UnitCost           float64    `json:"unit_cost" db:"unit_cost"`
	TotalCost          float64    `json:"total_cost" db:"total_cost"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
}

