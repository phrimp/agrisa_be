package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
