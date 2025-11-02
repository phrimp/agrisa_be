package minio

import (
	utils "agrisa_utils"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"policy-service/internal/config"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient wraps the MinIO client with policy service specific functionality
type MinioClient struct {
	client *minio.Client
	config config.MinioConfig
}

type FileUpload struct {
	FieldName string `json:"fieldName"`
	FileName  string `json:"fileName"`
	Data      string `json:"Data"`
}

type FileUploadRequest []FileUpload

type FileUploadValidationErr struct {
	FieldName string `json:"field_name"`
	FileName  string `json:"file_name"`
	Message   string `json:"message"`
}

type FileUploadedInfo struct {
	FieldName   string
	ResourceURL string
}

// Storage defines bucket names for different data types in policy service
var Storage = struct {
	PolicyService     string
	PolicyDocuments   string
	PolicyAttachments string
	DataSources       string
	ValidationReports string
}{
	PolicyService:     "policy-service",
	PolicyDocuments:   "policy-documents",
	PolicyAttachments: "policy-attachments",
	DataSources:       "data-sources",
	ValidationReports: "validation-reports",
}

// BucketNames contains all bucket names for policy service
var BucketNames = []string{
	Storage.PolicyService,
	Storage.PolicyDocuments,
	Storage.PolicyAttachments,
	Storage.DataSources,
	Storage.ValidationReports,
}

// NewMinioClient initializes a new MinIO client with the provided configuration
func NewMinioClient(cfg config.MinioConfig) (*MinioClient, error) {
	// Parse MinIO URL to extract endpoint
	endpoint := strings.TrimPrefix(cfg.MinioURL, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// Parse secure flag
	isSecure, err := strconv.ParseBool(cfg.MinioSecure)
	if err != nil {
		log.Printf("Invalid value for MinIO secure flag: %v. Defaulting to false.", err)
		isSecure = false
	}

	// Initialize MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: isSecure,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to list buckets to verify connection
	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MinIO server: %w", err)
	}

	log.Printf("Successfully connected to MinIO at %s", cfg.MinioURL)

	// Create MinioClient instance
	mc := &MinioClient{
		client: minioClient,
		config: cfg,
	}

	// Ensure all required buckets exist
	if err := mc.ensureRequiredBuckets(); err != nil {
		return nil, fmt.Errorf("failed to ensure required buckets: %w", err)
	}

	log.Printf("MinIO client initialized successfully with %d buckets", len(BucketNames))
	return mc, nil
}

// ensureRequiredBuckets creates all required buckets if they don't exist
func (mc *MinioClient) ensureRequiredBuckets() error {
	ctx := context.Background()

	for _, bucketName := range BucketNames {
		if err := mc.ensureBucket(ctx, bucketName); err != nil {
			return fmt.Errorf("failed to ensure bucket %s: %w", bucketName, err)
		}
	}

	// Set public read policy for policy documents bucket (for public policy access)
	if err := mc.SetPublicReadPolicy(ctx, Storage.PolicyDocuments); err != nil {
		log.Printf("Failed to set public policy for %s bucket: %v", Storage.PolicyDocuments, err)
		// Don't return error as this is not critical for basic functionality
	}

	return nil
}

// ensureBucket creates a bucket if it doesn't exist
func (mc *MinioClient) ensureBucket(ctx context.Context, bucketName string) error {
	// Check if bucket already exists
	exists, err := mc.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("error checking bucket existence: %w", err)
	}

	// Create bucket if it doesn't exist
	if !exists {
		err := mc.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: mc.config.MinioLocation,
		})
		if err != nil {
			return fmt.Errorf("error creating bucket %s: %w", bucketName, err)
		}
		log.Printf("Created bucket: %s", bucketName)
	} else {
		log.Printf("Bucket already exists: %s", bucketName)
	}

	return nil
}

// SetPublicReadPolicy sets a public read-only policy for a bucket
func (mc *MinioClient) SetPublicReadPolicy(ctx context.Context, bucketName string) error {
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": "*"},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucketName)

	err := mc.client.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		return fmt.Errorf("error setting public read policy for bucket %s: %w", bucketName, err)
	}

	log.Printf("Set public read policy for bucket: %s", bucketName)
	return nil
}

// UploadFile uploads a file to the specified bucket
func (mc *MinioClient) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	_, err := mc.client.PutObject(ctx, bucketName, objectName, reader, objectSize,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("failed to upload file %s to bucket %s: %w", objectName, bucketName, err)
	}

	log.Printf("Successfully uploaded file: %s to bucket: %s", objectName, bucketName)
	return nil
}

// UploadBytes uploads byte data to the specified bucket
func (mc *MinioClient) UploadBytes(ctx context.Context, bucketName, objectName string, data []byte, contentType string) error {
	reader := bytes.NewReader(data)
	_, err := mc.client.PutObject(ctx, bucketName, objectName, reader, int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("failed to upload bytes to %s in bucket %s: %w", objectName, bucketName, err)
	}

	log.Printf("Successfully uploaded %d bytes to: %s in bucket: %s", len(data), objectName, bucketName)
	return nil
}

// UploadFileFromPath uploads a file from local file system path
func (mc *MinioClient) UploadFileFromPath(ctx context.Context, bucketName, objectName, filePath, contentType string) error {
	_, err := mc.client.FPutObject(ctx, bucketName, objectName, filePath,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("failed to upload file from path %s to bucket %s: %w", filePath, bucketName, err)
	}

	log.Printf("Successfully uploaded file from path: %s to bucket: %s as %s", filePath, bucketName, objectName)
	return nil
}

// GetFile retrieves a file from the specified bucket
func (mc *MinioClient) GetFile(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	object, err := mc.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s from bucket %s: %w", objectName, bucketName, err)
	}

	return object, nil
}

// DeleteFile deletes a file from the specified bucket
func (mc *MinioClient) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	err := mc.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file %s from bucket %s: %w", objectName, bucketName, err)
	}

	log.Printf("Successfully deleted file: %s from bucket: %s", objectName, bucketName)
	return nil
}

// GetPresignedURL generates a presigned URL for temporary access to an object
func (mc *MinioClient) GetPresignedURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	presignedURL, err := mc.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s in bucket %s: %w", objectName, bucketName, err)
	}

	return presignedURL.String(), nil
}

// ListFiles lists all files in a bucket with optional prefix
func (mc *MinioClient) ListFiles(ctx context.Context, bucketName, prefix string) ([]minio.ObjectInfo, error) {
	var objects []minio.ObjectInfo

	objectCh := mc.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects in bucket %s: %w", bucketName, object.Err)
		}
		objects = append(objects, object)
	}

	return objects, nil
}

// FileExists checks if a file exists in the specified bucket
func (mc *MinioClient) FileExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := mc.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		// Check if error is "not found"
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("error checking file existence for %s in bucket %s: %w", objectName, bucketName, err)
	}

	return true, nil
}

// GetClient returns the underlying MinIO client for advanced operations
func (mc *MinioClient) GetClient() *minio.Client {
	return mc.client
}

// GetConfig returns the MinIO configuration
func (mc *MinioClient) GetConfig() config.MinioConfig {
	return mc.config
}

// Close performs any necessary cleanup (MinIO client doesn't require explicit closing)
func (mc *MinioClient) Close() error {
	log.Println("MinIO client connection closed")
	return nil
}

func GetContentType(objectName string) string {
	ext := strings.ToLower(filepath.Ext(objectName))

	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".doc":
		return "application/msword"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream" // fallback an toàn
	}
}

func FileSize(unit string, f *FileUpload) (float64, error) {
	data, err := base64.StdEncoding.DecodeString(f.Data)
	if err != nil {
		return 0, fmt.Errorf("decode base64 error: %w", err)
	}
	sizeInBytes := float64(len(data))
	unit = strings.ToLower(unit)
	switch unit {
	case "b":
		return sizeInBytes, nil
	case "kb":
		return sizeInBytes / 1024, nil
	case "mb":
		return sizeInBytes / (1024 * 1024), nil
	case "gb":
		return sizeInBytes / (1024 * 1024 * 1024), nil
	default:
		return 0, fmt.Errorf("invalid unit '%s', must be one of: b, kb, mb, gb", unit)
	}
}

func ValidateFiles(files FileUploadRequest, allowedExts []string, maxMB int64) *FileUploadValidationErr {
	for _, f := range files {
		if f.FileName == "" {
			return &FileUploadValidationErr{
				FieldName: "data",
				FileName:  "",
				Message:   "A file has an invalid name — please check it.",
			}
		}

		ext := strings.ToLower(filepath.Ext(f.FileName))
		found := false

		if len(allowedExts) == 0 {
			found = true
		}

		for _, allowed := range allowedExts {
			if strings.ToLower(allowed) == ext {
				found = true
				break
			}
		}
		if !found {
			return &FileUploadValidationErr{
				FieldName: "data",
				FileName:  f.FileName,
				Message:   "The file extension is invalid.",
			}
		}

		if maxMB != -1 {
			uploadFileSize, err := FileSize("mb", &f)
			if err != nil {
				return &FileUploadValidationErr{
					FieldName: "data",
					FileName:  f.FileName,
					Message:   "Invalid base64 data.",
				}
			}
			if uploadFileSize > float64(maxMB) {
				return &FileUploadValidationErr{
					FieldName: "data",
					FileName:  f.FileName,
					Message:   "There is a file that exceeds the maximum allowed size.",
				}
			}
		}

	}
	return nil
}

func CreateErrorFromValidation(validationErr FileUploadValidationErr) error {

	jsonBytes, err := utils.SerializeModel(validationErr)
	if err != nil {
		return fmt.Errorf("error serializing validation error: %w", err)
	}
	jsonString := string(jsonBytes)

	return errors.New(jsonString)
}

func Base64ToBytes(base64String string) ([]byte, error) {
	bytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func GetSafeFileName(fileName string) string {
	safeFileName := url.PathEscape(fileName)
	return safeFileName
}

func (mc *MinioClient) UploadFilesToMinio(
	filesUpload []FileUpload,
	ctx context.Context,
	bucketName string,
) ([]FileUploadedInfo, error) {
	var fileUploadedInfos []FileUploadedInfo

	resourceBaseURL := mc.GetConfig().MinioResourceURL

	for _, fu := range filesUpload {
		bytes, err := Base64ToBytes(fu.Data)
		if err != nil {
			fmt.Println("Lỗi:", err)
			return nil, err
		}

		contentType := GetContentType(fu.FileName)
		safeFileName := GetSafeFileName(fu.FileName)

		err = mc.UploadBytes(ctx, bucketName, safeFileName, bytes, contentType)
		if err != nil {
			log.Fatalf("Internal error - upload %s: %v", safeFileName, err)
			return nil, err
		}

		minioResourceURL := resourceBaseURL + bucketName + "/" + safeFileName

		fileUploadedInfo := FileUploadedInfo{
			FieldName:   "document_template_upload",
			ResourceURL: minioResourceURL,
		}

		fileUploadedInfos = append(fileUploadedInfos, fileUploadedInfo)
	}

	return fileUploadedInfos, nil
}

func (mc *MinioClient) FileProcessing(files FileUploadRequest, ctx context.Context, allowedExts []string, maxFileSizeMB int64) ([]FileUploadedInfo, error) {
	// check if files are empty or not
	if len(files) == 0 {
		return nil, CreateErrorFromValidation(FileUploadValidationErr{
			FieldName: "data",
			FileName:  "",
			Message:   "No files were uploaded.",
		})
	}

	// validate files
	validationErr := ValidateFiles(files, allowedExts, maxFileSizeMB)
	if validationErr != nil {
		return nil, CreateErrorFromValidation(*validationErr)
	}

	// Upload files to MinIO
	fileUploadedInfos, err := mc.UploadFilesToMinio(files, ctx, Storage.PolicyDocuments)
	if err != nil {
		return nil, fmt.Errorf("failed to upload files to MinIO: %w", err)
	}

	return fileUploadedInfos, nil
}
