package services

import (
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"time"
)

type DashboardService struct {
	registeredPolicyRepo repository.RegisteredPolicyRepository
}

func NewDashboardService(registeredPolicyRepo repository.RegisteredPolicyRepository) *DashboardService {
	return &DashboardService{
		registeredPolicyRepo: registeredPolicyRepo,
	}
}

func (s *DashboardService) GetAverageRevenuePerPolicy(year int, month int) (float64, error) {
	monthlyTotalRevenue, err := s.registeredPolicyRepo.GetTotalMonthlyRevenue(year, month, "active", "approved")
	if err != nil {
		slog.Error("failed to get total revenue", "year", year, "month", month, "error", err)
		return 0, err
	}

	monthlyTotalRPolicy, err := s.registeredPolicyRepo.GetMonthlyTotalRegisteredPolicyByStatus(year, month, "active", "approved")
	if err != nil {
		slog.Error("failed to get total registered policy", "year", year, "month", month, "error", err)
		return 0, err
	}

	if monthlyTotalRPolicy == 0 {
		return 0, nil
	}

	averageRevenue := monthlyTotalRevenue / float64(monthlyTotalRPolicy)
	return averageRevenue, nil
}

// GetMonthlyRevenue retrieves monthly revenue data for a specific month
func (s *DashboardService) GetMonthlyRevenue(year, month int) (*models.MonthlyRevenue, error) {
	totalRevenue, err := s.registeredPolicyRepo.GetTotalMonthlyRevenue(year, month, "active", "approved")
	if err != nil {
		slog.Error("failed to get monthly revenue", "year", year, "month", month, "error", err)
		return nil, err
	}

	totalPolicy, err := s.registeredPolicyRepo.GetMonthlyTotalRegisteredPolicyByStatus(year, month, "active", "approved")
	if err != nil {
		slog.Error("failed to get monthly total registered policies", "year", year, "month", month, "error", err)
		return nil, err
	}

	totalProvider, err := s.registeredPolicyRepo.GetTotalProvidersByMonth(year, month, "active", "approved")
	if err != nil {
		slog.Error("failed to get monthly total providers", "year", year, "month", month, "error", err)
		return nil, err
	}

	averageRevenuePerPolicy, err := s.GetAverageRevenuePerPolicy(year, month)
	if err != nil {
		slog.Error("failed to get average revenue per policy", "year", year, "month", month, "error", err)
		return nil, err
	}

	return &models.MonthlyRevenue{
		Year:                    year,
		Month:                   month,
		TotalRevenue:            totalRevenue,
		TotalPolicies:           totalPolicy,
		TotalProviders:          totalProvider,
		AverageRevenuePerPolicy: averageRevenuePerPolicy,
	}, nil
}

func (s *DashboardService) GetCurrentMonthRevenue() (*models.MonthlyRevenue, error) {
	now := time.Now()
	return s.GetMonthlyRevenue(now.Year(), int(now.Month()))
}

func (s *DashboardService) GetPreviousMonthRevenue() (*models.MonthlyRevenue, error) {
	previousMonth := time.Now().AddDate(0, -1, 0)
	return s.GetMonthlyRevenue(previousMonth.Year(), int(previousMonth.Month()))
}

func (s *DashboardService) CalculateMonthlyGrowthRate() (float64, error) {
	currentMonthRevenue, err := s.GetCurrentMonthRevenue()
	if err != nil {
		return 0, err
	}
	previousMonthRevenue, err := s.GetPreviousMonthRevenue()
	if err != nil {
		return 0, err
	}

	if previousMonthRevenue.TotalRevenue == 0 {
		return 100, nil
	}

	growthRate := ((currentMonthRevenue.TotalRevenue - previousMonthRevenue.TotalRevenue) / previousMonthRevenue.TotalRevenue) * 100
	return growthRate, nil
}
