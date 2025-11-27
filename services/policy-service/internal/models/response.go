package models

import "github.com/google/uuid"

type BasePolicyDataCost struct {
	BasePolicyID      uuid.UUID `json:"base_policy_id" db:"base_policy_id"`
	ProductName       string    `json:"product_name" db:"product_name"`
	ActivePolicyCount int       `json:"active_policy_count" db:"active_policy_count"`
	SumTotalDataCost  float64   `json:"sum_total_data_cost" db:"sum_total_data_cost"`
}

type MonthlyDataCostResponse struct {
	InsuranceProviderID     string               `json:"insurance_provider_id"`
	Month                   int                  `json:"month"`
	Year                    int                  `json:"year"`
	BasePolicyCosts         []BasePolicyDataCost `json:"base_policy_costs"`
	TotalActivePolicies     int                  `json:"total_active_policies"`
	TotalBasePolicyDataCost float64              `json:"total_base_policy_data_cost"`
	Currency                string               `json:"currency"`
}
