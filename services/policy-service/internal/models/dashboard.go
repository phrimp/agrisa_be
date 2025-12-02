package models

type MonthlyRevenue struct {
	Month                   int     `json:"month"`
	Year                    int     `json:"year"`
	TotalRevenue            float64 `json:"total_revenue"`
	TotalPolicies           int64     `json:"total_policies"`
	TotalProviders          int64     `json:"total_providers"`
	AverageRevenuePerPolicy float64 `json:"avg_revenue_per_policy"`
}
