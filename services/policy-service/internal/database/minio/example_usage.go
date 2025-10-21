package minio

import (
	"context"
	"log"
	"policy-service/internal/config"
	"strings"
	"time"
)

// ExampleUsage demonstrates how to use the MinIO client in the policy service
func ExampleUsage() {
	// Initialize configuration (normally this comes from your application setup)
	cfg := config.New()

	// Initialize MinIO client
	minioClient, err := NewMinioClient(cfg.MinioCfg)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}
	defer minioClient.Close()

	ctx := context.Background()

	// Example 1: Upload a policy document
	policyContent := "This is a sample policy document content"
	err = minioClient.UploadFile(
		ctx,
		Storage.PolicyDocuments,
		"sample-policy-123.pdf",
		strings.NewReader(policyContent),
		int64(len(policyContent)),
		"application/pdf",
	)
	if err != nil {
		log.Printf("Failed to upload policy document: %v", err)
	}

	// Example 2: Check if a file exists
	exists, err := minioClient.FileExists(ctx, Storage.PolicyDocuments, "sample-policy-123.pdf")
	if err != nil {
		log.Printf("Failed to check file existence: %v", err)
	} else {
		log.Printf("File exists: %v", exists)
	}

	// Example 3: Generate a presigned URL (valid for 1 hour)
	presignedURL, err := minioClient.GetPresignedURL(
		ctx,
		Storage.PolicyDocuments,
		"sample-policy-123.pdf",
		1*time.Hour,
	)
	if err != nil {
		log.Printf("Failed to generate presigned URL: %v", err)
	} else {
		log.Printf("Presigned URL: %s", presignedURL)
	}

	// Example 4: List all files in policy documents bucket
	files, err := minioClient.ListFiles(ctx, Storage.PolicyDocuments, "")
	if err != nil {
		log.Printf("Failed to list files: %v", err)
	} else {
		log.Printf("Found %d files in policy documents bucket", len(files))
		for _, file := range files {
			log.Printf("File: %s, Size: %d bytes, Modified: %v", 
				file.Key, file.Size, file.LastModified)
		}
	}

	// Example 5: Upload a validation report
	reportContent := `{
		"policy_id": "policy-123",
		"validation_status": "passed",
		"checks_passed": 15,
		"checks_failed": 0,
		"timestamp": "2024-01-01T12:00:00Z"
	}`
	err = minioClient.UploadFile(
		ctx,
		Storage.ValidationReports,
		"validation-report-policy-123.json",
		strings.NewReader(reportContent),
		int64(len(reportContent)),
		"application/json",
	)
	if err != nil {
		log.Printf("Failed to upload validation report: %v", err)
	}

	// Example 6: Download and read a file
	object, err := minioClient.GetFile(ctx, Storage.ValidationReports, "validation-report-policy-123.json")
	if err != nil {
		log.Printf("Failed to get validation report: %v", err)
	} else {
		// Note: In real usage, you would read the content from the object
		// buf := make([]byte, 1024)
		// n, err := object.Read(buf)
		// if err != nil && err != io.EOF {
		//     log.Printf("Failed to read object: %v", err)
		// } else {
		//     log.Printf("Read %d bytes: %s", n, string(buf[:n]))
		// }
		object.Close()
		log.Println("Successfully retrieved validation report")
	}

	// Example 7: Delete a file
	err = minioClient.DeleteFile(ctx, Storage.PolicyDocuments, "sample-policy-123.pdf")
	if err != nil {
		log.Printf("Failed to delete file: %v", err)
	} else {
		log.Println("Successfully deleted sample policy document")
	}
}

// Common usage patterns for different policy service operations

// UploadPolicyDocument uploads a policy document with proper naming convention
func UploadPolicyDocument(mc *MinioClient, ctx context.Context, policyID string, content []byte, contentType string) error {
	objectName := generatePolicyDocumentName(policyID)
	return mc.UploadFile(ctx, Storage.PolicyDocuments, objectName, strings.NewReader(string(content)), int64(len(content)), contentType)
}

// UploadValidationReport uploads a validation report for a policy
func UploadValidationReport(mc *MinioClient, ctx context.Context, policyID string, report []byte) error {
	objectName := generateValidationReportName(policyID)
	return mc.UploadFile(ctx, Storage.ValidationReports, objectName, strings.NewReader(string(report)), int64(len(report)), "application/json")
}

// UploadDataSourceFile uploads a data source related file
func UploadDataSourceFile(mc *MinioClient, ctx context.Context, dataSourceID string, filename string, content []byte, contentType string) error {
	objectName := generateDataSourceFileName(dataSourceID, filename)
	return mc.UploadFile(ctx, Storage.DataSources, objectName, strings.NewReader(string(content)), int64(len(content)), contentType)
}

// GetPolicyDocumentURL generates a presigned URL for accessing a policy document
func GetPolicyDocumentURL(mc *MinioClient, ctx context.Context, policyID string, expiry time.Duration) (string, error) {
	objectName := generatePolicyDocumentName(policyID)
	return mc.GetPresignedURL(ctx, Storage.PolicyDocuments, objectName, expiry)
}

// Helper functions for consistent naming conventions

func generatePolicyDocumentName(policyID string) string {
	return "policies/" + policyID + "/document.pdf"
}

func generateValidationReportName(policyID string) string {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	return "validation-reports/" + policyID + "/" + timestamp + ".json"
}

func generateDataSourceFileName(dataSourceID string, filename string) string {
	return "data-sources/" + dataSourceID + "/" + filename
}

// Cleanup functions

// DeletePolicyFiles removes all files related to a specific policy
func DeletePolicyFiles(mc *MinioClient, ctx context.Context, policyID string) error {
	// Delete from policy documents
	policyDocName := generatePolicyDocumentName(policyID)
	if err := mc.DeleteFile(ctx, Storage.PolicyDocuments, policyDocName); err != nil {
		log.Printf("Warning: Failed to delete policy document %s: %v", policyDocName, err)
	}

	// List and delete validation reports
	prefix := "validation-reports/" + policyID + "/"
	reports, err := mc.ListFiles(ctx, Storage.ValidationReports, prefix)
	if err != nil {
		return err
	}

	for _, report := range reports {
		if err := mc.DeleteFile(ctx, Storage.ValidationReports, report.Key); err != nil {
			log.Printf("Warning: Failed to delete validation report %s: %v", report.Key, err)
		}
	}

	return nil
}