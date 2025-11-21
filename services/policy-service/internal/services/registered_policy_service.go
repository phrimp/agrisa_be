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

	basePolicyIDStr, ok := params["base_policy_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid base_policy_id parameters")
	}
	basePolicyID, err := uuid.Parse(basePolicyIDStr)
	if err != nil {
		return fmt.Errorf("invalid base_policy_id format: %w", err)
	}

	farmIDStr, ok := params["farm_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid farm_id parameter")
	}

	farmID, err := uuid.Parse(farmIDStr)
	if err != nil {
		return fmt.Errorf("invalid farm_id format: %w", err)
	}

	dataSources, err := s.dataSourceRepo.GetDataSourcesByBasePolicyID(basePolicyID)
	if err != nil {
		return fmt.Errorf("retrieve data sources from base policy id failed: %w", err)
	}

	startDateFloat, ok := params["start_date"].(float64)
	if !ok {
		slog.Error("GetFarmPhotoJob: missing or invalid start_date parameter", "farm_id", farmID)
		return fmt.Errorf("missing or invalid start_date parameter")
	}
	startDate := int64(startDateFloat)

	endDateFloat, ok := params["end_date"].(float64)
	if !ok {
		slog.Error("GetFarmPhotoJob: missing or invalid end_date parameter", "farm_id", farmID)
		return fmt.Errorf("missing or invalid end_date parameter")
	}
	endDate := int64(endDateFloat)

	if endDate == 0 {
		endDate = time.Now().Add(24 * time.Hour).Unix()
	}

	// Check for existing data in range (skip if exists)
	ctx := context.Background()
	existingMonitorData, err := s.farmMonitoringDataRepo.CheckDataExistsInTimeRange(ctx, farmID, startDate, endDate)
	if err != nil {
		slog.Error("error checking existing monitor farm data", "error", err)
	}
	if existingMonitorData {
		slog.Info("existing monitor farm data found, skipping fetch job",
			"farm_id", farmID,
			"date_range", fmt.Sprintf("%d to %d", startDate, endDate))
		return nil
	}

	// Load farm to get boundary coordinates
	farm, err := s.farmService.GetByFarmID(ctx, farmID.String())
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

	// Convert Unix timestamps to YYYY-MM-DD format
	startDateStr := unixToDateString(startDate)
	endDateStr := unixToDateString(endDate)

	// Initialize worker pool
	numWorkers := min(10, len(dataSources))
	jobs := make(chan DataRequest, len(dataSources))
	results := make(chan DataResponse, len(dataSources))

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 120 * time.Second,
	}

	// Start workers
	for range numWorkers {
		go fetchMonitoringDataWorker(jobs, results, httpClient)
	}

	// Enqueue jobs for each data source
	for _, ds := range dataSources {
		jobs <- DataRequest{
			DataSource:                   ds,
			FarmID:                       farmID,
			FarmCoordinates:              farmCoordinates,
			AgroPolygonID:                farm.AgroPolygonID,
			StartDate:                    startDateStr,
			EndDate:                      endDateStr,
			BasePolicyTriggerConditionID: uuid.Nil, // Will be set by trigger conditions
			MaxCloudCover:                100.0,
			MaxImages:                    10,
			IncludeComponents:            true,
		}
	}
	close(jobs)

	// Collect results from all workers
	allMonitoringData := []models.FarmMonitoringData{}
	errorSummary := make(map[models.DataSourceParameterName]error)
	skipSummary := make(map[models.DataSourceParameterName]string)
	var agroPolygonID string // Store polygon ID from weather API

	for i := 0; i < len(dataSources); i++ {
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

	// Log summary
	successCount := len(dataSources) - len(errorSummary) - len(skipSummary)
	successRate := float64(successCount) / float64(len(dataSources)) * 100

	slog.Info("Farm monitoring data fetch completed",
		"policy_id", policyID,
		"farm_id", farmID,
		"base_policy_id", basePolicyID,
		"total_sources", len(dataSources),
		"successful", successCount,
		"failed", len(errorSummary),
		"skipped", len(skipSummary),
		"success_rate", fmt.Sprintf("%.1f%%", successRate),
		"records_stored", len(allMonitoringData))

	// Only fail if ALL sources failed
	if len(errorSummary) == len(dataSources) {
		return fmt.Errorf("all %d data sources failed to fetch", len(dataSources))
	}

	riskAnalysisJob := worker.JobPayload{
		JobID:      uuid.NewString(),
		Type:       "risk-analysis",
		Params:     map[string]any{"registered_policy_id": policyID, "force_reanalysis": false},
		MaxRetries: 10,
		OneTime:    true,
		RunNow:     true,
	}

	scheduler, ok := s.workerManager.GetSchedulerByPolicyID(policyID)
	if !ok {
		slog.Error("error get farm-imagery scheduler", "error", "scheduler doesn't exist")
	}

	scheduler.AddJob(riskAnalysisJob)

	return nil
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
	return s.registeredPolicyRepo.GetByID(policyID)
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
