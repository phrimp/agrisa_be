package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type WorkingPool struct {
	NumWorkers          int
	QueueName           string // e.g., "queue:general:pending"
	RunningQueueName    string // e.g., "queue:general:running"
	DeadLetterQueueName string // e.g., "queue:general:dlq"
	JobTimeout          time.Duration
	RedisClient         *redis.Client
	dispatcher          map[string]func(map[string]any) error
}

func NewWorkingPool(
	numWorkers int,
	queueNameBase string, // e.g., "queue:general"
	jobTimeout time.Duration,
	redisClient *redis.Client,
) *WorkingPool {
	return &WorkingPool{
		NumWorkers:          numWorkers,
		QueueName:           queueNameBase + ":pending",
		RunningQueueName:    queueNameBase + ":running",
		DeadLetterQueueName: queueNameBase + ":dlq",
		JobTimeout:          jobTimeout,
		RedisClient:         redisClient,
		dispatcher:          make(map[string]func(map[string]any) error),
	}
}

func (p *WorkingPool) RegisterJob(
	jobType string,
	jobFunc func(params map[string]any) error,
) {
	p.dispatcher[jobType] = jobFunc
}

func (p *WorkingPool) SubmitJob(ctx context.Context, job JobPayload) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return p.RedisClient.LPush(ctx, p.QueueName, payload).Err()
}

func (p *WorkingPool) Start(ctx context.Context, managerWg *sync.WaitGroup) {
	defer managerWg.Done()

	slog.Info("Working pool starting",
		"queue_name", p.QueueName,
		"num_workers", p.NumWorkers,
		"job_timeout", p.JobTimeout)

	// On start, move any "stale" jobs from the "running"
	// queue (from a previous crash) back to "pending".
	p.requeueStaleJobs(ctx)

	var workerWg sync.WaitGroup
	for i := 0; i < p.NumWorkers; i++ {
		workerWg.Add(1)
		go p.worker(ctx, &workerWg, i+1)
	}

	<-ctx.Done()
	workerWg.Wait()
	slog.Info("Working pool stopped, all workers exited", "queue_name", p.QueueName)
}

// worker is the main loop for a single worker goroutine.
func (p *WorkingPool) worker(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	slog.Info("Worker started", "worker_id", id, "queue_name", p.QueueName)

	for {
		// Atomically move a job from "pending" to "running".
		// This blocks until a job is available or 5s pass.
		jobPayload, err := p.RedisClient.BRPopLPush(
			ctx,
			p.QueueName,
			p.RunningQueueName,
			5*time.Second,
		).Result()

		if err == redis.Nil {
			// No job, just a timeout. Check for shutdown.
		} else if err != nil && err != context.Canceled {
			slog.Error("Redis error while fetching job",
				"worker_id", id,
				"queue_name", p.QueueName,
				"error", err)
			time.Sleep(1 * time.Second)
		} else if err == context.Canceled {
			// Context was canceled, stop trying to get jobs.
		} else if jobPayload != "" {
			// SUCCESS: We got a job. It is now safe in the "running" queue.
			slog.Debug("Worker received job", "worker_id", id, "queue_name", p.QueueName)

			jobErr := p.dispatchJob(ctx, jobPayload, id)

			// We MUST remove the job from the "running" queue,
			// whether it succeeded or failed.
			p.handleJobResult(ctx, jobPayload, jobErr, id)
		}

		// Check for shutdown signal
		if ctx.Err() != nil {
			slog.Info("Worker shutting down", "worker_id", id, "queue_name", p.QueueName)
			return
		}
	}
}

// dispatchJob runs a single job with panic recovery and timeouts.
func (p *WorkingPool) dispatchJob(ctx context.Context, payload string, workerID int) (jobErr error) {
	defer func() {
		if r := recover(); r != nil {
			jobErr = fmt.Errorf("panic recovered: %v", r)
			slog.Error("Job panic recovered",
				"worker_id", workerID,
				"panic", r)
		}
	}()

	var jobData JobPayload
	if err := json.Unmarshal([]byte(payload), &jobData); err != nil {
		slog.Error("Failed to unmarshal job payload",
			"worker_id", workerID,
			"error", err)
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	jobFunc, exists := p.dispatcher[jobData.Type]
	if !exists {
		slog.Error("Unknown job type",
			"worker_id", workerID,
			"job_id", jobData.JobID,
			"job_type", jobData.Type)
		return fmt.Errorf("unknown job type: %s", jobData.Type)
	}

	slog.Info("Executing job",
		"worker_id", workerID,
		"job_id", jobData.JobID,
		"job_type", jobData.Type,
		"retry_count", jobData.RetryCount,
		"max_retries", jobData.MaxRetries)

	jobCtx, cancel := context.WithTimeout(ctx, p.JobTimeout)
	defer cancel()

	done := make(chan error, 1) // Buffered channel
	go func() {
		done <- jobFunc(jobData.Params)
	}()

	select {
	case err := <-done:
		// Job finished, pass the error (or nil) up.
		if err != nil {
			slog.Error("Job execution failed",
				"worker_id", workerID,
				"job_id", jobData.JobID,
				"job_type", jobData.Type,
				"error", err)
		} else {
			slog.Info("Job execution completed successfully",
				"worker_id", workerID,
				"job_id", jobData.JobID,
				"job_type", jobData.Type)
		}
		return err
	case <-jobCtx.Done():
		// Job timed out.
		slog.Error("Job timed out",
			"worker_id", workerID,
			"job_id", jobData.JobID,
			"job_type", jobData.Type,
			"timeout", p.JobTimeout)
		return fmt.Errorf("job timed out after %v", p.JobTimeout)
	case <-ctx.Done():
		// Global shutdown signaled *during* job execution.
		slog.Warn("Job cancelled by global shutdown",
			"worker_id", workerID,
			"job_id", jobData.JobID,
			"job_type", jobData.Type)
		return fmt.Errorf("job cancelled by global shutdown")
	}
}

// handleJobResult cleans up a job from the "running" queue
// and retries or moves it to the DLQ if it failed.
func (p *WorkingPool) handleJobResult(
	ctx context.Context,
	jobPayload string,
	jobErr error,
	workerID int,
) {
	// ALWAYS remove the job from the "running" queue.
	// We use LRem to remove the specific job payload.
	if err := p.RedisClient.LRem(ctx, p.RunningQueueName, 1, jobPayload).Err(); err != nil {
		slog.Error("CRITICAL: Failed to remove job from running queue",
			"worker_id", workerID,
			"queue_name", p.RunningQueueName,
			"error", err)
		// If this fails, we're in big trouble.
	}

	if jobErr == nil {
		return // Success! Already logged in dispatchJob
	}

	// --- Job Failed. Handle Retry/DLQ ---
	var jobData JobPayload
	if err := json.Unmarshal([]byte(jobPayload), &jobData); err != nil {
		slog.Error("CRITICAL: Failed to unmarshal failed job, dropping it",
			"worker_id", workerID,
			"error", err)
		return
	}

	if jobData.RetryCount < jobData.MaxRetries {
		jobData.RetryCount++
		newPayload, _ := json.Marshal(jobData)
		slog.Info("Retrying job",
			"worker_id", workerID,
			"job_id", jobData.JobID,
			"job_type", jobData.Type,
			"retry_count", jobData.RetryCount,
			"max_retries", jobData.MaxRetries)

		err := p.RedisClient.LPush(ctx, p.QueueName, newPayload).Err()
		if err != nil {
			slog.Error("CRITICAL: Failed to requeue job for retry",
				"worker_id", workerID,
				"job_id", jobData.JobID,
				"job_type", jobData.Type,
				"error", err)
		}
	} else {
		// Max retries hit: Move to Dead-Letter Queue
		slog.Warn("Job exceeded max retries, moving to DLQ",
			"worker_id", workerID,
			"job_id", jobData.JobID,
			"job_type", jobData.Type,
			"retry_count", jobData.RetryCount,
			"dlq", p.DeadLetterQueueName)
		err := p.RedisClient.LPush(ctx, p.DeadLetterQueueName, jobPayload).Err()
		if err != nil {
			slog.Error("CRITICAL: Failed to move job to DLQ",
				"worker_id", workerID,
				"job_id", jobData.JobID,
				"job_type", jobData.Type,
				"dlq", p.DeadLetterQueueName,
				"error", err)
		}
	}
}

// requeueStaleJobs moves any jobs from "running" back to "pending"
// on startup. This handles jobs that were lost during a crash.
func (p *WorkingPool) requeueStaleJobs(ctx context.Context) {
	requeueCount := 0
	for {
		// Atomically move a job from "running" to "pending".
		// LPopLPush is the non-blocking version.
		jobPayload, err := p.RedisClient.RPopLPush(ctx, p.RunningQueueName, p.QueueName).Result()
		if err == redis.Nil {
			// No more stale jobs. We are done.
			if requeueCount > 0 {
				slog.Info("Finished requeuing stale jobs",
					"queue_name", p.QueueName,
					"requeued_count", requeueCount)
			} else {
				slog.Debug("No stale jobs found", "queue_name", p.QueueName)
			}
			return
		}
		if err != nil {
			slog.Error("CRITICAL: Could not requeue stale job",
				"queue_name", p.QueueName,
				"error", err)
			return
		}
		requeueCount++
		slog.Info("Requeued stale job",
			"queue_name", p.QueueName,
			"job_payload_length", len(jobPayload))
	}
}
