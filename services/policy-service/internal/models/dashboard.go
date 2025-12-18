package models

type MonthlyRevenue struct {
	Month                   int     `json:"month"`
	Year                    int     `json:"year"`
	TotalRevenue            float64 `json:"total_revenue"`
	TotalPolicies           int64   `json:"total_policies"`
	TotalProviders          int64   `json:"total_providers"`
	AverageRevenuePerPolicy float64 `json:"avg_revenue_per_policy"`
}

type MonthlyRevenueOptions struct {
	Year               int      `json:"year"`
	Month              int      `json:"month"`
	Status             []string `json:"status"`
	UnderwritingStatus []string `json:"underwriting_status"`
}

type AdminRevenueOverview struct {
	TotalActiveProviders int64          `json:"total_active_providers"`
	TotalActivePolicies  int64          `json:"total_active_policies"`
	CurrentMonth         MonthlyRevenue `json:"current_month"`
	PreviousMonth        MonthlyRevenue `json:"previous_month"`
	MonthlyGrowthRate    float64        `json:"monthly_growth_rate"`
}

type MonthlyLossRatio struct {
	Month            string  `json:"month" db:"month"`
	MonthlyPremium   float64 `json:"monthly_premium" db:"monthly_premium"`
	MonthlyPayout    float64 `json:"monthly_payout" db:"monthly_payout"`
	LossRatioPercent float64 `json:"loss_ratio_percent" db:"loss_ratio_percent"`
}

type PremiumGrowthMoM struct {
	Month                string   `json:"month" db:"month"`
	CurrentMonthPremium  float64  `json:"current_month_premium" db:"current_month_premium"`
	PreviousMonthPremium float64  `json:"previous_month_premium" db:"previous_month_premium"`
	MoMGrowthRatePercent *float64 `json:"mom_growth_rate_percent" db:"mom_growth_rate_percent"`
}

type PremiumGrowthYoY struct {
	Month                string   `json:"month" db:"month"`
	CurrentMonthPremium  float64  `json:"current_month_premium" db:"current_month_premium"`
	SameMonthLastYear    float64  `json:"same_month_last_year" db:"same_month_last_year"`
	YoYGrowthRatePercent *float64 `json:"yoy_growth_rate_percent" db:"yoy_growth_rate_percent"`
}

type MonthlyPayoutPerClaim struct {
	Month             string  `json:"month" db:"month"`
	AvgPayoutPerClaim float64 `json:"avg_payout_per_claim" db:"avg_payout_per_claim"`
	TotalPaidClaims   int64   `json:"total_paid_claims" db:"total_paid_claims"`
}

type FinancialSummary struct {
	TotalPremium        float64 `json:"total_premium" db:"total_premium"`
	TotalPayout         float64 `json:"total_payout" db:"total_payout"`
	TotalDataCost       float64 `json:"total_data_cost" db:"total_data_cost"`
	NetIncome           float64 `json:"net_income" db:"net_income"`
	ProfitMarginPercent float64 `json:"profit_margin_percent" db:"profit_margin_percent"`
}

type PartnerDashboardRequest struct {
	PartnerID string `json:"partner_id" validate:"required"`
	StartDate int64  `json:"start_date" validate:"required"`
	EndDate   int64  `json:"end_date" validate:"required"`
}

type PartnerDashboardOverview struct {
	// Financial Summary
	FinancialSummary FinancialSummary `json:"financial_summary"`

	// Premium Metrics
	TotalPremiumCollected   float64 `json:"total_premium_collected"`
	AveragePremiumPerPolicy float64 `json:"average_premium_per_policy"`
	OutstandingPremium      float64 `json:"outstanding_premium"`

	// Payout Metrics
	TotalPayoutDisbursed float64 `json:"total_payout_disbursed"`

	// Trends
	MonthlyLossRatioTrend      []MonthlyLossRatio      `json:"monthly_loss_ratio_trend"`
	PremiumGrowthMoM           []PremiumGrowthMoM      `json:"premium_growth_mom"`
	PremiumGrowthYoY           []PremiumGrowthYoY      `json:"premium_growth_yoy"`
	MonthlyPayoutPerClaimTrend []MonthlyPayoutPerClaim `json:"monthly_payout_per_claim_trend"`
}
