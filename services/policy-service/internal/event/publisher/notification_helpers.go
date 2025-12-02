package publisher

import (
	"context"
	"fmt"
)

// NotificationHelper provides convenient methods for publishing common notification types
type NotificationHelper struct {
	publisher *NotificationPublisher
}

// NewNotificationHelper creates a new notification helper
func NewNotificationHelper(publisher *NotificationPublisher) *NotificationHelper {
	return &NotificationHelper{
		publisher: publisher,
	}
}

// NotifyPolicyRegistered sends a notification when a policy is registered
func (h *NotificationHelper) NotifyPolicyRegistered(ctx context.Context, userID, policyNumber string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Policy Registered Successfully",
			Body:  fmt.Sprintf("Your policy %s has been registered and is now waiting for underwriting.", policyNumber),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

func (h *NotificationHelper) NotifyPolicyRegisteredPartner(ctx context.Context, userIDs []string, basePolicyNumber string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Policy Registered Successfully",
			Body:  fmt.Sprintf("Your base policy %s has been registered and is now waiting for underwriting.", basePolicyNumber),
		},
		UserIDs: userIDs,
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyPolicyExpiring sends a notification when a policy is about to expire
func (h *NotificationHelper) NotifyPolicyExpiring(ctx context.Context, userID, policyNumber string, daysRemaining int) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Policy Expiring Soon",
			Body:  fmt.Sprintf("Your policy %s will expire in %d days. Please renew to maintain coverage.", policyNumber, daysRemaining),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyPolicyExpired sends a notification when a policy has expired
func (h *NotificationHelper) NotifyPolicyExpired(ctx context.Context, userID, policyNumber string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Policy Expired",
			Body:  fmt.Sprintf("Your policy %s has expired. Please renew to continue coverage.", policyNumber),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyClaimGenerated sends a notification when a claim is automatically generated
func (h *NotificationHelper) NotifyClaimGenerated(ctx context.Context, userID, policyNumber, claimID string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Claim Generated",
			Body:  fmt.Sprintf("A claim (ID: %s) has been generated for your policy %s.", claimID, policyNumber),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyClaimApproved sends a notification when a claim is approved
func (h *NotificationHelper) NotifyClaimApproved(ctx context.Context, userID, claimID string, payoutAmount float64) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Claim Approved",
			Body:  fmt.Sprintf("Your claim (ID: %s) has been approved. Payout amount: %.2f", claimID, payoutAmount),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyClaimRejected sends a notification when a claim is rejected
func (h *NotificationHelper) NotifyClaimRejected(ctx context.Context, userID, claimID, reason string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Claim Rejected",
			Body:  fmt.Sprintf("Your claim (ID: %s) has been rejected. Reason: %s", claimID, reason),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyPaymentReceived sends a notification when payment is received
func (h *NotificationHelper) NotifyPaymentReceived(ctx context.Context, userID, policyNumber string, amount float64) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Payment Received",
			Body:  fmt.Sprintf("Payment of %.2f has been received for policy %s.", amount, policyNumber),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyUnderwritingCompleted sends a notification when underwriting is completed
func (h *NotificationHelper) NotifyUnderwritingCompleted(ctx context.Context, userID, policyNumber, status string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Underwriting Completed",
			Body:  fmt.Sprintf("Underwriting for policy %s is complete. Status: %s", policyNumber, status),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyRiskAnalysisCompleted sends a notification when risk analysis is completed
func (h *NotificationHelper) NotifyRiskAnalysisCompleted(ctx context.Context, userID, policyNumber, riskLevel string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: "Risk Analysis Completed",
			Body:  fmt.Sprintf("Risk analysis for policy %s is complete. Risk level: %s", policyNumber, riskLevel),
		},
		UserIDs: []string{userID},
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyCustom sends a custom notification
func (h *NotificationHelper) NotifyCustom(ctx context.Context, title, body string, userIDs []string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: title,
			Body:  body,
		},
		UserIDs: userIDs,
	}
	return h.publisher.PublishNotification(ctx, event)
}

// NotifyMultipleUsers sends the same notification to multiple users
func (h *NotificationHelper) NotifyMultipleUsers(ctx context.Context, title, body string, userIDs []string) error {
	event := NotificationEventPushModel{
		Notification: Notification{
			Title: title,
			Body:  body,
		},
		UserIDs: userIDs,
	}
	return h.publisher.PublishNotification(ctx, event)
}
