package event

// NotificationEventPushModel matches SendPayloadDto from noti-service
// TypeScript interface: { lstUserIds?: string[], title: string, body: string, data?: any }
type NotificationEventPushModel struct {
	LstUserIds []string               `json:"lstUserIds,omitempty"`
	Title      string                 `json:"title"`
	Body       string                 `json:"body"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// Deprecated: Use flat structure in NotificationEventPushModel instead
type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

const PushNotiQueue string = "push_noti_events"
