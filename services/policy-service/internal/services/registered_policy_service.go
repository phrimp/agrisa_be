package services

import (
	utils "agrisa_utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"policy-service/internal/ai/gemini"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/worker"
	"strconv"
	"strings"
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

// ============================================================================
// FARM MONITORING DATA FETCH TYPES
// ============================================================================

// DataRequest contains information needed to fetch monitoring data
type DataRequest struct {
	DataSource                   models.DataSource
	FarmID                       uuid.UUID
	FarmCoordinates              [][]float64 // GeoJSON polygon coordinates (first ring only)
	AgroPolygonID                string
	StartDate                    string // YYYY-MM-DD format
	EndDate                      string // YYYY-MM-DD format
	BasePolicyTriggerConditionID uuid.UUID
	MaxCloudCover                float64
	MaxImages                    int
	IncludeComponents            bool
}

// DataResponse contains the result of a monitoring data fetch operation
type DataResponse struct {
	DataSource     models.DataSource
	MonitoringData []models.FarmMonitoringData
	AgroPolygonID  string // Polygon ID from weather API for farm update
	Err            error
	SkipReason     string // Optional: reason for skipping (e.g., "unsupported_parameter")
}

type WeatherAPIResponse struct {
	PolygonID         string      `json:"polygon_id"`
	PolygonName       string      `json:"polygon_name"`
	PolygonCenter     []float64   `json:"polygon_center"`
	PolygonArea       float64     `json:"polygon_area"`
	PolygonReused     bool        `json:"polygon_reused"`      // True if existing polygon was reused
	PolygonCreatedNew bool        `json:"polygon_created_new"` // True if new polygon was created
	TimeRange         TimeRange   `json:"time_range"`
	Data              []DataPoint `json:"data"`
	TotalDataValue    float64     `json:"total_data_value"`
	DataPointCount    int         `json:"data_point_count"`
}

type DataPoint struct {
	Dt    int64   `json:"dt"`    // Unix timestamp
	Data  float64 `json:"data"`  // Precipitation in mm
	Count int     `json:"count"` // Number of measurements
	Unit  string  `json:"unit"`
}

type TimeRange struct {
	Start int64 `json:"start"` // Unix timestamp
	End   int64 `json:"end"`   // Unix timestamp
}

// SatelliteAPIResponse matches the satellite-data-service response structure
// StatValue represents a statistical value with unit and range
type StatValue struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Range string  `json:"range,omitempty"`
}

type SatelliteAPIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Summary struct {
			ImagesProcessed int `json:"images_processed"`
		} `json:"summary"`
		Images []struct {
			ImageIndex      int    `json:"image_index"`
			AcquisitionDate string `json:"acquisition_date"`
			CloudCover      struct {
				Value float64 `json:"value"`
				Unit  string  `json:"unit"`
			} `json:"cloud_cover"`
			// Support both ndmi_statistics and ndvi_statistics formats
			NDMIStatistics *struct {
				Mean   *StatValue `json:"mean,omitempty"`
				Median *StatValue `json:"median,omitempty"`
				Min    *StatValue `json:"min,omitempty"`
				Max    *StatValue `json:"max,omitempty"`
				StdDev *StatValue `json:"std_dev,omitempty"`
			} `json:"ndmi_statistics,omitempty"`
			NDVIStatistics *struct {
				Mean   *StatValue `json:"mean,omitempty"`
				Median *StatValue `json:"median,omitempty"`
				Min    *StatValue `json:"min,omitempty"`
				Max    *StatValue `json:"max,omitempty"`
				StdDev *StatValue `json:"std_dev,omitempty"`
			} `json:"ndvi_statistics,omitempty"`
			ComponentData *struct {
				NIR  *float64 `json:"nir,omitempty"`
				Red  *float64 `json:"red,omitempty"`
				SWIR *float64 `json:"swir,omitempty"`
			} `json:"component_data,omitempty"`
		} `json:"images"`
	} `json:"data"`
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
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("error retrieving policy by id", "id", policyID, "error", err)
		return fmt.Errorf("error retrieving policy by id: %w", err)
	}

	basePolicyID := policy.BasePolicyID

	farmID := policy.FarmID

	// Extract optional check_policy parameter (defaults to false)
	checkPolicy, _ := params["check_policy"].(bool)

	// Extract optional inject_test parameter for testing claim generation
	// inject_test should be an array of FarmMonitoringData objects
	var testMonitoringData []models.FarmMonitoringData
	if injectTestRaw, ok := params["inject_test"]; ok {
		// Parse test data from parameter
		if testDataArray, ok := injectTestRaw.([]interface{}); ok {
			for _, item := range testDataArray {
				if testDataMap, ok := item.(map[string]interface{}); ok {
					// Convert map to FarmMonitoringData
					testData := parseFarmMonitoringDataFromMap(testDataMap, farmID)
					testMonitoringData = append(testMonitoringData, testData)
				}
			}
			slog.Info("Test monitoring data injected for testing",
				"policy_id", policyID,
				"farm_id", farmID,
				"test_data_count", len(testMonitoringData))
		}
	}

	// Get triggers and conditions for this base policy
	triggers, err := s.basePolicyRepo.GetBasePolicyTriggersByPolicyID(basePolicyID)
	if err != nil {
		return fmt.Errorf("retrieve triggers from base policy id failed: %w", err)
	}

	if len(triggers) == 0 {
		return fmt.Errorf("no triggers found for base policy %s", basePolicyID)
	}

	riskAnalysisJob := worker.JobPayload{
		JobID:      uuid.NewString(),
		Type:       "risk-analysis",
		Params:     map[string]any{"registered_policy_id": policyID, "force_reanalysis": false},
		MaxRetries: 2,
		OneTime:    true,
		RunNow:     true,
	}

	scheduler, ok := s.workerManager.GetSchedulerByPolicyID(policyID)
	if !ok {
		slog.Error("error get farm-imagery scheduler", "error", "scheduler doesn't exist")
	}

	// Build list of conditions with their data sources
	type conditionWithDataSource struct {
		ConditionID uuid.UUID
		DataSource  models.DataSource
	}
	var conditionsWithDataSources []conditionWithDataSource

	for _, trigger := range triggers {
		conditions, err := s.basePolicyRepo.GetBasePolicyTriggerConditionsByTriggerID(trigger.ID)
		if err != nil {
			slog.Warn("Failed to get conditions for trigger", "trigger_id", trigger.ID, "error", err)
			continue
		}

		for _, cond := range conditions {
			ds, err := s.dataSourceRepo.GetDataSourceByID(cond.DataSourceID)
			if err != nil {
				slog.Warn("Failed to get data source for condition",
					"condition_id", cond.ID,
					"data_source_id", cond.DataSourceID,
					"error", err)
				continue
			}
			conditionsWithDataSources = append(conditionsWithDataSources, conditionWithDataSource{
				ConditionID: cond.ID,
				DataSource:  *ds,
			})
		}
	}

	if len(conditionsWithDataSources) == 0 {
		return fmt.Errorf("no conditions with data sources found for base policy %s", basePolicyID)
	}

	startDateFloat, ok := params["start_date"].(float64)
	if !ok {
		slog.Error("missing or invalid start_date parameter", "farm_id", farmID)
		return fmt.Errorf("missing or invalid start_date parameter")
	}
	startDate := int64(startDateFloat)

	endDateFloat, ok := params["end_date"].(float64)
	if !ok {
		slog.Error("missing or invalid end_date parameter", "farm_id", farmID)
		return fmt.Errorf("missing or invalid end_date parameter")
	}
	endDate := int64(endDateFloat)

	if endDate == 0 {
		endDate = time.Now().Unix()
	}
	if startDate == 0 {
		trigger, err := s.basePolicyRepo.GetBasePolicyTriggersByPolicyID(basePolicyID)
		if err != nil || len(trigger) == 0 {
			slog.Error("base policy trigger retrieve failed", "error", err, "trigger length", len(trigger))

			return fmt.Errorf("base policy trigger retrieve failed: %w", err)
		}
		switch trigger[0].MonitorFrequencyUnit {
		case models.MonitorFrequencyHour:
			startDate = time.Now().Add(-time.Duration(trigger[0].MonitorInterval) * time.Hour).Unix()
		case models.MonitorFrequencyDay:
			startDate = time.Now().AddDate(0, 0, -trigger[0].MonitorInterval).Unix()
		case models.MonitorFrequencyWeek:
			startDate = time.Now().AddDate(0, 0, -trigger[0].MonitorInterval*7).Unix()
		case models.MonitorFrequencyMonth:
			startDate = time.Now().AddDate(0, -trigger[0].MonitorInterval, 0).Unix()
		case models.MonitorFrequencyYear:
			startDate = time.Now().AddDate(-trigger[0].MonitorInterval, 0, 0).Unix()
		default:
			slog.Error("unsupported monitor frequency unit",
				"unit", trigger[0].MonitorFrequencyUnit,
				"basePolicyID", basePolicyID)
			return fmt.Errorf("unsupported monitor frequency unit: %v", trigger[0].MonitorFrequencyUnit)
		}
	}

	ctx := context.Background()

	// Initialize result variables
	var allMonitoringData []models.FarmMonitoringData
	var agroPolygonID string

	// If test data is injected, skip API fetching and use test data directly
	if len(testMonitoringData) > 0 {
		slog.Info("Using injected test data, skipping API fetch",
			"policy_id", policyID,
			"farm_id", farmID,
			"test_records", len(testMonitoringData))

		allMonitoringData = testMonitoringData

		// Store test monitoring data in database for consistency
		if err := s.farmMonitoringDataRepo.CreateBatch(ctx, allMonitoringData); err != nil {
			return fmt.Errorf("failed to store test monitoring data: %w", err)
		}
		slog.Info("Test monitoring data stored successfully",
			"farm_id", farmID,
			"total_records", len(allMonitoringData))

		// Check policy trigger conditions with test data
		if checkPolicy && len(allMonitoringData) > 0 {
			slog.Info("Checking policy trigger conditions with test data",
				"policy_id", policyID,
				"farm_id", farmID,
				"data_points", len(allMonitoringData))

			// Evaluate trigger conditions against test data
			triggeredConditions := s.evaluateTriggerConditions(ctx, triggers, allMonitoringData, farmID)

			if len(triggeredConditions) > 0 {
				slog.Info("Trigger conditions satisfied with test data",
					"policy_id", policyID,
					"triggered_conditions", len(triggeredConditions))

				// Generate Claim
				claim, err := s.generateClaimFromTrigger(ctx, policyID, basePolicyID, farmID, triggers[0].ID, triggeredConditions)
				if err != nil {
					slog.Error("Failed to generate claim from trigger",
						"policy_id", policyID,
						"error", err)
				} else {
					slog.Info("Claim generated successfully from test data",
						"claim_id", claim.ID,
						"claim_number", claim.ClaimNumber,
						"claim_amount", claim.ClaimAmount,
						"policy_id", policyID)

					for _, tc := range triggeredConditions {
						slog.Info("Triggered condition details",
							"claim_id", claim.ID,
							"condition_id", tc.ConditionID,
							"parameter", tc.ParameterName,
							"measured_value", tc.MeasuredValue,
							"threshold_value", tc.ThresholdValue,
							"operator", tc.Operator,
							"consecutive_days", tc.ConsecutiveDays,
							"is_early_warning", tc.IsEarlyWarning)
					}
				}
			}
		}

		// Schedule risk analysis
		scheduler.AddJob(riskAnalysisJob)
		return nil
	}

	// Load farm to get boundary coordinates
	var farm *models.Farm
	farm, err = s.farmService.GetByFarmID(ctx, farmID.String())
	if err != nil {
		return fmt.Errorf("failed to load farm: %w", err)
	}

	if farm.Boundary == nil {
		return fmt.Errorf("farm boundary is required for monitoring data fetch")
	}

	// Extract coordinates from GeoJSON polygon (first ring only)
	farmCoordinates := extractPolygonCoordinates(farm.Boundary)
	if len(farmCoordinates) < 3 {
		return fmt.Errorf("invalid farm boundary: need at least 3 coordinates")
	}

	// Convert end date to YYYY-MM-DD format (start date is calculated per-parameter)
	endDateStr := unixToDateString(endDate)

	// Initialize worker pool
	numWorkers := min(10, len(conditionsWithDataSources))
	jobs := make(chan DataRequest, len(conditionsWithDataSources))
	results := make(chan DataResponse, len(conditionsWithDataSources))

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 300 * time.Second,
	}

	// Start workers
	for range numWorkers {
		go fetchMonitoringDataWorker(jobs, results, httpClient)
	}

	// Enqueue jobs for each condition with data source
	// Track how many jobs were actually enqueued (some may be skipped if up to date)
	jobsEnqueued := 0
	for _, cds := range conditionsWithDataSources {
		// Check for existing data per parameter and adjust start date to fetch only missing data
		parameterName := string(cds.DataSource.ParameterName)
		paramStartDate := startDate

		latestTimestamp, err := s.farmMonitoringDataRepo.GetLatestTimestampByFarmIDAndParameterName(ctx, farmID, parameterName)
		if err != nil {
			slog.Warn("error getting latest timestamp for farm data by parameter",
				"farm_id", farmID,
				"parameter_name", parameterName,
				"error", err)
			// Continue with original startDate if error
		} else if latestTimestamp > 0 {
			// Adjust start date to day after latest data
			adjustedStartDate := latestTimestamp + (24 * 60 * 60) // Add 1 day
			if adjustedStartDate >= endDate {
				slog.Info("existing monitor farm data is up to date for parameter, skipping",
					"farm_id", farmID,
					"parameter_name", parameterName,
					"latest_data", latestTimestamp,
					"requested_end", endDate)
				continue // Skip this parameter, already up to date
			}
			slog.Info("adjusting start date to fetch only missing data for parameter",
				"farm_id", farmID,
				"parameter_name", parameterName,
				"original_start", startDate,
				"adjusted_start", adjustedStartDate,
				"latest_existing_data", latestTimestamp,
				"end_date", endDate)
			paramStartDate = adjustedStartDate
		}

		// Convert adjusted start date to string format
		paramStartDateStr := unixToDateString(paramStartDate)

		jobs <- DataRequest{
			DataSource:                   cds.DataSource,
			FarmID:                       farmID,
			FarmCoordinates:              farmCoordinates,
			AgroPolygonID:                farm.AgroPolygonID,
			StartDate:                    paramStartDateStr,
			EndDate:                      endDateStr,
			BasePolicyTriggerConditionID: cds.ConditionID,
			MaxCloudCover:                100.0,
			MaxImages:                    10,
			IncludeComponents:            true,
		}
		jobsEnqueued++
	}
	close(jobs)

	// If all parameters are up to date, schedule risk analysis and return
	if jobsEnqueued == 0 {
		slog.Info("all parameters are up to date, skipping fetch and scheduling risk analysis",
			"farm_id", farmID)
		scheduler.AddJob(riskAnalysisJob)
		return nil
	}

	// Collect results from all workers
	errorSummary := make(map[models.DataSourceParameterName]error)
	skipSummary := make(map[models.DataSourceParameterName]string)

	for i := 0; i < jobsEnqueued; i++ {
		resp := <-results

		if resp.Err != nil {
			errorSummary[resp.DataSource.ParameterName] = resp.Err
			slog.Error("Data source fetch failed",
				"parameter", resp.DataSource.ParameterName,
				"data_source_id", resp.DataSource.ID,
				"error", resp.Err)
		} else if resp.SkipReason != "" {
			skipSummary[resp.DataSource.ParameterName] = resp.SkipReason
			slog.Warn("Data source skipped",
				"parameter", resp.DataSource.ParameterName,
				"reason", resp.SkipReason)
		} else {
			allMonitoringData = append(allMonitoringData, resp.MonitoringData...)

			// Capture polygon ID from weather API response (if available)
			if resp.AgroPolygonID != "" && agroPolygonID == "" {
				agroPolygonID = resp.AgroPolygonID
				slog.Info("Captured Agro polygon ID from weather API",
					"polygon_id", agroPolygonID,
					"parameter", resp.DataSource.ParameterName)
			}

			slog.Info("Data source fetch succeeded",
				"parameter", resp.DataSource.ParameterName,
				"records_fetched", len(resp.MonitoringData))
		}
	}

	// Store monitoring data in database (batch insert)
	if len(allMonitoringData) > 0 {
		if err := s.farmMonitoringDataRepo.CreateBatch(ctx, allMonitoringData); err != nil {
			return fmt.Errorf("failed to store monitoring data: %w", err)
		}
		slog.Info("Monitoring data stored successfully",
			"farm_id", farmID,
			"total_records", len(allMonitoringData))
	}

	// Update farm's AgroPolygonID if we received one from weather API
	if agroPolygonID != "" && farm.AgroPolygonID != agroPolygonID {
		// Update farm with new polygon ID
		farm.AgroPolygonID = agroPolygonID
		if err := s.farmService.UpdateFarm(ctx, farm, "system", farmID.String()); err != nil {
			// Log error but don't fail the entire job
			slog.Error("Failed to update farm AgroPolygonID",
				"farm_id", farmID,
				"polygon_id", agroPolygonID,
				"error", err)
		} else {
			slog.Info("Updated farm AgroPolygonID successfully",
				"farm_id", farmID,
				"polygon_id", agroPolygonID)
		}
	}

	// Log summary (only for non-test data path)
	if len(testMonitoringData) == 0 {
		successCount := len(conditionsWithDataSources) - len(errorSummary) - len(skipSummary)
		successRate := float64(successCount) / float64(len(conditionsWithDataSources)) * 100

		slog.Info("Farm monitoring data fetch completed",
			"policy_id", policyID,
			"farm_id", farmID,
			"base_policy_id", basePolicyID,
			"total_conditions", len(conditionsWithDataSources),
			"successful", successCount,
			"failed", len(errorSummary),
			"skipped", len(skipSummary),
			"success_rate", fmt.Sprintf("%.1f%%", successRate),
			"records_stored", len(allMonitoringData))

		// Only fail if ALL sources failed
		if len(errorSummary) == len(conditionsWithDataSources) {
			return fmt.Errorf("all %d data sources failed to fetch", len(conditionsWithDataSources))
		}
	}

	// Check policy trigger conditions if enabled
	if checkPolicy && len(allMonitoringData) > 0 {
		slog.Info("Checking policy trigger conditions",
			"policy_id", policyID,
			"farm_id", farmID,
			"data_points", len(allMonitoringData))

		// Evaluate trigger conditions against fetched data
		triggeredConditions := s.evaluateTriggerConditions(ctx, triggers, allMonitoringData, farmID)

		if len(triggeredConditions) > 0 {
			slog.Info("Trigger conditions satisfied",
				"policy_id", policyID,
				"triggered_conditions", len(triggeredConditions))

			// Generate Claim
			claim, err := s.generateClaimFromTrigger(ctx, policyID, basePolicyID, farmID, triggers[0].ID, triggeredConditions)
			if err != nil {
				slog.Error("Failed to generate claim from trigger",
					"policy_id", policyID,
					"error", err)
			} else {
				slog.Info("Claim generated successfully",
					"claim_id", claim.ID,
					"claim_number", claim.ClaimNumber,
					"claim_amount", claim.ClaimAmount,
					"policy_id", policyID)

				// ================================================================
				// NOTIFICATION PLACEHOLDER - User will implement notifications
				// ================================================================
				// TODO: Send notification to farmer about claim generation
				// notification.SendToFarmer(claim.FarmerID, NotificationClaimGenerated, claim)

				// TODO: Send notification to insurance provider for review
				// notification.SendToProvider(policy.InsuranceProviderID, NotificationClaimPendingReview, claim)

				// TODO: Send notification to system administrators if claim amount exceeds threshold
				// if claim.ClaimAmount > HIGH_VALUE_CLAIM_THRESHOLD {
				//     notification.SendToAdmins(NotificationHighValueClaim, claim)
				// }
				// ================================================================

				for _, tc := range triggeredConditions {
					slog.Info("Triggered condition details",
						"claim_id", claim.ID,
						"condition_id", tc.ConditionID,
						"parameter", tc.ParameterName,
						"measured_value", tc.MeasuredValue,
						"threshold_value", tc.ThresholdValue,
						"operator", tc.Operator,
						"consecutive_days", tc.ConsecutiveDays,
						"is_early_warning", tc.IsEarlyWarning)
				}
			}
		}
	}

	scheduler.AddJob(riskAnalysisJob)

	return nil
}

// TriggeredCondition represents a condition that has been satisfied
type TriggeredCondition struct {
	ConditionID           uuid.UUID
	ParameterName         models.DataSourceParameterName
	MeasuredValue         float64
	ThresholdValue        float64
	Operator              models.ThresholdOperator
	Timestamp             int64
	BaselineValue         *float64 // Baseline value for change-based conditions
	ConsecutiveDays       int      // Number of consecutive days condition was met
	IsEarlyWarning        bool     // True if only early warning threshold was breached
	EarlyWarningThreshold *float64 // Early warning threshold value if applicable
}

// generateClaimFromTrigger creates a claim when trigger conditions are satisfied
func (s *RegisteredPolicyService) generateClaimFromTrigger(
	ctx context.Context,
	policyID uuid.UUID,
	basePolicyID uuid.UUID,
	farmID uuid.UUID,
	triggerID uuid.UUID,
	triggeredConditions []TriggeredCondition,
) (*models.Claim, error) {
	slog.Info("Generating claim from trigger",
		"policy_id", policyID,
		"trigger_id", triggerID,
		"conditions_count", len(triggeredConditions))

	// Check for duplicate claim within the last 24 hours
	recentClaim, err := s.registeredPolicyRepo.GetRecentClaimByPolicyAndTrigger(
		policyID,
		triggerID,
		24*60*60, // 24 hours in seconds
	)
	if err != nil {
		slog.Warn("Failed to check for recent claims", "error", err)
		// Continue anyway - better to potentially duplicate than miss a claim
	}
	if recentClaim != nil {
		slog.Info("Skipping claim generation - recent claim exists",
			"policy_id", policyID,
			"trigger_id", triggerID,
			"existing_claim_id", recentClaim.ID,
			"existing_claim_number", recentClaim.ClaimNumber,
			"existing_trigger_timestamp", recentClaim.TriggerTimestamp)
		return recentClaim, nil // Return existing claim instead of creating duplicate
	}

	// Get registered policy for coverage amount
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered policy: %w", err)
	}

	// Get base policy for payout calculation parameters
	basePolicy, err := s.basePolicyRepo.GetBasePolicyByID(basePolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base policy: %w", err)
	}

	// Calculate payout amounts
	fixPayout, thresholdPayout, totalPayout, overThresholdValue := s.calculateClaimPayouts(
		policy,
		basePolicy,
		triggeredConditions,
	)

	// Build evidence summary from triggered conditions
	evidenceSummary := s.buildEvidenceSummary(triggeredConditions)

	// Generate claim number
	claimNumber := "CLM" + utils.GenerateRandomStringWithLength(9)

	// Set auto-approval deadline (e.g., 7 days from now)
	autoApprovalDeadline := time.Now().AddDate(0, 0, 7).Unix()

	// Create the claim
	claim := &models.Claim{
		ID:                        uuid.New(),
		ClaimNumber:               claimNumber,
		RegisteredPolicyID:        policyID,
		BasePolicyID:              basePolicyID,
		FarmID:                    farmID,
		BasePolicyTriggerID:       triggerID,
		TriggerTimestamp:          time.Now().Unix(),
		OverThresholdValue:        overThresholdValue,
		CalculatedFixPayout:       &fixPayout,
		CalculatedThresholdPayout: &thresholdPayout,
		ClaimAmount:               totalPayout,
		Status:                    models.ClaimGenerated,
		AutoGenerated:             true,
		AutoApprovalDeadline:      &autoApprovalDeadline,
		AutoApproved:              false,
		EvidenceSummary:           evidenceSummary,
	}

	// Save claim to database
	if err := s.registeredPolicyRepo.CreateClaim(claim); err != nil {
		return nil, fmt.Errorf("failed to create claim: %w", err)
	}

	slog.Info("Claim generated and saved",
		"claim_id", claim.ID,
		"claim_number", claim.ClaimNumber,
		"fix_payout", fixPayout,
		"threshold_payout", thresholdPayout,
		"total_payout", totalPayout,
		"over_threshold_value", overThresholdValue)

	return claim, nil
}

// calculateClaimPayouts calculates the payout amounts for a claim
func (s *RegisteredPolicyService) calculateClaimPayouts(
	policy *models.RegisteredPolicy,
	basePolicy *models.BasePolicy,
	triggeredConditions []TriggeredCondition,
) (fixPayout float64, thresholdPayout float64, totalPayout float64, overThresholdValue *float64) {
	// Calculate fix payout (base payout amount)
	fixPayout = float64(basePolicy.FixPayoutAmount) * basePolicy.PayoutBaseRate

	// Calculate over-threshold payout based on how much the condition exceeded the threshold
	var maxOverThreshold float64
	for _, tc := range triggeredConditions {
		if tc.IsEarlyWarning {
			continue // Skip early warnings for payout calculation
		}

		// Calculate how much the measured value exceeded the threshold
		var overAmount float64
		switch tc.Operator {
		case models.ThresholdGT, models.ThresholdGTE, models.ThresholdChangeGT:
			// For "greater than" operators, the over amount is measured - threshold
			overAmount = tc.MeasuredValue - tc.ThresholdValue
		case models.ThresholdLT, models.ThresholdLTE, models.ThresholdChangeLT:
			// For "less than" operators, the over amount is threshold - measured
			overAmount = tc.ThresholdValue - tc.MeasuredValue
		}

		if overAmount > maxOverThreshold {
			maxOverThreshold = overAmount
		}
	}

	if maxOverThreshold > 0 {
		overThresholdValue = &maxOverThreshold
		// Calculate threshold payout using the over-threshold multiplier
		thresholdPayout = maxOverThreshold * basePolicy.OverThresholdMultiplier
	}

	// Total payout is fix + threshold payout
	totalPayout = fixPayout + thresholdPayout

	// Apply payout cap if specified
	if basePolicy.PayoutCap != nil && totalPayout > float64(*basePolicy.PayoutCap) {
		totalPayout = float64(*basePolicy.PayoutCap)
		slog.Info("Payout capped",
			"original_total", fixPayout+thresholdPayout,
			"capped_total", totalPayout,
			"payout_cap", *basePolicy.PayoutCap)
	}

	// Ensure payout doesn't exceed coverage amount
	if totalPayout > policy.CoverageAmount {
		totalPayout = policy.CoverageAmount
		slog.Info("Payout limited to coverage amount",
			"calculated_total", fixPayout+thresholdPayout,
			"coverage_amount", policy.CoverageAmount)
	}

	return fixPayout, thresholdPayout, totalPayout, overThresholdValue
}

// buildEvidenceSummary creates a JSON summary of triggered conditions for the claim
func (s *RegisteredPolicyService) buildEvidenceSummary(triggeredConditions []TriggeredCondition) utils.JSONMap {
	evidence := utils.JSONMap{
		"triggered_at":      time.Now().Unix(),
		"conditions_count":  len(triggeredConditions),
		"generation_method": "automatic",
	}

	conditions := make([]map[string]interface{}, 0, len(triggeredConditions))
	for _, tc := range triggeredConditions {
		condEvidence := map[string]interface{}{
			"condition_id":     tc.ConditionID.String(),
			"parameter":        string(tc.ParameterName),
			"measured_value":   tc.MeasuredValue,
			"threshold_value":  tc.ThresholdValue,
			"operator":         string(tc.Operator),
			"timestamp":        tc.Timestamp,
			"is_early_warning": tc.IsEarlyWarning,
		}

		if tc.BaselineValue != nil {
			condEvidence["baseline_value"] = *tc.BaselineValue
		}

		if tc.ConsecutiveDays > 0 {
			condEvidence["consecutive_days"] = tc.ConsecutiveDays
		}

		if tc.EarlyWarningThreshold != nil {
			condEvidence["early_warning_threshold"] = *tc.EarlyWarningThreshold
		}

		conditions = append(conditions, condEvidence)
	}

	evidence["conditions"] = conditions

	return evidence
}

// evaluateTriggerConditions checks if fetched monitoring data satisfies trigger conditions
func (s *RegisteredPolicyService) evaluateTriggerConditions(
	ctx context.Context,
	triggers []models.BasePolicyTrigger,
	monitoringData []models.FarmMonitoringData,
	farmID uuid.UUID,
) []TriggeredCondition {
	var triggeredConditions []TriggeredCondition
	currentTime := time.Now()

	for _, trigger := range triggers {
		// Check blackout periods - skip evaluation during blackout
		if s.isInBlackoutPeriod(trigger.BlackoutPeriods, currentTime) {
			slog.Info("Skipping trigger evaluation during blackout period",
				"trigger_id", trigger.ID,
				"current_time", currentTime)
			continue
		}

		conditions, err := s.basePolicyRepo.GetBasePolicyTriggerConditionsByTriggerID(trigger.ID)
		if err != nil {
			slog.Warn("Failed to get conditions for trigger evaluation",
				"trigger_id", trigger.ID,
				"error", err)
			continue
		}

		// Sort conditions by ConditionOrder for proper evaluation sequence
		sortConditionsByOrder(conditions)

		// Fetch historical data from database for comprehensive evaluation
		historicalData, err := s.farmMonitoringDataRepo.GetByFarmID(ctx, farmID)
		if err != nil {
			slog.Warn("Failed to get historical monitoring data",
				"farm_id", farmID,
				"error", err)
			// Continue with just the fetched data
			historicalData = nil
		}

		// Merge fetched data with historical data, avoiding duplicates
		allData := s.mergeMonitoringData(monitoringData, historicalData)

		// Group all monitoring data by condition ID
		dataByCondition := make(map[uuid.UUID][]models.FarmMonitoringData)
		for _, data := range allData {
			dataByCondition[data.BasePolicyTriggerConditionID] = append(
				dataByCondition[data.BasePolicyTriggerConditionID],
				data,
			)
		}

		// Evaluate each condition in order
		var conditionResults []bool
		var triggerConditionsForThisTrigger []TriggeredCondition

		for _, cond := range conditions {
			condData := dataByCondition[cond.ID]
			if len(condData) == 0 {
				slog.Debug("No data for condition", "condition_id", cond.ID)
				conditionResults = append(conditionResults, false)
				continue
			}

			// Sort data by timestamp for proper chronological analysis
			sortMonitoringDataByTimestamp(condData)

			// Apply aggregation function to get the current value
			aggregatedValue := s.applyAggregation(condData, cond.AggregationFunction, cond.AggregationWindowDays)

			// Calculate baseline if required for change-based operators
			var baselineValue *float64
			if cond.BaselineWindowDays != nil && cond.BaselineFunction != nil {
				baseline := s.calculateBaseline(condData, *cond.BaselineWindowDays, *cond.BaselineFunction, cond.AggregationWindowDays)
				baselineValue = &baseline

				// For change operators, calculate the change from baseline
				if cond.ThresholdOperator == models.ThresholdChangeGT || cond.ThresholdOperator == models.ThresholdChangeLT {
					aggregatedValue = aggregatedValue - baseline
				}
			}

			// Check if main threshold is satisfied
			isSatisfied := s.checkThreshold(aggregatedValue, cond.ThresholdValue, cond.ThresholdOperator)

			// Check early warning threshold if main threshold not satisfied
			isEarlyWarning := false
			if !isSatisfied && cond.EarlyWarningThreshold != nil {
				isEarlyWarning = s.checkThreshold(aggregatedValue, *cond.EarlyWarningThreshold, cond.ThresholdOperator)
			}

			// Check consecutive days requirement
			consecutiveDays := 0
			if cond.ConsecutiveRequired && isSatisfied {
				consecutiveDays = s.countConsecutiveDays(condData, cond.ThresholdValue, cond.ThresholdOperator, cond.AggregationFunction)
				// For consecutive requirement, we need at least ValidationWindowDays consecutive days
				if consecutiveDays < cond.ValidationWindowDays {
					slog.Info("Consecutive days requirement not met",
						"condition_id", cond.ID,
						"consecutive_days", consecutiveDays,
						"required_days", cond.ValidationWindowDays)
					isSatisfied = false
				}
			}

			// Validate within validation window if specified
			if isSatisfied && cond.ValidationWindowDays > 0 && !cond.ConsecutiveRequired {
				// Check if condition was satisfied within the validation window
				validationCutoff := currentTime.AddDate(0, 0, -cond.ValidationWindowDays).Unix()
				latestTimestamp := condData[len(condData)-1].MeasurementTimestamp
				if latestTimestamp < validationCutoff {
					slog.Info("Condition data outside validation window",
						"condition_id", cond.ID,
						"latest_data", latestTimestamp,
						"validation_cutoff", validationCutoff)
					isSatisfied = false
				}
			}

			if isSatisfied || isEarlyWarning {
				tc := TriggeredCondition{
					ConditionID:           cond.ID,
					ParameterName:         condData[0].ParameterName,
					MeasuredValue:         aggregatedValue,
					ThresholdValue:        cond.ThresholdValue,
					Operator:              cond.ThresholdOperator,
					Timestamp:             condData[len(condData)-1].MeasurementTimestamp,
					BaselineValue:         baselineValue,
					ConsecutiveDays:       consecutiveDays,
					IsEarlyWarning:        isEarlyWarning && !isSatisfied,
					EarlyWarningThreshold: cond.EarlyWarningThreshold,
				}
				triggerConditionsForThisTrigger = append(triggerConditionsForThisTrigger, tc)
			}

			conditionResults = append(conditionResults, isSatisfied)
		}

		// Check logical operator (AND/OR) for trigger
		triggerSatisfied := s.evaluateLogicalOperator(trigger.LogicalOperator, conditionResults)
		if triggerSatisfied {
			triggeredConditions = append(triggeredConditions, triggerConditionsForThisTrigger...)
		} else {
			// Log early warnings even if trigger not fully satisfied
			for _, tc := range triggerConditionsForThisTrigger {
				if tc.IsEarlyWarning {
					slog.Info("Early warning threshold breached (trigger not fully satisfied)",
						"condition_id", tc.ConditionID,
						"parameter", tc.ParameterName,
						"measured_value", tc.MeasuredValue,
						"early_warning_threshold", tc.EarlyWarningThreshold)
				}
			}
		}
	}

	return triggeredConditions
}

// isInBlackoutPeriod checks if current time falls within any blackout period
func (s *RegisteredPolicyService) isInBlackoutPeriod(blackoutPeriods utils.JSONMap, currentTime time.Time) bool {
	if blackoutPeriods == nil {
		return false
	}

	// Blackout periods expected format: {"periods": [{"start": "MM-DD", "end": "MM-DD"}, ...]}
	periods, ok := blackoutPeriods["periods"].([]interface{})
	if !ok {
		return false
	}

	currentMonthDay := currentTime.Format("01-02")

	for _, p := range periods {
		period, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		start, startOk := period["start"].(string)
		end, endOk := period["end"].(string)
		if !startOk || !endOk {
			continue
		}

		// Simple string comparison for MM-DD format
		if start <= end {
			// Normal range (e.g., 03-01 to 05-31)
			if currentMonthDay >= start && currentMonthDay <= end {
				return true
			}
		} else {
			// Wrapping range (e.g., 11-01 to 02-28)
			if currentMonthDay >= start || currentMonthDay <= end {
				return true
			}
		}
	}

	return false
}

// sortConditionsByOrder sorts conditions by their ConditionOrder field
func sortConditionsByOrder(conditions []models.BasePolicyTriggerCondition) {
	for i := 0; i < len(conditions)-1; i++ {
		for j := i + 1; j < len(conditions); j++ {
			if conditions[i].ConditionOrder > conditions[j].ConditionOrder {
				conditions[i], conditions[j] = conditions[j], conditions[i]
			}
		}
	}
}

// sortMonitoringDataByTimestamp sorts monitoring data by timestamp ascending
func sortMonitoringDataByTimestamp(data []models.FarmMonitoringData) {
	for i := 0; i < len(data)-1; i++ {
		for j := i + 1; j < len(data); j++ {
			if data[i].MeasurementTimestamp > data[j].MeasurementTimestamp {
				data[i], data[j] = data[j], data[i]
			}
		}
	}
}

// mergeMonitoringData merges fetched data with historical data, avoiding duplicates
func (s *RegisteredPolicyService) mergeMonitoringData(
	fetched []models.FarmMonitoringData,
	historical []models.FarmMonitoringData,
) []models.FarmMonitoringData {
	if historical == nil {
		return fetched
	}

	// Create a map of existing IDs from fetched data
	existingIDs := make(map[uuid.UUID]bool)
	for _, d := range fetched {
		existingIDs[d.ID] = true
	}

	// Add historical data that's not in fetched
	result := make([]models.FarmMonitoringData, len(fetched))
	copy(result, fetched)

	for _, d := range historical {
		if !existingIDs[d.ID] {
			result = append(result, d)
		}
	}

	return result
}

// calculateBaseline calculates the baseline value using historical data
func (s *RegisteredPolicyService) calculateBaseline(
	data []models.FarmMonitoringData,
	baselineWindowDays int,
	baselineFunction models.AggregationFunction,
	aggregationWindowDays int,
) float64 {
	if len(data) == 0 {
		return 0
	}

	// Baseline window is calculated before the aggregation window
	// e.g., if aggregation is last 7 days, baseline is the 30 days before that
	aggregationCutoff := time.Now().AddDate(0, 0, -aggregationWindowDays).Unix()
	baselineCutoff := time.Now().AddDate(0, 0, -(aggregationWindowDays + baselineWindowDays)).Unix()

	var baselineData []float64
	for _, d := range data {
		if d.MeasurementTimestamp >= baselineCutoff && d.MeasurementTimestamp < aggregationCutoff {
			baselineData = append(baselineData, d.MeasuredValue)
		}
	}

	if len(baselineData) == 0 {
		return 0
	}

	// Apply baseline aggregation function
	switch baselineFunction {
	case models.AggregationSum:
		var sum float64
		for _, v := range baselineData {
			sum += v
		}
		return sum
	case models.AggregationAvg:
		var sum float64
		for _, v := range baselineData {
			sum += v
		}
		return sum / float64(len(baselineData))
	case models.AggregationMin:
		minVal := baselineData[0]
		for _, v := range baselineData[1:] {
			if v < minVal {
				minVal = v
			}
		}
		return minVal
	case models.AggregationMax:
		maxVal := baselineData[0]
		for _, v := range baselineData[1:] {
			if v > maxVal {
				maxVal = v
			}
		}
		return maxVal
	default:
		// Return average as default
		var sum float64
		for _, v := range baselineData {
			sum += v
		}
		return sum / float64(len(baselineData))
	}
}

// countConsecutiveDays counts how many consecutive days the condition was met
func (s *RegisteredPolicyService) countConsecutiveDays(
	data []models.FarmMonitoringData,
	thresholdValue float64,
	operator models.ThresholdOperator,
	aggFunc models.AggregationFunction,
) int {
	if len(data) == 0 {
		return 0
	}

	// Group data by day
	dataByDay := make(map[string][]float64)
	for _, d := range data {
		day := time.Unix(d.MeasurementTimestamp, 0).Format("2006-01-02")
		dataByDay[day] = append(dataByDay[day], d.MeasuredValue)
	}

	// Sort days
	var days []string
	for day := range dataByDay {
		days = append(days, day)
	}
	for i := 0; i < len(days)-1; i++ {
		for j := i + 1; j < len(days); j++ {
			if days[i] > days[j] {
				days[i], days[j] = days[j], days[i]
			}
		}
	}

	// Count consecutive days from the most recent
	consecutiveCount := 0
	for i := len(days) - 1; i >= 0; i-- {
		dayData := dataByDay[days[i]]

		// Aggregate the day's data
		var dayValue float64
		switch aggFunc {
		case models.AggregationSum:
			for _, v := range dayData {
				dayValue += v
			}
		case models.AggregationAvg:
			for _, v := range dayData {
				dayValue += v
			}
			dayValue /= float64(len(dayData))
		case models.AggregationMin:
			dayValue = dayData[0]
			for _, v := range dayData[1:] {
				if v < dayValue {
					dayValue = v
				}
			}
		case models.AggregationMax:
			dayValue = dayData[0]
			for _, v := range dayData[1:] {
				if v > dayValue {
					dayValue = v
				}
			}
		default:
			dayValue = dayData[len(dayData)-1]
		}

		// Check if this day meets the threshold
		if s.checkThreshold(dayValue, thresholdValue, operator) {
			consecutiveCount++
		} else {
			break // Consecutive streak broken
		}

		// Check if days are actually consecutive
		if i > 0 {
			currentDay, _ := time.Parse("2006-01-02", days[i])
			prevDay, _ := time.Parse("2006-01-02", days[i-1])
			if currentDay.Sub(prevDay).Hours() > 48 { // Allow for 1 day gap
				break
			}
		}
	}

	return consecutiveCount
}

// applyAggregation applies the aggregation function to monitoring data
func (s *RegisteredPolicyService) applyAggregation(
	data []models.FarmMonitoringData,
	aggFunc models.AggregationFunction,
	windowDays int,
) float64 {
	if len(data) == 0 {
		return 0
	}

	// Filter data within aggregation window
	cutoffTime := time.Now().AddDate(0, 0, -windowDays).Unix()
	var windowData []float64
	for _, d := range data {
		if d.MeasurementTimestamp >= cutoffTime {
			windowData = append(windowData, d.MeasuredValue)
		}
	}

	if len(windowData) == 0 {
		return 0
	}

	switch aggFunc {
	case models.AggregationSum:
		var sum float64
		for _, v := range windowData {
			sum += v
		}
		return sum
	case models.AggregationAvg:
		var sum float64
		for _, v := range windowData {
			sum += v
		}
		return sum / float64(len(windowData))
	case models.AggregationMin:
		minVal := windowData[0]
		for _, v := range windowData[1:] {
			if v < minVal {
				minVal = v
			}
		}
		return minVal
	case models.AggregationMax:
		maxVal := windowData[0]
		for _, v := range windowData[1:] {
			if v > maxVal {
				maxVal = v
			}
		}
		return maxVal
	case models.AggregationChange:
		if len(windowData) < 2 {
			return 0
		}
		return windowData[len(windowData)-1] - windowData[0]
	default:
		return windowData[len(windowData)-1] // Return latest value
	}
}

// checkThreshold checks if the measured value satisfies the threshold condition
func (s *RegisteredPolicyService) checkThreshold(
	measuredValue float64,
	thresholdValue float64,
	operator models.ThresholdOperator,
) bool {
	switch operator {
	case models.ThresholdLT:
		return measuredValue < thresholdValue
	case models.ThresholdGT:
		return measuredValue > thresholdValue
	case models.ThresholdLTE:
		return measuredValue <= thresholdValue
	case models.ThresholdGTE:
		return measuredValue >= thresholdValue
	case models.ThresholdEQ:
		return measuredValue == thresholdValue
	case models.ThresholdNE:
		return measuredValue != thresholdValue
	case models.ThresholdChangeGT:
		return measuredValue > thresholdValue
	case models.ThresholdChangeLT:
		return measuredValue < thresholdValue
	default:
		return false
	}
}

// evaluateLogicalOperator evaluates conditions based on the logical operator
func (s *RegisteredPolicyService) evaluateLogicalOperator(
	operator models.LogicalOperator,
	results []bool,
) bool {
	if len(results) == 0 {
		return false
	}

	switch operator {
	case models.LogicalAND:
		for _, r := range results {
			if !r {
				return false
			}
		}
		return true
	case models.LogicalOR:
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	default:
		return results[0]
	}
}

// ============================================================================
// FARM MONITORING DATA FETCH HELPERS
// ============================================================================

// fetchMonitoringDataWorker processes DataRequest jobs and fetches monitoring data from satellite API
func fetchMonitoringDataWorker(
	jobs <-chan DataRequest,
	results chan<- DataResponse,
	httpClient *http.Client,
) {
	for req := range jobs {
		response := DataResponse{
			DataSource: req.DataSource,
		}

		// Determine API endpoint
		// endpoint, exists := endpointMap[req.DataSource.ParameterName]
		//	if !exists {
		//		response.SkipReason = fmt.Sprintf("unsupported parameter: %s", req.DataSource.ParameterName)
		//		slog.Warn("Skipping unsupported data source",
		//			"parameter", req.DataSource.ParameterName,
		//			"data_source_id", req.DataSource.ID)
		//		results <- response
		//		continue
		//	}

		// Fetch data with retry logic
		monitoringData, polygonID, err := fetchDataWithRetry(
			httpClient,
			*response.DataSource.APIEndpoint,
			req,
			3, // max retries
		)

		if err != nil {
			response.Err = err
		} else {
			response.MonitoringData = monitoringData
			response.AgroPolygonID = polygonID
		}

		results <- response
	}
}

// fetchSatelliteDataWithRetry fetches data with exponential backoff retry
func fetchDataWithRetry(
	client *http.Client,
	endpoint string,
	req DataRequest,
	maxRetries int,
) ([]models.FarmMonitoringData, string, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			slog.Info("Retrying satellite API call",
				"attempt", attempt,
				"backoff_seconds", backoff.Seconds(),
				"parameter", req.DataSource.ParameterName)
			time.Sleep(backoff)
		}

		if strings.Contains(endpoint, "satellite") {
			data, err := fetchSatelliteData(client, endpoint, req)
			if err == nil {
				return data, "", nil // Satellite data doesn't return polygon ID
			}

			lastErr = err
			slog.Warn("Satellite API call failed",
				"attempt", attempt,
				"parameter", req.DataSource.ParameterName,
				"error", err)
		} else if strings.Contains(endpoint, "weather") {
			data, polygonID, err := fetchWeatherData(client, endpoint, req)
			if err == nil {
				return data, polygonID, nil
			}

			lastErr = err
			slog.Warn("Weather API call failed",
				"attempt", attempt,
				"parameter", req.DataSource.ParameterName,
				"error", err)
		}
	}

	return nil, "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func fetchWeatherData(client *http.Client,
	endpoint string,
	req DataRequest,
) ([]models.FarmMonitoringData, string, error) {
	// Convert date strings (YYYY-MM-DD) to Unix timestamps
	startTime, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse start date: %w", err)
	}
	endTime, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse end date: %w", err)
	}

	// Add one day to end date to include the entire day
	endTime = endTime.Add(24 * time.Hour)

	// Extract first 4 coordinates for polygon (weather API expects 4 corners)
	if len(req.FarmCoordinates) < 4 {
		return nil, "", fmt.Errorf("insufficient coordinates: need at least 4 points, got %d", len(req.FarmCoordinates))
	}

	// Build query parameters for weather API
	params := url.Values{}
	for i := 0; i < 4 && i < len(req.FarmCoordinates); i++ {
		if len(req.FarmCoordinates[i]) < 2 {
			return nil, "", fmt.Errorf("invalid coordinate at index %d", i)
		}
		// Weather API expects lat/lon format
		params.Set(fmt.Sprintf("lat%d", i+1), fmt.Sprintf("%.6f", req.FarmCoordinates[i][1]))
		params.Set(fmt.Sprintf("lon%d", i+1), fmt.Sprintf("%.6f", req.FarmCoordinates[i][0]))
	}
	params.Set("start", strconv.FormatInt(startTime.Unix(), 10))
	params.Set("end", strconv.FormatInt(endTime.Unix(), 10))
	params.Set("polygon_id", req.AgroPolygonID)

	// Create HTTP request
	fullURL := endpoint + "?" + params.Encode()
	httpReq, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpReq = httpReq.WithContext(ctx)

	startRequestTime := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(startRequestTime)

	if err != nil {
		return nil, "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response - UnifiedAPIResponse from weather service
	var apiResp struct {
		PolygonID         string    `json:"polygon_id"`
		PolygonName       string    `json:"polygon_name"`
		PolygonCenter     []float64 `json:"polygon_center"`
		PolygonArea       float64   `json:"polygon_area"`
		PolygonReused     bool      `json:"polygon_reused"`
		PolygonCreatedNew bool      `json:"polygon_created_new"`
		TimeRange         struct {
			Start int64 `json:"start"`
			End   int64 `json:"end"`
		} `json:"time_range"`
		Data []struct {
			Dt    int64   `json:"dt"`
			Data  float64 `json:"data"`
			Count int     `json:"count"`
			Unit  string  `json:"unit"`
		} `json:"data"`
		TotalDataValue float64 `json:"total_data_value"`
		DataPointCount int     `json:"data_point_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	slog.Info("Weather API call completed",
		"parameter", req.DataSource.ParameterName,
		"data_points", apiResp.DataPointCount,
		"polygon_id", apiResp.PolygonID,
		"duration_ms", duration.Milliseconds())

	// Convert to FarmMonitoringData models
	var monitoringData []models.FarmMonitoringData

	for _, dataPoint := range apiResp.Data {
		// Determine data quality based on measurement count
		dataQuality := models.DataQualityGood
		if dataPoint.Count < 5 {
			dataQuality = models.DataQualityPoor
		} else if dataPoint.Count < 10 {
			dataQuality = models.DataQualityAcceptable
		}

		// Calculate confidence score based on data count
		confidenceScore := math.Min(1.0, float64(dataPoint.Count)/20.0)

		// Build component data
		componentData := utils.JSONMap{
			"measurement_count": dataPoint.Count,
			"polygon_id":        apiResp.PolygonID,
			"polygon_area_sqm":  apiResp.PolygonArea,
			"total_value":       apiResp.TotalDataValue,
		}

		monitoringData = append(monitoringData, models.FarmMonitoringData{
			ID:                           uuid.New(),
			FarmID:                       req.FarmID,
			BasePolicyTriggerConditionID: req.BasePolicyTriggerConditionID,
			ParameterName:                req.DataSource.ParameterName,
			MeasuredValue:                dataPoint.Data,
			Unit:                         &dataPoint.Unit,
			MeasurementTimestamp:         dataPoint.Dt,
			ComponentData:                componentData,
			DataQuality:                  dataQuality,
			ConfidenceScore:              &confidenceScore,
			MeasurementSource:            req.DataSource.DataProvider,
			CreatedAt:                    time.Now(),
		})
	}

	slog.Info("Converted weather data to monitoring records",
		"parameter", req.DataSource.ParameterName,
		"records_created", len(monitoringData),
		"polygon_id", apiResp.PolygonID)

	return monitoringData, apiResp.PolygonID, nil
}

// fetchSatelliteData performs a single API call to satellite service
func fetchSatelliteData(
	client *http.Client,
	endpoint string,
	req DataRequest,
) ([]models.FarmMonitoringData, error) {
	// Build query parameters
	coordsJSON, err := json.Marshal(req.FarmCoordinates)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal coordinates: %w", err)
	}

	params := url.Values{}
	params.Set("coordinates", string(coordsJSON))
	params.Set("start_date", req.StartDate)
	params.Set("end_date", req.EndDate)
	params.Set("max_cloud_cover", fmt.Sprintf("%.1f", req.MaxCloudCover))
	params.Set("max_images", strconv.Itoa(req.MaxImages))
	params.Set("include_components", strconv.FormatBool(req.IncludeComponents))

	// Create HTTP request
	fullURL := endpoint + "?" + params.Encode()
	slog.Info("Satellite API request",
		"full_url", fullURL,
		"parameter", req.DataSource.ParameterName,
		"start_date", req.StartDate,
		"end_date", req.EndDate,
		"max_cloud_cover", req.MaxCloudCover,
		"coordinates_length", len(string(coordsJSON)))

	httpReq, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpReq = httpReq.WithContext(ctx)

	startTime := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp SatelliteAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Status != "success" {
		return nil, fmt.Errorf("API returned non-success status: %s - %s", apiResp.Status, apiResp.Message)
	}

	slog.Info("Satellite API call completed",
		"parameter", req.DataSource.ParameterName,
		"images_processed", apiResp.Data.Summary.ImagesProcessed,
		"duration_ms", duration.Milliseconds())

	// Convert to FarmMonitoringData models
	return convertToMonitoringData(apiResp, req), nil
}

// convertToMonitoringData converts satellite API response to FarmMonitoringData models
func convertToMonitoringData(
	apiResp SatelliteAPIResponse,
	req DataRequest,
) []models.FarmMonitoringData {
	var monitoringData []models.FarmMonitoringData

	for _, image := range apiResp.Data.Images {
		// Determine which statistics field to use based on parameter type
		var meanValue, medianValue, minValue, maxValue, stddevValue *float64

		if image.NDMIStatistics != nil {
			if image.NDMIStatistics.Mean != nil {
				meanValue = &image.NDMIStatistics.Mean.Value
			}
			if image.NDMIStatistics.Median != nil {
				medianValue = &image.NDMIStatistics.Median.Value
			}
			if image.NDMIStatistics.Min != nil {
				minValue = &image.NDMIStatistics.Min.Value
			}
			if image.NDMIStatistics.Max != nil {
				maxValue = &image.NDMIStatistics.Max.Value
			}
			if image.NDMIStatistics.StdDev != nil {
				stddevValue = &image.NDMIStatistics.StdDev.Value
			}
		} else if image.NDVIStatistics != nil {
			if image.NDVIStatistics.Mean != nil {
				meanValue = &image.NDVIStatistics.Mean.Value
			}
			if image.NDVIStatistics.Median != nil {
				medianValue = &image.NDVIStatistics.Median.Value
			}
			if image.NDVIStatistics.Min != nil {
				minValue = &image.NDVIStatistics.Min.Value
			}
			if image.NDVIStatistics.Max != nil {
				maxValue = &image.NDVIStatistics.Max.Value
			}
			if image.NDVIStatistics.StdDev != nil {
				stddevValue = &image.NDVIStatistics.StdDev.Value
			}
		}

		// Log full image details for debugging
		slog.Info("Processing satellite image",
			"parameter", req.DataSource.ParameterName,
			"acquisition_date", image.AcquisitionDate,
			"cloud_cover", image.CloudCover.Value,
			"statistics_mean", meanValue,
			"statistics_median", medianValue,
			"statistics_min", minValue,
			"statistics_max", maxValue)

		// Skip images with no valid measurements
		if meanValue == nil {
			slog.Warn("Skipping image with no mean value",
				"parameter", req.DataSource.ParameterName,
				"acquisition_date", image.AcquisitionDate,
				"cloud_cover", image.CloudCover.Value,
				"has_median", medianValue != nil,
				"has_min", minValue != nil,
				"has_max", maxValue != nil)
			continue
		}

		// Parse acquisition date to Unix timestamp
		acquisitionTime, err := time.Parse("2006-01-02", image.AcquisitionDate)
		if err != nil {
			slog.Warn("Failed to parse acquisition date",
				"date", image.AcquisitionDate,
				"error", err)
			continue
		}

		// Build component data JSON
		componentData := utils.JSONMap{}
		if req.IncludeComponents && image.ComponentData != nil {
			if image.ComponentData.NIR != nil {
				componentData["nir"] = *image.ComponentData.NIR
			}
			if image.ComponentData.Red != nil {
				componentData["red"] = *image.ComponentData.Red
			}
			if image.ComponentData.SWIR != nil {
				componentData["swir"] = *image.ComponentData.SWIR
			}
		}

		// Add statistics to component data
		componentData["statistics"] = map[string]interface{}{
			"median": medianValue,
			"min":    minValue,
			"max":    maxValue,
			"stddev": stddevValue,
		}

		// Determine data quality based on cloud cover
		dataQuality := models.DataQualityGood
		if image.CloudCover.Value > 50 {
			dataQuality = models.DataQualityPoor
		} else if image.CloudCover.Value > 20 {
			dataQuality = models.DataQualityAcceptable
		}

		// Calculate confidence score (inverse of cloud cover)
		confidenceScore := math.Max(0.0, (100.0-image.CloudCover.Value)/100.0)

		monitoringData = append(monitoringData, models.FarmMonitoringData{
			ID:                           uuid.New(),
			FarmID:                       req.FarmID,
			BasePolicyTriggerConditionID: req.BasePolicyTriggerConditionID,
			ParameterName:                req.DataSource.ParameterName,
			MeasuredValue:                *meanValue,
			Unit:                         req.DataSource.Unit,
			MeasurementTimestamp:         acquisitionTime.Unix(),
			ComponentData:                componentData,
			DataQuality:                  dataQuality,
			ConfidenceScore:              &confidenceScore,
			MeasurementSource:            req.DataSource.DataProvider,
			CloudCoverPercentage:         &image.CloudCover.Value,
			CreatedAt:                    time.Now(),
		})
	}

	slog.Info("Converted satellite data to monitoring records",
		"parameter", req.DataSource.ParameterName,
		"records_created", len(monitoringData))

	return monitoringData
}

// extractPolygonCoordinates extracts coordinates from GeoJSON polygon (first ring only)
func extractPolygonCoordinates(polygon *models.GeoJSONPolygon) [][]float64 {
	if polygon == nil || len(polygon.Coordinates) == 0 {
		return nil
	}
	// Return first ring (outer boundary)
	return polygon.Coordinates[0]
}

// unixToDateString converts Unix timestamp to YYYY-MM-DD format
func unixToDateString(unixTime int64) string {
	return time.Unix(unixTime, 0).Format("2006-01-02")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseFarmMonitoringDataFromMap converts a map to FarmMonitoringData for test injection
func parseFarmMonitoringDataFromMap(dataMap map[string]interface{}, farmID uuid.UUID) models.FarmMonitoringData {
	data := models.FarmMonitoringData{
		ID:     uuid.New(),
		FarmID: farmID,
	}

	// Parse BasePolicyTriggerConditionID
	if condIDStr, ok := dataMap["base_policy_trigger_condition_id"].(string); ok {
		if condID, err := uuid.Parse(condIDStr); err == nil {
			data.BasePolicyTriggerConditionID = condID
		}
	}

	// Parse ParameterName
	if paramName, ok := dataMap["parameter_name"].(string); ok {
		data.ParameterName = models.DataSourceParameterName(paramName)
	}

	// Parse MeasuredValue
	if measuredValue, ok := dataMap["measured_value"].(float64); ok {
		data.MeasuredValue = measuredValue
	}

	// Parse Unit (optional)
	if unit, ok := dataMap["unit"].(string); ok {
		data.Unit = &unit
	}

	// Parse MeasurementTimestamp
	if timestamp, ok := dataMap["measurement_timestamp"].(float64); ok {
		data.MeasurementTimestamp = int64(timestamp)
	} else {
		// Default to current time if not provided
		data.MeasurementTimestamp = time.Now().Unix()
	}

	// Parse ComponentData (optional)
	if componentData, ok := dataMap["component_data"].(map[string]interface{}); ok {
		data.ComponentData = componentData
	}

	// Parse DataQuality (optional, defaults to "good")
	if quality, ok := dataMap["data_quality"].(string); ok {
		data.DataQuality = models.DataQuality(quality)
	} else {
		data.DataQuality = models.DataQualityGood
	}

	// Parse ConfidenceScore (optional)
	if score, ok := dataMap["confidence_score"].(float64); ok {
		data.ConfidenceScore = &score
	}

	// Parse MeasurementSource (optional)
	if source, ok := dataMap["measurement_source"].(string); ok {
		data.MeasurementSource = &source
	}

	// Set CreatedAt to now
	data.CreatedAt = time.Now()

	return data
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

	slog.Info("Successfully created partner policy underwriting",
		"underwriting_id", underwriting.ID,
		"policy_id", policyID,
		"status", req.UnderwritingStatus,
		"message", responseMessage)

	return &models.CreatePartnerPolicyUnderwritingResponse{
		UnderwritingID:     underwriting.ID.String(),
		PolicyID:           policyID.String(),
		UnderwritingStatus: req.UnderwritingStatus,
		ValidatedBy:        validatedBy,
		Message:            responseMessage,
	}, nil
}

func (s *RegisteredPolicyService) GetInsurancePartnerProfile(token string) (map[string]interface{}, error) {
	url := "https://agrisa-api.phrimp.io.vn/profile/protected/api/v1/insurance-partners/me/profile"
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
