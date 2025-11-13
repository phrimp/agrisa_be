package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"policy-service/internal/ai/gemini"
	"policy-service/internal/config"
	"policy-service/internal/database/minio"
	"policy-service/internal/database/postgres"
	"policy-service/internal/database/redis"
	"policy-service/internal/handlers"
	"policy-service/internal/repository"
	"policy-service/internal/services"
	"policy-service/internal/worker"
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

	app := fiber.New(fiber.Config{
		BodyLimit: 200 * 1024 * 1024,
	})
	app.Get("/checkhealth", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString("Policy service is healthy")
	})
	redisClient, err := redis.NewRedisClient(cfg.RedisCfg.Host, cfg.RedisCfg.Port, cfg.RedisCfg.Password, cfg.RedisCfg.DB)
	if err != nil {
		log.Printf("error connect to redis: %s", err)
	}

	geminiClient, err := gemini.NewGenAIClient(cfg.GeminiAPICfg.APIKey, cfg.GeminiAPICfg.FlashName, cfg.GeminiAPICfg.ProName)
	if err != nil {
		slog.Error("error initializing gemini client", "error", err)
	}

	// Initialize MinIO client
	minioClient, err := minio.NewMinioClient(cfg.MinioCfg)
	if err != nil {
		log.Printf("error initializing MinIO client: %s", err)
		log.Println("Warning: MinIO features will be disabled")
		minioClient = nil // Continue without MinIO
	}

	// Initialize repositories
	dataTierRepo := repository.NewDataTierRepository(db)
	basePolicyRepo := repository.NewBasePolicyRepository(db, redisClient.GetClient())
	dataSourceRepo := repository.NewDataSourceRepository(db)
	registeredPolicyRepo := repository.NewRegisteredPolicyRepository(db)
	farmRepo := repository.NewFarmRepository(db)

	// Initialize WorkerManagerV2
	workerManager := worker.NewWorkerManagerV2(db, redisClient)

	// Initialize services
	dataTierService := services.NewDataTierService(dataTierRepo)
	dataSourceService := services.NewDataSourceService(dataSourceRepo)
	basePolicyService := services.NewBasePolicyService(basePolicyRepo, dataSourceRepo, dataTierRepo, minioClient, geminiClient)
	farmService := services.NewFarmService(farmRepo, cfg, minioClient)
	registeredPolicyService := services.NewRegisteredPolicyService(registeredPolicyRepo, basePolicyRepo, basePolicyService, farmService, workerManager)
	expirationService := services.NewPolicyExpirationService(redisClient.GetClient(), basePolicyService, minioClient)

	// Expiration Listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := expirationService.StartListener(ctx); err != nil {
			log.Printf("Expiration service error: %v", err)
		}
	}()

	// Register job handlers with WorkerManagerV2
	workerManager.RegisterJobHandler("fetch-farm-monitoring-data", registeredPolicyService.FetchFarmMonitoringDataJob)
	workerManager.RegisterJobHandler("document-validation", basePolicyService.AIPolicyValidationJob)
	workerManager.RegisterJobHandler("farm-imagery", farmService.GetFarmPhotoJob)
	worker.AIWorkerPoolUUID, err = workerManager.CreateAIWorkerInfrastructure(workerManager.ManagerContext())
	if err != nil {
		slog.Error("error create AI worker pool", "error", err)
	} else {
		err = workerManager.StartAIWorkerInfrastructure(workerManager.ManagerContext(), *worker.AIWorkerPoolUUID)
		if err != nil {
			slog.Error("error starting AI worker pool", "error", err)
		}
	}

	// Recover active policy worker infrastructure after restart
	if err := registeredPolicyService.RecoverActivePolicies(ctx); err != nil {
		log.Printf("Warning: failed to recover active policies: %v", err)
		// Non-fatal: continue startup even if recovery fails
	}

	// Initialize handlers
	dataTierHandler := handlers.NewDataTierHandler(dataTierService)
	dataSourceHandler := handlers.NewDataSourceHandler(dataSourceService)
	basePolicyHandler := handlers.NewBasePolicyHandler(basePolicyService, minioClient, workerManager)
	farmHandler := handlers.NewFarmHandler(farmService, minioClient)
	policyHandler := handlers.NewPolicyHandler(registeredPolicyService)

	// Register routes
	dataTierHandler.Register(app)
	dataSourceHandler.Register(app)
	basePolicyHandler.Register(app)
	farmHandler.RegisterRoutes(app)
	policyHandler.Register(app)

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
	workerManager.Shutdown()
}
