package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"policy-service/internal/ai/gemini"
	"policy-service/internal/models"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RiskAnalysisJob performs AI-powered risk analysis on a registered policy
// Parameters:
//   - registered_policy_id (string, required): UUID of the policy to analyze
//   - force_reanalysis (bool, optional): Skip existing analysis check
func (s *RegisteredPolicyService) RiskAnalysisJob(params map[string]any) error {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("RiskAnalysisJob: recovered from panic", "panic", r)
		}
	}()

	ctx := context.Background()

	// 1. Extract and validate parameters
	policyIDStr, ok := params["registered_policy_id"].(string)
	if !ok || policyIDStr == "" {
		return fmt.Errorf("invalid or missing registered_policy_id parameter")
	}

	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return fmt.Errorf("invalid UUID format for registered_policy_id: %w", err)
	}

	forceReanalysis, _ := params["force_reanalysis"].(bool)

	slog.Info("Starting risk analysis job",
		"registered_policy_id", policyIDStr,
		"force_reanalysis", forceReanalysis)

	// 2. Get registered policy
	policy, err := s.registeredPolicyRepo.GetByID(policyID)
	if err != nil {
		slog.Error("Failed to get registered policy", "policy_id", policyIDStr, "error", err)
		return fmt.Errorf("failed to get registered policy: %w", err)
	}

	slog.Info("Retrieved registered policy",
		"policy_id", policyIDStr,
		"farm_id", policy.FarmID,
		"base_policy_id", policy.BasePolicyID)

	// 3. Check for existing analysis (skip if exists and not forced)
	if !forceReanalysis {
		existing, _ := s.registeredPolicyRepo.GetRiskAnalysesByPolicyID(policyID)
		if len(existing) > 0 {
			slog.Info("Risk analysis already exists, skipping",
				"policy_id", policyIDStr,
				"analysis_count", len(existing),
				"latest_analysis_id", existing[0].ID)
			return nil
		}
	}

	// 4. Parallel data fetching
	var (
		farm           *models.Farm
		farmPhotos     []models.FarmPhoto
		monitoringData []models.FarmMonitoringData
		trigger        *models.BasePolicyTrigger
		conditions     []models.BasePolicyTriggerCondition
		fetchErrors    []error
	)

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Fetch farm data
	wg.Add(1)
	go func() {
		defer wg.Done()
		farmData, err := s.farmService.GetByFarmID(ctx, policy.FarmID.String())
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Errorf("fetch farm: %w", err))
			return
		}
		farm = farmData
	}()

	// Fetch farm photos (will be fetched after farm data is available)
	// Photos are stored in the farm struct, so we'll extract them after farm fetch

	// Fetch 1-year historical monitoring data
	wg.Add(1)
	go func() {
		defer wg.Done()
		endTimestamp := time.Now().Unix()
		startTimestamp := endTimestamp - (365 * 24 * 60 * 60) // 1 year ago

		data, err := s.farmMonitoringDataRepo.GetByTimeRange(ctx, policy.FarmID, startTimestamp, endTimestamp)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			slog.Warn("Failed to fetch monitoring data, continuing with empty data",
				"farm_id", policy.FarmID,
				"error", err)
			monitoringData = []models.FarmMonitoringData{}
			return
		}
		monitoringData = data
	}()

	// Fetch trigger configuration
	wg.Add(1)
	go func() {
		defer wg.Done()
		triggers, err := s.basePolicyRepo.GetBasePolicyTriggersByPolicyID(policy.BasePolicyID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Errorf("fetch triggers: %w", err))
			return
		}
		if len(triggers) == 0 {
			fetchErrors = append(fetchErrors, fmt.Errorf("no triggers found for base policy %s", policy.BasePolicyID))
			return
		}
		trigger = &triggers[0]

		// Fetch conditions for the trigger
		conds, err := s.basePolicyRepo.GetBasePolicyTriggerConditionsByTriggerID(trigger.ID)
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Errorf("fetch conditions: %w", err))
			return
		}
		conditions = conds
	}()

	wg.Wait()

	// Check for critical fetch errors
	if len(fetchErrors) > 0 {
		for _, err := range fetchErrors {
			slog.Error("Critical data fetch error", "error", err)
		}
		return fmt.Errorf("failed to fetch required data: %v", fetchErrors)
	}

	if farm == nil {
		return fmt.Errorf("farm data is nil after fetch")
	}

	if trigger == nil {
		return fmt.Errorf("trigger data is nil after fetch")
	}

	// Extract farm photos from farm struct
	farmPhotos = farm.FarmPhotos
	if farmPhotos == nil {
		farmPhotos = []models.FarmPhoto{}
	}

	slog.Info("Successfully fetched all required data",
		"farm_id", farm.ID,
		"photos_count", len(farmPhotos),
		"monitoring_data_points", len(monitoringData),
		"conditions_count", len(conditions))

	// 5. Resolve data sources for all conditions
	dataSources := make(map[string]models.DataSource)
	for _, cond := range conditions {
		ds, err := s.dataSourceRepo.GetDataSourceByID(cond.DataSourceID)
		if err != nil {
			slog.Warn("Data source not found for condition",
				"condition_id", cond.ID,
				"data_source_id", cond.DataSourceID,
				"error", err)
			continue
		}
		dataSources[ds.ID.String()] = *ds
	}

	slog.Info("Resolved data sources", "count", len(dataSources))

	// 6. Download farm photos from MinIO concurrently
	farmPhotoData := make([]string, 0)
	if len(farmPhotos) > 0 && s.minioClient != nil {
		var downloadErr error
		farmPhotoData, downloadErr = s.downloadFarmPhotosParallel(ctx, farmPhotos)
		if downloadErr != nil {
			slog.Warn("Some photos failed to download", "error", downloadErr)
			// Continue with available photos
		}
	}

	slog.Info("Farm photos processed",
		"requested", len(farmPhotos),
		"downloaded", len(farmPhotoData))

	// 7. Build risk analysis prompt
	prompt := gemini.BuildRiskAnalysisPrompt(
		*farm,
		farmPhotos,
		farmPhotoData,
		monitoringData,
		*trigger,
		conditions,
		dataSources,
		*policy,
	)

	slog.Info("Risk analysis prompt constructed",
		"prompt_length", len(prompt),
		"photos_count", len(farmPhotos),
		"monitoring_data_points", len(monitoringData),
		"conditions_count", len(conditions))

	// 8. Call AI service with failover
	if s.geminiSelector == nil {
		return fmt.Errorf("gemini selector is not configured")
	}

	var aiResp map[string]any
	if len(farmPhotoData) > 0 {
		// Use multi-modal with images
		aiResp, err = gemini.SendAIWithImagesAndRetry(ctx, prompt, farmPhotoData, s.geminiSelector)
	} else {
		// Use text-only (no images available)
		// For text-only, we can use a simple wrapper or just send without images
		aiResp, err = gemini.SendAIWithImagesAndRetry(ctx, prompt, []string{}, s.geminiSelector)
	}

	if err != nil {
		slog.Error("AI risk analysis request failed", "error", err)
		return fmt.Errorf("AI risk analysis failed: %w", err)
	}

	// 9. Parse AI response into risk analysis structure
	var riskAnalysis models.RegisteredPolicyRiskAnalysis
	respBytes, err := json.Marshal(aiResp)
	if err != nil {
		return fmt.Errorf("failed to marshal AI response: %w", err)
	}

	slog.Info("AI risk analysis response received",
		"response_keys", getMapKeys(aiResp),
		"raw_response", string(respBytes))

	if err := json.Unmarshal(respBytes, &riskAnalysis); err != nil {
		slog.Error("Failed to unmarshal risk analysis response",
			"error", err,
			"raw_response", string(respBytes))
		return fmt.Errorf("failed to unmarshal risk analysis: %w", err)
	}

	// Set metadata fields
	riskAnalysis.ID = uuid.New()
	riskAnalysis.RegisteredPolicyID = policyID
	riskAnalysis.CreatedAt = time.Now()

	// Ensure analysis type is set
	if riskAnalysis.AnalysisType == "" {
		riskAnalysis.AnalysisType = models.RiskAnalysisTypeAIModel
	}

	// Ensure analysis timestamp is set and in seconds (not milliseconds)
	if riskAnalysis.AnalysisTimestamp == 0 {
		riskAnalysis.AnalysisTimestamp = time.Now().Unix()
	} else if riskAnalysis.AnalysisTimestamp > 9999999999 {
		// Convert milliseconds to seconds if timestamp is too large
		riskAnalysis.AnalysisTimestamp = riskAnalysis.AnalysisTimestamp / 1000
	}

	// Validate risk score is within expected range (0-100)
	if riskAnalysis.OverallRiskScore != nil {
		score := *riskAnalysis.OverallRiskScore
		if score < 0 {
			zero := 0.0
			riskAnalysis.OverallRiskScore = &zero
		} else if score > 100 {
			hundred := 100.0
			riskAnalysis.OverallRiskScore = &hundred
		}
	}

	// Cap all nested risk_score values in JSON maps to prevent numeric overflow
	capRiskScoresInMap(riskAnalysis.IdentifiedRisks)
	capRiskScoresInMap(riskAnalysis.Recommendations)
	capRiskScoresInMap(riskAnalysis.RawOutput)

	// Log actual values for debugging numeric overflow
	var scoreValue float64
	if riskAnalysis.OverallRiskScore != nil {
		scoreValue = *riskAnalysis.OverallRiskScore
	}

	slog.Info("Risk analysis parsed successfully",
		"analysis_id", riskAnalysis.ID,
		"analysis_status", riskAnalysis.AnalysisStatus,
		"overall_risk_score_value", scoreValue,
		"analysis_timestamp_value", riskAnalysis.AnalysisTimestamp,
		"overall_risk_level", riskAnalysis.OverallRiskLevel)

	// 10. Persist risk analysis
	if err := s.registeredPolicyRepo.CreateRiskAnalysis(&riskAnalysis); err != nil {
		slog.Error("Failed to persist risk analysis", "error", err)
		return fmt.Errorf("failed to persist risk analysis: %w", err)
	}

	// 11. Update underwriting status based on risk assessment
	newStatus := s.determineUnderwritingStatus(riskAnalysis)
	if err := s.registeredPolicyRepo.UpdateUnderwritingStatus(policyID, newStatus); err != nil {
		slog.Error("Failed to update underwriting status",
			"policy_id", policyIDStr,
			"new_status", newStatus,
			"error", err)
		// Don't fail the job, risk analysis is already saved
	} else {
		slog.Info("Updated underwriting status",
			"policy_id", policyIDStr,
			"new_status", newStatus)
	}

	slog.Info("Risk analysis job completed successfully",
		"registered_policy_id", policyIDStr,
		"risk_analysis_id", riskAnalysis.ID,
		"analysis_status", riskAnalysis.AnalysisStatus,
		"overall_risk_level", riskAnalysis.OverallRiskLevel)

	return nil
}

// downloadFarmPhotosParallel downloads farm photos from MinIO concurrently
func (s *RegisteredPolicyService) downloadFarmPhotosParallel(
	ctx context.Context,
	photos []models.FarmPhoto,
) ([]string, error) {
	if len(photos) == 0 {
		return []string{}, nil
	}

	photoData := make([]string, len(photos))
	var wg sync.WaitGroup
	errChan := make(chan error, len(photos))

	for i, photo := range photos {
		wg.Add(1)
		go func(idx int, p models.FarmPhoto) {
			defer wg.Done()

			// Extract bucket and object key from URL
			bucket, objectKey := extractBucketAndKeyFromURL(p.PhotoURL)
			if bucket == "" || objectKey == "" {
				slog.Warn("Could not extract bucket/key from photo URL",
					"photo_id", p.ID,
					"photo_url", p.PhotoURL,
					"extracted_bucket", bucket,
					"extracted_key", objectKey)
				return
			}

			slog.Info("Attempting to download photo",
				"photo_id", p.ID,
				"photo_url", p.PhotoURL,
				"bucket", bucket,
				"extracted_key", objectKey)

			obj, err := s.minioClient.GetFile(ctx, bucket, objectKey)
			if err != nil {
				errChan <- fmt.Errorf("photo %d (%s): %w", idx, p.ID, err)
				return
			}
			defer obj.Close()

			data, err := io.ReadAll(obj)
			if err != nil {
				errChan <- fmt.Errorf("read photo %d (%s): %w", idx, p.ID, err)
				return
			}

			photoData[idx] = base64.StdEncoding.EncodeToString(data)
		}(i, photo)
	}

	wg.Wait()
	close(errChan)

	// Collect errors for logging
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		slog.Warn("Some photo downloads failed",
			"total_photos", len(photos),
			"failed_count", len(errs),
			"errors", errs)
	}

	// Filter out empty strings (failed downloads)
	result := make([]string, 0, len(photos))
	for _, data := range photoData {
		if data != "" {
			result = append(result, data)
		}
	}

	return result, nil
}

// extractBucketAndKeyFromURL extracts the bucket name and object key from a MinIO URL
// Returns bucket name and object key
func extractBucketAndKeyFromURL(photoURL string) (string, string) {
	// Handle different URL formats:
	// - Full URL: http://minio:9000/bucket/path/to/file.jpg
	// - URL without protocol: hostname.com/bucket/path/to/file.jpg
	// - Relative path: /bucket/path/to/file.jpg
	// - Just bucket/key: bucket/path/to/file.jpg

	if photoURL == "" {
		return "", ""
	}

	url := photoURL

	// Remove protocol and host if present
	if strings.Contains(url, "://") {
		parts := strings.SplitN(url, "://", 2)
		if len(parts) == 2 {
			// Remove host part
			hostAndPath := parts[1]
			slashIdx := strings.Index(hostAndPath, "/")
			if slashIdx != -1 {
				url = hostAndPath[slashIdx:]
			}
		}
	} else {
		// Handle URL without protocol (e.g., hostname.com/bucket/path)
		// Check if first segment contains a dot (likely a hostname)
		parts := strings.SplitN(url, "/", 2)
		if len(parts) == 2 && strings.Contains(parts[0], ".") {
			url = "/" + parts[1]
		}
	}

	// Remove leading slash
	url = strings.TrimPrefix(url, "/")

	// Split into bucket and key
	parts := strings.SplitN(url, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "", url
}

// determineUnderwritingStatus determines the underwriting status based on risk analysis
func (s *RegisteredPolicyService) determineUnderwritingStatus(analysis models.RegisteredPolicyRiskAnalysis) models.UnderwritingStatus {
	// If analysis failed, mark for pending review
	if analysis.AnalysisStatus == models.ValidationFailed {
		return models.UnderwritingPending
	}

	// Base decision on risk level
	if analysis.OverallRiskLevel != nil {
		switch *analysis.OverallRiskLevel {
		case models.RiskLevelCritical:
			// Critical risk - reject
			return models.UnderwritingRejected
		case models.RiskLevelHigh:
			// High risk - needs manual review (pending)
			return models.UnderwritingPending
		case models.RiskLevelMedium:
			// Medium risk - may approve with conditions (pending for review)
			return models.UnderwritingPending
		case models.RiskLevelLow:
			// Low risk - can auto-approve
			return models.UnderwritingApproved
		}
	}

	// Also consider the risk score
	if analysis.OverallRiskScore != nil {
		score := *analysis.OverallRiskScore
		if score <= 25 {
			return models.UnderwritingApproved
		} else if score <= 50 {
			return models.UnderwritingPending
		} else if score <= 75 {
			return models.UnderwritingPending
		} else {
			return models.UnderwritingRejected
		}
	}

	// Default to pending review if we can't determine
	return models.UnderwritingPending
}

// getMapKeys returns the keys of a map for logging purposes
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// capRiskScoresInMap recursively caps all risk_score values in a map to 100
func capRiskScoresInMap(m map[string]any) {
	if m == nil {
		return
	}

	for key, value := range m {
		switch v := value.(type) {
		case float64:
			// Cap risk_score, fraud_score, and similar fields
			if strings.Contains(strings.ToLower(key), "score") && v > 100 {
				m[key] = 100.0
			}
		case map[string]any:
			capRiskScoresInMap(v)
		case []any:
			for _, item := range v {
				if itemMap, ok := item.(map[string]any); ok {
					capRiskScoresInMap(itemMap)
				}
			}
		}
	}
}
