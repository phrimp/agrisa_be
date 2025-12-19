package services

import (
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"time"
)

type DashboardService struct {
	registeredPolicyRepo *repository.RegisteredPolicyRepository
	dashboardRepo        *repository.DashboardRepository
}

func NewDashboardService(registeredPolicyRepo *repository.RegisteredPolicyRepository, dashboardRepo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{
		registeredPolicyRepo: registeredPolicyRepo,
		dashboardRepo:        dashboardRepo,
	}
}

func (s *DashboardService) GetAverageRevenuePerPolicy(options models.MonthlyRevenueOptions) (float64, error) {
	monthlyTotalRevenue, err := s.registeredPolicyRepo.GetTotalMonthlyRevenue(options.Year, options.Month, options.Status, options.UnderwritingStatus)
	if err != nil {
		slog.Error("failed to get total revenue", "year", options.Year, "month", options.Month, "error", err)
		return 0, err
	}

	monthlyTotalRPolicy, err := s.registeredPolicyRepo.GetMonthlyTotalRegisteredPolicyByStatus(options.Year, options.Month, options.Status, options.UnderwritingStatus)
	if err != nil {
		slog.Error("failed to get total registered policy", "year", options.Year, "month", options.Month, "error", err)
		return 0, err
	}

	if monthlyTotalRPolicy == 0 {
		return 0, nil
	}

	averageRevenue := monthlyTotalRevenue / float64(monthlyTotalRPolicy)
	return averageRevenue, nil
}

// GetMonthlyRevenue retrieves monthly revenue data for a specific month
func (s *DashboardService) GetMonthlyRevenue(options models.MonthlyRevenueOptions) (*models.MonthlyRevenue, error) {
	totalRevenue, err := s.registeredPolicyRepo.GetTotalMonthlyRevenue(options.Year, options.Month, options.Status, options.UnderwritingStatus)
	if err != nil {
		slog.Error("failed to get monthly revenue", "year", options.Year, "month", options.Month, "error", err)
		return nil, err
	}

	totalPolicy, err := s.registeredPolicyRepo.GetMonthlyTotalRegisteredPolicyByStatus(options.Year, options.Month, options.Status, options.UnderwritingStatus)
	if err != nil {
		slog.Error("failed to get monthly total registered policies", "year", options.Year, "month", options.Month, "error", err)
		return nil, err
	}

	totalProvider, err := s.registeredPolicyRepo.GetTotalProvidersByMonth(options.Year, options.Month, options.Status, options.UnderwritingStatus)
	if err != nil {
		slog.Error("failed to get monthly total providers", "year", options.Year, "month", options.Month, "error", err)
		return nil, err
	}

	averageRevenuePerPolicy, err := s.GetAverageRevenuePerPolicy(options)
	if err != nil {
		slog.Error("failed to get average revenue per policy", "year", options.Year, "month", options.Month, "error", err)
		return nil, err
	}

	return &models.MonthlyRevenue{
		Year:                    options.Year,
		Month:                   options.Month,
		TotalRevenue:            totalRevenue,
		TotalPolicies:           totalPolicy,
		TotalProviders:          totalProvider,
		AverageRevenuePerPolicy: averageRevenuePerPolicy,
	}, nil
}

func (s *DashboardService) GetCurrentMonthRevenue(options models.MonthlyRevenueOptions) (*models.MonthlyRevenue, error) {
	now := time.Now()
	options.Year = now.Year()
	options.Month = int(now.Month())
	return s.GetMonthlyRevenue(options)
}

func (s *DashboardService) GetPreviousMonthRevenue(options models.MonthlyRevenueOptions) (*models.MonthlyRevenue, error) {
	previousMonth := time.Now().AddDate(0, -1, 0)
	options.Year = previousMonth.Year()
	options.Month = int(previousMonth.Month())
	return s.GetMonthlyRevenue(options)
}

func (s *DashboardService) CalculateMonthlyGrowthRate(options models.MonthlyRevenueOptions) (float64, error) {
	currentMonthRevenue, err := s.GetCurrentMonthRevenue(options)
	if err != nil {
		return 0, err
	}
	previousMonthRevenue, err := s.GetPreviousMonthRevenue(options)
	if err != nil {
		return 0, err
	}

	if previousMonthRevenue.TotalRevenue == 0 {
		return 100, nil
	}

	growthRate := ((currentMonthRevenue.TotalRevenue - previousMonthRevenue.TotalRevenue) / previousMonthRevenue.TotalRevenue) * 100
	return growthRate, nil
}

func (s *DashboardService) GetAdminRevenueOverview(options models.MonthlyRevenueOptions) (*models.AdminRevenueOverview, error) {
	currentMonthRevenue, err := s.GetCurrentMonthRevenue(options)
	if err != nil {
		return nil, err
	}
	previousMonthRevenue, err := s.GetPreviousMonthRevenue(options)
	if err != nil {
		return nil, err
	}
	monthlyGrowthRate, err := s.CalculateMonthlyGrowthRate(options)
	if err != nil {
		return nil, err
	}
	totalActiveProviders, err := s.registeredPolicyRepo.GetTotalFilterStatusProviders(options.Status, options.UnderwritingStatus)
	if err != nil {
		return nil, err
	}
	totalActivePolicies, err := s.registeredPolicyRepo.GetTotalFilterStatusPolicies(options.Status, options.UnderwritingStatus)
	if err != nil {
		return nil, err
	}
	return &models.AdminRevenueOverview{
		TotalActiveProviders: totalActiveProviders,
		TotalActivePolicies:  totalActivePolicies,
		CurrentMonth:         *currentMonthRevenue,
		PreviousMonth:        *previousMonthRevenue,
		MonthlyGrowthRate:    monthlyGrowthRate,
	}, nil
}

// GetPartnerDashboardOverview retrieves comprehensive dashboard overview for a partner
func (s *DashboardService) GetPartnerDashboardOverview(req models.PartnerDashboardRequest) (*models.PartnerDashboardOverview, error) {
	startDate := time.Unix(req.StartDate, 0)
	endDate := time.Unix(req.EndDate, 0)

	// 1. Financial Summary (Net Income & Profit Margin)
	financialSummary, err := s.dashboardRepo.GetFinancialSummary(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get financial summary", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 2. Total Premium Collected
	totalPremium, err := s.dashboardRepo.GetTotalPremiumCollected(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get total premium collected", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 3. Average Premium per Policy
	avgPremium, err := s.dashboardRepo.GetAveragePremiumPerPolicy(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get average premium per policy", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 4. Outstanding Premium
	outstandingPremium, err := s.dashboardRepo.GetOutstandingPremium(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get outstanding premium", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 5. Total Payout Disbursed
	totalPayout, err := s.dashboardRepo.GetTotalPayoutDisbursed(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get total payout disbursed", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 6. Monthly Loss Ratio Trend
	lossRatioTrend, err := s.dashboardRepo.GetMonthlyLossRatioTrend(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get monthly loss ratio trend", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 7. Premium Growth MoM
	premiumGrowthMoM, err := s.dashboardRepo.GetPremiumGrowthMoM(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get premium growth MoM", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 8. Premium Growth YoY
	premiumGrowthYoY, err := s.dashboardRepo.GetPremiumGrowthYoY(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get premium growth YoY", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	// 9. Monthly Payout per Claim Trend
	payoutPerClaimTrend, err := s.dashboardRepo.GetMonthlyPayoutPerClaimTrend(req.PartnerID, startDate, endDate)
	if err != nil {
		slog.Error("failed to get monthly payout per claim trend", "partner_id", req.PartnerID, "error", err)
		return nil, err
	}

	return &models.PartnerDashboardOverview{
		FinancialSummary:           *financialSummary,
		TotalPremiumCollected:      totalPremium,
		AveragePremiumPerPolicy:    avgPremium,
		OutstandingPremium:         outstandingPremium,
		TotalPayoutDisbursed:       totalPayout,
		MonthlyLossRatioTrend:      lossRatioTrend,
		PremiumGrowthMoM:           premiumGrowthMoM,
		PremiumGrowthYoY:           premiumGrowthYoY,
		MonthlyPayoutPerClaimTrend: payoutPerClaimTrend,
	}, nil
}
