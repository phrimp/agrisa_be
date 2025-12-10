package phone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type PhoneService struct {
	Host     string
	Port     string
	Username string
	Password string
}

type smsPayload struct {
	TextMessage struct {
		Text string `json:"text"`
	} `json:"textMessage"`
	PhoneNumbers []string `json:"phoneNumbers"`
}

func NewPhoneService(host, port, username, password string) *PhoneService {
	return &PhoneService{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

func (p *PhoneService) SendSMS(title, content string, phoneNumbers []string) error {
	// --- 1. Preparation & URL Construction ---
	const op = "PhoneService.SendSMS"
	log := slog.With("operation", op)

	url := fmt.Sprintf("%s:%s/message", p.Host, p.Port)
	log.Info("Starting SMS delivery process",
		"target_url", url,
		"recipients_count", len(phoneNumbers),
		"title", title,
	)

	// --- 2. Payload Creation and Marshal ---
	payload := smsPayload{
		PhoneNumbers: phoneNumbers,
	}
	payload.TextMessage.Text = fmt.Sprintf("%s\n%s", title, content)

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		log.Error("Failed to marshal SMS payload",
			"error", err,
			"payload_struct", payload,
		)
		return fmt.Errorf("failed to marshal SMS payload: %w", err)
	}
	log.Info("Payload successfully marshaled", "payload_bytes", string(jsonBody))

	// --- 3. Request Creation and Setup ---
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error("Failed to create HTTP request", "error", err)
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Basic Auth credentials are set here but generally shouldn't be logged directly
	req.SetBasicAuth(p.Username, p.Password)
	req.Header.Set("Content-Type", "application/json")

	log.Info("HTTP request configured",
		"method", req.Method,
		"headers", req.Header,
		"auth_user", p.Username, // Log the username, but not the password
	)

	// --- 4. Request Execution ---
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Do(req)

	// Log the total request duration
	log.Info("Request execution complete", "duration", time.Since(startTime))

	if err != nil {
		log.Error("Failed to send SMS request (network/timeout error)",
			"error", err,
			"elapsed_time", time.Since(startTime),
		)
		return fmt.Errorf("failed to send SMS request: %w", err)
	}
	defer resp.Body.Close()

	// --- 5. Response Check and Error Logging ---
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		// Read the body for detailed error information from the server
		responseBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			responseBody = fmt.Appendf(nil, "failed to read response body: %v", readErr)
		}

		log.Error("External server returned non-success status",
			"status_code", resp.StatusCode,
			"status", resp.Status,
			"response_body", string(responseBody), // Log the server's error message
			"url", url,
		)
		return fmt.Errorf("external server returned non-success status: %s. Response body: %s", resp.Status, responseBody)
	}

	// --- 6. Success Log ---
	log.Info("SMS successfully sent",
		"status", resp.Status,
		"elapsed_time", time.Since(startTime),
	)

	return nil
}
