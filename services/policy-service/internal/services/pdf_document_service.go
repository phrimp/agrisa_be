package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"policy-service/internal/database/minio"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpuModel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type PDFService struct {
	minioClient *minio.MinioClient
	bucketName  string
}

func NewPDFService(minioClient *minio.MinioClient, bucketName string) *PDFService {
	return &PDFService{
		minioClient: minioClient,
		bucketName:  bucketName,
	}
}

// FillFromStorage retrieves PDF template from MinIO and fills it with values
func (s *PDFService) FillFromStorage(ctx context.Context, objectName string, values map[string]string) ([]byte, error) {
	slog.Info("Filling PDF from MinIO storage",
		"bucket", s.bucketName,
		"object", objectName,
		"values_count", len(values))

	// 1. Download from MinIO as []byte
	obj, err := s.minioClient.GetFile(ctx, s.bucketName, objectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get PDF from MinIO: %w", err)
	}
	defer obj.Close()

	// 2. Read into []byte
	templateData, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	// 3. Fill PDF
	filledPDF, err := s.FillPDFTemplate(templateData, values)
	if err != nil {
		return nil, fmt.Errorf("failed to fill PDF template: %w", err)
	}

	slog.Info("Successfully filled PDF from storage",
		"bucket", s.bucketName,
		"object", objectName,
		"output_size", len(filledPDF))

	return filledPDF, nil
}

// FillPDFTemplate fills PDF form fields with values from the map
func (s *PDFService) FillPDFTemplate(pdfData []byte, values map[string]string) ([]byte, error) {
	// Validate inputs
	if err := validateInputs(pdfData, values); err != nil {
		return nil, err
	}

	slog.Info("Filling PDF template",
		"pdf_size", len(pdfData),
		"values_count", len(values))

	// Try PDF form field approach
	filledPDF, err := s.fillFormFields(pdfData, values)
	if err != nil {
		return nil, fmt.Errorf("failed to fill PDF form fields: %w", err)
	}

	slog.Info("Successfully filled PDF using form fields")
	return filledPDF, nil
}

// fillFormFields fills PDF form fields (AcroForm)
func (s *PDFService) fillFormFields(pdfData []byte, values map[string]string) ([]byte, error) {
	reader := bytes.NewReader(pdfData)
	conf := pdfcpuModel.NewDefaultConfiguration()

	// Convert values map to JSON
	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal form data: %w", err)
	}
	jsonReader := bytes.NewReader(jsonData)

	// Fill form fields using FillForm API
	var filledBuf bytes.Buffer
	if err := api.FillForm(reader, jsonReader, &filledBuf, conf); err != nil {
		return nil, fmt.Errorf("failed to fill form: %w", err)
	}

	slog.Info("Successfully filled PDF form fields", "field_count", len(values))

	// Lock form fields to make them read-only (flatten)
	flatReader := bytes.NewReader(filledBuf.Bytes())
	var flatBuf bytes.Buffer

	// Get all field names to lock
	fieldNames := make([]string, 0, len(values))
	for key := range values {
		fieldNames = append(fieldNames, key)
	}

	if err := api.LockFormFields(flatReader, &flatBuf, fieldNames, conf); err != nil {
		slog.Warn("Failed to lock PDF form fields", "error", err)
		return filledBuf.Bytes(), nil // Return unlocked if locking fails
	}

	slog.Info("Successfully locked PDF form fields")
	return flatBuf.Bytes(), nil
}

// validateInputs validates PDF data and values map
func validateInputs(pdfData []byte, values map[string]string) error {
	// Check PDF data
	if len(pdfData) == 0 {
		return errors.New("empty PDF data")
	}

	// Check max size (50MB)
	const maxSize = 50 * 1024 * 1024
	if len(pdfData) > maxSize {
		return fmt.Errorf("PDF too large: %d bytes (max: %d)", len(pdfData), maxSize)
	}

	// Validate PDF format
	if !isPDF(pdfData) {
		return errors.New("invalid PDF format")
	}

	// Check values
	if len(values) == 0 {
		return errors.New("no values provided")
	}

	// Validate keys are not empty
	for key, value := range values {
		if strings.TrimSpace(key) == "" {
			return errors.New("empty key in values map")
		}
		slog.Info("Validating value", "key", key, "value_length", len(value))
	}

	return nil
}

// isPDF checks if data starts with PDF magic bytes
func isPDF(data []byte) bool {
	return len(data) > 4 && string(data[:4]) == "%PDF"
}

// normalizeKey normalizes a key for form field matching
func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// ExtractPlaceholders extracts all form field names from PDF
func (s *PDFService) ExtractPlaceholders(pdfData []byte) ([]string, error) {
	if !isPDF(pdfData) {
		return nil, errors.New("invalid PDF format")
	}

	reader := bytes.NewReader(pdfData)
	conf := pdfcpuModel.NewDefaultConfiguration()
	ctx, err := api.ReadContext(reader, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	// Extract field names
	if ctx.Form != nil {
		fieldNames := []string{}
		for fieldName := range ctx.Form {
			fieldNames = append(fieldNames, fieldName)
		}
		return fieldNames, nil
	}

	return []string{}, nil
}

// ExtractPlaceholdersFromPattern finds text placeholders with pattern: ____(key)____
func (s *PDFService) ExtractPlaceholdersFromPattern(text string) []string {
	pattern := regexp.MustCompile(`____+\((.*?)\)_+`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	placeholders := []string{}
	for _, match := range matches {
		if len(match) > 1 {
			placeholders = append(placeholders, match[1])
		}
	}

	return placeholders
}

func (s *PDFService) UploadFilledPDF(ctx context.Context, originalName string, filledPDF []byte) (string, error) {
	// Ensure .pdf extension
	newObjectName := s.generateFilledObjectName(originalName)

	slog.Info("Uploading filled PDF to MinIO",
		"bucket", s.bucketName,
		"original_name", originalName,
		"new_object_name", newObjectName,
		"size_bytes", len(filledPDF))

	// Upload to MinIO
	err := s.minioClient.UploadBytes(ctx, s.bucketName, newObjectName, filledPDF, "application/pdf")
	if err != nil {
		return "", fmt.Errorf("failed to upload filled PDF: %w", err)
	}

	slog.Info("Successfully uploaded filled PDF",
		"bucket", s.bucketName,
		"object_name", newObjectName)

	return newObjectName, nil
}

func (s *PDFService) FillFromStorageAndUpload(ctx context.Context, templateObjectName string, values map[string]string) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PDFService: recovered from panic", "panic", r)
		}
	}()
	slog.Info("Fill and upload operation started",
		"template", templateObjectName,
		"values_count", len(values))

	// 1. Fill PDF from storage
	filledPDF, err := s.FillFromStorage(ctx, templateObjectName, values)
	if err != nil {
		return "", fmt.Errorf("failed to fill PDF: %w", err)
	}

	// 2. Upload filled PDF
	uploadedObjectName, err := s.UploadFilledPDF(ctx, templateObjectName, filledPDF)
	if err != nil {
		return "", fmt.Errorf("failed to upload filled PDF: %w", err)
	}

	slog.Info("Fill and upload completed successfully",
		"template", templateObjectName,
		"filled_object", uploadedObjectName)

	return uploadedObjectName, nil
}

func (s *PDFService) generateFilledObjectName(originalName string) string {
	if strings.HasSuffix(strings.ToLower(originalName), ".pdf") {
		// Remove .pdf extension, add _filled, add .pdf back
		nameWithoutExt := originalName[:len(originalName)-4]
		return nameWithoutExt + "_filled.pdf"
	}

	return originalName + "_filled.pdf"
}
