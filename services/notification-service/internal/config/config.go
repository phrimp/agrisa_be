package config

import "os"

type NotificationService struct {
	Port         string
	RabbitMQCfg  RabbitMQConfig
	GoogleConfig GoogleConfig
}

type RabbitMQConfig struct {
	Username string
	Password string
	Port     string
}

type GoogleConfig struct {
	MailUsername string
	MailPassword string
}

func New() *NotificationService {
	return &NotificationService{
		Port: getEnvOrDefault("NOTIFICATION_SERVICE_PORT", "8088"),
		RabbitMQCfg: RabbitMQConfig{
			Username: getEnvOrDefault("RABBITMQ_USER", "admin"),
			Password: getEnvOrDefault("RABBITMQ_PWD", "admin"),
			Port:     getEnvOrDefault("RABBITMQ_PORT", "5672"),
		},
		GoogleConfig: GoogleConfig{
			MailUsername: getEnvOrDefault("GOOGLE_USERNAME", ""),
			MailPassword: getEnvOrDefault("GOOGLE_PASSWORD", "password"),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
