package repository

import (
	"log/slog"
	"time"

	"policy-service/internal/models"

	"github.com/jmoiron/sqlx"
)

type DashboardRepository struct {
	db *sqlx.DB
}

func NewDashboardRepository(db *sqlx.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// GetMonthlyLossRatioTrend returns monthly loss ratio trend for a partner within date range
func (r *DashboardRepository) GetMonthlyLossRatioTrend(partnerID string, startDate, endDate time.Time) ([]models.MonthlyLossRatio, error) {
	query := `
		WITH monthly_metrics AS (
			SELECT 
				DATE_TRUNC('month', TO_TIMESTAMP(rp.premium_paid_at)) AS month,
				
				-- Premium thu được trong tháng
				SUM(
					CASE 
						WHEN rp.premium_paid_by_farmer = true 
						THEN rp.total_farmer_premium 
						ELSE 0 
					END
				) AS premium_collected,
				
				-- Payout trả ra trong tháng (join với claim)
				SUM(
					CASE 
						WHEN c.status = 'paid' 
						 AND DATE_TRUNC('month', c.updated_at) = DATE_TRUNC('month', TO_TIMESTAMP(rp.premium_paid_at))
						THEN c.claim_amount 
						ELSE 0 
					END
				) AS payout_paid
			
			FROM registered_policy rp
			LEFT JOIN claim c ON c.registered_policy_id = rp.id
			JOIN base_policy bp ON bp.id = rp.base_policy_id
			
			WHERE bp.insurance_provider_id = $1
				AND rp.premium_paid_at IS NOT NULL
				AND TO_TIMESTAMP(rp.premium_paid_at) >= $2
				AND TO_TIMESTAMP(rp.premium_paid_at) < $3
			
			GROUP BY DATE_TRUNC('month', TO_TIMESTAMP(rp.premium_paid_at))
		)
		
		SELECT 
			month,
			premium_collected AS monthly_premium,
			payout_paid AS monthly_payout,
			
			ROUND(
				CASE 
					WHEN premium_collected = 0 THEN 0
					ELSE (payout_paid * 100.0 / premium_collected)
				END,
				2
			) AS loss_ratio_percent
		
		FROM monthly_metrics
		ORDER BY month DESC
	`

	var results []models.MonthlyLossRatio
	err := r.db.Select(&results, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get monthly loss ratio trend", "partner_id", partnerID, "error", err)
		return nil, err
	}

	return results, nil
}

// GetTotalPremiumCollected returns total premium collected for a partner within date range
func (r *DashboardRepository) GetTotalPremiumCollected(partnerID string, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT 
			COALESCE(
				SUM(
					CASE 
						WHEN rp.premium_paid_by_farmer = true 
						THEN rp.total_farmer_premium 
						ELSE 0 
					END
				), 
				0
			) AS total_premium_collected
		
		FROM registered_policy rp
		JOIN base_policy bp ON bp.id = rp.base_policy_id
		
		WHERE bp.insurance_provider_id = $1
			AND rp.premium_paid_at IS NOT NULL
			AND TO_TIMESTAMP(rp.premium_paid_at) >= $2
			AND TO_TIMESTAMP(rp.premium_paid_at) < $3
	`

	var totalPremium float64
	err := r.db.Get(&totalPremium, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get total premium collected", "partner_id", partnerID, "error", err)
		return 0, err
	}

	return totalPremium, nil
}

// GetPremiumGrowthMoM calculates month-over-month premium growth rate for a partner
func (r *DashboardRepository) GetPremiumGrowthMoM(partnerID string, startDate, endDate time.Time) ([]models.PremiumGrowthMoM, error) {
	query := `
		WITH monthly_premium AS (
			SELECT 
				TO_CHAR(TO_TIMESTAMP(rp.premium_paid_at), 'YYYY-MM') AS month,
				SUM(rp.total_farmer_premium) AS total_premium
			FROM registered_policy rp
			JOIN base_policy bp ON rp.base_policy_id = bp.id
			WHERE bp.insurance_provider_id = $1
				AND rp.premium_paid_at IS NOT NULL
				AND rp.premium_paid_by_farmer = true
				AND TO_TIMESTAMP(rp.premium_paid_at) >= $2
				AND TO_TIMESTAMP(rp.premium_paid_at) < $3
			GROUP BY TO_CHAR(TO_TIMESTAMP(rp.premium_paid_at), 'YYYY-MM')
			ORDER BY month
		),
		with_previous_month AS (
			SELECT 
				month,
				total_premium AS current_month_premium,
				LAG(total_premium) OVER (ORDER BY month) AS previous_month_premium
			FROM monthly_premium
		)
		SELECT 
			month,
			current_month_premium,
			COALESCE(previous_month_premium, 0) AS previous_month_premium,
			CASE 
				WHEN previous_month_premium IS NULL OR previous_month_premium = 0 THEN NULL
				ELSE ((current_month_premium - previous_month_premium) / previous_month_premium * 100)
			END AS mom_growth_rate_percent
		FROM with_previous_month
		ORDER BY month
	`

	var results []models.PremiumGrowthMoM
	err := r.db.Select(&results, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get premium growth MoM", "partner_id", partnerID, "error", err)
		return nil, err
	}

	return results, nil
}

// GetPremiumGrowthYoY calculates year-over-year premium growth rate for a partner
func (r *DashboardRepository) GetPremiumGrowthYoY(partnerID string, startDate, endDate time.Time) ([]models.PremiumGrowthYoY, error) {
	query := `
		WITH monthly_premium AS (
			SELECT 
				TO_CHAR(TO_TIMESTAMP(rp.premium_paid_at), 'YYYY-MM') AS month,
				SUM(rp.total_farmer_premium) AS total_premium
			FROM registered_policy rp
			JOIN base_policy bp ON rp.base_policy_id = bp.id
			WHERE bp.insurance_provider_id = $1
				AND rp.premium_paid_at IS NOT NULL
				AND rp.premium_paid_by_farmer = true
				AND TO_TIMESTAMP(rp.premium_paid_at) >= $2
				AND TO_TIMESTAMP(rp.premium_paid_at) < $3
			GROUP BY TO_CHAR(TO_TIMESTAMP(rp.premium_paid_at), 'YYYY-MM')
			ORDER BY month
		),
		with_same_month_last_year AS (
			SELECT 
				month,
				total_premium AS current_month_premium,
				LAG(total_premium, 12) OVER (ORDER BY month) AS same_month_last_year
			FROM monthly_premium
		)
		SELECT 
			month,
			current_month_premium,
			COALESCE(same_month_last_year, 0) AS same_month_last_year,
			CASE 
				WHEN same_month_last_year IS NULL OR same_month_last_year = 0 THEN NULL
				ELSE ((current_month_premium - same_month_last_year) / same_month_last_year * 100)
			END AS yoy_growth_rate_percent
		FROM with_same_month_last_year
		ORDER BY month
	`

	var results []models.PremiumGrowthYoY
	err := r.db.Select(&results, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get premium growth YoY", "partner_id", partnerID, "error", err)
		return nil, err
	}

	return results, nil
}

// GetAveragePremiumPerPolicy calculates the average premium per policy for a partner within date range
func (r *DashboardRepository) GetAveragePremiumPerPolicy(partnerID string, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(rp.total_farmer_premium) / NULLIF(COUNT(DISTINCT rp.id), 0), 0)
        	AS avg_premium_per_policy
		FROM registered_policy rp
		JOIN base_policy bp ON rp.base_policy_id = bp.id
		WHERE bp.insurance_provider_id = $1
			AND rp.premium_paid_at IS NOT NULL
			AND rp.premium_paid_by_farmer = true
			AND TO_TIMESTAMP(rp.premium_paid_at) >= $2
			AND TO_TIMESTAMP(rp.premium_paid_at) < $3
	`

	var avgPremium float64
	err := r.db.Get(&avgPremium, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get average premium per policy", "partner_id", partnerID, "error", err)
		return 0, err
	}

	return avgPremium, nil
}

// GetOutstandingPremium calculates total unpaid premium for policies created within date range
func (r *DashboardRepository) GetOutstandingPremium(partnerID string, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(rp.total_farmer_premium), 0) AS outstanding_premium
		FROM registered_policy rp
		JOIN base_policy bp ON rp.base_policy_id = bp.id
		WHERE bp.insurance_provider_id = $1
			AND rp.premium_paid_by_farmer = false
			AND rp.status = 'pending_review'
			AND rp.created_at >= $2
			AND rp.created_at < $3
	`

	var outstandingPremium float64
	err := r.db.Get(&outstandingPremium, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get outstanding premium", "partner_id", partnerID, "error", err)
		return 0, err
	}

	return outstandingPremium, nil
}

// GetTotalPayoutDisbursed calculates total payout amount actually disbursed to farmers within date range
func (r *DashboardRepository) GetTotalPayoutDisbursed(partnerID string, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(p.payout_amount), 0) AS total_payout_disbursed
		FROM payout p
		JOIN registered_policy rp ON p.registered_policy_id = rp.id
		JOIN base_policy bp ON rp.base_policy_id = bp.id
		WHERE bp.insurance_provider_id = $1
			AND p.status = 'completed'
			AND TO_TIMESTAMP(p.completed_at) >= $2
			AND TO_TIMESTAMP(p.completed_at) < $3
	`

	var totalPayout float64
	err := r.db.Get(&totalPayout, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get total payout disbursed", "partner_id", partnerID, "error", err)
		return 0, err
	}

	return totalPayout, nil
}

// GetMonthlyPayoutPerClaimTrend returns monthly average payout per claim trend for a partner
func (r *DashboardRepository) GetMonthlyPayoutPerClaimTrend(partnerID string, startDate, endDate time.Time) ([]models.MonthlyPayoutPerClaim, error) {
	query := `
		SELECT 
			TO_CHAR(TO_TIMESTAMP(p.completed_at), 'YYYY-MM') AS month,
			AVG(p.payout_amount) AS avg_payout_per_claim,
			COUNT(DISTINCT p.claim_id) AS total_paid_claims
		FROM payout p
		JOIN registered_policy rp ON p.registered_policy_id = rp.id
		JOIN base_policy bp ON rp.base_policy_id = bp.id
		WHERE bp.insurance_provider_id = $1
			AND p.status = 'completed'
			AND TO_TIMESTAMP(p.completed_at) >= $2
			AND TO_TIMESTAMP(p.completed_at) < $3
		GROUP BY TO_CHAR(TO_TIMESTAMP(p.completed_at), 'YYYY-MM')
		ORDER BY month
	`

	var results []models.MonthlyPayoutPerClaim
	err := r.db.Select(&results, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get monthly payout per claim trend", "partner_id", partnerID, "error", err)
		return nil, err
	}

	return results, nil
}

// GetFinancialSummary calculates net income and profit margin for a partner within date range
func (r *DashboardRepository) GetFinancialSummary(partnerID string, startDate, endDate time.Time) (*models.FinancialSummary, error) {
	query := `
		WITH metrics AS (
			SELECT 
				COALESCE(SUM(rp.total_farmer_premium), 0) AS total_premium
			FROM registered_policy rp
			JOIN base_policy bp ON rp.base_policy_id = bp.id
			WHERE bp.insurance_provider_id = $1
				AND rp.premium_paid_by_farmer = true
				AND rp.premium_paid_at IS NOT NULL
				AND TO_TIMESTAMP(rp.premium_paid_at) >= $2
				AND TO_TIMESTAMP(rp.premium_paid_at) < $3
		),
		payouts AS (
			SELECT 
				COALESCE(SUM(p.payout_amount), 0) AS total_payout
			FROM payout p
			JOIN registered_policy rp ON p.registered_policy_id = rp.id
			JOIN base_policy bp ON rp.base_policy_id = bp.id
			WHERE bp.insurance_provider_id = $1
				AND p.status = 'completed'
				AND TO_TIMESTAMP(p.completed_at) >= $2
				AND TO_TIMESTAMP(p.completed_at) < $3
		),
		data_costs AS (
			SELECT 
				COALESCE(SUM(rp.total_data_cost), 0) AS total_data_cost
			FROM registered_policy rp
			JOIN base_policy bp ON rp.base_policy_id = bp.id
			WHERE bp.insurance_provider_id = $1
				AND rp.status IN ('active', 'expired', 'payout', 'cancelled')
				AND rp.premium_paid_by_farmer = true
				AND TO_TIMESTAMP(rp.premium_paid_at) >= $2
				AND TO_TIMESTAMP(rp.premium_paid_at) < $3
		)
		SELECT 
			m.total_premium,
			p.total_payout,
			d.total_data_cost,
			(m.total_premium - p.total_payout - d.total_data_cost) AS net_income,
			CASE 
				WHEN m.total_premium = 0 THEN 0
				ELSE ROUND(((m.total_premium - p.total_payout - d.total_data_cost) / m.total_premium * 100), 2)
			END AS profit_margin_percent
		FROM metrics m, payouts p, data_costs d
	`

	var result models.FinancialSummary
	err := r.db.Get(&result, query, partnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get financial summary", "partner_id", partnerID, "error", err)
		return nil, err
	}

	return &result, nil
}
