package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/services"
	"policy-service/internal/worker"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	PaymentEventsQueue = "payment_events"
)

// PaymentEvent represents the payment event data from payment-service
type PaymentEvent struct {
	ID          string      `json:"id"`
	Amount      float64     `json:"amount"`
	Description string      `json:"description"`
	Status      string      `json:"status"`
	UserID      string      `json:"user_id"`
	CheckoutURL *string     `json:"checkout_url"`
	OrderCode   *string     `json:"order_code"`
	Type        *string     `json:"type"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DeletedAt   *time.Time  `json:"deleted_at"`
	PaidAt      *time.Time  `json:"paid_at"`
	ExpiredAt   *time.Time  `json:"expired_at"`
	OrderItems  []OrderItem `json:"orderItems"`
}

// OrderItem represents an order item in the payment event
type OrderItem struct {
	ID        string    `json:"id"`
	PaymentID string    `json:"payment_id"`
	ItemID    string    `json:"item_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PaymentEventHandler defines the interface for handling payment events
type PaymentEventHandler interface {
	HandlePaymentCompleted(ctx context.Context, event PaymentEvent) error
}

// PaymentConsumer consumes payment events from RabbitMQ
type PaymentConsumer struct {
	conn    *RabbitMQConnection
	handler PaymentEventHandler
}

// NewPaymentConsumer creates a new payment event consumer
func NewPaymentConsumer(conn *RabbitMQConnection, handler PaymentEventHandler) *PaymentConsumer {
	return &PaymentConsumer{
		conn:    conn,
		handler: handler,
	}
}

// Start begins consuming payment events
func (c *PaymentConsumer) Start(ctx context.Context) error {
	// Declare the queue (ensure it exists)
	_, err := c.conn.Channel.QueueDeclare(
		PaymentEventsQueue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	// Start consuming messages
	msgs, err := c.conn.Channel.Consume(
		PaymentEventsQueue,
		"",    // consumer tag (auto-generated)
		false, // auto-ack (we'll manually ack after processing)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	slog.Info("Payment consumer started", "queue", PaymentEventsQueue)

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Payment consumer stopped")
				return
			case msg, ok := <-msgs:
				if !ok {
					slog.Warn("Payment consumer channel closed")
					return
				}
				c.processMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *PaymentConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	var event PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("failed to unmarshal payment event", "error", err)
		// Reject the message and don't requeue (malformed message)
		msg.Nack(false, false)
		return
	}

	slog.Info("Received payment event",
		"payment_id", event.ID,
		"order_code", event.OrderCode,
		"amount", event.Amount,
		"status", event.Status,
	)

	if err := c.handler.HandlePaymentCompleted(ctx, event); err != nil {
		slog.Error("failed to handle payment event",
			"payment_id", event.ID,
			"error", err,
		)
		// Requeue the message for retry
		msg.Nack(false, true)
		return
	}

	// Acknowledge successful processing
	msg.Ack(false)
	slog.Info("Payment event processed successfully", "payment_id", event.ID)
}

// DefaultPaymentEventHandler is the default implementation of PaymentEventHandler
type DefaultPaymentEventHandler struct {
	registeredPolicyService *services.RegisteredPolicyService
	registeredPolicyRepo    *repository.RegisteredPolicyRepository
	workerManager           *worker.WorkerManagerV2
}

// NewDefaultPaymentEventHandler creates a new default payment event handler
func NewDefaultPaymentEventHandler(
	registeredPolicyService *services.RegisteredPolicyService,
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	workerManager *worker.WorkerManagerV2,
) *DefaultPaymentEventHandler {
	return &DefaultPaymentEventHandler{
		registeredPolicyService: registeredPolicyService,
		registeredPolicyRepo:    registeredPolicyRepo,
		workerManager:           workerManager,
	}
}

// HandlePaymentCompleted handles a completed payment event
func (h *DefaultPaymentEventHandler) HandlePaymentCompleted(ctx context.Context, event PaymentEvent) error {
	// Validate payment event type
	if event.Type == nil {
		slog.Warn("payment event has no type", "payment_id", event.ID)
		return nil
	}

	// Verify payment status (common validation for all payment types)
	if event.Status != "paid" && event.Status != "completed" {
		slog.Warn("payment not in paid status",
			"payment_id", event.ID,
			"status", event.Status)
		return nil
	}

	// Validate payment has PaidAt timestamp (common validation for all payment types)
	if event.PaidAt == nil {
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    "payment missing paid_at timestamp",
		}
	}

	// Route to appropriate handler based on payment type
	switch models.PaymentType(*event.Type) {
	case models.PaymentTypePolicyRegistration:
		return h.handlePolicyRegistrationPayment(ctx, event)

	// ============================================================================
	// TODO: ADD NEW PAYMENT TYPE HANDLERS HERE
	// ============================================================================
	//
	// To add a new payment type handler:
	//
	// 1. Add the payment type constant to internal/models/enums.go:
	//    const PaymentTypeYourNewType PaymentType = "your_payment_type"
	//
	// 2. Add a new case statement here:
	//    case models.PaymentTypeYourNewType:
	//        return h.handleYourNewTypePayment(ctx, event)
	//
	// 3. Implement the handler function following this pattern:
	//
	//    func (h *DefaultPaymentEventHandler) handleYourNewTypePayment(
	//        ctx context.Context,
	//        event PaymentEvent,
	//    ) error {
	//        paidAt := event.PaidAt.Unix()
	//
	//        slog.Info("processing your new type payment",
	//            "payment_id", event.ID,
	//            "order_items_count", len(event.OrderItems))
	//
	//        // Add your business logic here:
	//        // - Validate payment-type-specific requirements
	//        // - Process order items with transactions
	//        // - Update relevant records
	//        // - Trigger necessary workflows
	//
	//        return nil
	//    }
	//
	// Example: Policy Renewal Payment Handler
	//
	// case models.PaymentTypePolicyRenewal:
	//     return h.handlePolicyRenewalPayment(ctx, event)
	//
	// func (h *DefaultPaymentEventHandler) handlePolicyRenewalPayment(
	//     ctx context.Context,
	//     event PaymentEvent,
	// ) error {
	//     paidAt := event.PaidAt.Unix()
	//
	//     for _, orderItem := range event.OrderItems {
	//         // Parse policy ID
	//         policyID, err := uuid.Parse(orderItem.ItemID)
	//         if err != nil {
	//             return &PaymentValidationError{
	//                 PaymentID: event.ID,
	//                 Reason:    "invalid policy id format",
	//             }
	//         }
	//
	//         // Begin transaction
	//         tx, err := h.registeredPolicyRepo.BeginTransaction()
	//         if err != nil {
	//             return err
	//         }
	//         defer tx.Rollback()
	//
	//         // Get existing policy
	//         policy, err := h.registeredPolicyService.GetPolicyByID(policyID)
	//         if err != nil {
	//             return err
	//         }
	//
	//         // Verify policy can be renewed
	//         if policy.Status != models.PolicyExpired {
	//             return &PaymentValidationError{
	//                 PaymentID: event.ID,
	//                 Reason:    "policy is not in expired status",
	//             }
	//         }
	//
	//         // Update policy for renewal
	//         policy.Status = models.PolicyActive
	//         policy.CoverageStartDate = time.Now().Unix()
	//         // Calculate new end date based on base policy duration
	//
	//         // Update in transaction
	//         if err := h.registeredPolicyRepo.UpdateTx(tx, policy); err != nil {
	//             return err
	//         }
	//
	//         // Commit transaction
	//         if err := tx.Commit(); err != nil {
	//             return err
	//         }
	//
	//         // Restart monitoring if needed
	//         h.startPolicyMonitoring(policyID, orderItem.ItemID)
	//     }
	//
	//     return nil
	// }
	//
	// ============================================================================

	default:
		slog.Info("unsupported payment type, skipping",
			"payment_id", event.ID,
			"type", *event.Type)
		return nil
	}
}

// handlePolicyRegistrationPayment handles policy registration payment events
func (h *DefaultPaymentEventHandler) handlePolicyRegistrationPayment(
	ctx context.Context,
	event PaymentEvent,
) error {
	paidAt := event.PaidAt.Unix()

	slog.Info("processing policy registration payment",
		"payment_id", event.ID,
		"order_items_count", len(event.OrderItems),
		"amount", event.Amount)

	// Process each order item (policy)
	for _, orderItem := range event.OrderItems {
		if err := h.processPolicyPayment(ctx, event, orderItem, paidAt); err != nil {
			slog.Error("failed to process policy payment",
				"payment_id", event.ID,
				"order_item_id", orderItem.ID,
				"error", err)
			return err
		}
	}

	slog.Info("payment event processed successfully", "payment_id", event.ID)
	return nil
}

// processPolicyPayment processes payment for a single policy within a transaction
func (h *DefaultPaymentEventHandler) processPolicyPayment(
	ctx context.Context,
	event PaymentEvent,
	orderItem OrderItem,
	paidAt int64,
) error {
	// Parse policy ID
	registeredPolicyID, err := uuid.Parse(orderItem.ItemID)
	if err != nil {
		slog.Error("invalid policy id in order item",
			"order_item_id", orderItem.ID,
			"item_id", orderItem.ItemID,
			"error", err)
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    "invalid policy id format",
		}
	}

	// Begin transaction
	tx, err := h.registeredPolicyRepo.BeginTransaction()
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-panic after rollback
		}
	}()

	// Retrieve policy
	registeredPolicy, err := h.registeredPolicyService.GetPolicyByID(registeredPolicyID)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to retrieve registered policy",
			"policy_id", registeredPolicyID,
			"error", err)
		return err
	}

	// Idempotency check: skip if already processed
	if registeredPolicy.PremiumPaidByFarmer {
		tx.Rollback()
		slog.Warn("payment already processed for policy",
			"policy_id", registeredPolicyID,
			"payment_id", event.ID)
		return nil
	}

	// Verify payment amount matches premium
	expectedAmount := registeredPolicy.TotalFarmerPremium
	if math.Abs(orderItem.Price-expectedAmount) > 0.01 {
		tx.Rollback()
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    "payment amount mismatch",
			Details: map[string]interface{}{
				"expected": expectedAmount,
				"received": orderItem.Price,
				"policy_id": registeredPolicyID,
			},
		}
	}

	// Update policy with payment information
	now := time.Now().Unix()
	registeredPolicy.CoverageStartDate = now // Coverage starts immediately upon payment
	registeredPolicy.Status = models.PolicyActive
	registeredPolicy.PremiumPaidByFarmer = true
	registeredPolicy.PremiumPaidAt = &paidAt

	// Update policy in transaction
	err = h.registeredPolicyRepo.UpdateTx(tx, registeredPolicy)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to update registered policy",
			"policy_id", registeredPolicyID,
			"error", err)
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction",
			"policy_id", registeredPolicyID,
			"error", err)
		return err
	}

	slog.Info("policy activated successfully",
		"policy_id", registeredPolicyID,
		"payment_id", event.ID,
		"coverage_start_date", registeredPolicy.CoverageStartDate,
		"coverage_end_date", registeredPolicy.CoverageEndDate)

	// Start monitoring after successful database commit
	if err := h.startPolicyMonitoring(registeredPolicyID, orderItem.ItemID); err != nil {
		// Log error but don't fail the payment processing
		// Monitoring can be started manually if needed
		slog.Error("failed to start policy monitoring (payment still successful)",
			"policy_id", registeredPolicyID,
			"error", err)
	}

	return nil
}

// startPolicyMonitoring starts monitoring job for the policy
func (h *DefaultPaymentEventHandler) startPolicyMonitoring(
	registeredPolicyID uuid.UUID,
	policyIDString string,
) error {
	scheduler, ok := h.workerManager.GetSchedulerByPolicyID(registeredPolicyID)
	if !ok {
		return &MonitoringError{
			PolicyID: registeredPolicyID,
			Reason:   "scheduler not found for policy",
		}
	}

	dailyJob := worker.JobPayload{
		JobID: uuid.NewString(),
		Type:  "fetch-farm-monitoring-data",
		Params: map[string]any{
			"policy_id":  registeredPolicyID.String(),
			"start_date": 0,
			"end_date":   0,
		},
		MaxRetries: 5,
		RunNow:     true,
	}

	scheduler.AddJob(dailyJob)
	slog.Info("monitoring job started",
		"policy_id", registeredPolicyID,
		"job_id", dailyJob.JobID)

	return nil
}

// PaymentValidationError represents a payment validation error
type PaymentValidationError struct {
	PaymentID string
	Reason    string
	Details   map[string]interface{}
}

func (e *PaymentValidationError) Error() string {
	if len(e.Details) > 0 {
		return "payment validation failed: " + e.Reason + " (payment_id: " + e.PaymentID + ")"
	}
	return "payment validation failed: " + e.Reason + " (payment_id: " + e.PaymentID + ")"
}

// MonitoringError represents a monitoring setup error
type MonitoringError struct {
	PolicyID uuid.UUID
	Reason   string
}

func (e *MonitoringError) Error() string {
	return "monitoring setup failed: " + e.Reason + " (policy_id: " + e.PolicyID.String() + ")"
}
