package utils

import (
	"auth-service/internal/database/minio"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"auth-service/internal/config"
)

type FileInfo struct {
	FieldName    string
	OriginalName string
	SafeName     string
	Size         int64
	ContentType  string
	MinioURL     string
	BucketName   string
	ServiceName  string
}

type Utils struct {
	minioClient *minio.MinioClient
	cfg         *config.AuthServiceConfig
}

type SuccessResponse struct {
	Success bool  `json:"success"`
	Data    any   `json:"data"`
	Meta    *Meta `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success bool     `json:"success"`
	Error   APIError `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	Timestamp time.Time `json:"timestamp"`
}

func NewUtils(minioClient *minio.MinioClient, cfg *config.AuthServiceConfig) *Utils {
	return &Utils{
		minioClient: minioClient,
		cfg:         cfg,
	}
}

func (u *Utils) ProcessFiles(minioClient *minio.MinioClient,
	files map[string][]*multipart.FileHeader,
	serviceName string, allowedExts []string, maxMB int64) ([]FileInfo, error) {
	var uploadedFiles []FileInfo
	ctx := context.Background()

	for fieldName, fileHeaders := range files {
		for _, fileHeader := range fileHeaders {
			if err := ValidateFile(fileHeader, allowedExts, maxMB); err != nil {
				log.Printf("File validation failed: %v", err)
				return nil, fmt.Errorf("file validation failed: %v", err)
			}

			originalName := fileHeader.Filename
			safeName := GenerateSafeFilename(originalName)
			contentType := fileHeader.Header.Get("Content-Type")

			// Mở file
			file, err := fileHeader.Open()
			if err != nil {
				log.Printf("Failed to open file: %v", err)
				return nil, fmt.Errorf("failed to open file: %v", err)
			}
			defer file.Close()

			// Upload trực tiếp lên MinIO
			if err := minioClient.UploadFile(ctx, safeName, contentType, file, fileHeader.Size, serviceName); err != nil {
				log.Printf("Failed to upload file to MinIO: %v", err)
				return nil, fmt.Errorf("failed to upload file to MinIO: %v", err)
			}

			// Lấy URL
			bucketName := minioClient.GetBucketByServiceName(serviceName, minio.BucketNames)
			minioURL := BuildResourceURL(u.cfg.MinioCfg.MinioResourceUrl, bucketName, safeName)

			fileInfo := FileInfo{
				FieldName:    fieldName,
				OriginalName: originalName,
				SafeName:     safeName,
				Size:         fileHeader.Size,
				ContentType:  contentType,
				MinioURL:     minioURL,
				BucketName:   bucketName,
				ServiceName:  serviceName,
			}

			uploadedFiles = append(uploadedFiles, fileInfo)
		}
	}
	return uploadedFiles, nil
}

func GenerateSafeFilename(original string) string {
	// Get file extension
	ext := filepath.Ext(original)
	nameWithoutExt := strings.TrimSuffix(filepath.Base(original), ext)

	// Remove special characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	safeName := reg.ReplaceAllString(nameWithoutExt, "_")

	// Add timestamp to avoid conflicts
	timestamp := time.Now().Format("20060102_150405")

	return fmt.Sprintf("%s_%s%s", safeName, timestamp, ext)
}

func ValidateFile(fileHeader *multipart.FileHeader, allowedExts []string, maxMB int64) error {
	// Check file size (maxMB max)
	if fileHeader.Size > maxMB*1024*1024 {
		return fmt.Errorf("file too large: %s", fileHeader.Filename)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	//allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt"}

	isAllowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("file type not allowed: %s", ext)
	}

	return nil
}

func BuildResourceURL(baseURL, bucketName, resourceName string) string {
	return baseURL + path.Join(bucketName, resourceName)
}

func CreateErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: APIError{
			Code:    code,
			Message: message,
		},
	}
}

func CreateSuccessResponse(data any) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now(),
		},
	}
}
