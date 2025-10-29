package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	fmt.Printf("[Scheduler %s] Running.\n", s.Name)
	defer s.Ticker.Stop()

	for {
		select {
		case <-s.Ticker.C:
			fmt.Printf("[Scheduler %s] Ticker fired. Submitting jobs.\n", s.Name)
			s.submitJobs(ctx)

		case <-ctx.Done():
			// The manager signaled a global shutdown
			fmt.Printf("[Scheduler %s] Shutting down.\n", s.Name)
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
			fmt.Printf("[Scheduler %s] FAILED to submit job: %v\n", s.Name, err)
		}
		cancel()
	}
}
