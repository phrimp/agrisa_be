package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	"weather-service/internal/config"
	"weather-service/internal/handlers"
	"weather-service/internal/services"

	"github.com/gin-gonic/gin"
)

func setupLogging() (*os.File, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic: %v\n", r)
		}
	}()

	logDir := filepath.Join("/agrisa", "log", "weather_service")
	fmt.Println("Log directory:", logDir)
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
	// Setup logging
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Error setting up logging: %v", err)
	}
	defer logFile.Close()

	// Load configuration
	config := config.New()
	log.Printf("Weather Service Configuration: %+v", config)

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8086"
	}

	r := gin.Default()
	// Initialize and register routes
	// Initialize services and handlers here
	weatherService := services.NewWeatherService(*config)
	weatherHandler := handlers.NewWeatherHandler(weatherService)
	weatherHandler.RegisterRoutes(r)

	log.Printf("Starting weather-service on port %s", serverPort)
	if err := r.Run(":" + serverPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
