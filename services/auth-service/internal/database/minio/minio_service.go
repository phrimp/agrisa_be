package minio

import (
	"auth-service/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"log"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client *minio.Client
}

var Storage = struct {
	AuthService   string
	UserCardImage string
}{
	AuthService:   "auth-service",
	UserCardImage: "user-card-image",
}

// create slice for all bucket names
var BucketNames = GetAllStorageValues()

func NewMinioClient(cfg config.MinioConfig) (*MinioClient, error) {
	// Initialize MinIO client
	isSecure, err := strconv.ParseBool(cfg.MinioSecure)
	if err != nil {
		log.Printf("Invalid value for MinIO secure flag: %v. Defaulting to false.", err)
		isSecure = false
	}
	minioClient, err := minio.New(cfg.MinioUrl, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: isSecure,
	})

	if err != nil {
		fmt.Println("error connecting to MinIO client:", err)
		return nil, err
	}

	log.Printf("Bucketnames before creation: %v", BucketNames)

	// Ensure all buckets exist
	for _, v := range BucketNames {
		if err := ensureBucket(minioClient, v, cfg.MinioLocation); err != nil {
			return nil, err
		}
	}

	// Set public read-only policy for auth-service bucket
	err = SetPublicBucketPolicy(minioClient, Storage.AuthService)
	if err != nil {
		log.Printf("Failed to set public policy for auth-service bucket: %v", err.Error())
		return nil, err
	}
	//for compare markdown file
	// if err := enableVersioning(minioClient, Storage.CommonBucketName); err != nil {
	// 	log.Printf("Failed to enable versioning for common bucket: %v", err.Error())
	// 	return nil, err
	// }
	// log.Printf("Enabled versioning for common bucket")
	// log.Printf("MinIO client connected successfully")
	return &MinioClient{
		client: minioClient,
	}, nil
}

func SetPublicBucketPolicy(minioClient *minio.Client, bucketName string) error {
	// JSON policy cho public read-only (allow GetObject cho everyone)
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Action":    []string{"s3:GetObject"},
				"Effect":    "Allow",
				"Principal": map[string]any{"AWS": []string{"*"}},
				"Resource":  []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucketName)},
			},
		},
	}

	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("error marshalling policy: %w", err)
	}

	// Set policy cho bucket
	err = minioClient.SetBucketPolicy(context.Background(), bucketName, string(policyBytes))
	if err != nil {
		return fmt.Errorf("error setting bucket policy: %w", err)
	}

	return nil
}

func enableVersioning(minioClient *minio.Client, bucketName string) error {
	ctx := context.Background()
	config := minio.BucketVersioningConfiguration{
		Status: "Enabled",
	}
	return minioClient.SetBucketVersioning(ctx, bucketName, config)
}

// Ensure bucket exists or create it
func ensureBucket(client *minio.Client, bucketName, location string) error {
	ctx := context.Background()
	log.Printf("Bucketname: %s exists in bucket location: %s", bucketName, location)

	// Check if bucket already exists
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		log.Printf("error checking bucket existence: %v", err)
		return err
	}
	// If bucket doesn't exist, create it
	if !exists {
		err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
		if err != nil {
			fmt.Println("error creating bucket:", err)
			return err
		}
		fmt.Println("Bucket created successfully ", bucketName)
	} else {
		fmt.Println("Bucket already exists ", bucketName)
	}

	return nil
}

// Ensure bucket exists if not create bucket with lock enabled
func ensureBucketWithLocking(client *minio.Client, bucketName, location string) error {
	ctx := context.Background()

	// Check if bucket already exists
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		fmt.Println("error checking bucket existence:", err)
		return err
	}
	// If bucket doesn't exist, create it
	if !exists {
		err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location,
			ObjectLocking: true})
		if err != nil {
			fmt.Println("error creating bucket:", err)
			return err
		}
		fmt.Println("Bucket created successfully ", bucketName)
	} else {
		fmt.Println("Bucket already exists ", bucketName)
	}

	return nil
}

func (mc *MinioClient) FUploadFile(ctx context.Context, fileName, filePath, contentType string, serviceName string) error {
	bucket := mc.GetBucketByServiceName(serviceName, BucketNames)
	_, err := mc.client.FPutObject(ctx, bucket, fileName, filePath,
		minio.PutObjectOptions{ContentType: contentType},
	)
	return err
}
func (mc *MinioClient) UploadFile(ctx context.Context, fileName, contentType string, reader io.Reader, size int64, serviceName string) error {
	bucket := mc.GetBucketByServiceName(serviceName, BucketNames)
	_, err := mc.client.PutObject(ctx, bucket, fileName, reader, size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	return err
}

//	func (mc *MinioClient) UploadFilePublic(ctx context.Context, fileName string, r io.Reader, size int64, contentType string) error {
//		_, err := mc.client.PutObject(ctx, Storage.PublicBucket, fileName, r, size, minio.PutObjectOptions{ContentType: contentType})
//		return err
//	}
func (mc *MinioClient) GetFile(ctx context.Context, ext, fileName string, serviceName string) (io.Reader, error) {
	bucket := mc.GetBucketByServiceName(serviceName, BucketNames)
	object, err := mc.client.GetObject(ctx, bucket, fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return object, nil
}
func (mc *MinioClient) GetSignedURL(ctx context.Context, bucketName, fileName string, expiry time.Duration) (string, error) {
	presignedURL, err := mc.client.PresignedGetObject(ctx, bucketName, fileName, expiry, nil)
	if err != nil {
		fmt.Println("error generating presigned URL for object:", err)
		return "", err
	}
	return presignedURL.String(), nil
}
func (mc *MinioClient) DeleteFolder(ctx context.Context, bucketName, folderPath string) error {
	// Ensure the folder path ends with a slash
	if !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}

	objectsCh := mc.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    folderPath,
		Recursive: true,
	})

	for object := range objectsCh {
		if object.Err != nil {
			fmt.Println(" failed listing objects:", object.Err)
			return object.Err
		}
		err := mc.client.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			fmt.Println(" failed deleting object:", err)
			return err
		}
	}
	return nil
}
func (mc *MinioClient) DeleteFile(ctx context.Context, fileName string, serviceName string) error {
	if fileName == "" {
		return fmt.Errorf("fileName cannot be empty")
	}
	bucket := mc.GetBucketByServiceName(serviceName, BucketNames)
	err := mc.client.RemoveObject(ctx, bucket, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		log.Printf("failed to delete file from minio: %v", err)
		return fmt.Errorf("failed to delete file %s: %w", fileName, err)
	}
	log.Printf("file deleted successfully from minio: %s", fileName)
	return nil
}

// List all versions of the object
// func (m *MinioClient) ListVersions(fileName string) ([]storage.FileVersion, error) {
// 	bucketName := m.GetBucketByExt(fileName)
// 	ctx := context.Background()
// 	var versions []minio.ObjectInfo
// 	var fileVersions = []storage.FileVersion{}
// 	opts := minio.ListObjectsOptions{
// 		Prefix:       fileName,
// 		Recursive:    false,
// 		WithVersions: true,
// 	}

// 	for object := range m.client.ListObjects(ctx, bucketName, opts) {
// 		if object.Err != nil {
// 			return nil, object.Err
// 		}
// 		versions = append(versions, object)
// 	}

// 	for _, version := range versions {
// 		fileVersions = append(fileVersions, storage.FileVersion{
// 			VersionID: version.VersionID,
// 			IsLatest:  version.IsLatest,
// 		})
// 	}
// 	return fileVersions, nil
// }

// // Get specific version of object
// func (m *MinioClient) GetFileVersion(fileName, versionID string) (io.Reader, error) {
// 	bucket := m.GetBucketByExt(fileName)
// 	ctx := context.Background()

// 	opts := minio.GetObjectOptions{}
// 	if versionID != "" {
// 		opts.VersionID = versionID
// 	}
// 	obj, err := m.client.GetObject(ctx, bucket, fileName, opts)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get object version: %v", err)
// 	}
// 	return obj, nil
// }

// func (mc *MinioClient) GetBucketByExt(ext string) string {
// 	if util.IsImageFile(ext) {
// 		return storage.ImageBucketName
// 	}
// 	if util.IsVideoFile(ext) {
// 		return storage.VideoBucketName
// 	}
// 	return storage.CommonBucketName
// }

func (mc *MinioClient) GetBucketByServiceName(serviceName string, bucketNames []string) string {
	for _, bucket := range bucketNames {
		if strings.EqualFold(bucket, serviceName) {
			return bucket
		}
	}
	return ""
}

func GetAllStorageValues() []string {
	v := reflect.ValueOf(Storage)
	values := make([]string, 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		values = append(values, v.Field(i).String())
	}
	return values
}
