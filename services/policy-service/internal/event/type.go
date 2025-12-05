package event

type NotificationEventPushModel struct {
	Notification Notification `json:"notification"`
	UserIDs      []string     `json:"user_ids"`
}

type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

const PushNotiQueue string = "push_noti_events"
