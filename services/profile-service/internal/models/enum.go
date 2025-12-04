package models

const (
	CropTypeRice   = "lúa nước"
	CropTypeCoffee = "cà phê"
)

type DeletionRequestStatus string

const (
	DeletionRequestPending   DeletionRequestStatus = "pending"
	DeletionRequestApproved  DeletionRequestStatus = "approved"
	DeletionRequestRejected  DeletionRequestStatus = "rejected"
	DeletionRequestCancelled DeletionRequestStatus = "cancelled"
	DeletionRequestCompleted DeletionRequestStatus = "completed"
)
