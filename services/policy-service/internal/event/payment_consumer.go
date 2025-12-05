package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"policy-service/internal/models"
	"policy-service/internal/repository"
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
	conn              *RabbitMQConnection
	handler           PaymentEventHandler
	messagesProcessed int64
	messagesFailed    int64
	lastMessageTime   time.Time
	isRunning         bool
}

// NewPaymentConsumer creates a new payment event consumer
func NewPaymentConsumer(conn *RabbitMQConnection, handler PaymentEventHandler) *PaymentConsumer {
	return &PaymentConsumer{
		conn:            conn,
		handler:         handler,
		lastMessageTime: time.Now(),
		isRunning:       false,
	}
}

// Start begins consuming payment events with automatic reconnection
func (c *PaymentConsumer) Start(ctx context.Context) error {
	slog.Info("Starting payment consumer with auto-reconnect")

	c.isRunning = true

	go func() {
		defer func() {
			c.isRunning = false
		}()

		for {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				slog.Info("Payment consumer stopped - context cancelled")
				return
			default:
			}

			// Start consumer loop (will block until error or context cancelled)
			err := c.startConsumerLoop(ctx)

			if ctx.Err() != nil {
				slog.Info("Payment consumer stopped - context done")
				return
			}

			if err != nil {
				slog.Error("Payment consumer loop failed, reconnecting in 5 seconds",
					"error", err)
				time.Sleep(5 * time.Second)

				// Attempt to recreate channel if connection is alive
				if c.conn.Connection != nil && !c.conn.Connection.IsClosed() {
					ch, chErr := c.conn.Connection.Channel()
					if chErr == nil {
						if c.conn.Channel != nil {
							c.conn.Channel.Close() // Close old channel
						}
						c.conn.Channel = ch
						slog.Info("RabbitMQ channel recreated successfully")
					} else {
						slog.Error("Failed to recreate channel",
							"error", chErr)
					}
				} else {
					slog.Error("RabbitMQ connection is closed, waiting for reconnection")
				}
			}
		}
	}()

	return nil
}

// startConsumerLoop runs the actual message consumption loop
func (c *PaymentConsumer) startConsumerLoop(ctx context.Context) error {
	// Configure QoS - limit to 10 unacked messages at a time
	err := c.conn.Channel.Qos(
		10,    // prefetch count - process 10 messages at a time
		0,     // prefetch size (0 = no limit)
		false, // global (false = apply to this channel only)
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare the queue with dead letter queue configuration
	args := amqp.Table{
		"x-dead-letter-exchange":    "dlx.payment",
		"x-dead-letter-routing-key": "payment_events.failed",
	}

	_, err = c.conn.Channel.QueueDeclare(
		PaymentEventsQueue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		args,  // arguments for DLQ
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
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
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	slog.Info("Payment consumer started successfully",
		"queue", PaymentEventsQueue,
		"prefetch_count", 10)

	// Process messages until channel closes or context cancelled
	for {
		select {
		case <-ctx.Done():
			slog.Info("Consumer loop stopping - context cancelled")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				slog.Warn("Payment consumer channel closed")
				return fmt.Errorf("message channel closed")
			}
			c.processMessage(ctx, msg)
		}
	}
}

func (c *PaymentConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	// Add timeout to prevent hanging indefinitely
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var event PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("failed to unmarshal payment event", "error", err)
		c.messagesFailed++
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

	if err := c.handler.HandlePaymentCompleted(processCtx, event); err != nil {
		slog.Error("failed to handle payment event",
			"payment_id", event.ID,
			"error", err,
		)
		c.messagesFailed++
		// Requeue the message for retry
		msg.Nack(false, true)
		return
	}

	// Acknowledge successful processing
	msg.Ack(false)
	c.messagesProcessed++
	c.lastMessageTime = time.Now()
	slog.Info("Payment event processed successfully", "payment_id", event.ID)
}

// DefaultPaymentEventHandler is the default implementation of PaymentEventHandler
type DefaultPaymentEventHandler struct {
	registeredPolicyRepo *repository.RegisteredPolicyRepository
	basePolicyRepo       *repository.BasePolicyRepository
	workerManager        *worker.WorkerManagerV2
	claimRepo            *repository.ClaimRepository
	payoutRepo           *repository.PayoutRepository
	notievent            *NotificationHelper
}

// NewDefaultPaymentEventHandler creates a new default payment event handler
func NewDefaultPaymentEventHandler(
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	basePolicyRepo *repository.BasePolicyRepository,
	workerManager *worker.WorkerManagerV2,
	claimRepo *repository.ClaimRepository,
	payoutRepo *repository.PayoutRepository,
	notievent *NotificationHelper,
) *DefaultPaymentEventHandler {
	return &DefaultPaymentEventHandler{
		registeredPolicyRepo: registeredPolicyRepo,
		basePolicyRepo:       basePolicyRepo,
		workerManager:        workerManager,
		claimRepo:            claimRepo,
		payoutRepo:           payoutRepo,
		notievent:            notievent,
	}
}

// HandlePaymentCompleted handles a completed payment event
func (h *DefaultPaymentEventHandler) HandlePaymentCompleted(ctx context.Context, event PaymentEvent) error {
	// Validate payment event type - CRITICAL: must return error to retry
	if event.Type == nil {
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    "payment event has no type",
		}
	}

	// Verify payment status - CRITICAL: must return error to retry
	// This could be a timing issue where event was sent before payment completed
	if event.Status != "paid" && event.Status != "completed" {
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    fmt.Sprintf("payment not in paid/completed status: %s", event.Status),
		}
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
	case models.PaymentTypePolicyPayout:
		return h.handlePolicyPayoutPayment(ctx, event)

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
		// CRITICAL: Return error for unsupported types to allow retry
		// This could be a new payment type that hasn't been deployed yet
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    fmt.Sprintf("unsupported payment type: %s", *event.Type),
		}
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

func (h *DefaultPaymentEventHandler) handlePolicyPayoutPayment(
	ctx context.Context,
	event PaymentEvent,
) error {
	paidAt := event.PaidAt.Unix()
	slog.Info("processing policy payout payment",
		"payment_id", event.ID,
		"order_items_count", len(event.OrderItems),
		"amount", event.Amount)

	// Process each order item (policy)
	for _, orderItem := range event.OrderItems {
		if err := h.processPolicyPayoutPayment(ctx, event, orderItem, paidAt); err != nil {
			slog.Error("failed to process policy payout payment",
				"payment_id", event.ID,
				"order_item_id", orderItem.ID,
				"error", err)
			return err
		}
	}

	slog.Info("payment event processed successfully", "payment_id", event.ID)

	return nil
}

func (h *DefaultPaymentEventHandler) processPolicyPayoutPayment(
	ctx context.Context,
	event PaymentEvent,
	orderItem OrderItem,
	paidAt int64,
) error {
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

	claims, err := h.claimRepo.GetByRegisteredPolicyID(ctx, registeredPolicyID)
	if err != nil {
		slog.Error("failed to retrieve claim", "error", err)
		return err
	}
	if len(claims) == 0 {
		slog.Error("there are no claims for this policy", "error", err)
		return err
	}
	claim := claims[0]

	if claim.Status != models.ClaimApproved {
		slog.Warn("only approved claim is allowed to be processed", "actual status", claim.Status)
		return fmt.Errorf("invalid claim status=%v", claim.Status)
	}

	payout, err := h.payoutRepo.GetByClaimID(ctx, claim.ID)
	if err != nil {
		slog.Error("failed to retrieve payout", "error", err)
		return err
	}
	if payout.Status != models.PayoutProcessing {
		slog.Warn("only processing payout is allowed to be processed", "actual status", payout.Status)
		return fmt.Errorf("invalid payout status=%v", payout.Status)
	}

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
	registeredPolicy, err := h.registeredPolicyRepo.GetByID(registeredPolicyID)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to retrieve registered policy",
			"policy_id", registeredPolicyID,
			"error", err)
		return err
	}

	if registeredPolicy.Status != models.PolicyActive {
		tx.Rollback()
		slog.Warn("only active policy is allowed to be processed", "actual status", registeredPolicy.Status)
		return fmt.Errorf("invalid policy status=%v", registeredPolicy.Status)
	}

	// Verify payment amount matches premium
	expectedAmount := payout.PayoutAmount
	if math.Abs(orderItem.Price-expectedAmount) > 0.01 {
		tx.Rollback()
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    "payment amount mismatch",
			Details: map[string]any{
				"expected":  expectedAmount,
				"received":  orderItem.Price,
				"policy_id": registeredPolicyID,
			},
		}
	}

	// Update policy with payment information
	payout.Status = models.PayoutCompleted
	payout.CompletedAt = &paidAt

	registeredPolicy.Status = models.PolicyPayout

	// Update policy in transaction
	err = h.registeredPolicyRepo.UpdateTx(tx, registeredPolicy)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to update registered policy",
			"policy_id", registeredPolicyID,
			"error", err)
		return err
	}

	err = h.claimRepo.UpdateStatusTX(tx, ctx, claim.ID, models.ClaimPaid)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to update claim",
			"claim id", claim.ID,
			"error", err)
		return err
	}

	err = h.payoutRepo.UpdatePayoutTx(tx, payout)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to update payout",
			"payout id", claim.ID,
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

	slog.Info("payout processed successfully",
		"policy_id", registeredPolicyID,
		"payment_id", event.ID,
		"coverage_start_date", registeredPolicy.CoverageStartDate,
		"coverage_end_date", registeredPolicy.CoverageEndDate)

	if err := h.workerManager.CleanupWorkerInfrastructure(ctx, registeredPolicyID); err != nil {
		slog.Error("error cleanup worker infrastructure for policy", "policy_id", registeredPolicyID, "error", err)
	}

	go func() {
		for {
			err := h.notievent.NotifyPayoutCompleted(ctx, registeredPolicy.FarmerID, registeredPolicy.PolicyNumber, payout.PayoutAmount)
			if err == nil {
				slog.Info("payout completed notification sent", "policy id", registeredPolicy.PolicyNumber)
				return
			}
			slog.Error("error sending payout completed notification", "error", err)
			time.Sleep(10 * time.Second)
		}
	}()
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
	registeredPolicy, err := h.registeredPolicyRepo.GetByID(registeredPolicyID)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to retrieve registered policy",
			"policy_id", registeredPolicyID,
			"error", err)
		return err
	}

	basePolicy, err := h.basePolicyRepo.GetBasePolicyByID(registeredPolicy.BasePolicyID)
	if err != nil {
		tx.Rollback()
		slog.Error("failed to retrieve base policy", "error", err)
		return err
	}

	if registeredPolicy.UnderwritingStatus != models.UnderwritingApproved {
		tx.Rollback()
		slog.Warn("only policy with underwriting approved are allowed to be processed", "actual status", registeredPolicy.UnderwritingStatus)
		return nil
	}

	if registeredPolicy.Status != models.PolicyPendingPayment {
		tx.Rollback()
		slog.Warn("only policy pending payment are allowed to be processed", "actual status", registeredPolicy.Status)
		return nil
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
				"expected":  expectedAmount,
				"received":  orderItem.Price,
				"policy_id": registeredPolicyID,
			},
		}
	}

	// Update policy with payment information
	now := time.Now().Unix()
	if registeredPolicy.CoverageStartDate == 0 {
		registeredPolicy.CoverageStartDate = max(now, int64(*basePolicy.InsuranceValidFromDay))
	}
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
			"policy_id":    registeredPolicyID.String(),
			"start_date":   0,
			"end_date":     0,
			"check_policy": true,
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

// ConsumerHealthStatus represents the health status of the payment consumer
type ConsumerHealthStatus struct {
	IsHealthy         bool      `json:"is_healthy"`
	IsRunning         bool      `json:"is_running"`
	MessagesProcessed int64     `json:"messages_processed"`
	MessagesFailed    int64     `json:"messages_failed"`
	LastMessageTime   time.Time `json:"last_message_time"`
	ConnectionStatus  string    `json:"connection_status"`
	HealthMessage     string    `json:"health_message,omitempty"`
}

// HealthCheck returns the current health status of the consumer
func (c *PaymentConsumer) HealthCheck() ConsumerHealthStatus {
	status := ConsumerHealthStatus{
		IsRunning:         c.isRunning,
		MessagesProcessed: c.messagesProcessed,
		MessagesFailed:    c.messagesFailed,
		LastMessageTime:   c.lastMessageTime,
	}

	// Check connection status
	if c.conn.Connection == nil || c.conn.Connection.IsClosed() {
		status.ConnectionStatus = "disconnected"
		status.IsHealthy = false
		status.HealthMessage = "RabbitMQ connection is closed"
		return status
	}

	status.ConnectionStatus = "connected"

	// Check if consumer is running
	if !c.isRunning {
		status.IsHealthy = false
		status.HealthMessage = "Consumer is not running"
		return status
	}

	// Check if we've processed messages recently (within last 10 minutes)
	// This check is only relevant if we've processed at least one message
	if c.messagesProcessed > 0 {
		timeSinceLastMessage := time.Since(c.lastMessageTime)
		if timeSinceLastMessage > 10*time.Minute {
			status.IsHealthy = false
			status.HealthMessage = fmt.Sprintf("No messages processed in last %v", timeSinceLastMessage)
			return status
		}
	}

	// Check failure rate
	if c.messagesProcessed > 0 {
		totalMessages := c.messagesProcessed + c.messagesFailed
		failureRate := float64(c.messagesFailed) / float64(totalMessages)
		if failureRate > 0.5 {
			status.IsHealthy = false
			status.HealthMessage = fmt.Sprintf("High failure rate: %.1f%%", failureRate*100)
			return status
		}
	}

	status.IsHealthy = true
	status.HealthMessage = "Consumer is healthy"
	return status
}

// GetMetrics returns consumer metrics
func (c *PaymentConsumer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"messages_processed":  c.messagesProcessed,
		"messages_failed":     c.messagesFailed,
		"last_message_time":   c.lastMessageTime,
		"is_running":          c.isRunning,
		"connection_alive":    c.conn.Connection != nil && !c.conn.Connection.IsClosed(),
		"total_messages":      c.messagesProcessed + c.messagesFailed,
		"success_rate":        c.calculateSuccessRate(),
		"time_since_last_msg": time.Since(c.lastMessageTime).String(),
	}
}

// calculateSuccessRate calculates the success rate of message processing
func (c *PaymentConsumer) calculateSuccessRate() float64 {
	total := c.messagesProcessed + c.messagesFailed
	if total == 0 {
		return 100.0
	}
	return (float64(c.messagesProcessed) / float64(total)) * 100.0
}
