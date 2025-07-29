package main

import (
	"auth-service/internal/config"
	"auth-service/internal/database/postgres"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func setupLogging() (*os.File, error) {
	logDir := filepath.Join("/agrisa", "log", "auth_service")
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	currentTime := time.Now()
	logFileName := fmt.Sprintf("log_%s.log", currentTime.Format("2006-01-02"))
	logFile := filepath.Join(logDir, logFileName)

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
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

	db, err := postgres.ConnectAndCreateDB(cfg.PostgresCfg)
	if err != nil {
		log.Printf("error connect to database: %s", err)
		go postgres.RetryConnectOnFailed(30*time.Second, db, cfg.PostgresCfg)
	}
}
