package services

import (
	utils "agrisa_utils"
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/database/minio"
	"policy-service/internal/event/publisher"
	"policy-service/internal/models"
	"policy-service/internal/repository"
	"policy-service/internal/worker"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// PolicyExpirationService handles auto-commit of expired archive policies
type PolicyExpirationService struct {
	redisClient               *redis.Client
	minioClient               *minio.MinioClient
	policyService             *BasePolicyService
	stopChannel               chan struct{}
	stats                     *ExpirationStats
	policyRenewalOrchestrator *PolicyRenewalOrchestrator
	basePolicyRepo            *repository.BasePolicyRepository
	notiPublisher             *publisher.NotificationHelper
}

// ExpirationStats tracks processing statistics
type ExpirationStats struct {
	TotalExpired      int64
	SuccessfulCommits int64
	FailedCommits     int64
	LastProcessed     time.Time
	mu                sync.RWMutex
}

// NewPolicyExpirationService creates a new expiration service instance
func NewPolicyExpirationService(redisClient *redis.Client, policyService *BasePolicyService, minioClient *minio.MinioClient, policyRepo *repository.RegisteredPolicyRepository, basePolicyRepo *repository.BasePolicyRepository, notiPublisher *publisher.NotificationHelper, workerManager *worker.WorkerManagerV2) *PolicyExpirationService {
	validityCalculator := NewBasePolicyValidityCalculator()
	policyRenewalOrchestrator := NewPolicyRenewalOrchestrator(basePolicyRepo, policyRepo, validityCalculator, workerManager, notiPublisher)
	return &PolicyExpirationService{
		minioClient:   minioClient,
		redisClient:   redisClient,
		policyService: policyService,
		stopChannel:   make(chan struct{}),
		stats: &ExpirationStats{
			LastProcessed: time.Now(),
		},
		policyRenewalOrchestrator: policyRenewalOrchestrator,
		basePolicyRepo:            basePolicyRepo,
		notiPublisher:             notiPublisher,
	}
}

// StartListener begins listening for Redis expiration events
func (s *PolicyExpirationService) StartListener(ctx context.Context) error {
	slog.Info("Starting policy expiration listener")

	// Subscribe to Redis expiration events
	pubsub := s.redisClient.PSubscribe(ctx, "__keyevent@*__:expired")
	defer pubsub.Close()

	// Listen for expiration events
	for {
		select {
		case msg := <-pubsub.Channel():
			if s.isArchivePolicyKey(msg.Payload) {
				go s.processExpiredDraftPolicy(ctx, msg.Payload)
			}
			if s.isValidDateKey(msg.Payload) {
				slog.Info("DEBUG Expiration key catched", "key", msg.Payload)
				go s.processExpiredPolicy(ctx, msg.Payload)
			}
			if s.isEnrollmentClosed(msg.Payload) {
				slog.Info("DEBUG Expiration key catched", "key", msg.Payload)
				go s.processEnrollmentClosed(ctx, msg.Payload)
			}
		case <-ctx.Done():
			slog.Info("Policy expiration listener stopped")
			return ctx.Err()
		case <-s.stopChannel:
			slog.Info("Policy expiration listener stopped gracefully")
			return nil
		}
	}
}

// Stop gracefully stops the expiration listener
func (s *PolicyExpirationService) Stop() {
	close(s.stopChannel)
}

// isArchivePolicyKey checks if the expired key is a BasePolicy with archive:true
func (s *PolicyExpirationService) isArchivePolicyKey(expiredKey string) bool {
	// Pattern: {provider}--{policyID}--BasePolicy--archive:true
	if strings.Contains(expiredKey, "--BasePolicy--archive:true--COMMIT_EVENT") {
		return true
	}
	expiredKey = strings.Split(expiredKey, "--COMMIT_EVENT")[0]
	go s.processUnArchivedExpiredPolicy(context.Background(), expiredKey)
	return false
}

func (s *PolicyExpirationService) isEnrollmentClosed(expiredKey string) bool {
	return strings.Contains(expiredKey, "--BasePolicy--EnrollmentClosed")
}

func (s *PolicyExpirationService) isValidDateKey(expiredKey string) bool {
	return strings.Contains(expiredKey, "--BasePolicy--ValidDate")
}

func (s *PolicyExpirationService) processEnrollmentClosed(ctx context.Context, expiredKey string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CRITICAL: Panic recovery", "panic", r)
		}
	}()
	expiredKey = strings.Split(expiredKey, "--BasePolicy--EnrollmentClosed")[0]
	basePolicyID, err := uuid.Parse(expiredKey)
	if err != nil {
		slog.Error("error parsing base_policy_id retry spamming", "base_policy_id", expiredKey, "error", err)
		time.Sleep(10 * time.Second)
		s.processExpiredPolicy(ctx, expiredKey)
	}
	basePolicy, err := s.basePolicyRepo.GetBasePolicyByID(basePolicyID)
	if err != nil {
		slog.Error("error retriving base policy", "base_policy_id", basePolicyID, "error", err)
		return
	}
	if basePolicy.Status == models.BasePolicyArchived {
		slog.Error("invalid operation: base policy status invalid", "status", basePolicy.Status)
		return
	}
	basePolicy.Status = models.BasePolicyClosed
	err = s.basePolicyRepo.UpdateBasePolicy(basePolicy)
	if err != nil {
		slog.Error("error updating base policy", "base_policy_id", basePolicyID, "error", err)
		return
	}
}

func (s *PolicyExpirationService) processExpiredPolicy(ctx context.Context, expiredKey string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CRITICAL: Panic recovery", "panic", r)
		}
	}()
	expiredKey = strings.Split(expiredKey, "--BasePolicy--ValidDate")[0]
	basePolicyID, err := uuid.Parse(expiredKey)
	if err != nil {
		slog.Error("error parsing base_policy_id retry spamming", "base_policy_id", expiredKey, "error", err)
		time.Sleep(10 * time.Second)
		s.processExpiredPolicy(ctx, expiredKey)
	}
	result, err := s.policyRenewalOrchestrator.PrepareRenewal(ctx, basePolicyID)
	if err != nil {
		slog.Error("error policy renew process", "base_policy_id", basePolicyID, "error", err)
		return
	}

	slog.Info("policy renew successfully", "result", result)

	if result.IsExpired {
		go func() {
			for {
				err := s.notiPublisher.NotifyPolicyExpiredBatch(ctx, result.FarmerIDs, result.PolicyCode)
				if err == nil {
					slog.Info("policy expiration notification sent", "policy id", result.PolicyCode)
					return
				}
				slog.Error("error sending policy expiration notification", "error", err)
				time.Sleep(10 * time.Second)
			}
		}()
	} else {
		go func() {
			for {
				err := s.notiPublisher.NotifyPolicyRenewedBatch(ctx, result.FarmerIDs, result.PolicyCode)
				if err == nil {
					slog.Info("policy renewed notification sent", "policy id", result.PolicyCode)
					return
				}
				slog.Error("error sending policy renewed notification", "error", err)
				time.Sleep(10 * time.Second)
			}
		}()
	}
}

func (s *PolicyExpirationService) processUnArchivedExpiredPolicy(ctx context.Context, expiredKey string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovery", "panic", r)
		}
	}()
	policyData, err := s.policyService.basePolicyRepo.GetTempBasePolicyModels(ctx, expiredKey)
	if err != nil {
		slog.Error("Failed to extract Policy data", "error", err)
		return
	}
	var policy models.BasePolicy
	err = utils.DeserializeModel(policyData, &policy)
	if err != nil {
		slog.Error("Failed to deserialze policy model", "error", err)
		return
	}

	err = s.minioClient.DeleteFile(ctx, minio.Storage.PolicyDocuments, *policy.TemplateDocumentURL)
	if err != nil {
		slog.Error("Failed to delete Temp Policy Document", "error", err)
	}
}

// processExpiredPolicy handles a single expired archive policy
func (s *PolicyExpirationService) processExpiredDraftPolicy(ctx context.Context, expiredKey string) {
	slog.Info("Processing expired archive policy", "expired_key", expiredKey)

	s.updateStats(true, false) // Mark as processed

	expiredKey = strings.Split(expiredKey, "--COMMIT_EVENT")[0]
	// Extract policy information from expired key
	policyInfo, err := s.extractPolicyInfo(expiredKey)
	if err != nil {
		slog.Error("Failed to extract policy info", "expired_key", expiredKey, "error", err)
		s.updateStats(false, true)
		return
	}

	// Auto-commit to database
	commitRequest := &models.CommitPoliciesRequest{
		BasePolicyID:    policyInfo.PolicyID,
		ProviderID:      policyInfo.ProviderID,
		ArchiveStatus:   "true",
		DeleteFromRedis: true,
		BatchSize:       1,
	}

	response, err := s.policyService.CommitPolicies(ctx, commitRequest)
	if err != nil {
		slog.Error("Auto-commit failed",
			"expired_key", expiredKey,
			"policy_id", policyInfo.PolicyID,
			"provider_id", policyInfo.ProviderID,
			"error", err)
		s.updateStats(false, true)
		return
	}

	// Log success
	slog.Info("Auto-commit completed successfully",
		"expired_key", expiredKey,
		"policy_id", policyInfo.PolicyID,
		"provider_id", policyInfo.ProviderID,
		"committed_count", response.TotalCommitted)

	s.updateStats(false, false) // Mark as successful
}

// PolicyInfo holds extracted policy information from Redis key
type PolicyInfo struct {
	ProviderID string
	PolicyID   string
}

// extractPolicyInfo extracts policy information from expired Redis key
func (s *PolicyExpirationService) extractPolicyInfo(expiredKey string) (*PolicyInfo, error) {
	// Expected format: {provider}--{policyID}--BasePolicy--archive:true
	parts := strings.Split(expiredKey, "--")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid key format: %s", expiredKey)
	}

	return &PolicyInfo{
		ProviderID: parts[0],
		PolicyID:   parts[1],
	}, nil
}

// updateStats updates processing statistics
func (s *PolicyExpirationService) updateStats(processed bool, failed bool) {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	if processed {
		s.stats.TotalExpired++
		s.stats.LastProcessed = time.Now()
	}

	if failed {
		s.stats.FailedCommits++
	} else if processed {
		s.stats.SuccessfulCommits++
	}
}

// GetStats returns current processing statistics
func (s *PolicyExpirationService) GetStats() ExpirationStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	return *s.stats
}

// Health check for monitoring
func (s *PolicyExpirationService) HealthCheck() error {
	stats := s.GetStats()

	// Check if service has processed events recently (last 10 minutes)
	if time.Since(stats.LastProcessed) > 10*time.Minute && stats.TotalExpired > 0 {
		return fmt.Errorf("no expirations processed in last 10 minutes")
	}

	// Check failure rate (if more than 50% failures)
	if stats.TotalExpired > 0 {
		failureRate := float64(stats.FailedCommits) / float64(stats.TotalExpired)
		if failureRate > 0.5 {
			return fmt.Errorf("high failure rate: %.1f%%", failureRate*100)
		}
	}

	return nil
}

// ============================================================================
// USAGE EXAMPLE
// ============================================================================

/*
// Example usage in main application:

func main() {
    // Initialize Redis client
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // Enable Redis keyspace notifications (required once)
    redisClient.ConfigSet(context.Background(), "notify-keyspace-events", "Ex")

    // Initialize repositories and services
    basePolicyRepo := repository.NewBasePolicyRepository(db, redisClient)
    policyService := services.NewBasePolicyService(basePolicyRepo, dataSourceRepo, dataTierRepo)

    // Initialize expiration service
    expirationService := services.NewPolicyExpirationService(redisClient, policyService)

    // Start the expiration listener in a goroutine
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        if err := expirationService.StartListener(ctx); err != nil {
            log.Printf("Expiration service error: %v", err)
        }
    }()

    // Your application logic here...

    // Graceful shutdown
    expirationService.Stop()
}
*/
