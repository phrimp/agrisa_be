package phone

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	url := fmt.Sprintf("http://%s:%s/message", p.Host, p.Port)

	payload := smsPayload{
		PhoneNumbers: phoneNumbers,
	}
	payload.TextMessage.Text = fmt.Sprintf("%s\n%s", title, content)

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SMS payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.SetBasicAuth(p.Username, p.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second, // Set a timeout for the request
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("external server returned non-success status: %s", resp.Status)
	}

	return nil
}
