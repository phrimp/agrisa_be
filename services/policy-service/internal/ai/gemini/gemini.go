package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

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
