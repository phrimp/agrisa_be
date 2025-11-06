package services

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/worker"

	"github.com/google/uuid"
)

// RegisteredPolicyService handles registered policy operations and worker infrastructure lifecycle
type RegisteredPolicyService struct {
	registeredPolicyRepo *repository.RegisteredPolicyRepository
	basePolicyRepo       *repository.BasePolicyRepository
	farmRepo             *repository.FarmRepository
	workerManager        *worker.WorkerManagerV2
}

// NewRegisteredPolicyService creates a new registered policy service
func NewRegisteredPolicyService(
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	basePolicyRepo *repository.BasePolicyRepository,
	workerManager *worker.WorkerManagerV2,
) *RegisteredPolicyService {
	return &RegisteredPolicyService{
		registeredPolicyRepo: registeredPolicyRepo,
		basePolicyRepo:       basePolicyRepo,
		workerManager:        workerManager,
	}
}

// CreatePolicyWithWorkerInfrastructure creates a registered policy and its worker infrastructure HELPER FUNCTION -- NOT BUSINESS FUNCTION
func (s *RegisteredPolicyService) CreatePolicyWithWorkerInfrastructure(
	ctx context.Context,
	policy *models.RegisteredPolicy,
) error {
	slog.Info("Creating registered policy with worker infrastructure",
		"policy_id", policy.ID,
		"base_policy_id", policy.BasePolicyID,
		"farm_id", policy.FarmID)

	// 1. Create the registered policy
	if err := s.registeredPolicyRepo.Create(policy); err != nil {
		return fmt.Errorf("failed to create registered policy: %w", err)
	}

	// 2. Load base policy and trigger information
	basePolicy, err := s.basePolicyRepo.GetBasePolicyByID(policy.BasePolicyID)
	if err != nil {
		return fmt.Errorf("failed to load base policy: %w", err)
	}

	// Load base policy trigger, there is only 1 trigger at the moment but use slice anyway
	triggers, err := s.basePolicyRepo.GetBasePolicyTriggersByPolicyID(policy.BasePolicyID)
	if err != nil {
		return fmt.Errorf("failed to load base policy triggers: %w", err)
	}

	if len(triggers) == 0 {
		slog.Warn("No triggers found for base policy, skipping worker infrastructure creation",
			"base_policy_id", policy.BasePolicyID)
		return nil
	}

	basePolicyTrigger := &triggers[0]

	conditions, err := s.basePolicyRepo.GetBasePolicyTriggerConditionsByTriggerID(basePolicyTrigger.ID)
	if err != nil {
		return fmt.Errorf("failed to load base policy trigger conditions %w", err)
	}

	// 3. Create worker infrastructure
	if err := s.workerManager.CreatePolicyWorkerInfrastructure(ctx, policy, basePolicy, basePolicyTrigger, conditions); err != nil {
		return fmt.Errorf("failed to create worker infrastructure: %w", err)
	}

	slog.Info("Successfully created registered policy with worker infrastructure",
		"policy_id", policy.ID)

	return nil
}

// StartPolicyMonitoring starts the worker infrastructure for a policy
func (s *RegisteredPolicyService) StartPolicyMonitoring(ctx context.Context, policyID uuid.UUID) error {
	slog.Info("Starting policy monitoring", "policy_id", policyID)

	if err := s.workerManager.StartPolicyWorkerInfrastructure(ctx, policyID); err != nil {
		return fmt.Errorf("failed to start worker infrastructure: %w", err)
	}

	slog.Info("Successfully started policy monitoring", "policy_id", policyID)
	return nil
}

// StopPolicyMonitoring stops the worker infrastructure for a policy
func (s *RegisteredPolicyService) StopPolicyMonitoring(ctx context.Context, policyID uuid.UUID) error {
	slog.Info("Stopping policy monitoring", "policy_id", policyID)

	if err := s.workerManager.StopWorkerInfrastructure(ctx, policyID); err != nil {
		return fmt.Errorf("failed to stop worker infrastructure: %w", err)
	}

	slog.Info("Successfully stopped policy monitoring", "policy_id", policyID)
	return nil
}

// ArchiveExpiredPolicy archives the worker infrastructure for an expired policy
func (s *RegisteredPolicyService) ArchiveExpiredPolicy(ctx context.Context, policyID uuid.UUID) error {
	slog.Info("Archiving expired policy", "policy_id", policyID)

	if err := s.workerManager.ArchiveWorkerInfrastructure(ctx, policyID); err != nil {
		return fmt.Errorf("failed to archive worker infrastructure: %w", err)
	}

	slog.Info("Successfully archived expired policy", "policy_id", policyID)
	return nil
}

// RecoverActivePolicies recovers worker infrastructure for all active policies after restart
func (s *RegisteredPolicyService) RecoverActivePolicies(ctx context.Context) error {
	slog.Info("Recovering active policy worker infrastructure")

	// Load active policy IDs from database
	activePolicyIDs, err := s.workerManager.GetPersistor().LoadActiveWorkerInfrastructure(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active policies: %w", err)
	}

	slog.Info("Found active policies to recover", "count", len(activePolicyIDs))

	// Recover each policy's infrastructure
	successCount := 0
	for _, policyID := range activePolicyIDs {
		if err := s.recoverPolicyInfrastructure(ctx, policyID); err != nil {
			slog.Error("Failed to recover policy infrastructure",
				"policy_id", policyID,
				"error", err)
			continue
		}
		successCount++
	}

	slog.Info("Worker infrastructure recovery completed",
		"total", len(activePolicyIDs),
		"successful", successCount,
		"failed", len(activePolicyIDs)-successCount)

	return nil
}

// recoverPolicyInfrastructure recovers worker infrastructure for a single policy
func (s *RegisteredPolicyService) recoverPolicyInfrastructure(ctx context.Context, policyID uuid.UUID) error {
	slog.Info("Recovering policy infrastructure", "policy_id", policyID)

	// 1. Load registered policy
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		return fmt.Errorf("failed to load registered policy: %w", err)
	}

	// 2. Load base policy
	basePolicy, err := s.basePolicyRepo.GetBasePolicyByID(policy.BasePolicyID)
	if err != nil {
		return fmt.Errorf("failed to load base policy: %w", err)
	}

	// 3. Load base policy trigger
	triggers, err := s.basePolicyRepo.GetBasePolicyTriggersByPolicyID(policy.BasePolicyID)
	if err != nil {
		return fmt.Errorf("failed to load base policy triggers: %w", err)
	}

	if len(triggers) == 0 {
		return fmt.Errorf("no triggers found for base policy %s", policy.BasePolicyID)
	}

	basePolicyTrigger := &triggers[0]

	conditions, err := s.basePolicyRepo.GetBasePolicyTriggerConditionsByTriggerID(basePolicyTrigger.ID)
	if err != nil {
		return fmt.Errorf("failed to load base policy trigger conditions %w", err)
	}

	// 4. Recreate worker infrastructure
	if err := s.workerManager.CreatePolicyWorkerInfrastructure(ctx, policy, basePolicy, basePolicyTrigger, conditions); err != nil {
		return fmt.Errorf("failed to create worker infrastructure: %w", err)
	}

	// 5. Start worker infrastructure
	if err := s.workerManager.StartPolicyWorkerInfrastructure(ctx, policyID); err != nil {
		return fmt.Errorf("failed to start worker infrastructure: %w", err)
	}

	slog.Info("Successfully recovered policy infrastructure", "policy_id", policyID)
	return nil
}

// FetchFarmMonitoringDataJob is the job handler for fetching farm monitoring data
func (s *RegisteredPolicyService) FetchFarmMonitoringDataJob(params map[string]any) error {
	slog.Info("Executing farm monitoring data fetch job", "params", params)

	// Extract parameters
	policyIDStr, ok := params["policy_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid policy_id parameter")
	}

	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return fmt.Errorf("invalid policy_id format: %w", err)
	}

	farmIDStr, ok := params["farm_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid farm_id parameter")
	}

	farmID, err := uuid.Parse(farmIDStr)
	if err != nil {
		return fmt.Errorf("invalid farm_id format: %w", err)
	}

	// TODO: Implement actual farm monitoring data fetch logic
	// This would typically:
	// 1. Query data source APIs (weather, satellite, soil sensors, etc.)
	// 2. Store raw data in database
	// 3. Evaluate trigger conditions
	// 4. Generate alerts if conditions are met
	// 5. Update policy status if needed

	slog.Info("Farm monitoring data fetch completed",
		"policy_id", policyID,
		"farm_id", farmID)

	return nil
}

// ============================================================================
// Validation
// ============================================================================

func (s *RegisteredPolicyService) validateRegisteredPolicy(policy *models.RegisteredPolicy) error {
	return nil
}

// ============================================================================
// BUSINESS PROCESS
// ============================================================================

func (s *RegisteredPolicyService) RegisterAPolicy(request models.RegisterAPolicyRequest) (*models.RegisterAPolicyResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("recover from panic", "panic", r)
		}
	}()
	tx, err := s.registeredPolicyRepo.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error begining registered policy transaction: %w", err)
	}

	if request.IsNewFarm {
		// create new farm
		// err := s.farmRepo.ValidateFarm(request.Farm)
		err := s.farmRepo.CreateTx(tx, &request.Farm)
		if err != nil {
			return nil, fmt.Errorf("error creating farm: %w", err)
		}
	}
	return nil, nil
}
