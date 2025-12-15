package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// NotificationPublisher publishes notification events to RabbitMQ
type NotificationPublisher struct {
	conn              *RabbitMQConnection
	messagesPublished int64
	messagesFailed    int64
	lastPublishTime   time.Time
}

// NewNotificationPublisher creates a new notification event publisher
func NewNotificationPublisher(conn *RabbitMQConnection) *NotificationPublisher {
	return &NotificationPublisher{
		conn:            conn,
		lastPublishTime: time.Now(),
	}
}

// PublishNotification publishes a notification event to the push_noti_events queue
func (p *NotificationPublisher) PublishNotification(ctx context.Context, event NotificationEventPushModel) error {
	// Ensure the queue exists
	_, err := p.conn.Channel.QueueDeclare(
		PushNotiQueue, // queue name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		p.messagesFailed++
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Marshal the event to JSON
	body, err := json.Marshal(event)
	if err != nil {
		p.messagesFailed++
		return fmt.Errorf("failed to marshal notification event: %w", err)
	}

	// Publish the message
	err = p.conn.Channel.PublishWithContext(
		ctx,
		"",            // exchange
		PushNotiQueue, // routing key (queue name)
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		p.messagesFailed++
		return fmt.Errorf("failed to publish notification event: %w", err)
	}

	p.messagesPublished++
	p.lastPublishTime = time.Now()

	slog.Info("Notification event published",
		"queue", PushNotiQueue,
		"title", event.Title,
		"user_count", len(event.LstUserIds),
	)

	return nil
}

// PublishNotificationBatch publishes multiple notification events in batch
func (p *NotificationPublisher) PublishNotificationBatch(ctx context.Context, events []NotificationEventPushModel) error {
	// Ensure the queue exists
	_, err := p.conn.Channel.QueueDeclare(
		PushNotiQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	successCount := 0
	failureCount := 0

	for _, event := range events {
		// Marshal the event to JSON
		body, err := json.Marshal(event)
		if err != nil {
			failureCount++
			slog.Error("Failed to marshal notification event", "error", err)
			continue
		}

		// Publish the message
		err = p.conn.Channel.PublishWithContext(
			ctx,
			"",
			PushNotiQueue,
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         body,
				Timestamp:    time.Now(),
			},
		)
		if err != nil {
			failureCount++
			slog.Error("Failed to publish notification event", "error", err)
			continue
		}

		successCount++
	}

	p.messagesPublished += int64(successCount)
	p.messagesFailed += int64(failureCount)
	p.lastPublishTime = time.Now()

	slog.Info("Notification batch published",
		"queue", PushNotiQueue,
		"success", successCount,
		"failed", failureCount,
		"total", len(events),
	)

	if failureCount > 0 {
		return fmt.Errorf("batch publish completed with %d failures out of %d total events", failureCount, len(events))
	}

	return nil
}

// GetMetrics returns publisher metrics
func (p *NotificationPublisher) GetMetrics() map[string]any {
	return map[string]any{
		"messages_published": p.messagesPublished,
		"messages_failed":    p.messagesFailed,
		"last_publish_time":  p.lastPublishTime,
		"queue":              PushNotiQueue,
	}
}

// HealthCheck returns the health status of the publisher
func (p *NotificationPublisher) HealthCheck() PublisherHealthStatus {
	isHealthy := p.conn != nil && p.conn.Connection != nil && !p.conn.Connection.IsClosed()

	return PublisherHealthStatus{
		IsHealthy:         isHealthy,
		MessagesPublished: p.messagesPublished,
		MessagesFailed:    p.messagesFailed,
		LastPublishTime:   p.lastPublishTime,
		Queue:             PushNotiQueue,
	}
}

// PublisherHealthStatus represents the health status of the publisher
type PublisherHealthStatus struct {
	IsHealthy         bool      `json:"is_healthy"`
	MessagesPublished int64     `json:"messages_published"`
	MessagesFailed    int64     `json:"messages_failed"`
	LastPublishTime   time.Time `json:"last_publish_time"`
	Queue             string    `json:"queue"`
}
