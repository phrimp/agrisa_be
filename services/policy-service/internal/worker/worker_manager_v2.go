package worker

import (
	"context"
	"fmt"
	"log/slog"
	"policy-service/internal/database/redis"
	"policy-service/internal/models"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	goredis "github.com/redis/go-redis/v9"
)

// WorkerManagerV2 is the refactored worker manager with persistence and lifecycle management
type WorkerManagerV2 struct {
	// Pool and scheduler storage by policy ID
	pools      map[uuid.UUID]Pool
	schedulers map[uuid.UUID]*JobScheduler

	// Reverse lookup by name (for backward compatibility)
	poolsByName      map[string]Pool
	schedulersByName map[string]*JobScheduler

	// Pool context cancellation tracking
	poolCancels map[uuid.UUID]context.CancelFunc

	// Concurrency control
	mu sync.RWMutex

	// Context management
	managerCtx    context.Context
	managerCancel context.CancelFunc

	// Dependencies
	redisClient *redis.Client
	db          *sqlx.DB
	persistor   WorkerPersistor

	// Job handler registry
	jobHandlers map[string]func(map[string]any) error
	handlersMu  sync.RWMutex

	// Wait group for graceful shutdown
	wg *sync.WaitGroup
}

// NewWorkerManagerV2 creates a new worker manager with persistence support
func NewWorkerManagerV2(db *sqlx.DB, redisClient *redis.Client) *WorkerManagerV2 {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerManagerV2{
		pools:            make(map[uuid.UUID]Pool),
		schedulers:       make(map[uuid.UUID]*JobScheduler),
		poolsByName:      make(map[string]Pool),
		schedulersByName: make(map[string]*JobScheduler),
		poolCancels:      make(map[uuid.UUID]context.CancelFunc),
		managerCtx:       ctx,
		managerCancel:    cancel,
		redisClient:      redisClient,
		db:               db,
		persistor:        NewPostgresPersistor(db),
		jobHandlers:      make(map[string]func(map[string]any) error),
		wg:               new(sync.WaitGroup),
	}
}

// RegisterJobHandler registers a job handler function
func (m *WorkerManagerV2) RegisterJobHandler(jobType string, handler func(map[string]any) error) {
	m.handlersMu.Lock()
	defer m.handlersMu.Unlock()

	m.jobHandlers[jobType] = handler
	slog.Info("Registered job handler", "job_type", jobType)
}

// GetJobHandler retrieves a registered job handler
func (m *WorkerManagerV2) GetJobHandler(jobType string) (func(map[string]any) error, bool) {
	m.handlersMu.RLock()
	defer m.handlersMu.RUnlock()

	handler, exists := m.jobHandlers[jobType]
	return handler, exists
}

// CreateWorkerInfrastructure creates pool + scheduler for a registered policy
func (m *WorkerManagerV2) CreateWorkerInfrastructure(
	ctx context.Context,
	registeredPolicy *models.RegisteredPolicy,
	basePolicy *models.BasePolicy,
	basePolicyTrigger *models.BasePolicyTrigger,
) error {
	// Validate inputs first before accessing fields
	if registeredPolicy == nil || basePolicy == nil || basePolicyTrigger == nil {
		return fmt.Errorf("invalid parameters: all parameters must be non-nil")
	}

	slog.Info("Creating worker infrastructure",
		"policy_id", registeredPolicy.ID,
		"base_policy_id", basePolicy.ID)

	// Convert monitor interval to duration
	var monitorInterval time.Duration
	switch basePolicyTrigger.MonitorFrequencyUnit {
	case models.MonitorFrequencyHour:
		monitorInterval = time.Duration(basePolicyTrigger.MonitorInterval) * time.Hour
	case models.MonitorFrequencyDay:
		monitorInterval = time.Duration(basePolicyTrigger.MonitorInterval) * 24 * time.Hour
	case models.MonitorFrequencyWeek:
		monitorInterval = time.Duration(basePolicyTrigger.MonitorInterval) * 7 * 24 * time.Hour
	case models.MonitorFrequencyMonth:
		monitorInterval = time.Duration(basePolicyTrigger.MonitorInterval) * 30 * 24 * time.Hour
	default:
		return fmt.Errorf("unsupported monitor frequency unit: %s", basePolicyTrigger.MonitorFrequencyUnit)
	}

	// Create pool and scheduler in transaction
	err := m.persistor.WithTransaction(ctx, func(ctx context.Context, txPersistor WorkerPersistor) error {
		// 1. Create pool
		poolName := fmt.Sprintf("policy-%s-pool", registeredPolicy.ID)
		queueNameBase := fmt.Sprintf("policy-%s", registeredPolicy.ID)

		var goRedisClient *goredis.Client
		if m.redisClient != nil {
			goRedisClient = m.redisClient.GetClient()
		}

		pool := NewWorkingPool(
			5, // numWorkers - TODO: make configurable
			poolName,
			30*time.Minute, // jobTimeout - TODO: make configurable
			goRedisClient,
		)

		// Register job handler for farm monitoring data fetch
		handler, exists := m.GetJobHandler("fetch-farm-monitoring-data")
		if !exists {
			return fmt.Errorf("job handler not registered: fetch-farm-monitoring-data")
		}
		pool.RegisterJob("fetch-farm-monitoring-data", handler)

		poolState := &WorkerPoolState{
			PolicyID:      registeredPolicy.ID,
			PoolName:      poolName,
			QueueNameBase: queueNameBase,
			NumWorkers:    5,
			JobTimeout:    30 * time.Minute,
			PoolStatus:    PoolStatusCreated,
			CreatedAt:     time.Now(),
			Metadata:      map[string]any{"base_policy_id": basePolicy.ID.String()},
		}

		if err := txPersistor.CreatePoolState(ctx, poolState); err != nil {
			return fmt.Errorf("failed to create pool state: %w", err)
		}

		// 2. Create scheduler
		schedulerName := fmt.Sprintf("policy-%s-scheduler", registeredPolicy.ID)

		scheduler := NewJobScheduler(schedulerName, monitorInterval, pool)

		// TODO: Add jobs for each data source endpoint
		// This requires loading trigger conditions which reference data sources
		// For now, we'll create a placeholder job that can be populated later
		job := JobPayload{
			JobID: uuid.NewString(),
			Type:  "fetch-farm-monitoring-data",
			Params: map[string]any{
				"policy_id":      registeredPolicy.ID.String(),
				"base_policy_id": basePolicy.ID.String(),
				"farm_id":        registeredPolicy.FarmID.String(),
				"trigger_id":     basePolicyTrigger.ID.String(),
			},
			MaxRetries: 3,
		}
		scheduler.AddJob(job)

		schedulerState := &WorkerSchedulerState{
			PolicyID:             registeredPolicy.ID,
			SchedulerName:        schedulerName,
			MonitorInterval:      monitorInterval,
			MonitorFrequencyUnit: string(basePolicyTrigger.MonitorFrequencyUnit),
			SchedulerStatus:      SchedulerStatusCreated,
			CreatedAt:            time.Now(),
			Metadata:             map[string]any{"base_policy_id": basePolicy.ID.String()},
		}

		if err := txPersistor.CreateSchedulerState(ctx, schedulerState); err != nil {
			return fmt.Errorf("failed to create scheduler state: %w", err)
		}

		// 3. Store in memory
		m.mu.Lock()
		m.pools[registeredPolicy.ID] = pool
		m.poolsByName[poolName] = pool
		m.schedulers[registeredPolicy.ID] = scheduler
		m.schedulersByName[schedulerName] = scheduler
		m.mu.Unlock()

		slog.Info("Worker infrastructure created successfully",
			"policy_id", registeredPolicy.ID,
			"pool_name", poolName,
			"scheduler_name", schedulerName,
			"monitor_interval", monitorInterval)

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create worker infrastructure: %w", err)
	}

	return nil
}

// StartWorkerInfrastructure starts pool + scheduler for a policy
func (m *WorkerManagerV2) StartWorkerInfrastructure(ctx context.Context, policyID uuid.UUID) error {
	slog.Info("Starting worker infrastructure", "policy_id", policyID)

	m.mu.RLock()
	pool, poolExists := m.pools[policyID]
	scheduler, schedulerExists := m.schedulers[policyID]
	m.mu.RUnlock()

	if !poolExists || !schedulerExists {
		return fmt.Errorf("worker infrastructure not found for policy %s", policyID)
	}

	// Update statuses in transaction
	err := m.persistor.WithTransaction(ctx, func(ctx context.Context, txPersistor WorkerPersistor) error {
		if err := txPersistor.SetPoolStatus(ctx, policyID, PoolStatusActive); err != nil {
			return err
		}
		return txPersistor.SetSchedulerStatus(ctx, policyID, SchedulerStatusActive)
	})
	if err != nil {
		return fmt.Errorf("failed to update worker infrastructure status: %w", err)
	}

	// Start pool workers with dedicated context
	poolCtx, poolCancel := context.WithCancel(m.managerCtx)
	m.mu.Lock()
	m.poolCancels[policyID] = poolCancel
	m.mu.Unlock()

	m.wg.Add(1)
	go pool.Start(poolCtx, m.wg)

	// Start scheduler
	go scheduler.Run(m.managerCtx)

	slog.Info("Worker infrastructure started successfully", "policy_id", policyID)

	return nil
}

// StopWorkerInfrastructure stops pool + scheduler for a policy
func (m *WorkerManagerV2) StopWorkerInfrastructure(ctx context.Context, policyID uuid.UUID) error {
	slog.Info("Stopping worker infrastructure", "policy_id", policyID)

	m.mu.RLock()
	_, poolExists := m.pools[policyID]
	scheduler, schedulerExists := m.schedulers[policyID]
	poolCancel, cancelExists := m.poolCancels[policyID]
	m.mu.RUnlock()

	if !poolExists || !schedulerExists {
		return fmt.Errorf("worker infrastructure not found for policy %s", policyID)
	}

	// Stop pool by canceling its context
	if cancelExists {
		poolCancel()
		m.mu.Lock()
		delete(m.poolCancels, policyID)
		m.mu.Unlock()
	}

	// Stop scheduler (stops ticker)
	scheduler.Ticker.Stop()

	// Update statuses in transaction
	err := m.persistor.WithTransaction(ctx, func(ctx context.Context, txPersistor WorkerPersistor) error {
		if err := txPersistor.SetPoolStatus(ctx, policyID, PoolStatusStopped); err != nil {
			return err
		}
		return txPersistor.SetSchedulerStatus(ctx, policyID, SchedulerStatusStopped)
	})
	if err != nil {
		return fmt.Errorf("failed to update worker infrastructure status: %w", err)
	}

	slog.Info("Worker infrastructure stopped successfully", "policy_id", policyID)

	return nil
}

// ArchiveWorkerInfrastructure archives pool + scheduler for expired policy
func (m *WorkerManagerV2) ArchiveWorkerInfrastructure(ctx context.Context, policyID uuid.UUID) error {
	slog.Info("Archiving worker infrastructure", "policy_id", policyID)

	// First stop if not already stopped
	if err := m.StopWorkerInfrastructure(ctx, policyID); err != nil {
		slog.Warn("Failed to stop before archive (may already be stopped)", "error", err)
	}

	// Update statuses to archived
	err := m.persistor.WithTransaction(ctx, func(ctx context.Context, txPersistor WorkerPersistor) error {
		if err := txPersistor.SetPoolStatus(ctx, policyID, PoolStatusArchived); err != nil {
			return err
		}
		return txPersistor.SetSchedulerStatus(ctx, policyID, SchedulerStatusArchived)
	})
	if err != nil {
		return fmt.Errorf("failed to archive worker infrastructure: %w", err)
	}

	// Remove from in-memory maps
	m.mu.Lock()
	defer m.mu.Unlock()

	if pool, exists := m.pools[policyID]; exists {
		delete(m.poolsByName, pool.GetName())
		delete(m.pools, policyID)
	}

	if scheduler, exists := m.schedulers[policyID]; exists {
		delete(m.schedulersByName, scheduler.Name)
		delete(m.schedulers, policyID)
	}

	// Clean up cancel function if still exists
	if _, exists := m.poolCancels[policyID]; exists {
		delete(m.poolCancels, policyID)
	}

	slog.Info("Worker infrastructure archived successfully", "policy_id", policyID)

	return nil
}

// RecoverWorkerInfrastructure recovers all active worker infrastructure after restart
// Note: This method signature will need repository dependencies to be fully functional
func (m *WorkerManagerV2) RecoverWorkerInfrastructure(ctx context.Context) error {
	slog.Info("Recovering worker infrastructure from database")

	// Load all active policy IDs
	activePolicyIDs, err := m.persistor.LoadActiveWorkerInfrastructure(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active policies: %w", err)
	}

	slog.Info("Found active policies to recover", "count", len(activePolicyIDs))

	// TODO: This requires repositories to be injected
	// For each active policy, we need to:
	// 1. Load RegisteredPolicy from repository
	// 2. Load BasePolicy from repository
	// 3. Load BasePolicyTrigger from repository
	// 4. Call CreateWorkerInfrastructure
	// 5. Call StartWorkerInfrastructure
	//
	// This will be completed in Phase 5 when integrating with services

	slog.Info("Worker infrastructure recovery completed", "recovered_count", len(activePolicyIDs))

	return nil
}

// GetPool retrieves a pool by name (for backward compatibility)
func (m *WorkerManagerV2) GetPool(poolName string) (Pool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, exists := m.poolsByName[poolName]
	return pool, exists
}

// GetPoolByPolicyID retrieves a pool by policy ID
func (m *WorkerManagerV2) GetPoolByPolicyID(policyID uuid.UUID) (Pool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, exists := m.pools[policyID]
	return pool, exists
}

// GetSchedulerByPolicyID retrieves a scheduler by policy ID
func (m *WorkerManagerV2) GetSchedulerByPolicyID(policyID uuid.UUID) (*JobScheduler, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	scheduler, exists := m.schedulers[policyID]
	return scheduler, exists
}

// ManagerContext returns the manager's context
func (m *WorkerManagerV2) ManagerContext() context.Context {
	return m.managerCtx
}

// Shutdown gracefully shuts down all worker infrastructure
func (m *WorkerManagerV2) Shutdown() {
	slog.Info("Shutting down worker manager")

	m.mu.RLock()
	policyIDs := make([]uuid.UUID, 0, len(m.pools))
	for policyID := range m.pools {
		policyIDs = append(policyIDs, policyID)
	}
	m.mu.RUnlock()

	// Stop all infrastructure
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, policyID := range policyIDs {
		if err := m.StopWorkerInfrastructure(ctx, policyID); err != nil {
			slog.Error("Failed to stop worker infrastructure during shutdown",
				"policy_id", policyID,
				"error", err)
		}
	}

	// Cancel manager context
	m.managerCancel()

	// Wait for all goroutines
	m.wg.Wait()

	slog.Info("Worker manager shutdown complete")
}

// GetPersistor returns the persistor (for testing)
func (m *WorkerManagerV2) GetPersistor() WorkerPersistor {
	return m.persistor
}
