package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

// PolicyRenewalOrchestrator orchestrates the renewal process for policies
type PolicyRenewalOrchestrator struct {
	basePolicyRepo       *repository.BasePolicyRepository
	registeredPolicyRepo *repository.RegisteredPolicyRepository
	validityCalculator   *BasePolicyValidityCalculator
}

// NewPolicyRenewalOrchestrator creates a new renewal orchestrator instance
func NewPolicyRenewalOrchestrator(
	basePolicyRepo *repository.BasePolicyRepository,
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	validityCalculator *BasePolicyValidityCalculator,
) *PolicyRenewalOrchestrator {
	return &PolicyRenewalOrchestrator{
		basePolicyRepo:       basePolicyRepo,
		registeredPolicyRepo: registeredPolicyRepo,
		validityCalculator:   validityCalculator,
	}
}

// RenewalResult contains the results of a renewal operation
type RenewalResult struct {
	BasePolicyID            uuid.UUID
	UpdatedValidityWindow   *ValidityWindow
	RenewedPolicyCount      int
	RenewalPremiumApplied   bool
	RenewalDiscountRate     float64
	OriginalPremium         float64
	RenewedPremium          float64
	NotificationsSent       int
	Errors                  []error
}

// PrepareRenewal prepares all policies for renewal
func (o *PolicyRenewalOrchestrator) PrepareRenewal(
	ctx context.Context,
	basePolicy *models.BasePolicy,
	registeredPolicies []models.RegisteredPolicy,
) (*RenewalResult, error) {
	slog.Info("Preparing policy renewal",
		"base_policy_id", basePolicy.ID,
		"policy_count", len(registeredPolicies))

	result := &RenewalResult{
		BasePolicyID: basePolicy.ID,
		Errors:       make([]error, 0),
	}

	// Step 1: Calculate and update base policy validity window
	nextWindow, err := o.validityCalculator.CalculateNextValidityWindow(basePolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate next validity window: %w", err)
	}

	// Validate the calculated window
	if err := o.validityCalculator.ValidateValidityWindow(nextWindow, basePolicy.CoverageDurationDays); err != nil {
		return nil, fmt.Errorf("invalid validity window calculated: %w", err)
	}

	// Update base policy validity window
	basePolicy.InsuranceValidFromDay = &nextWindow.FromDay
	basePolicy.InsuranceValidToDay = &nextWindow.ToDay
	basePolicy.UpdatedAt = time.Now()

	if err := o.basePolicyRepo.UpdateBasePolicy(basePolicy); err != nil {
		return nil, fmt.Errorf("failed to update base policy validity window: %w", err)
	}

	result.UpdatedValidityWindow = nextWindow

	slog.Info("Updated base policy validity window",
		"base_policy_id", basePolicy.ID,
		"new_window", fmt.Sprintf("Day %d-%d", nextWindow.FromDay, nextWindow.ToDay))

	// Step 2: Calculate renewal premium (if discount applies)
	if len(registeredPolicies) > 0 {
		var discountRate float64
		if basePolicy.RenewalDiscountRate != nil {
			discountRate = *basePolicy.RenewalDiscountRate
		}

		// Use first policy as reference for premium calculation
		samplePolicy := registeredPolicies[0]
		originalPremium := samplePolicy.TotalFarmerPremium
		renewedPremium := o.calculateRenewalPremium(originalPremium, discountRate)

		result.RenewalDiscountRate = discountRate
		result.OriginalPremium = originalPremium
		result.RenewedPremium = renewedPremium
		result.RenewalPremiumApplied = discountRate > 0

		slog.Info("Calculated renewal premium",
			"base_policy_id", basePolicy.ID,
			"original_premium", originalPremium,
			"discount_rate", discountRate,
			"renewed_premium", renewedPremium)

		// Step 3: Update all registered policies
		// Note: Payment reset and status update are handled by the expiration handler
		// Here we focus on premium adjustment if needed
		if result.RenewalPremiumApplied {
			for _, policy := range registeredPolicies {
				policy.TotalFarmerPremium = o.calculateRenewalPremium(policy.TotalFarmerPremium, discountRate)
				policy.UpdatedAt = time.Now()

				if err := o.registeredPolicyRepo.Update(&policy); err != nil {
					errMsg := fmt.Errorf("failed to update policy %s premium: %w", policy.ID, err)
					result.Errors = append(result.Errors, errMsg)
					slog.Error("Failed to update policy premium",
						"policy_id", policy.ID,
						"error", err)
					// Continue with other policies
					continue
				}
			}
		}

		result.RenewedPolicyCount = len(registeredPolicies) - len(result.Errors)
	}

	// Step 4: Send renewal notifications (stub for now)
	// TODO: Integrate with notification service
	result.NotificationsSent = 0

	slog.Info("Renewal preparation completed",
		"base_policy_id", basePolicy.ID,
		"validity_window", fmt.Sprintf("Day %d-%d", nextWindow.FromDay, nextWindow.ToDay),
		"renewed_policies", result.RenewedPolicyCount,
		"errors", len(result.Errors))

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("renewal completed with %d errors", len(result.Errors))
	}

	return result, nil
}

// calculateRenewalPremium calculates the renewal premium with discount applied
func (o *PolicyRenewalOrchestrator) calculateRenewalPremium(
	originalPremium float64,
	discountRate float64,
) float64 {
	if discountRate <= 0 || discountRate >= 100 {
		return originalPremium
	}

	// Apply discount: new_premium = original * (1 - discount_rate/100)
	discountMultiplier := 1.0 - (discountRate / 100.0)
	renewedPremium := originalPremium * discountMultiplier

	slog.Debug("Calculated renewal premium",
		"original", originalPremium,
		"discount_rate", discountRate,
		"multiplier", discountMultiplier,
		"renewed", renewedPremium)

	return renewedPremium
}

// ValidateRenewalEligibility checks if a base policy is eligible for renewal
func (o *PolicyRenewalOrchestrator) ValidateRenewalEligibility(
	basePolicy *models.BasePolicy,
) error {
	if basePolicy == nil {
		return fmt.Errorf("basePolicy cannot be nil")
	}

	if !basePolicy.AutoRenewal {
		return fmt.Errorf("base policy does not have auto-renewal enabled")
	}

	if basePolicy.Status != models.BasePolicyActive {
		return fmt.Errorf("base policy is not active: status=%s", basePolicy.Status)
	}

	if basePolicy.InsuranceValidFromDay == nil || basePolicy.InsuranceValidToDay == nil {
		return fmt.Errorf("validity window not set")
	}

	if basePolicy.CoverageDurationDays <= 0 {
		return fmt.Errorf("invalid coverage duration: %d", basePolicy.CoverageDurationDays)
	}

	return nil
}

// CalculateRenewalCoverageDates calculates the new coverage dates for renewed policies
func (o *PolicyRenewalOrchestrator) CalculateRenewalCoverageDates(
	currentCoverageEndDate int64,
	coverageDurationDays int,
) (startDate int64, endDate int64) {
	// New coverage starts the day after the current coverage ends
	startDate = currentCoverageEndDate + (24 * 60 * 60) // Add 1 day in seconds

	// Calculate end date based on coverage duration
	durationSeconds := int64(coverageDurationDays) * 24 * 60 * 60
	endDate = startDate + durationSeconds - (24 * 60 * 60) // Subtract 1 day to make it inclusive

	return startDate, endDate
}
