package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"policy-service/internal/config"
	"policy-service/internal/database/postgres"

	"github.com/gofiber/fiber/v3"
)

func setupLogging() (*os.File, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic: %v\n", r)
		}
	}()

	logDir := filepath.Join("/agrisa", "log", "policy_service")
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
	log.Printf("Connecting to PostgreSQL with: host=%s, port=%s, user=%s, dbname=auth_service",
		cfg.PostgresCfg.Host, cfg.PostgresCfg.Port, cfg.PostgresCfg.Username)
	db, err := postgres.ConnectAndCreateDB(cfg.PostgresCfg)
	if err != nil {
		log.Printf("error connect to database: %s", err)
		go postgres.RetryConnectOnFailed(30*time.Second, &db, cfg.PostgresCfg)
	}

	app := fiber.New()
	app.Get("/checkhealth", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("Policy service is healthy")
	})
}
