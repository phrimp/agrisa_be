package services

import (
	utils "agrisa_utils"
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/worker"
	"time"

	"github.com/google/uuid"
)

// RegisteredPolicyService handles registered policy operations and worker infrastructure lifecycle
type RegisteredPolicyService struct {
	registeredPolicyRepo *repository.RegisteredPolicyRepository
	basePolicyRepo       *repository.BasePolicyRepository
	basePolicyService    *BasePolicyService
	farmService          *FarmService
	workerManager        *worker.WorkerManagerV2
	pdfDocumentService   *PDFService
}

// NewRegisteredPolicyService creates a new registered policy service
func NewRegisteredPolicyService(
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	basePolicyRepo *repository.BasePolicyRepository,
	basePolicyService *BasePolicyService,
	farmService *FarmService,
	workerManager *worker.WorkerManagerV2,
) *RegisteredPolicyService {
	return &RegisteredPolicyService{
		registeredPolicyRepo: registeredPolicyRepo,
		basePolicyRepo:       basePolicyRepo,
		basePolicyService:    basePolicyService,
		farmService:          farmService,
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

func (s *RegisteredPolicyService) validateRegisteredPolicy(policy *models.RegisteredPolicy, actualTotalPremium, actualDatacost float64) error {
	if policy.TotalDataCost != actualDatacost {
		return fmt.Errorf("total data cost invalid")
	}
	if policy.TotalFarmerPremium != actualTotalPremium {
		return fmt.Errorf("total premium cost invalid")
	}
	return nil
}

func (s *RegisteredPolicyService) validateEnrollmentDate(startDay, endDate, enrolldate int64) error {
	if startDay > enrolldate {
		return fmt.Errorf("policy have not started yet")
	}
	if enrolldate > endDate {
		return fmt.Errorf("policy enrollment date is over")
	}
	return nil
}

func (s *RegisteredPolicyService) validatePolicyTags(tags map[string]string, requiredTags []string) error {
	for _, tag := range requiredTags {
		if _, exists := tags[tag]; !exists {
			return fmt.Errorf("missing required tag: %s", tag)
		}
	}
	return nil
}

// ============================================================================
// BUSINESS PROCESS
// ============================================================================

func (s *RegisteredPolicyService) RegisterAPolicy(request models.RegisterAPolicyRequest, ctx context.Context) (*models.RegisterAPolicyResponse, error) {
	var err error
	tx, err := s.registeredPolicyRepo.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error beginning registered policy transaction: %w", err)
	}

	var panicErr error
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered", "panic", r)
			panicErr = fmt.Errorf("panic during policy registration: %v", r)
			err = panicErr
		}

		// Single rollback point
		if err != nil && tx != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("failed to rollback transaction", "rollback_error", rbErr, "original_error", err)
			}
		}
	}()

	var farm *models.Farm

	if request.IsNewFarm {
		// create new farm
		return nil, fmt.Errorf("feature unimplemented, comeback later") // TODO: delete later
		farm = &request.Farm
		slog.Info("new farm creation request for a new registered policy", "farm", farm)
		err := s.farmService.CreateFarmTx(farm, request.RegisteredPolicy.FarmerID, tx)
		if err != nil {
			slog.Error("error creating new farm", "error", err)
			return nil, fmt.Errorf("error creating farm: %w", err)
		}
	} else {
		farm, err = s.farmService.GetByFarmID(ctx, request.FarmID)
		if err != nil {
			slog.Error("error getting farm by id", "id", request.FarmID, "error", err)
			return nil, fmt.Errorf("error getting farm by ID: %w", err)
		}
		// verify ownership
	}
	// log current farm
	slog.Info("farm processing completed", "farm", farm)
	// processing base policy
	completeBasePolicy, err := s.basePolicyService.GetCompletePolicyDetail(ctx, models.PolicyDetailFilterRequest{ID: &request.RegisteredPolicy.BasePolicyID, IncludePDF: true})
	if err != nil {
		slog.Error("error processing base policy for registered policy", "error", err)
		return nil, fmt.Errorf("error processing base policy for registered policy: %w", err)
	}

	if completeBasePolicy.BasePolicy.EnrollmentStartDay == nil ||
		completeBasePolicy.BasePolicy.EnrollmentEndDay == nil {
		return nil, fmt.Errorf("internal: enrollment dates are required")
	}

	err = s.validateEnrollmentDate(int64(*completeBasePolicy.BasePolicy.EnrollmentStartDay), int64(*completeBasePolicy.BasePolicy.EnrollmentEndDay), time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("enrollment date validation failed: %w", err)
	}
	// processing register policy
	request.RegisteredPolicy.ID = uuid.New()
	request.RegisteredPolicy.FarmID = farm.ID
	request.RegisteredPolicy.PolicyNumber = "AGP" + utils.GenerateRandomStringWithLength(9)
	request.RegisteredPolicy.UnderwritingStatus = models.UnderwritingPending

	request.RegisteredPolicy.CoverageStartDate = 0 // start day only start after underwriting
	request.RegisteredPolicy.CoverageEndDate = int64(*completeBasePolicy.BasePolicy.InsuranceValidToDay)
	request.RegisteredPolicy.PremiumPaidByFarmer = false

	calculatedTotalPremium := s.calculateFarmerPremium(farm.AreaSqm, completeBasePolicy.BasePolicy.PremiumBaseRate, completeBasePolicy.BasePolicy.FixPremiumAmount)

	// validate register policy
	err = s.validateRegisteredPolicy(&request.RegisteredPolicy, completeBasePolicy.Metadata.TotalDataCost, calculatedTotalPremium)
	if err != nil {
		slog.Error("error validating registered policy", "policy", request.RegisteredPolicy, "error", err)
		return nil, fmt.Errorf("error validating registered policy: %w", err)
	}

	// validate tags
	documentRequiredTags := completeBasePolicy.BasePolicy.DocumentTags.KeySlice()
	err = s.validatePolicyTags(request.PolicyTags, documentRequiredTags)
	if err != nil {
		return nil, fmt.Errorf("error validating document tags: %w", err)
	}

	// populate base policy pdf document
	documentLocation := completeBasePolicy.Document.DocumentURL
	signedDocumentLocation, err := s.pdfDocumentService.FillFromStorageAndUpload(ctx, *documentLocation, request.PolicyTags)
	if err != nil {
		return nil, fmt.Errorf("error generate signed document: %w", err)
	}
	request.RegisteredPolicy.SignedPolicyDocumentURL = &signedDocumentLocation

	// create new register policy
	err = s.registeredPolicyRepo.CreateTx(tx, &request.RegisteredPolicy)
	if err != nil {
		slog.Error("error creating new registered policy", "policy", request.RegisteredPolicy, "error", err)
		return nil, fmt.Errorf("error creating new registered policy: %w", err)
	}
	basePolicyTrigger, err := s.basePolicyRepo.GetBasePolicyTriggersByPolicyID(request.RegisteredPolicy.BasePolicyID)
	if err != nil {
		slog.Error("error getting base policy trigger", "error", err)
		return nil, fmt.Errorf("error getting base policy trigger: %w", err)
	}
	if len(basePolicyTrigger) == 0 {
		slog.Error("base policy trigger", "error", err)
		return nil, fmt.Errorf("internal: basePolicyTrigger len is 0")
	}

	// commit
	if err := tx.Commit(); err != nil {
		slog.Error("error commiting registered policy transaction", "error", err)
		return nil, fmt.Errorf("error commiting registered policy transaction: %w", err)
	}
	// start create worker infrastructure and data jobs
	go func() {
		retryWait := 0.5
		for {
			retryWait = retryWait * 2
			time.Sleep(time.Duration(retryWait) * time.Second)
			err = s.workerManager.CreatePolicyWorkerInfrastructure(ctx, &request.RegisteredPolicy, &completeBasePolicy.BasePolicy,
				&basePolicyTrigger[0],
				completeBasePolicy.Triggers[0].Conditions)
			if err != nil {
				slog.Error("error creating worker infrastructure for policy", "policy", request.RegisteredPolicy, "error", err)
				continue
			}
			err = s.workerManager.StartPolicyWorkerInfrastructure(ctx, request.RegisteredPolicy.ID)
			if err != nil {
				slog.Error("error starting worker infrastructure for policy", "policy", request.RegisteredPolicy, "error", err)
				continue
			}
			break
		}
	}()

	return &models.RegisterAPolicyResponse{
		RegisterPolicyID:             request.RegisteredPolicy.ID.String(),
		SignedPolicyDocumentLocation: signedDocumentLocation,
	}, nil
}

func (s *RegisteredPolicyService) calculateFarmerPremium(areasqm, basePremiumRate float64, fixPremiumAmount int) float64 {
	if areasqm <= 0 {
		areasqm = 1
	}
	if basePremiumRate <= 0 {
		basePremiumRate = 1
	}
	if fixPremiumAmount <= 0 {
		fixPremiumAmount = 1
	}

	return areasqm * basePremiumRate * float64(fixPremiumAmount)
}
