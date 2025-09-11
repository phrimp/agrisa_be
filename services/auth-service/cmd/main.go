package main

import (
	"auth-service/internal/config"
	"auth-service/internal/database/postgres"
	"auth-service/internal/handlers"
	"auth-service/internal/minio"
	"auth-service/internal/repository"
	"auth-service/internal/services"
	"auth-service/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func setupLogging() (*os.File, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic: %v\n", r)
		}
	}()

	logDir := filepath.Join("/agrisa", "log", "auth_service")
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
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Failed to set up logging: %v", err)
	}
	defer logFile.Close()

	cfg := config.New()
	log.Printf("Line 65 - main.go: Connecting to PostgreSQL with: host=%s, port=%s, user=%s, dbname=auth_service",
		cfg.PostgresCfg.Host, cfg.PostgresCfg.Port, cfg.PostgresCfg.Username)

	db, err := postgres.ConnectAndCreateDB(cfg.PostgresCfg)
	if err != nil {
		log.Printf("error connect to database: %s", err)
		go postgres.RetryConnectOnFailed(30*time.Second, &db, cfg.PostgresCfg)
	}

	//minio client
	mc, err := minio.NewMinioClient(cfg.MinioCfg)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	// utils
	utils := utils.NewUtils(mc, cfg)

	// repositories
	userRepo := repository.NewUserRepository(db)
	userCardRepo := repository.NewUserCardRepository(db)
	ekycProgressRepo := repository.NewUserEkycProgressRepository(db)

	// services
	userService := services.NewUserService(userRepo, mc, cfg, utils, userCardRepo, ekycProgressRepo)
	userHandler := handlers.NewUserHandler(userService)

	// Setup Gin router
	r := gin.Default()

	// Register routes
	handlers.RegisterRoutes(r, userHandler)

	// Start HTTP server
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8083"
	}

	log.Printf("Starting auth-service on port %s", serverPort)
	if err := r.Run(":" + serverPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
