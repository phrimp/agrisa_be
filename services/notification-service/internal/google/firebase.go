package google

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type PushNotificationPayload struct {
	Token       string            `json:"token"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Data        map[string]string `json:"data,omitempty"`
	ImageURL    string            `json:"image_url,omitempty"`
	ClickAction string            `json:"click_action,omitempty"`
	Badge       *int              `json:"badge,omitempty"`
	Sound       string            `json:"sound,omitempty"`
}
type FirebaseService struct {
	app    *firebase.App
	client *messaging.Client
	config *FirebaseConfig
}

type FirebaseConfig struct {
	CredentialsPath string
	ProjectID       string
	BatchSize       int // For batch sending
}

func NewFirebaseService(cfg *FirebaseConfig) (*FirebaseService, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile(cfg.CredentialsPath)
	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: cfg.ProjectID,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %v", err)
	}

	return &FirebaseService{
		app:    app,
		client: client,
		config: cfg,
	}, nil
}

// Send single notification
func (f *FirebaseService) SendPushNotification(ctx context.Context, payload *PushNotificationPayload) (string, error) {
	message := &messaging.Message{
		Token: payload.Token,
		Notification: &messaging.Notification{
			Title:    payload.Title,
			Body:     payload.Body,
			ImageURL: payload.ImageURL,
		},
		Data: payload.Data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ClickAction: payload.ClickAction,
				Sound:       payload.Sound,
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Badge: payload.Badge,
					Sound: payload.Sound,
				},
			},
		},
	}

	response, err := f.client.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("error sending message: %v", err)
	}

	return response, nil
}

// Batch send for efficiency
func (f *FirebaseService) SendBatchNotifications(ctx context.Context, messages []*messaging.Message) (*messaging.BatchResponse, error) {
	if len(messages) > 500 {
		return nil, fmt.Errorf("batch size exceeds FCM limit of 500")
	}

	response, err := f.client.SendEach(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("error sending batch: %v", err)
	}

	return response, nil
}
