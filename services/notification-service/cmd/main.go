package main

import (
	"context"
	"fmt"
	"log"
	"notification-service/internal/config"
	"notification-service/internal/event"
	"notification-service/internal/google"
	"notification-service/internal/handlers"
	"notification-service/internal/phone"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
)

func setupLogging() (*os.File, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic: %v\n", r)
		}
	}()

	logDir := filepath.Join("/agrisa", "log", "notification_service")
	fmt.Println("Log directory:", logDir)
	err := os.MkdirAll(logDir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	currentTime := time.Now()
	logFileName := fmt.Sprintf("log_%s.log", currentTime.Format("2006-01-02"))
	logFile := filepath.Join(logDir, logFileName)

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	if _, err := os.Stat(logFile); err == nil {
		absPath, err := filepath.Abs(logFile)
		if err != nil {
			fmt.Printf("Failed to get absolute path of log file: %v\n", err)
		} else {
			fmt.Printf("Log file exists at absolute path: %s\n", absPath)
		}
	} else if os.IsNotExist(err) {
		fmt.Println("Log file does not exist (it will be created)")
	} else {
		fmt.Printf("Error checking log file existence: %v\n", err)
	}

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return file, nil
}

func main() {
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Failed to set up logging: %v", err)
	}
	defer logFile.Close()
	cfg := config.New()
	app := fiber.New()
	app.Get("/checkhealth", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("Policy service is healthy")
	})

	emailService := google.NewEmailService(cfg.GoogleConfig.MailUsername, cfg.GoogleConfig.MailPassword)

	emailHandler := handlers.NewEmailHandler(emailService)

	emailHandler.Register(app)

	firebaseConfig := &google.FirebaseConfig{
		CredentialsPath: cfg.GoogleConfig.FirebaseCredentials,
		ProjectID:       cfg.GoogleConfig.FirebaseProjectID,
		BatchSize:       500,
	}

	firebaseService, err := google.NewFirebaseService(firebaseConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}
	phoneService := phone.NewPhoneService(cfg.PhoneServerConfig.Host, cfg.PhoneServerConfig.Port, cfg.PhoneServerConfig.Username, cfg.PhoneServerConfig.Password)

	// Setup queue consumer
	consumerConfig := &event.ConsumerConfig{
		RabbitMQURL: fmt.Sprintf("amqp://%s:%s@rabbitmq:%s/",
			cfg.RabbitMQCfg.Username,
			cfg.RabbitMQCfg.Password,
			cfg.RabbitMQCfg.Port),
		QueueName:       "notifications",
		DeadLetterQueue: "notifications.dlq",
		PrefetchCount:   10,
	}

	consumer, err := event.NewQueueConsumer(consumerConfig, firebaseService, emailService, phoneService)
	if err != nil {
		log.Fatalf("Failed to setup queue consumer: %v", err)
	}

	// Start consuming in goroutine
	go func() {
		if err := consumer.StartConsuming(context.Background()); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Add cleanup on shutdown
	defer consumer.Close()

	shutdownChan := make(chan os.Signal, 1)
	doneChan := make(chan bool, 1)

	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Starting server on port %s", cfg.Port)
		if err := app.Listen(fmt.Sprintf("0.0.0.0:%s", cfg.Port)); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
		doneChan <- true
	}()

	<-shutdownChan
	log.Println("Shutting down server...")
}
