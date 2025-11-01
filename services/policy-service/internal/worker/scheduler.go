package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

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
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Scheduler panic", "scheduler_name", s.Name, "panic", r)
		}
	}()

	defer s.Ticker.Stop()

	for {
		select {
		case <-s.Ticker.C:
			s.mu.RLock()
			jobCount := len(s.Jobs)
			s.mu.RUnlock()
			slog.Info("...", "job_count", jobCount)
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

	newJobs := make([]JobPayload, len(jobsToRun))
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
			slog.Info("Job submitted successfully",
				"scheduler_name", s.Name,
				"job_id", job.JobID,
				"job_type", job.Type)
		}
		cancel()
		if !job.OneTime {
			newJobs = append(newJobs, job)
		}
	}
	s.mu.Lock()
	s.Jobs = newJobs
	s.mu.Unlock()
}
