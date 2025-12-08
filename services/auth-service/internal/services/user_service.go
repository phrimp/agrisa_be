package services

import (
	agrisa_utils "agrisa_utils"
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"auth-service/internal/config"
	"auth-service/internal/database/minio"
	"auth-service/internal/event"
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"auth-service/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type IUserService interface {
	RegisterNewUser(phone, email, password, nationalID string, phoneVerificationStatus, isDefault bool) (*models.User, error)
	Login(email, phone, password string, deviceInfo, ipAddress *string) (*models.User, *models.UserSession, error)
	GetUserByID(userID string) (*models.User, error)
	BanUser(userID string, until int64) error
	UnbanUser(userID string) error
	GetAllUsers(limit, offset int) (*models.GetAllUsersResponse, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserEkycProgressByUserID(userID string) (*models.UserEkycProgress, error)
	UploadToMinIO(c *gin.Context, file io.Reader, header *multipart.FileHeader, serviceName string) error
	ProcessAndUploadFiles(files map[string][]*multipart.FileHeader, serviceName string, allowedExts []string, maxMB int64) ([]utils.FileInfo, error)
	OCRNationalIDCard(form *multipart.Form) (interface{}, error)
	VerifyFaceLiveness(form *multipart.Form) (interface{}, error)
	VerifyLandCertificate(userID string, NationalIDInput string) (result bool, err error)
	CheckExistEmailOrPhone(input string) (bool, error)
	GetUserCardByUserID(userID string) (*models.UserCard, error)
	ResetEkycData(userID string) error
	UpdateUserCardByUserID(userID string, req models.UpdateUserCardRequest) error
	GeneratePhoneOTP(ctx context.Context, phoneNumber string) error
	ValidatePhoneOTP(ctx context.Context, phoneNumber, otp string) error
}

type UserService struct {
	userRepo         repository.IUserRepository
	minioClient      *minio.MinioClient
	cfg              *config.AuthServiceConfig
	utils            *utils.Utils
	userCardRepo     repository.IUserCardRepository
	ekycProgressRepo repository.IUserEkycProgressRepository
	sessionService   *SessionService
	roleService      *RoleService
	jwtService       *JWTService
	eventPublisher   *event.NotificationPublisher

	globalLoginAttempt map[string]int
	mu                 *sync.Mutex
	redisClient        *redis.Client
}

func NewUserService(userRepo repository.IUserRepository, minioClient *minio.MinioClient, cfg *config.AuthServiceConfig, utils *utils.Utils, userCardRepo repository.IUserCardRepository, ekycProgressRepo repository.IUserEkycProgressRepository, sessionService *SessionService, jwtService *JWTService, roleService *RoleService, eventPublisher *event.NotificationPublisher) IUserService {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisCfg.Host, cfg.RedisCfg.Port),
		Password: cfg.RedisCfg.Password,
		DB:       cfg.RedisCfg.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	return &UserService{
		userRepo:           userRepo,
		minioClient:        minioClient,
		cfg:                cfg,
		utils:              utils,
		userCardRepo:       userCardRepo,
		ekycProgressRepo:   ekycProgressRepo,
		sessionService:     sessionService,
		jwtService:         jwtService,
		roleService:        roleService,
		globalLoginAttempt: make(map[string]int),
		mu:                 &sync.Mutex{},
		redisClient:        rdb,
		eventPublisher:     eventPublisher,
	}
}

func (s *UserService) GetUserEkycProgressByUserID(userID string) (*models.UserEkycProgress, error) {
	return s.ekycProgressRepo.GetUserEkycProgressByUserID(userID)
}

func (s *UserService) UploadToMinIO(c *gin.Context, file io.Reader, header *multipart.FileHeader, serviceName string) error {
	// Lấy thông tin file
	fileName := header.Filename
	fileSize := header.Size
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx := c.Request.Context()
	return s.minioClient.UploadFile(ctx, fileName, contentType, file, fileSize, serviceName)
}

func (s *UserService) GetUserByID(userID string) (*models.User, error) {
	return s.userRepo.GetUserByID(userID)
}

func (s *UserService) ProcessAndUploadFiles(files map[string][]*multipart.FileHeader,
	serviceName string, allowedExts []string, maxMB int64,
) ([]utils.FileInfo, error) {
	return s.utils.ProcessFiles(s.minioClient, files, serviceName, allowedExts, maxMB)
}

func (s *UserService) OCRNationalIDCard(form *multipart.Form) (interface{}, error) {
	userIDs := form.Value["user_id"]
	if len(userIDs) == 0 {
		log.Printf("Error: user_id is required in the form data")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "user_id is required",
			},
		}, nil
	}

	userID := userIDs[0]

	frontFiles := form.File["cccd_front"]
	if len(frontFiles) == 0 {
		log.Printf("Error: cccd_front file is required")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "cccd_front file is required",
			},
		}, nil
	}

	backFiles := form.File["cccd_back"]
	if len(backFiles) == 0 {
		log.Printf("Error: cccd_back file is required")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "cccd_back file is required",
			},
		}, nil
	}
	frontHeader := frontFiles[0]
	backHeader := backFiles[0]

	// Step 1: GetUserEkycProgressByUserID
	userEkycProgress, err := s.ekycProgressRepo.GetUserEkycProgressByUserID(userID)
	if err != nil {
		log.Printf("Failed to get user ekyc progress: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get user ekyc progress",
			},
		}, nil
	}
	// Step 1.1: Found progress
	if userEkycProgress.IsOcrDone {
		// Step 2.1: Already verified
		log.Printf("User %s has already completed OCR Card verification", userID)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "ALREADY_OCR_DONE",
				Message: "User has already completed OCR Card verification",
			},
		}, nil
	}
	// Step 3: Create OCR front request
	frontFile, err := frontHeader.Open()
	if err != nil {
		log.Printf("Error when opening cccd_front: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "Error when opening cccd_front",
			},
		}, nil
	}
	defer frontFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", frontHeader.Filename)
	if err != nil {
		log.Printf("Error when creating form file: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when creating form file",
			},
		}, nil
	}
	_, err = io.Copy(part, frontFile)
	if err != nil {
		log.Printf("Error when copying front file: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when copying front file",
			},
		}, nil
	}
	writer.Close()

	// Step 4: Send OCR front request
	req, err := http.NewRequest("POST", s.cfg.AuthCfg.FptOcrUrl, body)
	if err != nil {
		log.Printf("Failed to create OCR front request: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create OCR front request",
			},
		}, nil
	}
	req.Header.Add("api-key", s.cfg.AuthCfg.FptEkycApiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error when sending front OCR request: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when sending front OCR request",
			},
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error when reading front OCR response: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when reading front OCR response",
			},
		}, nil
	}

	// Step 5: Check front OCR response status
	if resp.StatusCode != http.StatusOK {
		// Step 5.1: Return FPT response
		log.Printf("FPT OCR front API error: %s", string(respBody))
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "EXTERNAL_API_ERROR",
				Message: "FPT OCR front API error",
			},
		}, nil
	}

	// Step 6: Parse front OCR response
	var idCardOcrFrontResponse map[string]interface{}
	if err := json.Unmarshal(respBody, &idCardOcrFrontResponse); err != nil {
		log.Printf("Error when parsing front OCR response: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when parsing front OCR response",
			},
		}, nil
	}

	data, ok := idCardOcrFrontResponse["data"].([]interface{})
	if !ok || len(data) == 0 {
		log.Printf("Error: missing or invalid data in front OCR response")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "missing or invalid data in front OCR response",
			},
		}, nil
	}
	frontData := data[0].(map[string]interface{})
	nationalID, ok := frontData["id"].(string)
	if !ok {
		log.Printf("Error: missing id in front OCR response")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "missing id in front OCR response",
			},
		}, nil
	}

	// Step 7: Create OCR back request
	backFile, err := backHeader.Open()
	if err != nil {
		log.Printf("Error when opening cccd_back: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "Error when opening cccd_back",
			},
		}, nil
	}
	defer backFile.Close()

	bodyBack := &bytes.Buffer{}
	writerBack := multipart.NewWriter(bodyBack)
	partBack, err := writerBack.CreateFormFile("image", backHeader.Filename)
	if err != nil {
		log.Printf("Error when creating form file for back: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when creating form file for back",
			},
		}, nil
	}
	_, err = io.Copy(partBack, backFile)
	if err != nil {
		log.Printf("Error when copying back file: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when copying back file",
			},
		}, nil
	}
	writerBack.Close()

	// Step 9: Send OCR back request
	reqBack, err := http.NewRequest("POST", s.cfg.AuthCfg.FptOcrUrl, bodyBack)
	if err != nil {
		log.Printf("Error when creating back OCR request: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when creating back OCR request",
			},
		}, nil
	}
	reqBack.Header.Add("api-key", s.cfg.AuthCfg.FptEkycApiKey)
	reqBack.Header.Set("Content-Type", writerBack.FormDataContentType())

	respBack, err := client.Do(reqBack)
	if err != nil {
		log.Printf("Error when sending back OCR request: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when sending back OCR request",
			},
		}, nil
	}
	defer respBack.Body.Close()

	respBodyBack, err := io.ReadAll(respBack.Body)
	if err != nil {
		log.Printf("Error when reading back OCR response: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when reading back OCR response",
			},
		}, nil
	}

	// Step 11: Check back OCR response status
	if respBack.StatusCode != http.StatusOK {
		// Step 11.1: Return FPT back response
		log.Printf("FPT OCR back API error: %s", string(respBodyBack))
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "EXTERNAL_API_ERROR",
				Message: "FPT OCR back API error",
			},
		}, nil
	}

	// Step 10: Parse back OCR response
	var idCardOcrBackResponse map[string]interface{}
	if err := json.Unmarshal(respBodyBack, &idCardOcrBackResponse); err != nil {
		log.Printf("Error when parsing back OCR response: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when parsing back OCR response",
			},
		}, nil
	}

	// Step 12: UpdateUserNationalID
	err = s.userRepo.UpdateUserNationalID(userID, nationalID)
	if err != nil {
		// Step 12.1: Log error and return internal error
		log.Printf("Failed to update user national ID: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update user national ID",
			},
		}, nil
	}

	// Step 13: Create URL variables
	var cccdFrontAccessURL, cccdBackAccessURL string

	// Step 14: Upload files to MinIO
	uploadedFiles, err := (s.utils.ProcessFiles(s.minioClient, form.File, "auth-service", []string{".jpg", ".png", ".jpeg"}, 50))
	if err != nil {
		// Step 14: Handle upload error
		log.Printf("Failed to upload files to MinIO: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to upload files to storage",
			},
		}, nil
	}

	// Step 15: Assign URLs from uploaded files
	for _, fileInfo := range uploadedFiles {
		if fileInfo.FieldName == "cccd_front" {
			cccdFrontAccessURL = fileInfo.MinioURL
		} else if fileInfo.FieldName == "cccd_back" {
			cccdBackAccessURL = fileInfo.MinioURL
		}
	}

	// Extract front data
	frontFieldData, ok := idCardOcrFrontResponse["data"].([]interface{})
	if !ok || len(frontFieldData) == 0 {
		log.Printf("invalid front OCR response data: %v", idCardOcrFrontResponse)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "invalid front OCR response data",
			},
		}, nil
	}
	front := frontFieldData[0].(map[string]interface{})

	// Extract back data
	backFieldData, ok := idCardOcrBackResponse["data"].([]interface{})
	if !ok || len(backFieldData) == 0 {
		log.Printf("invalid back OCR response data: %v", idCardOcrBackResponse)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "invalid back OCR response data",
			},
		}, nil
	}
	back := backFieldData[0].(map[string]interface{})

	// Convert mrz array to string
	mrzArray, ok := back["mrz"].([]interface{})
	if !ok {
		log.Printf("invalid mrz format in back OCR response: %v", back)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "invalid mrz format in back OCR response",
			},
		}, nil
	}
	mrzStrings := make([]string, len(mrzArray))
	for i, v := range mrzArray {
		mrzStrings[i], _ = v.(string)
	}
	mrz := strings.Join(mrzStrings, ", ")

	userCard := models.UserCard{
		NationalID:        front["id"].(string),
		Name:              front["name"].(string),
		Dob:               front["dob"].(string),
		Sex:               front["sex"].(string),
		Nationality:       front["nationality"].(string),
		Home:              front["home"].(string),
		Address:           front["address"].(string),
		Doe:               front["doe"].(string),
		NumberOfNameLines: front["number_of_name_lines"].(string),
		Features:          back["features"].(string),
		IssueDate:         back["issue_date"].(string),
		Mrz:               mrz,
		IssueLoc:          back["issue_loc"].(string),
		ImageFront:        cccdFrontAccessURL,
		ImageBack:         cccdBackAccessURL,
		UserID:            userID,
	}

	_, err = s.userCardRepo.CreateUserCard(&userCard)
	if err != nil {
		log.Printf("Failed to create user card record: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create user card record",
			},
		}, nil
	}

	// Step 16: Update Ekyc Progress
	err = s.ekycProgressRepo.UpdateOCRDone(userID, true, nationalID)
	if err != nil {
		log.Printf("Failed to update ekyc progress: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update ekyc progress",
			},
		}, nil
	}
	// Step 17: Get ekyc progress to include in response
	updatedEkycProgress, err := s.ekycProgressRepo.GetUserEkycProgressByUserID(userID)
	if err != nil {
		log.Printf("Failed to get updated ekyc progress: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get updated ekyc progress",
			},
		}, nil
	}

	// Step 18: Return success response
	return utils.SuccessResponse{
		Success: true,
		Data:    updatedEkycProgress,
		Meta:    &utils.Meta{Timestamp: time.Now()},
	}, nil
}

func (s *UserService) VerifyFaceLiveness(form *multipart.Form) (interface{}, error) {
	userIDs := form.Value["user_id"]
	if len(userIDs) == 0 {
		log.Printf("Error: user_id is required in the form data")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "user_id is required",
			},
		}, nil
	}

	userID := userIDs[0]

	// Verify if user has done OCR before
	ekycProgress, err := s.ekycProgressRepo.GetUserEkycProgressByUserID(userID)
	if err != nil {
		log.Printf("Failed to get ekyc progress: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get ekyc progress",
			},
		}, nil
	}
	if !ekycProgress.IsOcrDone {
		log.Printf("Error: user has not completed OCR")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "User has not completed OCR",
			},
		}, nil
	}
	if ekycProgress.IsFaceVerified {
		log.Printf("Error: user has already completed face liveness")
		return utils.CreateErrorResponse("ALREADY_FACE_LIVENESS_DONE", "User has already completed face liveness"), nil
	}

	// Get video file
	videos := form.File["video"]
	if len(videos) == 0 {
		log.Printf("Error: failed to get video file")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "Failed to get video file",
			},
		}, nil
	}

	videoHeader := videos[0]
	srcVideo, err := videoHeader.Open()
	if err != nil {
		log.Printf("Error when opening video file: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "Error when opening video file",
			},
		}, nil
	}
	defer srcVideo.Close()

	videoBuffer := &bytes.Buffer{}
	_, err = io.Copy(videoBuffer, srcVideo)
	if err != nil {
		log.Printf("Error when reading video file: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when reading video file",
			},
		}, nil
	}

	// getimage file
	images := form.File["cmnd"]
	if len(images) == 0 {
		log.Printf("Error: failed to get cmnd file")
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "Failed to get cmnd file",
			},
		}, nil
	}

	imageHeader := images[0]
	srcImage, err := imageHeader.Open()
	if err != nil {
		log.Printf("Error when opening cmnd file: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "BAD_REQUEST",
				Message: "Error when opening cmnd file",
			},
		}, nil
	}
	defer srcImage.Close()

	imageBuffer := &bytes.Buffer{}
	_, err = io.Copy(imageBuffer, srcImage)
	if err != nil {
		log.Printf("Error when reading cmnd file: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when reading cmnd file",
			},
		}, nil
	}

	// send API
	url := s.cfg.AuthCfg.FptFaceLivenessUrl
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	partVideo, err := writer.CreateFormFile("video", "face_video.mp4")
	if err != nil {
		log.Printf("Error when creating form file for video: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when creating form file for video",
			},
		}, nil
	}

	_, err = io.Copy(partVideo, videoBuffer)
	if err != nil {
		log.Printf("Error when copying video file to form: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when copying video file to form",
			},
		}, nil
	}

	partImage, err := writer.CreateFormFile("cmnd", "cccc_front.jpg")
	if err != nil {
		log.Printf("Error when creating form file for image: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when creating form file for image",
			},
		}, nil
	}
	_, err = io.Copy(partImage, imageBuffer)
	if err != nil {
		log.Printf("Error when copying image file to form: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when copying image file to form",
			},
		}, nil
	}

	err = writer.Close()
	if err != nil {
		log.Printf("Error when closing multipart writer: %v", err)
		return utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "Error when closing multipart writer",
			},
		}, nil
	}

	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		log.Printf("Error when creating request: %v", err)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Error when creating request"), nil
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("api-key", s.cfg.AuthCfg.FptEkycApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error when sending request: %v", err)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Error when sending request"), nil
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error when reading response body: %v", err)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Error when reading response body"), nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		log.Printf("Error when parsing response body: %v", err)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Error when parsing response body"), nil
	}

	if code, ok := result["code"].(string); ok {
		if code != "200" {
			message, ok := result["message"].(string)
			if ok {
				return utils.CreateErrorResponse("EXTERNAL_API_ERROR", "Face liveness failed: "+message), nil
			}
			return utils.CreateErrorResponse("EXTERNAL_API_ERROR", "Face liveness failed: Unknown error"), nil
		}
	}

	videoAccessURL := ""

	fileInfos, err := s.utils.ProcessFiles(s.minioClient, form.File, "auth-service", []string{".mp4", ".jpg", ".png"}, 50)
	if err != nil {
		log.Printf("Error when processing files: %v", err)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Error when processing files"), nil
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.FieldName == "video" {
			videoAccessURL = fileInfo.MinioURL
			break
		}
	}

	// Update user face liveness URL
	errorUpdateUserFaceLiveness := s.userRepo.UpdateUserFaceLiveness(userID, videoAccessURL)
	if errorUpdateUserFaceLiveness != nil {
		log.Printf("Failed to update user face liveness: %v", errorUpdateUserFaceLiveness)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Failed to update user face liveness"), nil
	}

	// Update ekyc progress (face liveness done)
	errorUpdateEkycProgress := s.ekycProgressRepo.UpdateFaceLivenessDone(userID, true)
	if errorUpdateEkycProgress != nil {
		log.Printf("Failed to update ekyc progress: %v", errorUpdateEkycProgress)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Failed to update ekyc progress"), nil
	}

	ekycProgressUpdated, err := s.ekycProgressRepo.GetUserEkycProgressByUserID(userID)
	if err != nil {
		log.Printf("Failed to get updated ekyc progress: %v", err)
		return utils.CreateErrorResponse("INTERNAL_ERROR", "Failed to get updated ekyc progress"), nil
	}

	if ekycProgressUpdated.IsOcrDone && ekycProgressUpdated.IsFaceVerified {
		errorUpdateUserEkycStatus := s.userRepo.UpdateUserKycStatus(userID, true)
		if errorUpdateUserEkycStatus != nil {
			log.Printf("Failed to update user status: %v", errorUpdateUserEkycStatus)
			return utils.CreateErrorResponse("INTERNAL_ERROR", "Failed to update user status"), nil
		}
	}

	return utils.CreateSuccessResponse(ekycProgressUpdated), nil
}

func (s *UserService) RegisterNewUser(phone, email, password, nationalID string, phoneVerificationStatus, isDefault bool) (*models.User, error) {
	if isDefault {
		newUser := models.User{
			ID:            "UC" + agrisa_utils.GenerateRandomStringWithLength(8),
			PhoneNumber:   phone,
			Email:         email,
			PasswordHash:  password,
			NationalID:    nationalID,
			Status:        models.UserStatusPendingVerification,
			PhoneVerified: phoneVerificationStatus,
			LockedUntil:   0,
			FaceLiveness:  nil,
		}
		err := s.userRepo.CreateUser(&newUser)
		if err != nil {
			return nil, fmt.Errorf("error creating new default user: %s", err)
		}
		return &newUser, nil

	}

	if isvalid, err := agrisa_utils.ValidateEmail(email); !isvalid {
		return nil, fmt.Errorf("error validating email: %s", err)
	}

	if isvalid, err := agrisa_utils.ValidatePhone(phone); !isvalid {
		return nil, fmt.Errorf("error validating phone: %s", err)
	}

	passwordNumberRegex := regexp.MustCompile(`[0-9]`)
	passwordLetterRegex := regexp.MustCompile(`[a-zA-Z]`)
	passwordSpecialRegex := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`)

	if len(password) < 8 || !passwordNumberRegex.MatchString(password) || !passwordLetterRegex.MatchString(password) || !passwordSpecialRegex.MatchString(password) {
		return nil, fmt.Errorf("error: password format incorrect")
	}

	if !agrisa_utils.ValidateCCCD(nationalID) {
		return nil, fmt.Errorf("error: cccd format incorrect")
	}

	newUser := models.User{
		ID:            "UC" + agrisa_utils.GenerateRandomStringWithLength(8),
		PhoneNumber:   phone,
		Email:         email,
		PasswordHash:  password,
		NationalID:    nationalID,
		Status:        models.UserStatusPendingVerification,
		PhoneVerified: phoneVerificationStatus,
		LockedUntil:   0,
		FaceLiveness:  nil,
	}
	err := s.userRepo.CreateUser(&newUser)
	if err != nil {
		return nil, fmt.Errorf("error creating new user: %s", err)
	}

	// create farmer profile
	isSuccess, err := s.CreateFarmerProfile(newUser.ID, phone, email)
	if err != nil && !isSuccess {
		slog.Error("failed to create farmer profile", "error", err)
		return nil, err
	}

	// create ekyc progress
	ekycProgress := models.UserEkycProgress{
		UserID:         newUser.ID,
		CicNo:          nationalID,
		IsOcrDone:      false,
		OcrDoneAt:      nil,
		IsFaceVerified: false,
		FaceVerifiedAt: nil,
	}
	err = s.ekycProgressRepo.CreateUserEkycProgress(&ekycProgress)
	if err != nil {
		log.Printf("error creating ekyc progress for user %s: %s", newUser.ID, err)
		return nil, err
	}
	return &newUser, nil
}

func (s *UserService) CreateFarmerProfile(userID string, phone string, email string) (bool, error) {
	payload := map[string]interface{}{
		"user_id":           userID,
		"role_id":           "user",
		"partner_id":        nil,
		"full_name":         "",
		"display_name":      "",
		"date_of_birth":     "1970-01-01",
		"gender":            "",
		"nationality":       "VN",
		"email":             email,
		"primary_phone":     phone,
		"alternate_phone":   "",
		"permanent_address": "",
		"current_address":   "",
		"province_code":     "",
		"province_name":     "",
		"district_code":     "",
		"district_name":     "",
		"ward_code":         "",
		"ward_name":         "",
		"postal_code":       "",
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal request body", "error", err)
		return false, fmt.Errorf("badrequest: failed to marshal request body: %w", err)
	}

	apiURL := s.cfg.AuthCfg.CreateUserProfileURL

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		slog.Error("failed to create HTTP request", "error", err)
		return false, fmt.Errorf("internal_error: failed to create HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")

	req.Host = s.cfg.AuthCfg.CreateUserProfileHostAPI
	slog.Info("sending request to Verify National ID API", "url", apiURL, "host", req.Host)

	client := &http.Client{Timeout: 120 * time.Second}
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		slog.Error("failed to create farmer profile", "status_code", resp.StatusCode, "response_body", string(body))
		return false, fmt.Errorf("badrequest: failed to create farmer profile, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return true, nil
}

func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	return s.userRepo.GetUserByEmail(email)
}

func (s *UserService) Login(email, phone, password string, deviceInfo, ipAddress *string) (*models.User, *models.UserSession, error) {
	if email != "" && phone != "" {
		log.Println("SUSPICIOUS ACTIVITY DETECTED : email & phone present reached service layer and blocked")
		return nil, nil, fmt.Errorf("action forbidden")
	}
	var login_attempt_user *models.User
	var err error

	// Try cache first, then database
	if email != "" {
		login_attempt_user = s.getCachedUserByEmail(email)
		if login_attempt_user == nil {
			login_attempt_user, err = s.userRepo.GetUserByEmail(email)
			if err != nil {
				log.Printf("user searching failed: %s \n", err)
				return nil, nil, fmt.Errorf("email or password incorrect: %s", err)
			}
			// Cache the user for future requests
			s.cacheUser(login_attempt_user)
		}
	}
	if phone != "" {
		login_attempt_user = s.getCachedUserByPhone(phone)
		if login_attempt_user == nil {
			login_attempt_user, err = s.userRepo.GetUserByPhone(phone)
			if err != nil {
				log.Printf("user searching failed: %s \n", err)
				return nil, nil, fmt.Errorf("phone number or password incorrect: %s", err)
			}
			// Cache the user for future requests
			s.cacheUser(login_attempt_user)
		}
	}
	if login_attempt_user == nil {
		return nil, nil, fmt.Errorf("UNEXPECTED ERROR : user found but still null")
	}

	if !s.userRepo.CheckPasswordHash(password, login_attempt_user.PasswordHash) {
		attemptCount := s.incrementLoginAttempts(login_attempt_user.ID)

		if attemptCount%5 == 0 {
			// event to notification service to send email/phone of suspicious login activities
			log.Printf("Suspicious login activity detected for user %s: %d attempts", login_attempt_user.ID, attemptCount)
		}
		if attemptCount%10 == 0 {
			log.Println("account blocked due to too many failed login attempts")
			// lock account
			s.BanUser(login_attempt_user.ID, time.Now().Unix()+(int64(attemptCount)*60))
			return nil, nil, fmt.Errorf("account blocked due to too many failed login attempts")
		}
		return nil, nil, fmt.Errorf("invalid password")
	}
	if login_attempt_user.Status == models.UserStatusSuspended {
		// Check if the ban period has expired
		if login_attempt_user.LockedUntil > 0 && time.Now().Unix() > login_attempt_user.LockedUntil {
			// Automatically unban the user
			err := s.UnbanUser(login_attempt_user.ID)
			if err != nil {
				log.Printf("Failed to automatically unban user %s: %v", login_attempt_user.ID, err)
				return nil, nil, fmt.Errorf("account blocked, check email for further information")
			}
			// Update the user object status for this login session
			login_attempt_user.Status = models.UserStatusActive
			login_attempt_user.LockedUntil = 0
		} else {
			// Still banned
			return nil, nil, fmt.Errorf("account blocked, check email for further information")
		}
	}
	if login_attempt_user.Status == models.UserStatusDeactivated {
		// event to email for deactivated account
		return nil, nil, fmt.Errorf("account blocked, check email for further information")
	}

	// get roles
	roles, err := s.roleService.GetUserRoles(login_attempt_user.ID, true)
	if err != nil {
		log.Println("error get user roles: ", err)
		return nil, nil, fmt.Errorf("error get user roles: %s", err)
	}
	roleNames := []string{}
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}

	// gen token
	token, err := s.jwtService.GenerateNewToken(roleNames, login_attempt_user.PhoneNumber, login_attempt_user.Email, login_attempt_user.ID)
	if err != nil {
		log.Println("error generating token: ", err)
		return nil, nil, fmt.Errorf("error generating token: %s", err)
	}

	// gen Login Session
	finalSession := &models.UserSession{}
	// check exist sessions
	sessions, err := s.sessionService.GetUserSessions(context.Background(), login_attempt_user.ID)
	newSessionSignal := true
	if len(sessions) != 0 {
		log.Printf("User %s session exists: %v", login_attempt_user.ID, len(sessions))
		// process existing session
		for _, session := range sessions {
			if *deviceInfo == *session.DeviceInfo {
				log.Printf("New login in the same device, retrieve old session (user id: %s --- session id: %s)", login_attempt_user.ID, session.ID)
				finalSession = session
				newSessionSignal = false
				break
			}
		}
	}

	if newSessionSignal {
		finalSession, err = s.sessionService.CreateSession(context.Background(), login_attempt_user.ID, token, &token, deviceInfo, ipAddress)
		if err != nil {
			log.Println("error creating new session: ", err)
			return nil, nil, fmt.Errorf("error creating new session: %s", err)
		}
		log.Printf("New session created (user id: %s --- session id: %s)", login_attempt_user.ID, finalSession.ID)
	}

	// Reset login attempts on successful login
	s.resetLoginAttempts(login_attempt_user.ID)

	return login_attempt_user, finalSession, nil
}

// Cache helper methods
func (s *UserService) getCachedUserByEmail(email string) *models.User {
	if s.redisClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	key := fmt.Sprintf("user:email:%s", email)
	val, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil // Cache miss or error
	}

	buf := bytes.NewBuffer(val)
	decoder := gob.NewDecoder(buf)
	var user models.User
	if err := decoder.Decode(&user); err != nil {
		log.Println("error decoding cached user")
		return nil
	}
	return &user
}

func (s *UserService) getCachedUserByPhone(phone string) *models.User {
	if s.redisClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	key := fmt.Sprintf("user:phone:%s", phone)
	val, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil // Cache miss or error
	}

	buf := bytes.NewBuffer(val)
	decoder := gob.NewDecoder(buf)
	var user models.User
	if err := decoder.Decode(&user); err != nil {
		log.Println("error decoding cached user")
		return nil
	}
	return &user
}

func (s *UserService) cacheUser(user *models.User) {
	if s.redisClient == nil || user == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(user); err != nil {
		log.Println("error encoding user caching")
		return
	}

	// Cache for 15 minutes
	ttl := 15 * time.Minute
	s.redisClient.Set(ctx, fmt.Sprintf("user:email:%s", user.Email), buf.Bytes(), ttl)
	s.redisClient.Set(ctx, fmt.Sprintf("user:phone:%s", user.PhoneNumber), buf.Bytes(), ttl)
}

func (s *UserService) incrementLoginAttempts(userID string) int {
	if s.redisClient == nil {
		// Fallback to in-memory with proper locking
		s.mu.Lock()
		s.globalLoginAttempt[userID]++
		attempts := s.globalLoginAttempt[userID]
		s.mu.Unlock()
		return attempts
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	key := fmt.Sprintf("login_attempts:%s", userID)
	count, err := s.redisClient.Incr(ctx, key).Result()
	if err != nil {
		// Fallback to in-memory
		s.mu.Lock()
		s.globalLoginAttempt[userID]++
		attempts := s.globalLoginAttempt[userID]
		s.mu.Unlock()
		return attempts
	}

	// Set TTL for 24 hours on first attempt
	if count == 1 {
		s.redisClient.Expire(ctx, key, 24*time.Hour)
	}

	return int(count)
}

func (s *UserService) resetLoginAttempts(userID string) {
	if s.redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		key := fmt.Sprintf("login_attempts:%s", userID)
		s.redisClient.Del(ctx, key)
	}

	// Also clear in-memory
	s.mu.Lock()
	delete(s.globalLoginAttempt, userID)
	s.mu.Unlock()
}

// BanUser bans a user by setting status to suspended and locked_until timestamp
func (s *UserService) BanUser(userID string, until int64) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Update user status to suspended with locked_until timestamp
	err := s.userRepo.UpdateUserStatus(userID, models.UserStatusSuspended, &until)
	if err != nil {
		log.Printf("Failed to ban user %s: %v", userID, err)
		return fmt.Errorf("failed to ban user: %w", err)
	}

	// Invalidate all user sessions to force re-authentication
	err = s.sessionService.InvalidateUserSessions(context.Background(), userID)
	if err != nil {
		log.Printf("Failed to invalidate sessions for banned user %s: %v", userID, err)
		// Don't fail the ban operation if session invalidation fails
	}

	log.Printf("User %s has been banned until %v", userID, time.Unix(until, 0))
	return nil
}

// UnbanUser unbans a user by setting status back to active and clearing locked_until
func (s *UserService) UnbanUser(userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Update user status to active and clear locked_until timestamp
	err := s.userRepo.UpdateUserStatus(userID, models.UserStatusActive, nil)
	if err != nil {
		log.Printf("Failed to unban user %s: %v", userID, err)
		return fmt.Errorf("failed to unban user: %w", err)
	}

	// Clear failed login attempts
	s.resetLoginAttempts(userID)

	log.Printf("User %s has been unbanned and reactivated", userID)
	return nil
}

func (s *UserService) VerifyLandCertificate(userID string, NationalIDInput string) (bool, error) {
	var result bool = false
	var isNationalIDMatch bool = true
	userCard, err := s.userCardRepo.GetUserCardByUserID(userID)
	if err != nil {
		log.Printf("Failed to get user card: %v", err)
		return false, fmt.Errorf(err.Error())
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return false, err
	}

	if userCard.NationalID != NationalIDInput {
		isNationalIDMatch = false
		return result, fmt.Errorf("bad_request: National ID does not match")
	}

	if user.KYCVerified && isNationalIDMatch {
		result = true
	} else {
		return result, fmt.Errorf("forbidden: User has not completed KYC verification")
	}

	return result, nil
}

func (s *UserService) GetAllUsers(limit, offset int) (*models.GetAllUsersResponse, error) {
	users, err := s.userRepo.GetAllUsers(limit, offset)
	if err != nil {
		log.Printf("Failed to get all users: %v", err)
		return nil, err
	}

	return &models.GetAllUsersResponse{
		Users:  users,
		Total:  len(users),
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *UserService) CheckExistEmailOrPhone(input string) (bool, error) {
	exists, err := s.userRepo.CheckExistEmailOrPhone(input)
	return exists, err
}

func (s *UserService) GetUserCardByUserID(userID string) (*models.UserCard, error) {
	return s.userCardRepo.GetUserCardByUserID(userID)
}

func (s *UserService) ResetEkycData(userID string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		slog.Error("user not found when resetting ekyc data", "user_id", userID)
		return fmt.Errorf("note_found: user not found")
	}

	err = s.userRepo.ResetEkycData(user.ID)
	if err != nil {
		return fmt.Errorf("failed to delete user card data: %w", err)
	}
	return nil
}

func (s *UserService) UpdateUserCardByUserID(userID string, req models.UpdateUserCardRequest) error {
	// check if user exists
	_, error := s.userCardRepo.GetUserCardByUserID(userID)
	if error != nil {
		log.Printf("Failed to get user card by user ID: %v", error)
		return fmt.Errorf("not_found: user card not found")
	}

	return s.userCardRepo.UpdateUserCardByUserID(userID, req)
}

func (s *UserService) GeneratePhoneOTP(ctx context.Context, phoneNumber string) error {
	otp := agrisa_utils.GenerateRandomStringWithLength(6)
	err := s.redisClient.Set(ctx, phoneNumber, otp, 5*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("error generating otp=%w", err)
	}
	go func() {
		event := event.NotificationEventPushModel{
			Notification: event.Notification{
				Title: "Xac Thuc So Dien Thoai",
				Body:  fmt.Sprintf("Ma OTP cua ban la %s. Luu y: khong cung cap ma OTP cho nguoi khac."),
			},
			Destinations: []string{phoneNumber},
		}

		for {
			err := s.eventPublisher.PublishNotification(context.Background(), event)
			if err == nil {
				slog.Info("phone number verification sent", "phone_number", phoneNumber)
				return
			}
			slog.Error("error sending phone number verification ", "error", err)
			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

func (s *UserService) ValidatePhoneOTP(ctx context.Context, phoneNumber, otp string) error {
	generatedOTP := s.redisClient.Get(ctx, phoneNumber).String()
	if otp != generatedOTP {
		return fmt.Errorf("incorrect otp")
	}
	return nil
}
