package event

import (
	"fmt"
	"log/slog"
	"policy-service/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQConnection holds the RabbitMQ connection and channel
type RabbitMQConnection struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

// ConnectRabbitMQ establishes a connection to RabbitMQ
func ConnectRabbitMQ(cfg config.RabbitMQConfig) (*RabbitMQConnection, error) {
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
	)

	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	slog.Info("Connected to RabbitMQ", "host", cfg.Host, "port", cfg.Port)

	return &RabbitMQConnection{
		Connection: conn,
		Channel:    ch,
	}, nil
}

// Close closes the RabbitMQ connection and channel
func (r *RabbitMQConnection) Close() error {
	if r.Channel != nil {
		if err := r.Channel.Close(); err != nil {
			slog.Error("failed to close RabbitMQ channel", "error", err)
		}
	}
	if r.Connection != nil {
		if err := r.Connection.Close(); err != nil {
			slog.Error("failed to close RabbitMQ connection", "error", err)
			return err
		}
	}
	slog.Info("RabbitMQ connection closed")
	return nil
}
