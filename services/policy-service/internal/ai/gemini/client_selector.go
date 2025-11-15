package gemini

import (
	"fmt"
	"log/slog"
	"sync"
)

// GeminiClientSelector manages round-robin selection and failover across multiple Gemini clients
type GeminiClientSelector struct {
	clients      []GeminiClient
	currentIndex int
	mutex        sync.Mutex
}

// NewGeminiClientSelector creates a new client selector with round-robin support
func NewGeminiClientSelector(clients []GeminiClient) *GeminiClientSelector {
	return &GeminiClientSelector{
		clients:      clients,
		currentIndex: 0,
	}
}

// GetNextClient returns the next client in round-robin order
func (s *GeminiClientSelector) GetNextClient() (*GeminiClient, int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.clients) == 0 {
		return nil, -1
	}

	client := &s.clients[s.currentIndex]
	index := s.currentIndex

	// Move to next client for next request
	s.currentIndex = (s.currentIndex + 1) % len(s.clients)

	return client, index
}

// GetClientCount returns total number of clients
func (s *GeminiClientSelector) GetClientCount() int {
	return len(s.clients)
}

// TryAllClients attempts the operation with all clients until one succeeds
func (s *GeminiClientSelector) TryAllClients(operation func(*GeminiClient, int) error) error {
	clientCount := s.GetClientCount()
	if clientCount == 0 {
		return fmt.Errorf("no Gemini clients available")
	}

	var lastErr error
	errorsCollected := make([]string, 0, clientCount)

	for attempt := 0; attempt < clientCount; attempt++ {
		client, clientIdx := s.GetNextClient()

		slog.Info("Attempting Gemini API request",
			"client_index", clientIdx,
			"attempt", attempt+1,
			"total_clients", clientCount)

		err := operation(client, clientIdx)
		if err == nil {
			slog.Info("Gemini API request succeeded",
				"client_index", clientIdx,
				"attempt", attempt+1)
			return nil
		}

		lastErr = err
		errorsCollected = append(errorsCollected, fmt.Sprintf("client[%d]: %v", clientIdx, err))

		slog.Warn("Gemini API request failed, trying next client",
			"client_index", clientIdx,
			"attempt", attempt+1,
			"error", err)
	}

	// All clients failed
	slog.Error("All Gemini clients exhausted",
		"total_attempts", clientCount,
		"errors", errorsCollected)

	return fmt.Errorf("all %d Gemini clients failed, last error: %w", clientCount, lastErr)
}
