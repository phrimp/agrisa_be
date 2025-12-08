package event

import (
	utils "agrisa_utils"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

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
		NotiQueue, // queue name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		p.messagesFailed++
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	totalEvent := NotificationMessage{
		ID:           utils.GenerateRandomStringWithLength(6),
		Type:         TypeSMS,
		Priority:     PriorityHigh,
		RecipientID:  "",
		Payload:      map[string]any{"payload": event},
		RetryCount:   0,
		MaxRetries:   5,
		CreatedAt:    time.Now(),
		ScheduledFor: nil,
	}

	// Marshal the event to JSON
	body, err := json.Marshal(totalEvent)
	if err != nil {
		p.messagesFailed++
		return fmt.Errorf("failed to marshal notification event: %w", err)
	}

	// Publish the message
	err = p.conn.Channel.PublishWithContext(
		ctx,
		"",        // exchange
		NotiQueue, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
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
		"queue", NotiQueue,
		"title", event.Notification.Title,
	)

	return nil
}
