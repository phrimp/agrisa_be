package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var GeminiClients []GeminiClient

type GeminiClient struct {
	Client     *genai.Client
	FlashModel *genai.GenerativeModel
	ProModel   *genai.GenerativeModel
}

func NewGenAIClient(apiKey, flashModelName, proModelName string) (*GeminiClient, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("genai client init failed: %w", err)
	}

	return &GeminiClient{
		Client:     client,
		FlashModel: client.GenerativeModel(flashModelName),
		ProModel:   client.GenerativeModel(proModelName),
	}, nil
}

func (g *GeminiClient) SendAIWithPDF(ctx context.Context, prompt string, data map[string]any) (map[string]any, error) {
	fileData := data["pdf"].([]byte)

	resp, err := g.ProModel.GenerateContent(ctx,
		genai.Text(prompt),
		genai.Blob{
			MIMEType: "application/pdf",
			Data:     fileData,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("no content returned from AI")
	}
	textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("response part is not text, received %T", resp.Candidates[0].Content.Parts[0])
	}
	aiResponse := string(textPart)
	if strings.HasPrefix(aiResponse, "```json") {
		aiResponse = strings.TrimPrefix(aiResponse, "```json\n")
		aiResponse = strings.TrimSuffix(aiResponse, "\n```")
	}
	aiResponse = strings.TrimSpace(aiResponse)
	var resultMap map[string]any
	err = json.Unmarshal([]byte(aiResponse), &resultMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI response to JSON: %w. \nRaw response was: %s", err, aiResponse)
	}
	return resultMap, nil
}

// SendAIWithPDFAndRetry attempts the request with automatic failover across multiple clients
func SendAIWithPDFAndRetry(ctx context.Context, prompt string, data map[string]any, selector *GeminiClientSelector) (map[string]any, error) {
	var result map[string]any

	err := selector.TryAllClients(func(client *GeminiClient, clientIdx int) error {
		resp, err := client.SendAIWithPDF(ctx, prompt, data)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// SendAIWithImages sends a prompt with multiple images (base64 encoded) to the AI model
func (g *GeminiClient) SendAIWithImages(ctx context.Context, prompt string, imageData []string) (map[string]any, error) {
	parts := []genai.Part{genai.Text(prompt)}

	for i, imgBase64 := range imageData {
		if imgBase64 == "" {
			slog.Warn("Empty image data at index, skipping", "index", i)
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(imgBase64)
		if err != nil {
			slog.Warn("Failed to decode image base64", "index", i, "error", err)
			continue
		}

		// Detect MIME type based on magic bytes
		mimeType := detectImageMIMEType(decoded)

		parts = append(parts, genai.Blob{
			MIMEType: mimeType,
			Data:     decoded,
		})
	}

	slog.Info("Sending AI request with images",
		"prompt_length", len(prompt),
		"image_count", len(parts)-1) // -1 for the text prompt

	resp, err := g.ProModel.GenerateContent(ctx, parts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content with images: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("no content returned from AI")
	}

	textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("response part is not text, received %T", resp.Candidates[0].Content.Parts[0])
	}

	aiResponse := string(textPart)

	// Clean up markdown JSON wrapper if present
	if strings.HasPrefix(aiResponse, "```json") {
		aiResponse = strings.TrimPrefix(aiResponse, "```json\n")
		aiResponse = strings.TrimSuffix(aiResponse, "\n```")
	}
	aiResponse = strings.TrimSpace(aiResponse)

	var resultMap map[string]any
	err = json.Unmarshal([]byte(aiResponse), &resultMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI response to JSON: %w. \nRaw response was: %s", err, aiResponse)
	}

	return resultMap, nil
}

// SendAIWithImagesAndRetry attempts the request with automatic failover across multiple clients
func SendAIWithImagesAndRetry(ctx context.Context, prompt string, imageData []string, selector *GeminiClientSelector) (map[string]any, error) {
	var result map[string]any

	err := selector.TryAllClients(func(client *GeminiClient, clientIdx int) error {
		resp, err := client.SendAIWithImages(ctx, prompt, imageData)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// detectImageMIMEType detects the MIME type of an image based on magic bytes
func detectImageMIMEType(data []byte) string {
	if len(data) < 8 {
		return "image/jpeg" // default fallback
	}

	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}

	// WebP: 52 49 46 46 ... 57 45 42 50
	if data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
		if len(data) > 11 && data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
			return "image/webp"
		}
	}

	// BMP: 42 4D
	if data[0] == 0x42 && data[1] == 0x4D {
		return "image/bmp"
	}

	// Default to JPEG as it's most common
	return "image/jpeg"
}
