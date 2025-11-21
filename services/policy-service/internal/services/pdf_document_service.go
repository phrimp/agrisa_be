package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"policy-service/internal/database/minio"
	"regexp"
	"strings"
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

// fillFormFields fills PDF form fields using pdftk
func (s *PDFService) fillFormFields(pdfData []byte, values map[string]string) ([]byte, error) {
	// Create temporary files for pdftk
	inputFile, err := os.CreateTemp("", "pdf_input_*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(inputFile.Name())
	defer inputFile.Close()

	fdfFile, err := os.CreateTemp("", "pdf_fdf_*.fdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp FDF file: %w", err)
	}
	defer os.Remove(fdfFile.Name())
	defer fdfFile.Close()

	outputFile, err := os.CreateTemp("", "pdf_output_*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	// Write PDF data to temp file
	if _, err := inputFile.Write(pdfData); err != nil {
		return nil, fmt.Errorf("failed to write PDF to temp file: %w", err)
	}
	inputFile.Close()

	// First, get available field names using pdftk dump_data_fields
	fieldNames, err := s.getFormFieldNames(inputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get form field names: %w", err)
	}

	if len(fieldNames) == 0 {
		return nil, errors.New("PDF has no form fields (AcroForm)")
	}

	slog.Info("PDF form fields available", "fields", fieldNames)

	// Log provided keys
	providedKeys := make([]string, 0, len(values))
	for key := range values {
		providedKeys = append(providedKeys, key)
	}
	slog.Info("Provided form data keys", "keys", providedKeys)

	// Generate FDF content
	fdfContent := generateFDF(values)
	if _, err := fdfFile.WriteString(fdfContent); err != nil {
		return nil, fmt.Errorf("failed to write FDF file: %w", err)
	}
	fdfFile.Close()

	// Run pdftk to fill form and flatten
	cmd := exec.Command("pdftk", inputFile.Name(), "fill_form", fdfFile.Name(), "output", outputFile.Name(), "flatten")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftk fill_form failed: %w, stderr: %s", err, stderr.String())
	}

	// Read output file
	filledPDF, err := os.ReadFile(outputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read filled PDF: %w", err)
	}

	slog.Info("Successfully filled PDF form fields", "field_count", len(values))
	return filledPDF, nil
}

// getFormFieldNames extracts form field names using pdftk dump_data_fields
func (s *PDFService) getFormFieldNames(pdfPath string) ([]string, error) {
	cmd := exec.Command("pdftk", pdfPath, "dump_data_fields")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftk dump_data_fields failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse output to extract field names
	// Format: FieldName: fieldname
	var fieldNames []string
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "FieldName: ") {
			fieldName := strings.TrimPrefix(line, "FieldName: ")
			fieldName = strings.TrimSpace(fieldName)
			if fieldName != "" {
				fieldNames = append(fieldNames, fieldName)
			}
		}
	}

	return fieldNames, nil
}

// generateFDF generates FDF content for pdftk
func generateFDF(values map[string]string) string {
	var builder strings.Builder

	// FDF header
	builder.WriteString("%FDF-1.2\n")
	builder.WriteString("1 0 obj\n")
	builder.WriteString("<<\n")
	builder.WriteString("/FDF\n")
	builder.WriteString("<<\n")
	builder.WriteString("/Fields [\n")

	// Add field values
	for key, value := range values {
		// Escape special characters for FDF
		escapedValue := escapeFDFString(value)
		builder.WriteString(fmt.Sprintf("<< /T (%s) /V (%s) >>\n", escapeFDFString(key), escapedValue))
	}

	// FDF footer
	builder.WriteString("]\n")
	builder.WriteString(">>\n")
	builder.WriteString(">>\n")
	builder.WriteString("endobj\n")
	builder.WriteString("trailer\n")
	builder.WriteString("<<\n")
	builder.WriteString("/Root 1 0 R\n")
	builder.WriteString(">>\n")
	builder.WriteString("%%EOF\n")

	return builder.String()
}

// escapeFDFString escapes special characters for FDF format
func escapeFDFString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
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

// ExtractPlaceholders extracts all form field names from PDF using pdftk
func (s *PDFService) ExtractPlaceholders(pdfData []byte) ([]string, error) {
	if !isPDF(pdfData) {
		return nil, errors.New("invalid PDF format")
	}

	// Create temp file for pdftk
	tmpFile, err := os.CreateTemp("", "pdf_extract_*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(pdfData); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Get field names using pdftk
	fieldNames, err := s.getFormFieldNames(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to extract form fields: %w", err)
	}

	return fieldNames, nil
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
