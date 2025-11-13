package services

import (
	utils "agrisa_utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"policy-service/internal/config"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/worker"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FarmService struct {
	farmRepository *repository.FarmRepository
	config         *config.PolicyServiceConfig
	minioClient    *minio.MinioClient
	workerManager  *worker.WorkerManagerV2
}

func NewFarmService(farmRepo *repository.FarmRepository, cfg *config.PolicyServiceConfig, minioClient *minio.MinioClient, workerManager *worker.WorkerManagerV2) *FarmService {
	return &FarmService{farmRepository: farmRepo, config: cfg, minioClient: minioClient, workerManager: workerManager}
}

func (s *FarmService) GetFarmByOwnerID(ctx context.Context, userID string) ([]models.Farm, error) {
	farms, err := s.farmRepository.GetByOwnerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return farms, nil
}

func (s *FarmService) CreateFarm(farm *models.Farm, ownerID string) error {
	if farm.ID == uuid.Nil {
		farm.ID = uuid.New()
	}
	farm.OwnerID = ownerID
	farmcode := utils.GenerateRandomStringWithLength(10)
	farm.FarmCode = &farmcode
	// // Check if farmer has already owned a farm
	// existingFarm, err := s.farmRepository.GetByOwnerID(context.Background(), ownerID)
	// if err != nil && strings.Contains(err.Error(), "no rows in result set") {
	// 	// no existing farm, proceed to create
	// } else if existingFarm != nil {
	// 	return fmt.Errorf("badrequest: farmer has already owned a farm")
	// }

	err := s.farmRepository.Create(farm)
	if err != nil {
		return fmt.Errorf("error creating farm: %w", err)
	}
	poolId, err := s.workerManager.CreateFarmImageryWorkerInfrastructure(context.Background(), farm.ID)
	if err != nil {
		return fmt.Errorf("error creating imagery worker infra: %w", err)
	}
	err = s.workerManager.StartFarmImageryWorkerInfrastructure(context.Background(), *poolId)
	if err != nil {
		return fmt.Errorf("error starting imagery worker infra: %w", err)
	}

	currentTime := time.Now()
	previousYearTime := currentTime.AddDate(-1, 0, 0)
	formattedTime := currentTime.Format("2006-01-02")
	previousYearFormattedTime := previousYearTime.Format("2006-01-02")

	// send job
	fullYearJob := worker.JobPayload{
		JobID:      uuid.NewString(),
		Type:       "farm-imagery",
		Params:     map[string]any{"farm_id": farm.ID, "start_date": previousYearFormattedTime, "end_date": formattedTime},
		MaxRetries: 100,
		OneTime:    true,
		RunNow:     true,
	}
	everydayJob := worker.JobPayload{
		JobID:      uuid.NewString(),
		Type:       "farm-imagery",
		Params:     map[string]any{"farm_id": farm.ID, "start_date": "", "end_date": "now"},
		MaxRetries: 100,
		OneTime:    false,
	}
	scheduler, ok := s.workerManager.GetSchedulerByPolicyID(farm.ID)
	if !ok {
		slog.Error("error get farm-imagery scheduler", "error", "scheduler doesn't exist")
	}
	scheduler.AddJob(fullYearJob)
	scheduler.AddJob(everydayJob)
	return nil
}

func (s *FarmService) CreateFarmTx(farm *models.Farm, ownerID string, tx *sqlx.Tx) error {
	if farm.ID == uuid.Nil {
		farm.ID = uuid.New()
	}
	farm.OwnerID = ownerID
	farmcode := utils.GenerateRandomStringWithLength(10)
	farm.FarmCode = &farmcode
	// // Check if farmer has already owned a farm
	// existingFarm, err := s.farmRepository.GetByOwnerID(context.Background(), ownerID)
	// if err != nil && strings.Contains(err.Error(), "no rows in result set") {
	// 	// no existing farm, proceed to create
	// } else if existingFarm != nil {
	// 	return fmt.Errorf("badrequest: farmer has already owned a farm")
	// }
	err := s.farmRepository.CreateTx(tx, farm)
	if err != nil {
		return fmt.Errorf("error creating farm: %w", err)
	}
	poolId, err := s.workerManager.CreateFarmImageryWorkerInfrastructure(context.Background(), farm.ID)
	if err != nil {
		return fmt.Errorf("error creating imagery worker infra: %w", err)
	}
	err = s.workerManager.StartFarmImageryWorkerInfrastructure(context.Background(), *poolId)
	if err != nil {
		return fmt.Errorf("error starting imagery worker infra: %w", err)
	}

	currentTime := time.Now()
	previousYearTime := currentTime.AddDate(-1, 0, 0)
	formattedTime := currentTime.Format("2006-01-02")
	previousYearFormattedTime := previousYearTime.Format("2006-01-02")

	// send job
	fullYearJob := worker.JobPayload{
		JobID:      uuid.NewString(),
		Type:       "farm-imagery",
		Params:     map[string]any{"farm_id": farm.ID, "start_date": previousYearFormattedTime, "end_date": formattedTime},
		MaxRetries: 100,
		OneTime:    true,
		RunNow:     true,
	}
	everydayJob := worker.JobPayload{
		JobID:      uuid.NewString(),
		Type:       "farm-imagery",
		Params:     map[string]any{"farm_id": farm.ID, "start_date": "", "end_date": "now"},
		MaxRetries: 100,
		OneTime:    false,
	}
	scheduler, ok := s.workerManager.GetSchedulerByPolicyID(farm.ID)
	if !ok {
		slog.Error("error get farm-imagery scheduler", "error", "scheduler doesn't exist")
	}
	scheduler.AddJob(fullYearJob)
	scheduler.AddJob(everydayJob)
	return nil
}

func (s *FarmService) GetAllFarms(ctx context.Context) ([]models.Farm, error) {
	return s.farmRepository.GetAll(ctx)
}

func (s *FarmService) GetByFarmID(ctx context.Context, farmID string) (*models.Farm, error) {
	return s.farmRepository.GetFarmByID(ctx, farmID)
}

func (s *FarmService) UpdateFarm(ctx context.Context, farm *models.Farm, updatedBy string, farmID string) error {
	// check if farm exists
	_, err := s.farmRepository.GetFarmByID(ctx, farmID)
	if err != nil {
		return err
	}

	farm.ID, err = uuid.Parse(farmID)

	// Validate required fields
	if farm.CropType == "" {
		return fmt.Errorf("badrequest: crop_type is required")
	}
	if farm.AreaSqm <= 0 {
		return fmt.Errorf("badrequest: area_sqm must be greater than 0")
	}
	isDuplicateFarmCode := false
	// check if farm_code has already existed
	if farm.FarmCode != nil {
		existingFarm, err := s.farmRepository.GetFarmByFarmCode(*farm.FarmCode)
		if err != nil && strings.Contains(err.Error(), "no rows in result set") {
			isDuplicateFarmCode = false
		}

		if existingFarm != nil && existingFarm.ID != farm.ID {
			isDuplicateFarmCode = true
		}
	}

	if isDuplicateFarmCode {
		return fmt.Errorf("badrequest: farm_code already exists")
	}

	// Validate expected_harvest_date is after or equal to planting_date if provided
	if farm.ExpectedHarvestDate != nil {
		if farm.PlantingDate == nil {
			return fmt.Errorf("badrequest: planting_date is required when expected_harvest_date is provided")
		}
		if *farm.ExpectedHarvestDate < *farm.PlantingDate {
			return fmt.Errorf("badrequest: expected_harvest_date must be greater than or equal to planting_date")
		}
	}

	return s.farmRepository.Update(farm)
}

func (s *FarmService) DeleteFarm(ctx context.Context, id string, deletedBy string) error {
	// check if farm exists
	farmID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid farm ID: %w", err)
	}

	existFarm, err := s.farmRepository.GetFarmByID(ctx, id)
	if err != nil {
		return err
	}

	if existFarm == nil {
		return fmt.Errorf("not found: farm not found")
	}

	// check if user is authorized to delete farm
	if deletedBy != "" {
		farm, err := s.farmRepository.GetFarmByID(ctx, id)
		if err != nil {
			return err
		}
		if farm.OwnerID != deletedBy {
			return fmt.Errorf("unauthorized to delete farm")
		}
	}

	return s.farmRepository.Delete(farmID)
}

func (s *FarmService) VerifyLandCertificateAPI(nationalIDInput string, token string) (bool, error) {
	apiURl := s.config.VerifyNationalIDURL
	requestBody := models.VerifyNationalIDRequest{
		NationalID: nationalIDInput,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		slog.Error("failed to marshal request body", "error", err)
		return false, fmt.Errorf("badrequest: failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURl, bytes.NewBuffer(jsonBody))
	if err != nil {
		slog.Error("failed to create HTTP request", "error", err)
		return false, fmt.Errorf("internal_error: failed to create HTTP request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	req.Host = s.config.VerifyLandCertificateHostAPI
	slog.Info("sending request to Verify National ID API", "url", apiURl, "host", req.Host)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to send HTTP request", "error", err)
		return false, fmt.Errorf("internal_error: failed to send HTTP request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body", "error", err)
		return false, fmt.Errorf("internal_error: failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp models.VerifyNationalIDErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			slog.Error("failed to unmarshal error response of API Verify nationalID", "error", err, "response_body", string(body))
			return false, fmt.Errorf("internal_error")
		}
		return false, fmt.Errorf("API Verify nationalID error: code=%s, message=%s", errorResp.Error.Code, errorResp.Error.Message)
	}

	var successResp models.VerifyNationalIDResponse
	if err := json.Unmarshal(body, &successResp); err != nil {
		return false, fmt.Errorf("internal_error: failed to unmarshal success response: %w", err)
	}

	return successResp.Data.IsValid, nil
}

func (s *FarmService) VerifyLandCertificate(verifyRequest models.VerifyLandCertificateRequest, farm *models.Farm) (err error) {
	isLandCertificateVerify, err := s.VerifyLandCertificateAPI(verifyRequest.OwnerNationalID, verifyRequest.Token)
	if err != nil {
		return err
	}
	if !isLandCertificateVerify {
		return fmt.Errorf("unauthorized: land certificate verification failed")
	}

	farm.LandOwnershipVerified = true
	secondnow := time.Now().Unix()
	farm.LandOwnershipVerifiedAt = &secondnow

	// upload land certificate image to MinIO
	fileuploadRquest := []minio.FileUpload{}
	for _, photo := range verifyRequest.LandCertificatePhotos {
		fileUpload := minio.FileUpload{
			FileName:  photo.FileName,
			FieldName: photo.FieldName,
			Data:      photo.Data,
		}
		fileuploadRquest = append(fileuploadRquest, fileUpload)
	}

	fileUploadedInfos, err := s.minioClient.FileProcessing(fileuploadRquest, context.Background(), []string{".jpg", ".png", ".jpeg", ".webp"}, 5, "mb")
	if err != nil {
		return err
	}

	landCertificateURLs := minio.JoinResourceURLs(fileUploadedInfos)
	farm.LandCertificateURL = &landCertificateURLs
	return nil
}

// SatelliteImageryResponse represents the response from satellite service
type SatelliteImageryResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    struct {
		Summary struct {
			TotalImages     int `json:"total_images"`
			ImagesProcessed int `json:"images_processed"`
		} `json:"summary"`
		FarmInfo struct {
			Boundary interface{} `json:"boundary"`
			Area     struct {
				Value float64 `json:"value"`
				Unit  string  `json:"unit"`
			} `json:"area"`
		} `json:"farm_info"`
		Images []struct {
			ImageIndex      int    `json:"image_index"`
			ImageID         string `json:"image_id,omitempty"`
			ProductID       string `json:"product_id,omitempty"`
			AcquisitionDate string `json:"acquisition_date"`
			CloudCover      struct {
				Value float64 `json:"value"`
				Unit  string  `json:"unit"`
			} `json:"cloud_cover"`
			Visualization struct {
				NaturalColor struct {
					URL         string `json:"url"`
					Description string `json:"description"`
				} `json:"natural_color"`
			} `json:"visualization"`
		} `json:"images"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GetFarmPhotoJob fetches satellite imagery from satellite-data-service and saves to database
func (s *FarmService) GetFarmPhotoJob(params map[string]any) error {
	// Extract farm_id from params
	farmIDStr, ok := params["farm_id"].(string)
	if !ok {
		slog.Error("GetFarmPhotoJob: missing or invalid farm_id parameter")
		return fmt.Errorf("missing or invalid farm_id parameter")
	}

	farmID, err := uuid.Parse(farmIDStr)
	if err != nil {
		slog.Error("GetFarmPhotoJob: invalid farm_id format", "error", err)
		return fmt.Errorf("invalid farm_id format: %w", err)
	}

	slog.Info("GetFarmPhotoJob: starting fetch", "farm_id", farmID)

	// 1. Get farm details to retrieve boundary
	farm, err := s.farmRepository.GetFarmByID(context.Background(), farmID.String())
	if err != nil {
		slog.Error("GetFarmPhotoJob: failed to get farm", "farm_id", farmID, "error", err)
		return fmt.Errorf("failed to get farm: %w", err)
	}

	if farm.Boundary == nil {
		slog.Error("GetFarmPhotoJob: farm has no boundary defined", "farm_id", farmID)
		return fmt.Errorf("farm has no boundary defined")
	}

	// 2. Extract coordinates from GeoJSON boundary (first ring of polygon)
	coordinates := farm.Boundary.Coordinates[0]
	coordsJSON, err := json.Marshal(coordinates)
	if err != nil {
		slog.Error("GetFarmPhotoJob: failed to marshal coordinates", "farm_id", farmID, "error", err)
		return fmt.Errorf("failed to marshal coordinates: %w", err)
	}

	SatelliteDataServiceURL := "http://satellite-data-service:8000"

	// 3. Build GET request with query parameters
	apiURL := fmt.Sprintf("%s/satellite/public/boundary/imagery", SatelliteDataServiceURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		slog.Error("GetFarmPhotoJob: failed to create HTTP request", "farm_id", farmID, "error", err)
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	startDate, ok := params["start_date"].(string)
	if !ok {
		slog.Error("GetFarmPhotoJob: missing or invalid start_date parameter", "farm_id", farmID)
		return fmt.Errorf("missing or invalid start_date parameter")
	}
	endDate, ok := params["end_date"].(string)
	if !ok {
		slog.Error("GetFarmPhotoJob: missing or invalid end_date parameter", "farm_id", farmID)
		return fmt.Errorf("missing or invalid end_date parameter")
	}

	if endDate == "now" {
		currentTime := time.Now()
		lastDay := currentTime.Add(24 * time.Hour)
		endDate = currentTime.Format("2006-01-02")
		startDate = lastDay.Format("2006-01-02")
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("coordinates", string(coordsJSON))
	q.Add("start_date", startDate)
	q.Add("end_date", endDate)
	q.Add("max_cloud_cover", "100.0")
	req.URL.RawQuery = q.Encode()

	slog.Info("GetFarmPhotoJob: calling satellite service", "farm_id", farmID, "url", req.URL.String())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("GetFarmPhotoJob: failed to call satellite service", "farm_id", farmID, "error", err)
		return fmt.Errorf("failed to call satellite service: %w", err)
	}
	defer resp.Body.Close()

	// 4. Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("GetFarmPhotoJob: failed to read response body", "farm_id", farmID, "error", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("GetFarmPhotoJob: satellite service returned error", "farm_id", farmID, "status_code", resp.StatusCode, "response_body", string(body))

		var errorResp SatelliteImageryResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != nil {
			return fmt.Errorf("satellite service error: %s - %s", errorResp.Error.Code, errorResp.Error.Message)
		}
		return fmt.Errorf("satellite service returned status %d", resp.StatusCode)
	}

	var satelliteResp SatelliteImageryResponse
	if err := json.Unmarshal(body, &satelliteResp); err != nil {
		slog.Error("GetFarmPhotoJob: failed to unmarshal response", "farm_id", farmID, "error", err)
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if satelliteResp.Status != "success" {
		slog.Error("GetFarmPhotoJob: satellite service returned error status", "farm_id", farmID, "status", satelliteResp.Status)
		if satelliteResp.Error != nil {
			return fmt.Errorf("satellite service error: %s - %s", satelliteResp.Error.Code, satelliteResp.Error.Message)
		}
		return fmt.Errorf("satellite service returned status=%s", satelliteResp.Status)
	}

	// 5. Save images to database
	if len(satelliteResp.Data.Images) == 0 {
		slog.Info("GetFarmPhotoJob: no images returned", "farm_id", farmID)
		return nil // Not an error, just no images available
	}

	slog.Info("GetFarmPhotoJob: retrieved images from satellite service", "farm_id", farmID, "image_count", len(satelliteResp.Data.Images))

	savedCount := 0
	for _, img := range satelliteResp.Data.Images {
		// Parse acquisition date to Unix timestamp
		var takenAt *int64
		if img.AcquisitionDate != "" {
			t, err := time.Parse("2006-01-02", img.AcquisitionDate)
			if err == nil {
				timestamp := t.Unix()
				takenAt = &timestamp
			} else {
				slog.Warn("GetFarmPhotoJob: failed to parse date", "farm_id", farmID, "date", img.AcquisitionDate, "error", err)
			}
		}

		photo := &models.FarmPhoto{
			FarmID:    farmID,
			PhotoURL:  img.Visualization.NaturalColor.URL,
			PhotoType: models.PhotoSatellite,
			TakenAt:   takenAt,
		}

		err := s.farmRepository.CreateFarmPhoto(photo)
		if err != nil {
			slog.Error("GetFarmPhotoJob: failed to save photo", "farm_id", farmID, "url", img.Visualization.NaturalColor.URL, "error", err)
			// Continue with other photos even if one fails
			continue
		}
		savedCount++
	}

	slog.Info("GetFarmPhotoJob: successfully saved photos", "farm_id", farmID, "saved_count", savedCount, "total_images", len(satelliteResp.Data.Images))

	if savedCount == 0 && len(satelliteResp.Data.Images) > 0 {
		return fmt.Errorf("failed to save any photos to database")
	}

	return nil
}

func (s *FarmService) CreateFarmValidate(farm *models.Farm, token string) error {
	// Validate required fields
	if farm.CropType == "" {
		return fmt.Errorf("bad_request: crop_type is required")
	}
	if farm.AreaSqm <= 0 {
		return fmt.Errorf("bad_request: area_sqm must be greater than 0")
	}

	// Validate harvest date if provided
	if farm.ExpectedHarvestDate != nil {
		if farm.PlantingDate == nil {
			return fmt.Errorf("bad_request: planting_date is required when expected_harvest_date is provided")
		}
		if *farm.ExpectedHarvestDate < *farm.PlantingDate {
			return fmt.Errorf("bad_request: expected_harvest_date must be greater than or equal to planting_date")
		}
	}

	if farm.OwnerNationalID == nil {
		return fmt.Errorf("bad_request: owner_national_id is required")
	}

	verifyLandCerRequest := models.VerifyLandCertificateRequest{
		OwnerNationalID:       *farm.OwnerNationalID,
		Token:                 token,
		LandCertificatePhotos: farm.LandCertificatePhotos,
	}

	if err := s.VerifyLandCertificate(verifyLandCerRequest, farm); err != nil {
		return err
	}
	return nil
}

func (s *FarmService) CheckFarmOwner(ownerID string, farmID string) (bool, error) {
	farm, err := s.farmRepository.GetFarmByID(context.Background(), farmID)
	if err != nil {
		return false, err
	}
	if farm.OwnerID != ownerID {
		slog.Warn("owner ID mismatch", "farm_id", farmID, "expected_owner", farm.OwnerID, "provided_owner", ownerID)
		return false, fmt.Errorf("unauthorize: ower id mismatch")
	}
	return true, nil
}

func (s *FarmService) FarmJobRecovery() error {
	slog.Info("FarmJobRecovery: starting farm job recovery process")

	// 1. Get all farms from database
	farms, err := s.farmRepository.GetAll(context.Background())
	if err != nil {
		slog.Error("FarmJobRecovery: failed to get all farms", "error", err)
		return fmt.Errorf("failed to get all farms: %w", err)
	}

	slog.Info("FarmJobRecovery: found farms to recover", "farm_count", len(farms))

	successCount := 0
	failCount := 0

	// 2. Process each farm
	for _, farm := range farms {
		slog.Info("FarmJobRecovery: processing farm", "farm_id", farm.ID, "owner_id", farm.OwnerID)

		// Create worker infrastructure for this farm
		poolId, err := s.workerManager.CreateFarmImageryWorkerInfrastructure(context.Background(), farm.ID)
		if err != nil {
			slog.Error("FarmJobRecovery: failed to create imagery worker infra", "farm_id", farm.ID, "error", err)
			failCount++
			continue // Continue with next farm
		}

		// Start worker infrastructure
		err = s.workerManager.StartFarmImageryWorkerInfrastructure(context.Background(), *poolId)
		if err != nil {
			slog.Error("FarmJobRecovery: failed to start imagery worker infra", "farm_id", farm.ID, "error", err)
			failCount++
			continue // Continue with next farm
		}

		// Calculate date ranges
		currentTime := time.Now()
		previousYearTime := currentTime.AddDate(-1, 0, 0)
		formattedTime := currentTime.Format("2006-01-02")
		previousYearFormattedTime := previousYearTime.Format("2006-01-02")

		// Create full year job (one-time, run now)
		fullYearJob := worker.JobPayload{
			JobID:      uuid.NewString(),
			Type:       "farm-imagery",
			Params:     map[string]any{"farm_id": farm.ID, "start_date": previousYearFormattedTime, "end_date": formattedTime},
			MaxRetries: 100,
			OneTime:    true,
			RunNow:     true,
		}

		// Create everyday job (recurring)
		everydayJob := worker.JobPayload{
			JobID:      uuid.NewString(),
			Type:       "farm-imagery",
			Params:     map[string]any{"farm_id": farm.ID, "start_date": "", "end_date": "now"},
			MaxRetries: 100,
			OneTime:    false,
		}

		// Get scheduler and add jobs
		scheduler, ok := s.workerManager.GetSchedulerByPolicyID(*worker.AIWorkerPoolUUID)
		if !ok {
			slog.Error("FarmJobRecovery: failed to get farm-imagery scheduler", "farm_id", farm.ID, "error", "scheduler doesn't exist")
			failCount++
			continue
		}

		scheduler.AddJob(fullYearJob)
		scheduler.AddJob(everydayJob)

		slog.Info("FarmJobRecovery: successfully recovered farm", "farm_id", farm.ID, "jobs_added", 2)
		successCount++
	}

	slog.Info("FarmJobRecovery: recovery complete", "success_count", successCount, "fail_count", failCount, "total_farms", len(farms))

	if failCount > 0 {
		return fmt.Errorf("farm job recovery completed with %d failures out of %d farms", failCount, len(farms))
	}

	return nil
}
