package services

import (
	utils "agrisa_utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"policy-service/internal/database/minio"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"strings"
	"time"

	"policy-service/internal/config"

	"github.com/google/uuid"
)

type FarmService struct {
	farmRepository *repository.FarmRepository
	config         *config.PolicyServiceConfig
	minioClient    *minio.MinioClient
}

func NewFarmService(farmRepo *repository.FarmRepository, cfg *config.PolicyServiceConfig, minioClient *minio.MinioClient) *FarmService {
	return &FarmService{farmRepository: farmRepo, config: cfg, minioClient: minioClient}
}

func (s *FarmService) GetFarmByOwnerID(ctx context.Context, userID string) ([]models.Farm, error) {

	farms, err := s.farmRepository.GetByOwnerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return farms, nil
}

func (s *FarmService) CreateFarm(farm *models.Farm, ownerID string) error {
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

	return s.farmRepository.Create(farm)
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
		log.Printf("failed to marshal request body: %v", err)
		return false, fmt.Errorf("badrequest: failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", apiURl, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("failed to create HTTP request: %v", err)
		return false, fmt.Errorf("internal_error: failed to create HTTP request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	req.Host = "localhost"
	// log api url
	log.Printf("Sending request to Verify National ID API: %s", apiURl)
	// log host
	log.Printf("Host: %s", req.Host)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send HTTP request: %v", err)
		return false, fmt.Errorf("internal_error: failed to send HTTP request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response body: %v", err)
		return false, fmt.Errorf("internal_error: failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp models.VerifyNationalIDErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			log.Printf("Failed to unmarshal error response of API Verify nationalID: %v", err)
			log.Printf("Response body of API Verify nationalID: %s", string(body))
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
