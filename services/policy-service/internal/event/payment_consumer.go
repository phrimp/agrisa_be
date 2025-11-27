package event

import (
	"context"
	"encoding/json"
	"log/slog"
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
}

// NewDefaultPaymentEventHandler creates a new default payment event handler
func NewDefaultPaymentEventHandler(registeredPolicyService *services.RegisteredPolicyService) *DefaultPaymentEventHandler {
	return &DefaultPaymentEventHandler{
		registeredPolicyService: registeredPolicyService,
	}
}

// HandlePaymentCompleted handles a completed payment event
func (h *DefaultPaymentEventHandler) HandlePaymentCompleted(ctx context.Context, event PaymentEvent) error {
	for _, orderItem := range event.OrderItems {
		dailyJob := worker.JobPayload{
			JobID: uuid.NewString(),
			Type:  "fetch-farm-monitoring-data",
			Params: map[string]any{
				"policy_id":  orderItem.ItemID,
				"start_date": 0,
				"end_date":   0,
			},
			MaxRetries: 10,
			RunNow:     true,
		}
		slog.Info("starting job", "job", dailyJob)
	}
	// TODO: Implement payment completion logic
	// This should:
	// 1. Find the registered policy associated with this payment (via order_code or payment type)
	// 2. Update the policy status to "active" or "paid"
	// 3. Trigger any necessary workflows (e.g., start monitoring, send notifications)
	// 4. Update billing records
	//
	// Example implementation:
	// - Parse event.Type to determine what kind of payment this is (e.g., "policy_registration", "policy_renewal")
	// - Look up the associated policy using event.OrderCode or event.OrderItems[].ItemID
	// - Update policy status and payment records
	// - Start monitoring workers if this is a new policy activation

	slog.Info("TODO: Handle payment completed",
		"payment_id", event.ID,
		"order_code", event.OrderCode,
		"type", event.Type,
		"amount", event.Amount,
		"user_id", event.UserID,
	)

	return nil
}
