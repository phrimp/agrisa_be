package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"policy-service/internal/repository"
	"policy-service/internal/worker"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ProfileEventQueue = "profile_events"
)

type ProfileEventType string

const (
	ProfilePendingDelete = "pending_delete"
	ProfileCancelDelete  = "delete_cancelled"
	ProfleConfirmDelete  = "confirm_delete"
)

type ProfileEvent struct {
	ID        string           `json:"id"`
	EventType ProfileEventType `json:"event_type"`
	UserID    string           `json:"user_id"`
	ProfileID string           `json:"profile_id"`
}

type ProfileConsumer struct {
	conn              *RabbitMQConnection
	handler           ProfileEventHandler
	messagesProcessed int64
	messagesFailed    int64
	lastMessageTime   time.Time
	isRunning         bool
}

type ProfileEventHandler interface {
	HandleProfileEvent(ctx context.Context, event ProfileEvent) error
}

// NewProfileConsumer creates a new consumer event consumer
func NewProfileConsumer(conn *RabbitMQConnection, handler ProfileEventHandler) *ProfileConsumer {
	return &ProfileConsumer{
		conn:            conn,
		handler:         handler,
		lastMessageTime: time.Now(),
		isRunning:       false,
	}
}

func (c *ProfileConsumer) Start(ctx context.Context) error {
	slog.Info("Starting profile consumer with auto-reconnect")

	c.isRunning = true

	go func() {
		defer func() {
			c.isRunning = false
		}()

		for {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				slog.Info("Profile consumer stopped - context cancelled")
				return
			default:
			}

			// Start consumer loop (will block until error or context cancelled)
			err := c.startConsumerLoop(ctx)

			if ctx.Err() != nil {
				slog.Info("Profile consumer stopped - context done")
				return
			}

			if err != nil {
				slog.Error("Profile consumer loop failed, reconnecting in 5 seconds",
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

func (c *ProfileConsumer) startConsumerLoop(ctx context.Context) error {
	// Configure QoS - limit to 10 unacked messages at a time
	err := c.conn.Channel.Qos(
		10,    // prefetch count - process 10 messages at a time
		0,     // prefetch size (0 = no limit)
		false, // global (false = apply to this channel only)
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	_, err = c.conn.Channel.QueueDeclare(
		ProfileEventQueue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Start consuming messages
	msgs, err := c.conn.Channel.Consume(
		ProfileEventQueue,
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

	slog.Info("Profile consumer started successfully",
		"queue", ProfileEventQueue,
		"prefetch_count", 10)

	// Process messages until channel closes or context cancelled
	for {
		select {
		case <-ctx.Done():
			slog.Info("Consumer loop stopping - context cancelled")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				slog.Warn("Profile consumer channel closed")
				return fmt.Errorf("message channel closed")
			}
			c.processMessage(ctx, msg)
		}
	}
}

func (c *ProfileConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	// Add timeout to prevent hanging indefinitely
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var event ProfileEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		slog.Error("failed to unmarshal profile event", "error", err)
		c.messagesFailed++
		// Reject the message and don't requeue (malformed message)
		msg.Nack(false, false)
		return
	}

	slog.Info("Received profile event",
		"event_id", event.ID,
		"user_id", event.UserID,
		"profile_id", event.ProfileID,
	)

	if err := c.handler.HandleProfileEvent(processCtx, event); err != nil {
		slog.Error("failed to handle profile event",
			"event_id", event.ID,
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
	slog.Info("Profile event processed successfully", "event_id", event.ID)
}

type DefaultProfileEventHandler struct {
	registeredPolicyRepo *repository.RegisteredPolicyRepository
	cancelRequestRepo    *repository.CancelRequestRepository
	basePolicyRepo       *repository.BasePolicyRepository
	workerManager        *worker.WorkerManagerV2
}

// NewDefaultPaymentEventHandler creates a new default payment event handler
func NewDefaultProfileEventHandler(
	registeredPolicyRepo *repository.RegisteredPolicyRepository,
	basePolicyRepo *repository.BasePolicyRepository,
	workerManager *worker.WorkerManagerV2,
	cancelRequestRepo *repository.CancelRequestRepository,
) *DefaultProfileEventHandler {
	return &DefaultProfileEventHandler{
		registeredPolicyRepo: registeredPolicyRepo,
		basePolicyRepo:       basePolicyRepo,
		workerManager:        workerManager,
		cancelRequestRepo:    cancelRequestRepo,
	}
}

func (h *DefaultProfileEventHandler) HandleProfileEvent(ctx context.Context, event ProfileEvent) error {
	slog.Info("Profile Event Consumed", "event", event)
	if event.EventType == "" {
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    "payment event has no type",
		}
	}
	switch event.EventType {
	case ProfleConfirmDelete:
	case ProfileCancelDelete:
	default:
		return &PaymentValidationError{
			PaymentID: event.ID,
			Reason:    fmt.Sprintf("unsupported payment type: %s", event.EventType),
		}
	}

	return nil
}

func (h *DefaultProfileEventHandler) handleProfileConfirmDelete(ctx context.Context, event ProfileEvent) error {
	slog.Info("CONFIRM DELETE PROFILE EVENT", "event", event)
	return nil
}
