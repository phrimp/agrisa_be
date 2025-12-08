package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"notification-service/internal/google"
	"notification-service/internal/phone"
	"time"

	"github.com/streadway/amqp"
)

type QueueConsumer struct {
	conn            *amqp.Connection
	channel         *amqp.Channel
	firebaseService *google.FirebaseService
	emailService    *google.EmailService
	phoneService    *phone.PhoneService
	queueName       string
	deadLetterQueue string
}

type ConsumerConfig struct {
	RabbitMQURL     string
	QueueName       string
	DeadLetterQueue string
	PrefetchCount   int
}

func NewQueueConsumer(cfg *ConsumerConfig, email *google.EmailService, phoneService *phone.PhoneService) (*QueueConsumer, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}

	// Set QoS for controlled processing
	err = ch.Qos(
		cfg.PrefetchCount, // prefetch count
		0,                 // prefetch size
		false,             // global
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %v", err)
	}

	// Declare main queue with DLX

	_, err = ch.QueueDeclare(
		cfg.QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %v", err)
	}

	// Declare dead letter queue
	_, err = ch.QueueDeclare(
		cfg.DeadLetterQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare DLQ: %v", err)
	}

	return &QueueConsumer{
		conn:            conn,
		channel:         ch,
		emailService:    email,
		phoneService:    phoneService,
		queueName:       cfg.QueueName,
		deadLetterQueue: cfg.DeadLetterQueue,
	}, nil
}

func (q *QueueConsumer) StartConsuming(ctx context.Context) error {
	msgs, err := q.channel.Consume(
		q.queueName,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %v", err)
	}

	for {
		select {
		case msg := <-msgs:
			if err := q.processMessage(ctx, msg); err != nil {
				log.Printf("Error processing message: %v", err)

				// Check retry count
				retryCount := 0
				if val, ok := msg.Headers["x-retry-count"].(int32); ok {
					retryCount = int(val)
				}

				if retryCount < 3 {
					// Requeue with exponential backoff
					q.requeueMessage(msg, retryCount+1)
				} else {
					// Send to DLQ
					msg.Nack(false, false)
					log.Printf("Message sent to DLQ after %d retries", retryCount)
				}
			} else {
				msg.Ack(false)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (q *QueueConsumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	var notification NotificationMessage
	if err := json.Unmarshal(msg.Body, &notification); err != nil {
		return fmt.Errorf("failed to unmarshal message: %v", err)
	}

	switch notification.Type {
	case TypeSMS:
		return q.processSMS(ctx, &notification)
		//	case TypeEmail:
		//		return q.processEmailNotification(ctx, &notification)
	default:
		return fmt.Errorf("unsupported notification type: %s", notification.Type)
	}
}

func (q *QueueConsumer) processSMS(ctx context.Context, notif *NotificationMessage) error {
	payloadBytes, err := json.Marshal(notif.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}
	var smsPayload NotificationEventPushModelPayload
	if err := json.Unmarshal(payloadBytes, &smsPayload); err != nil {
		return fmt.Errorf("failed to unmarshal push payload: %v", err)
	}
	slog.Info("SMS event receive", "payload", smsPayload)
	err = q.phoneService.SendSMS(smsPayload.Payload.Notification.Title, smsPayload.Payload.Notification.Body, smsPayload.Payload.Destinations)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

func (q *QueueConsumer) processPushNotification(ctx context.Context, notif *NotificationMessage) error {
	// Parse payload
	payloadBytes, err := json.Marshal(notif.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	var pushPayload google.PushNotificationPayload
	if err := json.Unmarshal(payloadBytes, &pushPayload); err != nil {
		return fmt.Errorf("failed to unmarshal push payload: %v", err)
	}

	// Send via Firebase
	messageID, err := q.firebaseService.SendPushNotification(ctx, &pushPayload)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %v", err)
	}

	log.Printf("Successfully sent push notification: %s", messageID)
	return nil
}

func (q *QueueConsumer) requeueMessage(msg amqp.Delivery, retryCount int) error {
	// Add retry count to headers
	headers := msg.Headers
	if headers == nil {
		headers = amqp.Table{}
	}
	headers["x-retry-count"] = int32(retryCount)

	// Calculate backoff delay
	delay := time.Duration(retryCount*retryCount) * time.Second

	// Publish with delay
	return q.channel.Publish(
		"",          // exchange
		q.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg.Body,
			Headers:     headers,
			Expiration:  fmt.Sprintf("%d", delay.Milliseconds()),
		},
	)
}

func (q *QueueConsumer) Close() error {
	if err := q.channel.Close(); err != nil {
		return err
	}
	return q.conn.Close()
}
