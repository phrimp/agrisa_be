package services

import (
	"fmt"
	"log/slog"
	"policy-service/internal/models"
)

// BasePolicyValidityCalculator handles validity window calculations for base policy renewals
type BasePolicyValidityCalculator struct{}

// NewBasePolicyValidityCalculator creates a new validity calculator instance
func NewBasePolicyValidityCalculator() *BasePolicyValidityCalculator {
	return &BasePolicyValidityCalculator{}
}

// ValidityWindow represents a policy validity period
type ValidityWindow struct {
	FromDay int
	ToDay   int
}

// CalculateNextValidityWindow calculates the next validity window for a renewed policy
// Example progression: Day 1-120 → Day 121-240 → Day 241-360
func (c *BasePolicyValidityCalculator) CalculateNextValidityWindow(
	basePolicy *models.BasePolicy,
) (*ValidityWindow, error) {
	if basePolicy == nil {
		return nil, fmt.Errorf("basePolicy cannot be nil")
	}

	if basePolicy.InsuranceValidFromDay == nil || basePolicy.InsuranceValidToDay == nil {
		return nil, fmt.Errorf("current validity window is not set: from_day=%v, to_day=%v",
			basePolicy.InsuranceValidFromDay, basePolicy.InsuranceValidToDay)
	}

	if basePolicy.CoverageDurationDays <= 0 {
		return nil, fmt.Errorf("invalid coverage duration: %d", basePolicy.CoverageDurationDays)
	}

	currentFromDay := *basePolicy.InsuranceValidFromDay
	currentToDay := *basePolicy.InsuranceValidToDay

	// Validate current window
	expectedDuration := currentToDay - currentFromDay + 1
	if expectedDuration != basePolicy.CoverageDurationDays {
		slog.Warn("Current validity window duration mismatch",
			"expected_duration", basePolicy.CoverageDurationDays,
			"actual_duration", expectedDuration,
			"from_day", currentFromDay,
			"to_day", currentToDay)
	}

	// Calculate next window: starts the day after current window ends
	nextFromDay := currentToDay + 1
	nextToDay := nextFromDay + basePolicy.CoverageDurationDays - 1

	slog.Info("Calculated next validity window",
		"base_policy_id", basePolicy.ID,
		"current_window", fmt.Sprintf("Day %d-%d", currentFromDay, currentToDay),
		"next_window", fmt.Sprintf("Day %d-%d", nextFromDay, nextToDay),
		"coverage_duration", basePolicy.CoverageDurationDays)

	return &ValidityWindow{
		FromDay: nextFromDay,
		ToDay:   nextToDay,
	}, nil
}

// ValidateValidityWindow validates that a validity window is logically correct
func (c *BasePolicyValidityCalculator) ValidateValidityWindow(
	window *ValidityWindow,
	coverageDurationDays int,
) error {
	if window == nil {
		return fmt.Errorf("validity window cannot be nil")
	}

	if window.FromDay <= 0 {
		return fmt.Errorf("from_day must be positive: %d", window.FromDay)
	}

	if window.ToDay <= window.FromDay {
		return fmt.Errorf("to_day must be greater than from_day: from=%d, to=%d",
			window.FromDay, window.ToDay)
	}

	actualDuration := window.ToDay - window.FromDay + 1
	if actualDuration != coverageDurationDays {
		return fmt.Errorf("validity window duration mismatch: expected=%d, actual=%d",
			coverageDurationDays, actualDuration)
	}

	return nil
}

// GetCurrentValidityWindow extracts the current validity window from a base policy
func (c *BasePolicyValidityCalculator) GetCurrentValidityWindow(
	basePolicy *models.BasePolicy,
) (*ValidityWindow, error) {
	if basePolicy == nil {
		return nil, fmt.Errorf("basePolicy cannot be nil")
	}

	if basePolicy.InsuranceValidFromDay == nil || basePolicy.InsuranceValidToDay == nil {
		return nil, fmt.Errorf("validity window not set")
	}

	return &ValidityWindow{
		FromDay: *basePolicy.InsuranceValidFromDay,
		ToDay:   *basePolicy.InsuranceValidToDay,
	}, nil
}

// IsWithinValidityWindow checks if a given day is within the validity window
func (c *BasePolicyValidityCalculator) IsWithinValidityWindow(
	day int,
	window *ValidityWindow,
) bool {
	if window == nil {
		return false
	}
	return day >= window.FromDay && day <= window.ToDay
}

// CalculateRenewalCount calculates how many renewals have occurred based on validity window
func (c *BasePolicyValidityCalculator) CalculateRenewalCount(
	currentWindow *ValidityWindow,
	coverageDurationDays int,
) int {
	if currentWindow == nil || coverageDurationDays <= 0 {
		return 0
	}

	// Renewal count = (fromDay - 1) / coverageDurationDays
	// Example: Day 1-120 (renewal 0), Day 121-240 (renewal 1), Day 241-360 (renewal 2)
	return (currentWindow.FromDay - 1) / coverageDurationDays
}
