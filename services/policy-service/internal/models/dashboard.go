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
	Year               int
	Month              int
	Status             []string
	UnderwritingStatus []string
}

type AdminRevenueOverview struct {
	TotalActiveProviders int64          `json:"total_active_providers"`
	TotalActivePolicies  int64          `json:"total_active_policies"`
	CurrentMonth         MonthlyRevenue `json:"current_month"`
	PreviousMonth        MonthlyRevenue `json:"previous_month"`
	MonthlyGrowthRate    float64        `json:"monthly_growth_rate"`
}
