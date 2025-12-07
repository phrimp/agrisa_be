package services

import (
	utils "agrisa_utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"policy-service/internal/ai/gemini"
	"policy-service/internal/database/minio"
	"policy-service/internal/event"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/worker"
	"time"

	"github.com/google/uuid"
)

// RegisteredPolicyService handles registered policy operations and worker infrastructure lifecycle
type RegisteredPolicyService struct {
	registeredPolicyRepo   *repository.RegisteredPolicyRepository
	basePolicyRepo         *repository.BasePolicyRepository
	basePolicyService      *BasePolicyService
	farmService            *FarmService
	workerManager          *worker.WorkerManagerV2
	pdfDocumentService     *PDFService
	dataSourceRepo         *repository.DataSourceRepository
	farmMonitoringDataRepo *repository.FarmMonitoringDataRepository
	minioClient            *minio.MinioClient
	notievent              *event.NotificationHelper
	geminiSelector         *gemini.GeminiClientSelector
}

// NewRegisteredPolicyService creates a new registered policy service
func NewRegisteredPolicyService(
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	basePolicyRepo *repository.BasePolicyRepository,
	basePolicyService *BasePolicyService,
	farmService *FarmService,
	workerManager *worker.WorkerManagerV2,
	pdfDocumentService *PDFService,
	dataSourceRepo *repository.DataSourceRepository,
	farmMonitoringDataRepo *repository.FarmMonitoringDataRepository,
	minioClient *minio.MinioClient,
	notievent *event.NotificationHelper,
	geminiSelector *gemini.GeminiClientSelector,
) *RegisteredPolicyService {
	return &RegisteredPolicyService{
		registeredPolicyRepo:   registeredPolicyRepo,
		basePolicyRepo:         basePolicyRepo,
		basePolicyService:      basePolicyService,
		farmService:            farmService,
		workerManager:          workerManager,
		pdfDocumentService:     pdfDocumentService,
		dataSourceRepo:         dataSourceRepo,
		farmMonitoringDataRepo: farmMonitoringDataRepo,
		minioClient:            minioClient,
		notievent:              notievent,
		geminiSelector:         geminiSelector,
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

func (s *RegisteredPolicyService) RegisterAPolicy(request models.RegisterAPolicyRequest, ctx context.Context, partnerUserIDs []string) (*models.RegisterAPolicyResponse, error) {
	now := time.Now()
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
		// Create new farm with validation
		farm = &request.Farm
		slog.Info("new farm creation request for a new registered policy", "farm", farm)

		// Validate required fields (using same validation logic as CreateFarmValidate)
		if farm.CropType == "" {
			return nil, fmt.Errorf("crop_type is required")
		}

		if farm.AreaSqm <= 0 {
			return nil, fmt.Errorf("area_sqm must be greater than 0")
		}

		if !ValidateCroptype(farm.CropType) {
			return nil, fmt.Errorf("invalid crop_type (only rice or coffee allowed)")
		}

		if farm.SoilType == nil {
			return nil, fmt.Errorf("soil_type is required")
		}

		if !ValidateSoilType(farm.SoilType, farm.CropType) {
			return nil, fmt.Errorf("invalid soil_type for the given crop_type")
		}

		// Validate harvest date if provided
		if farm.ExpectedHarvestDate != nil {
			if farm.PlantingDate == nil {
				return nil, fmt.Errorf("planting_date is required when expected_harvest_date is provided")
			}
			if *farm.ExpectedHarvestDate < *farm.PlantingDate {
				return nil, fmt.Errorf("expected_harvest_date must be greater than or equal to planting_date")
			}
		}

		if farm.OwnerNationalID == nil {
			return nil, fmt.Errorf("owner_national_id is required")
		}

		// Create farm in transaction
		err = s.farmService.CreateFarmTx(farm, request.RegisteredPolicy.FarmerID, tx)
		if err != nil {
			slog.Error("error creating new farm", "error", err)
			return nil, fmt.Errorf("error creating farm: %w", err)
		}

		slog.Info("new farm created successfully", "farm_id", farm.ID)
	} else {
		farmID, err := uuid.Parse(request.FarmID)
		if err != nil {
			slog.Error("error parsing farm id", "error", err)
			return nil, fmt.Errorf("error parsing farm id: %w", err)
		}
		existingPolicy, err := s.registeredPolicyRepo.GetByBasePolicyIDAndFarmID(request.RegisteredPolicy.BasePolicyID, farmID)
		if existingPolicy != nil {
			slog.Error("farm already registered to this base policy, additional error", "error", err)
			return nil, fmt.Errorf("farm already registered to this base policy")
		}

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

	if completeBasePolicy.BasePolicy.Status != models.BasePolicyActive {
		return nil, fmt.Errorf("base policy is not active: status=%s", completeBasePolicy.BasePolicy.Status)
	}

	if completeBasePolicy.BasePolicy.InsuranceValidToDay != nil {
		if now.Unix() > int64(*completeBasePolicy.BasePolicy.InsuranceValidToDay) {
			return nil, fmt.Errorf("base policy is invalid")
		}
	}

	if completeBasePolicy.BasePolicy.EnrollmentStartDay == nil ||
		completeBasePolicy.BasePolicy.EnrollmentEndDay == nil {
		return nil, fmt.Errorf("internal: enrollment dates are required")
	}

	err = s.validateEnrollmentDate(int64(*completeBasePolicy.BasePolicy.EnrollmentStartDay), int64(*completeBasePolicy.BasePolicy.EnrollmentEndDay), now.Unix())
	if err != nil {
		return nil, fmt.Errorf("enrollment date validation failed: %w", err)
	}
	// processing register policy
	request.RegisteredPolicy.ID = uuid.New()
	request.RegisteredPolicy.FarmID = farm.ID
	request.RegisteredPolicy.PolicyNumber = "AGP" + utils.GenerateRandomStringWithLength(9)
	request.RegisteredPolicy.UnderwritingStatus = models.UnderwritingPending

	request.RegisteredPolicy.CoverageStartDate = 0 // start day only start after payment
	request.RegisteredPolicy.CoverageEndDate = int64(*completeBasePolicy.BasePolicy.InsuranceValidToDay)
	request.RegisteredPolicy.PremiumPaidByFarmer = false
	request.RegisteredPolicy.Status = models.PolicyPendingReview

	calculatedTotalPremium := s.calculateFarmerPremium(farm.AreaSqm, completeBasePolicy.BasePolicy.PremiumBaseRate, completeBasePolicy.BasePolicy.FixPremiumAmount)
	slog.Info("Total Calculated Premium", "premium", calculatedTotalPremium)
	calculatedCoverageAmount := s.calculateCoverageAmount(completeBasePolicy.BasePolicy.PayoutBaseRate, farm.AreaSqm, completeBasePolicy.BasePolicy.FixPayoutAmount, completeBasePolicy.BasePolicy.IsPerHectare)
	slog.Info("Total Coverage Amount", "coverage amount", calculatedCoverageAmount)
	request.RegisteredPolicy.CoverageAmount = calculatedCoverageAmount

	// validate register policy
	err = s.validateRegisteredPolicy(&request.RegisteredPolicy, calculatedTotalPremium, completeBasePolicy.Metadata.TotalDataCost)
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
		slog.Error("error generate signed document", "error", err)
		// return nil, fmt.Errorf("error generate signed document: %w", err)
		request.RegisteredPolicy.SignedPolicyDocumentURL = documentLocation
	} else {
		request.RegisteredPolicy.SignedPolicyDocumentURL = &signedDocumentLocation
	}

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
		currentTime := time.Now()
		previousYearTime := currentTime.AddDate(-1, 0, 0)

		// send job
		fullYearJob := worker.JobPayload{
			JobID: uuid.NewString(),
			Type:  "fetch-farm-monitoring-data",
			Params: map[string]any{
				"policy_id":      request.RegisteredPolicy.ID,
				"base_policy_id": completeBasePolicy.BasePolicy.ID,
				"farm_id":        farm.ID,
				"start_date":     previousYearTime.Unix(), // int64
				"end_date":       currentTime.Unix(),      // int64
			},
			MaxRetries: 10,
			OneTime:    true,
			RunNow:     true,
		}
		scheduler, ok := s.workerManager.GetSchedulerByPolicyID(request.RegisteredPolicy.ID)
		if !ok {
			slog.Error("error get farm-imagery scheduler", "error", "scheduler doesn't exist")
		}
		scheduler.AddJob(fullYearJob)
	}()

	go func() {
		for {
			err := s.notievent.NotifyPolicyRegistered(context.Background(), request.RegisteredPolicy.FarmerID, request.RegisteredPolicy.PolicyNumber)
			if err == nil {
				slog.Info("policy registeration notification sent", "policy id", request.RegisteredPolicy.ID)
				return
			}
			slog.Error("error sending policy registeration notification", "error", err)
			time.Sleep(10 * time.Second)
		}
	}()

	go func() {
		for {
			err := s.notievent.NotifyPolicyRegisteredPartner(context.Background(), partnerUserIDs, request.RegisteredPolicy.PolicyNumber)
			if err == nil {
				slog.Info("policy registeration partner notification sent", "policy id", request.RegisteredPolicy.ID)
				return
			}
			slog.Error("error sending policy registeration partner notification", "error", err)
			time.Sleep(10 * time.Second)
		}
	}()

	return &models.RegisterAPolicyResponse{
		RegisterPolicyID:             request.RegisteredPolicy.ID.String(),
		SignedPolicyDocumentLocation: signedDocumentLocation,
	}, nil
}

func (s *RegisteredPolicyService) calculateCoverageAmount(payoutBaseRate, hectare float64, baseCoverageAmount int, isPerHactare bool) float64 {
	if isPerHactare {
		return float64(baseCoverageAmount) * hectare * payoutBaseRate
	}
	return float64(baseCoverageAmount) * payoutBaseRate
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

// GetPolicyStats retrieves policy statistics (optionally filtered by provider)
func (s *RegisteredPolicyService) GetPolicyStats(providerID string) (map[string]interface{}, error) {
	return s.registeredPolicyRepo.GetPolicyStats(providerID)
}

// UpdatePolicyStatus updates the status of a registered policy
func (s *RegisteredPolicyService) UpdatePolicyStatus(policyID uuid.UUID, status models.PolicyStatus) error {
	return s.registeredPolicyRepo.UpdateStatus(policyID, status)
}

// UpdateUnderwritingStatus updates the underwriting status of a registered policy
func (s *RegisteredPolicyService) UpdateUnderwritingStatus(policyID uuid.UUID, status models.UnderwritingStatus) error {
	return s.registeredPolicyRepo.UpdateUnderwritingStatus(policyID, status)
}

// GetPolicyByID retrieves a single policy by ID
func (s *RegisteredPolicyService) GetPolicyByID(policyID uuid.UUID) (*models.RegisteredPolicy, error) {
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		return nil, err
	}
	link, err := s.minioClient.GetPresignedURL(context.Background(), minio.Storage.PolicyDocuments, *policy.SignedPolicyDocumentURL, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	policy.SignedPolicyDocumentURL = &link
	return policy, nil
}

// GetPoliciesByFarmerID retrieves all policies for a specific farmer
func (s *RegisteredPolicyService) GetPoliciesByFarmerID(farmerID string) ([]models.RegisteredPolicy, error) {
	return s.registeredPolicyRepo.GetByFarmerID(farmerID)
}

// GetPoliciesByProviderID retrieves all policies for a specific insurance provider
func (s *RegisteredPolicyService) GetPoliciesByProviderID(providerID string) ([]models.RegisteredPolicy, error) {
	return s.registeredPolicyRepo.GetByInsuranceProviderID(providerID)
}

// GetAllPolicies retrieves all registered policies
func (s *RegisteredPolicyService) GetAllPolicies() ([]models.RegisteredPolicy, error) {
	return s.registeredPolicyRepo.GetAll()
}

// GetRegisteredPoliciesWithFilters retrieves registered policies with optional filters and presigned URLs
func (s *RegisteredPolicyService) GetRegisteredPoliciesWithFilters(ctx context.Context, filter models.RegisteredPolicyFilterRequest) (*models.RegisteredPolicyFilterResponse, error) {
	// Get filtered policies from repository
	policies, err := s.registeredPolicyRepo.GetWithFilters(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered policies: %w", err)
	}

	// Build response with additional details
	var policiesWithDetails []models.RegisteredPolicyWithDetails
	for _, policy := range policies {
		policyDetail := models.RegisteredPolicyWithDetails{
			RegisteredPolicy: policy,
		}

		// Fetch minimal farm info
		if policy.FarmID != uuid.Nil {
			farm, err := s.farmService.GetByFarmID(ctx, policy.FarmID.String())
			if err == nil && farm != nil {
				policyDetail.Farm = &models.MinimalFarmInfo{
					ID:             farm.ID,
					FarmName:       farm.FarmName,
					FarmCode:       farm.FarmCode,
					AreaSqm:        farm.AreaSqm,
					Province:       farm.Province,
					District:       farm.District,
					Commune:        farm.Commune,
					CropType:       farm.CropType,
					CenterLocation: farm.CenterLocation,
				}
			}
		}

		// Fetch minimal base policy info
		basePolicy, err := s.basePolicyRepo.GetBasePolicyByID(policy.BasePolicyID)
		if err == nil && basePolicy != nil {
			policyDetail.BasePolicy = &models.MinimalBasePolicyInfo{
				ID:                   basePolicy.ID,
				ProductName:          basePolicy.ProductName,
				CropType:             basePolicy.CropType,
				CoverageCurrency:     basePolicy.CoverageCurrency,
				CoverageDurationDays: basePolicy.CoverageDurationDays,
				Status:               basePolicy.Status,
			}
		}

		// Generate presigned URL if requested and document exists
		if filter.IncludePresignedURL && policy.SignedPolicyDocumentURL != nil && *policy.SignedPolicyDocumentURL != "" {
			expiryDuration := time.Duration(filter.URLExpiryHours) * time.Hour
			presignedURL, err := s.minioClient.GetPresignedURL(ctx, minio.Storage.PolicyDocuments, *policy.SignedPolicyDocumentURL, expiryDuration)
			if err != nil {
				slog.Warn("Failed to generate presigned URL",
					"policy_id", policy.ID,
					"document_url", *policy.SignedPolicyDocumentURL,
					"error", err)
			} else {
				policyDetail.PresignedDocumentURL = &presignedURL
				expiryTime := time.Now().Add(expiryDuration)
				policyDetail.PresignedURLExpiryTime = &expiryTime
			}
		}

		policiesWithDetails = append(policiesWithDetails, policyDetail)
	}

	return &models.RegisteredPolicyFilterResponse{
		Policies:   policiesWithDetails,
		TotalCount: len(policiesWithDetails),
		Filters:    filter,
	}, nil
}

func (s *RegisteredPolicyService) GetStatsOverview(ownerID string) (models.FarmStatsOverview, error) {
	activeFarmCount, err := s.farmService.CountActiveFarmsByOwnerID(ownerID)
	if err != nil {
		slog.Error("failed to count active farms", "owner_id", ownerID, "error", err)
		return models.FarmStatsOverview{}, err
	}

	inactiveFarmCount, err := s.farmService.CountInactiveFarmsByOwnerID(ownerID)
	if err != nil {
		slog.Error("failed to count inactive farms", "owner_id", ownerID, "error", err)
		return models.FarmStatsOverview{}, err
	}

	activeRegisteredPolicyCount, err := s.registeredPolicyRepo.CountActivePoliciesByFarmerID(ownerID)
	if err != nil {
		slog.Error("failed to count active registered policies", "owner_id", ownerID, "error", err)
		return models.FarmStatsOverview{}, err
	}
	return models.FarmStatsOverview{
		FarmActiveCount:       activeFarmCount,
		FarmInactiveCount:     inactiveFarmCount,
		RegisteredPolicyCount: activeRegisteredPolicyCount,
	}, nil
}

// GetAllMonitoringDataWithPolicyStatus retrieves all farm monitoring data with associated policy status
func (s *RegisteredPolicyService) GetAllMonitoringDataWithPolicyStatus(ctx context.Context, startTimestamp, endTimestamp *int64) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Info("Getting all farm monitoring data with policy status",
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	data, err := s.farmMonitoringDataRepo.GetAllWithPolicyStatus(ctx, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get monitoring data with policy status", "error", err)
		return nil, fmt.Errorf("failed to get monitoring data with policy status: %w", err)
	}

	slog.Info("Successfully retrieved monitoring data with policy status", "count", len(data))
	return data, nil
}

// GetMonitoringDataWithPolicyStatusByFarmID retrieves farm monitoring data with policy status for a specific farm
func (s *RegisteredPolicyService) GetMonitoringDataWithPolicyStatusByFarmID(ctx context.Context, farmID uuid.UUID, startTimestamp, endTimestamp *int64) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Info("Getting farm monitoring data with policy status",
		"farm_id", farmID,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	data, err := s.farmMonitoringDataRepo.GetAllWithPolicyStatusByFarmID(ctx, farmID, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get monitoring data with policy status by farm ID", "farm_id", farmID, "error", err)
		return nil, fmt.Errorf("failed to get monitoring data with policy status: %w", err)
	}

	slog.Info("Successfully retrieved monitoring data with policy status", "farm_id", farmID, "count", len(data))
	return data, nil
}

// GetMonitoringDataByFarmAndParameter retrieves monitoring data filtered by farm ID and parameter name
func (s *RegisteredPolicyService) GetMonitoringDataByFarmAndParameter(
	ctx context.Context,
	farmID uuid.UUID,
	parameterName models.DataSourceParameterName,
	startTimestamp, endTimestamp *int64,
) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Info("Getting farm monitoring data by farm and parameter",
		"farm_id", farmID,
		"parameter_name", parameterName,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	data, err := s.farmMonitoringDataRepo.GetByFarmIDAndParameterNameWithPolicyStatus(ctx, farmID, parameterName, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get monitoring data by farm and parameter",
			"farm_id", farmID,
			"parameter_name", parameterName,
			"error", err)
		return nil, fmt.Errorf("failed to get monitoring data: %w", err)
	}

	slog.Info("Successfully retrieved monitoring data",
		"farm_id", farmID,
		"parameter_name", parameterName,
		"count", len(data))
	return data, nil
}

// GetFarmerMonitoringData retrieves monitoring data for a farmer's own farm with ownership verification
func (s *RegisteredPolicyService) GetFarmerMonitoringData(
	ctx context.Context,
	userID string,
	farmID uuid.UUID,
	startTimestamp, endTimestamp *int64,
) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Info("Getting farmer monitoring data",
		"user_id", userID,
		"farm_id", farmID,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	// Verify farm ownership
	farm, err := s.farmService.GetByFarmID(ctx, farmID.String())
	if err != nil {
		slog.Error("Failed to get farm for ownership verification", "farm_id", farmID, "error", err)
		return nil, fmt.Errorf("failed to verify farm ownership: %w", err)
	}

	if farm.OwnerID != userID {
		slog.Warn("User does not own farm",
			"user_id", userID,
			"farm_id", farmID,
			"farm_owner_id", farm.OwnerID)
		return nil, fmt.Errorf("user does not own this farm")
	}

	// Get monitoring data
	data, err := s.farmMonitoringDataRepo.GetAllWithPolicyStatusByFarmID(ctx, farmID, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get farmer monitoring data", "farm_id", farmID, "error", err)
		return nil, fmt.Errorf("failed to get monitoring data: %w", err)
	}

	slog.Info("Successfully retrieved farmer monitoring data",
		"user_id", userID,
		"farm_id", farmID,
		"count", len(data))
	return data, nil
}

func (s *RegisteredPolicyService) GetUnderwritingByPolicyID(policyID uuid.UUID) ([]models.RegisteredPolicyUnderwriting, error) {
	return s.registeredPolicyRepo.GetUnderwritingsByPolicyID(policyID)
}

func (s *RegisteredPolicyService) GetUnderwritingByPolicyIDAndFarmerID(policyID uuid.UUID, farmerID string) ([]models.RegisteredPolicyUnderwriting, error) {
	return s.registeredPolicyRepo.GetUnderwritingsByPolicyIDAndFarmerID(policyID, farmerID)
}

func (s *RegisteredPolicyService) GetAllUnderwriting() ([]models.RegisteredPolicyUnderwriting, error) {
	return s.registeredPolicyRepo.GetAllUnderwriting()
}

func (s *RegisteredPolicyService) GetInsuranceProviderIDByID(policyID uuid.UUID) (string, error) {
	return s.registeredPolicyRepo.GetInsuranceProviderIDByID(policyID)
}

// GetFarmerMonitoringDataByParameter retrieves monitoring data for a specific parameter from a farmer's own farm
func (s *RegisteredPolicyService) GetFarmerMonitoringDataByParameter(
	ctx context.Context,
	userID string,
	farmID uuid.UUID,
	parameterName models.DataSourceParameterName,
	startTimestamp, endTimestamp *int64,
) ([]models.FarmMonitoringDataWithPolicyStatus, error) {
	slog.Info("Getting farmer monitoring data by parameter",
		"user_id", userID,
		"farm_id", farmID,
		"parameter_name", parameterName,
		"start_timestamp", startTimestamp,
		"end_timestamp", endTimestamp)

	// Verify farm ownership
	farm, err := s.farmService.GetByFarmID(ctx, farmID.String())
	if err != nil {
		slog.Error("Failed to get farm for ownership verification", "farm_id", farmID, "error", err)
		return nil, fmt.Errorf("failed to verify farm ownership: %w", err)
	}

	if farm.OwnerID != userID {
		slog.Warn("User does not own farm",
			"user_id", userID,
			"farm_id", farmID,
			"farm_owner_id", farm.OwnerID)
		return nil, fmt.Errorf("user does not own this farm")
	}

	// Get monitoring data by parameter
	data, err := s.farmMonitoringDataRepo.GetByFarmIDAndParameterNameWithPolicyStatus(ctx, farmID, parameterName, startTimestamp, endTimestamp)
	if err != nil {
		slog.Error("Failed to get farmer monitoring data by parameter",
			"farm_id", farmID,
			"parameter_name", parameterName,
			"error", err)
		return nil, fmt.Errorf("failed to get monitoring data: %w", err)
	}

	slog.Info("Successfully retrieved farmer monitoring data by parameter",
		"user_id", userID,
		"farm_id", farmID,
		"parameter_name", parameterName,
		"count", len(data))
	return data, nil
}

// CreatePartnerPolicyUnderwriting creates an underwriting record, updates policy status, and dispatches monitoring job
func (s *RegisteredPolicyService) CreatePartnerPolicyUnderwriting(
	ctx context.Context,
	policyID uuid.UUID,
	req models.CreatePartnerPolicyUnderwritingRequest,
	validatedBy string,
) (*models.CreatePartnerPolicyUnderwritingResponse, error) {
	slog.Info("Creating partner policy underwriting",
		"policy_id", policyID,
		"underwriting_status", req.UnderwritingStatus,
		"validated_by", validatedBy)

	// 1. Get the policy to verify it exists and get required info
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("Failed to get policy for underwriting", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if policy.UnderwritingStatus != models.UnderwritingPending {
		return nil, fmt.Errorf("invalid operation: policy underwriting status=%v", policy.UnderwritingStatus)
	}

	if policy.Status != models.PolicyPendingReview {
		return nil, fmt.Errorf("invalid operation: policy status=%v", policy.Status)
	}

	// 2. Create underwriting record
	underwriting := &models.RegisteredPolicyUnderwriting{
		ID:                  uuid.New(),
		RegisteredPolicyID:  policyID,
		ValidationTimestamp: time.Now().Unix(),
		UnderwritingStatus:  req.UnderwritingStatus,
		Recommendations:     req.Recommendations,
		Reason:              req.Reason,
		ReasonEvidence:      req.ReasonEvidence,
		ValidatedBy:         &validatedBy,
		ValidationNotes:     req.ValidationNotes,
	}

	if err := s.registeredPolicyRepo.CreateUnderwriting(underwriting); err != nil {
		slog.Error("Failed to create underwriting record", "policy_id", policyID, "error", err)
		return nil, fmt.Errorf("failed to create underwriting: %w", err)
	}

	// 4. If approved, update policy status
	responseMessage := "Underwriting record created"
	policy.UnderwritingStatus = req.UnderwritingStatus
	if req.UnderwritingStatus == models.UnderwritingApproved {
		// Update policy status to active and set coverage start date
		policy.Status = models.PolicyPendingPayment
		if err := s.registeredPolicyRepo.Update(policy); err != nil {
			slog.Error("Failed to update policy status to active", "policy_id", policyID, "error", err)
			return nil, fmt.Errorf("failed to update policy status: %w", err)
		}

		responseMessage = "Underwriting approved, policy activated, and monitoring job dispatched"
	} else if req.UnderwritingStatus == models.UnderwritingRejected {
		// Update policy status to rejected
		policy.Status = models.PolicyRejected
		if err := s.registeredPolicyRepo.Update(policy); err != nil {
			slog.Error("Failed to update policy status to rejected", "policy_id", policyID, "error", err)
			return nil, fmt.Errorf("failed to update policy status: %w", err)
		}
		responseMessage = "Underwriting rejected, policy rejected"
	}

	go func() {
		for {
			err := s.notievent.NotifyUnderwritingCompleted(context.Background(), policy.FarmerID, policy.PolicyNumber)
			if err == nil {
				slog.Info("policy underwriting notification sent", "policy id", policy.ID)
				return
			}
			slog.Error("error sending policy underwriting notification", "error", err)
			time.Sleep(10 * time.Second)
		}
	}()

	slog.Info("Successfully created partner policy underwriting",
		"underwriting_id", underwriting.ID,
		"policy_id", policyID,
		"status", req.UnderwritingStatus,
		"message", responseMessage)

	// Payment Window
	if req.UnderwritingStatus == models.UnderwritingApproved {

		basePolicy, err := s.basePolicyRepo.GetBasePolicyByID(policy.BasePolicyID)
		if err != nil {
			slog.Error("CRITICAL: retrieve base policy failed", "error", err)
		}

		go func() {
			slog.Info("underwriting approved, start payment window: 24h before policy auto cancel", "policy_id", policyID)
			time.Sleep(time.Duration(*basePolicy.MaxPremiumPaymentProlong) * time.Minute) // TODO: Change to hour
			policy, err := s.registeredPolicyRepo.GetByID(policyID)
			if err != nil {
				slog.Error("CRITICAL: policy not found skip payment window", "error", err)
				return
			}
			if policy.Status != models.PolicyPendingPayment {
				slog.Info("policy status invalid skip payment window", "status", policy.Status)
				return
			}
			policy.Status = models.PolicyCancelled
			err = s.registeredPolicyRepo.Update(policy)
			if err != nil {
				slog.Info("error updating policy", "error", err)
				return
			}
			slog.Info("payment pending due: policy status set to cancelled", "policy_id", policy.ID)

			go func() {
				for {
					err := s.notievent.NotifyPolicyCancel(context.Background(), policy.FarmerID, policy.PolicyNumber, "Quá hạn thanh toán")
					if err == nil {
						slog.Info("policy cancel notification sent", "policy id", policy.ID)
						return
					}
					slog.Error("error sending policy cancel notification", "error", err)
					time.Sleep(10 * time.Second)
				}
			}()
		}()
	}

	return &models.CreatePartnerPolicyUnderwritingResponse{
		UnderwritingID:     underwriting.ID.String(),
		PolicyID:           policyID.String(),
		UnderwritingStatus: req.UnderwritingStatus,
		ValidatedBy:        validatedBy,
		Message:            responseMessage,
	}, nil
}

func (s *RegisteredPolicyService) GetInsurancePartnerProfile(token string) (map[string]any, error) {
	url := "http://profile-service:8087/profile/protected/api/v1/insurance-partners/me/profile"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Error creating request for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error making request for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response body for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		slog.Error("insurance partner profile not found", "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("insurance partner profile not found %d, body: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected status code for insurance partner profile", "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("Error parsing JSON for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return result, nil
}

func (s *RegisteredPolicyService) UpdateRegisteredPolicy(policy *models.RegisteredPolicy) error {
	return s.registeredPolicyRepo.Update(policy)
}

func (s *RegisteredPolicyService) GetPartnerID(result map[string]interface{}) (string, error) {
	// Lấy object "data"
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data field not found or invalid")
	}

	// Lấy "partner_id" từ data
	partnerID, ok := data["partner_id"].(string)
	if !ok {
		return "", fmt.Errorf("partner_id not found or not a string")
	}

	return partnerID, nil
}

func (s *RegisteredPolicyService) GetMonthlyDataCost(
	request models.MonthlyDataCostRequest,
	insuranceProviderID string,
) (*models.MonthlyDataCostResponse, error) {
	slog.Info("Calculating monthly data cost",
		"provider_id", insuranceProviderID,
		"month", request.Month,
		"year", request.Year,
		"direction", request.Direction,
		"status", request.UnderwritingStatus,
		"underwriting_status", request.UnderwritingStatus,
	)

	// Get base policy costs
	basePolicyCosts, err := s.registeredPolicyRepo.GetMonthlyDataCostByProvider(
		insuranceProviderID,
		request.Year,
		request.Month,
		request.Direction,
		request.Status,
		request.UnderwritingStatus,
		request.OrderBy,
	)
	if err != nil {
		return nil, err
	}

	// Calculate totals
	var totalActivePolicies int
	var totalDataCost float64

	for _, cost := range basePolicyCosts {
		totalActivePolicies += cost.ActivePolicyCount
		totalDataCost += cost.SumTotalDataCost
	}

	response := &models.MonthlyDataCostResponse{
		InsuranceProviderID:     insuranceProviderID,
		Month:                   request.Month,
		Year:                    request.Year,
		BasePolicyCosts:         basePolicyCosts,
		TotalActivePolicies:     totalActivePolicies,
		TotalBasePolicyDataCost: totalDataCost,
		Currency:                "VND",
	}

	return response, nil
}

func (s *RegisteredPolicyService) GetAllUserIDsFromInsuranceProvider(providerID string, token string) ([]string, error) {
	url := "http://profile-service:8087/profile/public/api/v1/users/" + providerID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Error creating request for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error making request for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response body for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected status code for insurance partner profile", "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("Error parsing JSON for insurance partner profile", "error", err)
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}
	profileDatas, ok := result["data"].([]any)
	if !ok {
		slog.Error("profile data not fould", "full response", result)
		return nil, fmt.Errorf("profile data not fould")
	}
	userID := []string{}
	for _, userRawData := range profileDatas {
		userData := userRawData.(map[string]any)
		id, ok := userData["user_id"].(string)
		if !ok {
			slog.Warn("user id not found in profile data", "profile data", userRawData)
			continue
		}
		userID = append(userID, id)
	}
	return userID, nil
}

func (s *RegisteredPolicyService) GetByBasePolicy(ctx context.Context, basePolicyID uuid.UUID) ([]models.RegisteredPolicy, error) {
	return s.registeredPolicyRepo.GetByBasePolicyID(ctx, basePolicyID)
}
