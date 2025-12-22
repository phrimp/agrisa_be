package event

const ProfileQueue string = "profile_events"

type ProfileEvent struct {
	ID        string           `json:"id"`
	EventType ProfileEventType `json:"event_type"`
	UserID    string           `json:"user_id"`
	ProfileID string           `json:"profile_id"`
}

type ProfileEventType string

const (
	ProfilePendingDelete = "pending_delete"
	ProfileCancelDelete  = "delete_cancelled"
	ProfleConfirmDelete  = "confirm_delete"
)
