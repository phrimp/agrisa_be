package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"policy-service/internal/database/redis"
	"sync"
	"time"

	"github.com/google/uuid"
)

type JobSchedulerRecord struct {
	Name     string       `json:"name"`
	Interval string       `json:"interval"`
	PoolName string       `json:"pool_name"`
	Jobs     []JobPayload `json:"jobs"`
}

var JobSchedulerRecords JobSchedulerRecord

func LoadSchedules(filePath string) ([]JobSchedulerRecord, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read schedules file: %w", err)
	}

	var schedules []JobSchedulerRecord
	if err := json.Unmarshal(file, &schedules); err != nil {
		return nil, fmt.Errorf("could not parse schedules.json: %w", err)
	}

	return schedules, nil
}

func InitDefaultSchedulers(manager *WorkerManager, redisClient *redis.Client) error {
	slog.Info("Initializing default schedulers")

	// Try to load schedules from file
	schedulerRecords, err := LoadSchedules("schedules.json")
	if err != nil {
		slog.Warn("Could not load schedules from file, initializing defaults", "error", err)

		// Create schedules file if it doesn't exist
		if err := CreateSchedulerSaveFile(); err != nil {
			slog.Error("Failed to create schedules file", "error", err)
		}

		// Init all default schedulers
		dailyPool := NewWorkingPool(100, "DailyPool", 5*time.Minute, redisClient.GetClient())
		dailyScheduler := NewJobScheduler("DailyScheduler", 24*time.Hour, dailyPool)
		go dailyScheduler.Run(manager.ManagerContext())

		slog.Info("Default schedulers initialized")

		// Init Auto WriteToSaveFile goroutine
		go func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Auto-save goroutine panicked", "panic", r)
				}
			}()

			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()

			slog.Info("Auto-save goroutine started", "interval", "5m")

			for {
				select {
				case <-ticker.C:
					slog.Debug("Auto-saving scheduler state to file")
					if err := WriteToSaveFile(); err != nil {
						slog.Error("Failed to auto-save scheduler state", "error", err)
					}

				case <-ctx.Done():
					slog.Info("Auto-save goroutine shutting down")
					// Final save before shutdown
					if err := WriteToSaveFile(); err != nil {
						slog.Error("Failed to save scheduler state on shutdown", "error", err)
					}
					return
				}
			}
		}(manager.ManagerContext())

		return nil
	}

	// Load schedulers from file
	slog.Info("Loading schedulers from file", "count", len(schedulerRecords))

	for _, record := range schedulerRecords {
		pool, exists := manager.GetPool(record.PoolName)
		if !exists {
			slog.Error("Pool not found for scheduler, skipping",
				"scheduler_name", record.Name,
				"pool_name", record.PoolName)
			continue
		}

		interval, err := time.ParseDuration(record.Interval)
		if err != nil {
			slog.Error("Invalid interval for scheduler, skipping",
				"scheduler_name", record.Name,
				"interval", record.Interval,
				"error", err)
			continue
		}

		sched := NewJobScheduler(record.Name, interval, pool)

		for _, job := range record.Jobs {
			sched.AddJob(job)
		}

		slog.Info("Scheduler loaded from file",
			"scheduler_name", record.Name,
			"interval", interval,
			"job_count", len(record.Jobs))

		go sched.Run(manager.ManagerContext())
	}

	slog.Info("All schedulers initialized successfully", "count", len(schedulerRecords))
	return nil
}

// CreateSchedulerSaveFile creates the schedules.json file if it doesn't exist
func CreateSchedulerSaveFile() error {
	fileName := "schedules.json"

	// Check if file already exists
	if _, err := os.Stat(fileName); err == nil {
		slog.Info("Schedules file already exists", "file", fileName)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check file existence: %w", err)
	}

	// Create an empty schedules array as default
	defaultSchedules := []JobSchedulerRecord{}

	// Marshal to JSON with pretty formatting
	data, err := json.MarshalIndent(defaultSchedules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal default schedules: %w", err)
	}

	// Write to file with proper permissions
	if err := os.WriteFile(fileName, data, 0644); err != nil {
		return fmt.Errorf("failed to create schedules file: %w", err)
	}

	slog.Info("Successfully created schedules file", "file", fileName)
	return nil
}

// WriteToSaveFile writes the current JobSchedulerRecords to schedules.json
func WriteToSaveFile() error {
	fileName := "schedules.json"

	slog.Info("Writing schedules to file", "file", fileName)

	// Marshal the current scheduler records to JSON with pretty formatting
	data, err := json.MarshalIndent(JobSchedulerRecords, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal scheduler records", "error", err)
		return fmt.Errorf("failed to marshal scheduler records: %w", err)
	}

	// Write to file with proper permissions
	if err := os.WriteFile(fileName, data, 0644); err != nil {
		slog.Error("Failed to write schedules file", "file", fileName, "error", err)
		return fmt.Errorf("failed to write schedules file: %w", err)
	}

	slog.Info("Successfully wrote schedules to file",
		"file", fileName,
		"scheduler_name", JobSchedulerRecords.Name,
		"job_count", len(JobSchedulerRecords.Jobs))
	return nil
}

// WriteSchedulersToFile writes a slice of scheduler records to schedules.json
func WriteSchedulersToFile(schedulers []JobSchedulerRecord) error {
	fileName := "schedules.json"

	slog.Info("Writing scheduler records to file", "file", fileName, "count", len(schedulers))

	// Marshal to JSON with pretty formatting
	data, err := json.MarshalIndent(schedulers, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal scheduler records", "error", err)
		return fmt.Errorf("failed to marshal scheduler records: %w", err)
	}

	// Write to file with proper permissions
	if err := os.WriteFile(fileName, data, 0644); err != nil {
		slog.Error("Failed to write schedules file", "file", fileName, "error", err)
		return fmt.Errorf("failed to write schedules file: %w", err)
	}

	slog.Info("Successfully wrote scheduler records to file",
		"file", fileName,
		"count", len(schedulers))
	return nil
}

// AddSchedulerToFile adds a new scheduler record to the schedules.json file
func AddSchedulerToFile(record JobSchedulerRecord) error {
	// Load existing schedules
	schedules, err := LoadSchedules("schedules.json")
	if err != nil {
		// If file doesn't exist, create new slice
		if os.IsNotExist(err) {
			schedules = []JobSchedulerRecord{}
		} else {
			return fmt.Errorf("failed to load existing schedules: %w", err)
		}
	}

	// Check if scheduler with same name already exists
	for i, existing := range schedules {
		if existing.Name == record.Name {
			slog.Info("Scheduler already exists, updating", "scheduler_name", record.Name)
			schedules[i] = record
			return WriteSchedulersToFile(schedules)
		}
	}

	// Add new scheduler
	schedules = append(schedules, record)
	slog.Info("Adding new scheduler to file", "scheduler_name", record.Name)

	return WriteSchedulersToFile(schedules)
}

// RemoveSchedulerFromFile removes a scheduler record from schedules.json by name
func RemoveSchedulerFromFile(schedulerName string) error {
	// Load existing schedules
	schedules, err := LoadSchedules("schedules.json")
	if err != nil {
		return fmt.Errorf("failed to load existing schedules: %w", err)
	}

	// Find and remove scheduler
	found := false
	newSchedules := make([]JobSchedulerRecord, 0, len(schedules))
	for _, existing := range schedules {
		if existing.Name != schedulerName {
			newSchedules = append(newSchedules, existing)
		} else {
			found = true
			slog.Info("Removing scheduler from file", "scheduler_name", schedulerName)
		}
	}

	if !found {
		return fmt.Errorf("scheduler not found: %s", schedulerName)
	}

	return WriteSchedulersToFile(newSchedules)
}

// JobScheduler runs a list of jobs on a fixed schedule.
type JobScheduler struct {
	Name   string
	Ticker *time.Ticker
	Jobs   []JobPayload // <-- Uses JobPayload
	Pool   Pool         // <-- Uses Pool interface
	mu     sync.RWMutex
}

func NewJobScheduler(name string, interval time.Duration, pool Pool) *JobScheduler {
	return &JobScheduler{
		Name:   name,
		Ticker: time.NewTicker(interval),
		Jobs:   make([]JobPayload, 0),
		Pool:   pool,
	}
}

func (s *JobScheduler) AddJob(job JobPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Jobs = append(s.Jobs, job)
}

func (s *JobScheduler) Run(ctx context.Context) {
	slog.Info("Job scheduler starting", "scheduler_name", s.Name, "job_count", len(s.Jobs))
	defer s.Ticker.Stop()

	for {
		select {
		case <-s.Ticker.C:
			slog.Info("Scheduler ticker fired, submitting jobs", "scheduler_name", s.Name, "job_count", len(s.Jobs))
			s.submitJobs(ctx)

		case <-ctx.Done():
			slog.Info("Scheduler shutting down", "scheduler_name", s.Name)
			return
		}
	}
}

func (s *JobScheduler) submitJobs(ctx context.Context) {
	s.mu.RLock()
	jobsToRun := make([]JobPayload, len(s.Jobs))
	copy(jobsToRun, s.Jobs)
	s.mu.RUnlock()

	for _, job := range jobsToRun {
		job.JobID = uuid.NewString()
		job.RetryCount = 0

		// Use a short timeout for the submit itself
		submitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if err := s.Pool.SubmitJob(submitCtx, job); err != nil {
			slog.Error("Failed to submit job to pool",
				"scheduler_name", s.Name,
				"job_id", job.JobID,
				"job_type", job.Type,
				"error", err)
		} else {
			slog.Debug("Job submitted successfully",
				"scheduler_name", s.Name,
				"job_id", job.JobID,
				"job_type", job.Type)
		}
		cancel()
	}
}
