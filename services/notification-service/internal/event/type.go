package event

import "time"

type NotificationType string

const (
	TypeEmail NotificationType = "email"
	TypeSMS   NotificationType = "sms"
	TypeInApp NotificationType = "in_app"
)

type NotificationPriority int

const (
	PriorityLow    NotificationPriority = 1
	PriorityNormal NotificationPriority = 5
	PriorityHigh   NotificationPriority = 10
)

type NotificationMessage struct {
	ID           string               `json:"id"`
	Type         NotificationType     `json:"type"`
	Priority     NotificationPriority `json:"priority"`
	RecipientID  string               `json:"recipient_id"`
	Payload      map[string]any       `json:"payload"`
	RetryCount   int                  `json:"retry_count"`
	MaxRetries   int                  `json:"max_retries"`
	CreatedAt    time.Time            `json:"created_at"`
	ScheduledFor *time.Time           `json:"scheduled_for,omitempty"`
}

type NotificationEventPushModelPayload struct {
	Payload NotificationEventPushModel `json:"payload"`
}

type NotificationEventPushModel struct {
	Notification Notification `json:"notification"`
	Destinations []string     `json:"destinations"`
}

type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
