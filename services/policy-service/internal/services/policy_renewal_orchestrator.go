package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/event"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/worker"
	"time"

	"github.com/google/uuid"
)

// PolicyRenewalOrchestrator orchestrates the renewal process for policies
type PolicyRenewalOrchestrator struct {
	basePolicyRepo       *repository.BasePolicyRepository
	registeredPolicyRepo *repository.RegisteredPolicyRepository
	validityCalculator   *BasePolicyValidityCalculator
	workerManager        *worker.WorkerManagerV2
	notievent            *event.NotificationHelper
}

// NewPolicyRenewalOrchestrator creates a new renewal orchestrator instance
func NewPolicyRenewalOrchestrator(
	basePolicyRepo *repository.BasePolicyRepository,
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	validityCalculator *BasePolicyValidityCalculator,
	workerManager *worker.WorkerManagerV2,
	notievent *event.NotificationHelper,
) *PolicyRenewalOrchestrator {
	return &PolicyRenewalOrchestrator{
		basePolicyRepo:       basePolicyRepo,
		registeredPolicyRepo: registeredPolicyRepo,
		validityCalculator:   validityCalculator,
		workerManager:        workerManager,
		notievent:            notievent,
	}
}

// RenewalResult contains the results of a renewal operation
type RenewalResult struct {
	BasePolicyID          uuid.UUID
	UpdatedValidityWindow *ValidityWindow
	RenewedPolicyCount    int
	RenewalPremiumApplied bool
	RenewalDiscountRate   float64
	PolicyCode            string
	FarmerIDs             []string
	IsExpired             bool
	Errors                []error
}

// PrepareRenewal prepares all policies for renewal
func (o *PolicyRenewalOrchestrator) PrepareRenewal(
	ctx context.Context,
	basePolicyID uuid.UUID,
) (*RenewalResult, error) {
	slog.Info("Preparing policy renewal -- initializing -- retrieving base policy and all related registered policies", "base_policy_id", basePolicyID)
	basePolicy, err := o.basePolicyRepo.GetBasePolicyByID(basePolicyID)
	if err != nil {
		slog.Error("Error retrieving base policy", "base_policy_id", basePolicyID)
		return nil, fmt.Errorf("error retrieving base policy: %w", err)
	}
	registeredPolicies, err := o.registeredPolicyRepo.GetByBasePolicyID(ctx, basePolicy.ID)
	if err != nil {
		slog.Warn("No registered policy found", "base_policy_id", basePolicy.ID)
	}

	if basePolicy.Status == models.BasePolicyArchived {
		return nil, fmt.Errorf("base policy is archived")
	}

	if !basePolicy.AutoRenewal {
		slog.Info("Preparing policy renewal: no auto renew -- expiration processing",
			"base_policy_id", basePolicy.ID,
			"policy_count", len(registeredPolicies))
		return o.PrepareExpired(ctx, basePolicy, registeredPolicies)
	}

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

		result.RenewalDiscountRate = discountRate
		result.RenewalPremiumApplied = discountRate > 0

		allowedStatus := map[models.PolicyStatus]bool{
			models.PolicyActive: true,
			models.PolicyPayout: true,
		}
		skipStatus := map[models.PolicyStatus]bool{
			models.PolicyCancelled: true,
			models.PolicyRejected:  true,
		}
		// Step 3: Update all registered policies
		for _, policy := range registeredPolicies {

			if skipStatus[policy.Status] {
				continue
			}

			if !allowedStatus[policy.Status] {
				policy.Status = models.PolicyExpired
				continue
			} else {
				originalPremium := policy.TotalFarmerPremium
				policy.TotalFarmerPremium = o.calculateRenewalPremium(originalPremium, discountRate)
				policy.CoverageEndDate = int64(*basePolicy.InsuranceValidToDay)
				policy.PremiumPaidAt = nil
				policy.PremiumPaidByFarmer = false
				policy.Status = models.PolicyPendingPayment

				slog.Info("Calculated renewal premium",
					"base_policy_id", basePolicy.ID,
					"original_premium", originalPremium,
					"discount_rate", discountRate,
					"renewed_premium", policy.TotalFarmerPremium)
			}

			policy.UpdatedAt = time.Now()
			slog.Info("updating registered policy after expired", "policy", policy)

			if err := o.registeredPolicyRepo.Update(&policy); err != nil {
				errMsg := fmt.Errorf("failed to update policy %s : %w", policy.ID, err)
				result.Errors = append(result.Errors, errMsg)
				slog.Error("Failed to update policy",
					"policy_id", policy.ID,
					"error", err)
				// Continue with other policies
				continue
			}
			err := o.workerManager.CleanupWorkerInfrastructure(ctx, policy.ID)
			if err != nil {
				slog.Error("Failed to cleanup worker infrastructure",
					"policy_id", policy.ID,
					"error", err)
			}

			go func() {
				for {
					err := o.notievent.NotifyClaimGenerated(context.Background(), policy.FarmerID, policy.PolicyNumber)
					if err == nil {
						slog.Info("policy underwriting notification sent", "policy id", policy.ID)
						return
					}
					slog.Error("error sending policy underwriting notification", "error", err)
					time.Sleep(10 * time.Second)
				}
			}()

			result.FarmerIDs = append(result.FarmerIDs, policy.FarmerID)
		}

		result.RenewedPolicyCount = len(registeredPolicies) - len(result.Errors)
	}
	result.PolicyCode = *basePolicy.ProductCode

	slog.Info("Renewal preparation completed",
		"base_policy_id", basePolicy.ID,
		"validity_window", fmt.Sprintf("Day %d-%d", nextWindow.FromDay, nextWindow.ToDay),
		"renewed_policies", result.RenewedPolicyCount,
		"errors", len(result.Errors))

	return result, nil
}

func (o *PolicyRenewalOrchestrator) PrepareExpired(
	ctx context.Context,
	basePolicy *models.BasePolicy,
	registeredPolicies []models.RegisteredPolicy,
) (*RenewalResult, error) {
	result := &RenewalResult{
		BasePolicyID: basePolicy.ID,
		Errors:       make([]error, 0),
	}

	basePolicy.Status = models.BasePolicyArchived
	basePolicy.UpdatedAt = time.Now()

	if err := o.basePolicyRepo.UpdateBasePolicy(basePolicy); err != nil {
		return nil, fmt.Errorf("failed to update base policy validity window: %w", err)
	}

	if len(registeredPolicies) > 0 {

		skipStatus := map[models.PolicyStatus]bool{
			models.PolicyCancelled: true,
			models.PolicyRejected:  true,
		}
		for _, policy := range registeredPolicies {

			if skipStatus[policy.Status] {
				slog.Info("skipping status for expired policy", "status", policy.Status, "id", policy.ID)
				continue
			}
			policy.Status = models.PolicyExpired
			policy.UpdatedAt = time.Now()

			slog.Info("Calculated renewal premium",
				"base_policy_id", basePolicy.ID,
			)
			if err := o.registeredPolicyRepo.Update(&policy); err != nil {
				errMsg := fmt.Errorf("failed to update policy %s premium: %w", policy.ID, err)
				result.Errors = append(result.Errors, errMsg)
				slog.Error("Failed to update policy premium",
					"policy_id", policy.ID,
					"error", err)
				// Continue with other policies
				continue
			}
			err := o.workerManager.CleanupWorkerInfrastructure(ctx, policy.ID)
			if err != nil {
				slog.Error("Failed to cleanup worker infrastructure",
					"policy_id", policy.ID,
					"error", err)
			}
			result.FarmerIDs = append(result.FarmerIDs, policy.FarmerID)
		}

		result.RenewedPolicyCount = len(registeredPolicies) - len(result.Errors)
	}
	result.PolicyCode = *basePolicy.ProductCode
	result.IsExpired = true

	slog.Info("Renewal preparation completed",
		"base_policy_id", basePolicy.ID,
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
