package worker

import (
	"context"
	"encoding/json"
	"fmt"
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
	fmt.Printf("[Pool %s] All workers stopped.\n", p.QueueName)
}

// worker is the main loop for a single worker goroutine.
func (p *WorkingPool) worker(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	fmt.Printf("[Worker %d] Started. Waiting for jobs on %s.\n", id, p.QueueName)

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
			fmt.Printf("[Worker %d] Redis error: %v. Retrying in 1s.\n", id, err)
			time.Sleep(1 * time.Second)
		} else if err == context.Canceled {
			// Context was canceled, stop trying to get jobs.
		} else if jobPayload != "" {
			// SUCCESS: We got a job. It is now safe in the "running" queue.

			jobErr := p.dispatchJob(ctx, jobPayload, id)

			// We MUST remove the job from the "running" queue,
			// whether it succeeded or failed.
			p.handleJobResult(ctx, jobPayload, jobErr, id)
		}

		// Check for shutdown signal
		if ctx.Err() != nil {
			fmt.Printf("[Worker %d] Shutdown signaled. Exiting.\n", id)
			return
		}
	}
}

// dispatchJob runs a single job with panic recovery and timeouts.
func (p *WorkingPool) dispatchJob(ctx context.Context, payload string, workerID int) (jobErr error) {
	defer func() {
		if r := recover(); r != nil {
			jobErr = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	var jobData JobPayload
	if err := json.Unmarshal([]byte(payload), &jobData); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	jobFunc, exists := p.dispatcher[jobData.Type]
	if !exists {
		return fmt.Errorf("unknown job type: %s", jobData.Type)
	}

	jobCtx, cancel := context.WithTimeout(ctx, p.JobTimeout)
	defer cancel()

	done := make(chan error, 1) // Buffered channel
	go func() {
		done <- jobFunc(jobData.Params)
	}()

	select {
	case err := <-done:
		// Job finished, pass the error (or nil) up.
		return err
	case <-jobCtx.Done():
		// Job timed out.
		return fmt.Errorf("job timed out after %v", p.JobTimeout)
	case <-ctx.Done():
		// Global shutdown signaled *during* job execution.
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
		fmt.Printf("[Worker %d] CRITICAL: Failed to remove job from running queue: %v\n", workerID, err)
		// If this fails, we're in big trouble.
	}

	if jobErr == nil {
		fmt.Printf("[Worker %d] Job completed successfully.\n", workerID)
		return // Success!
	}

	// --- Job Failed. Handle Retry/DLQ ---
	fmt.Printf("[Worker %d] Job failed: %v\n", workerID, jobErr)

	var jobData JobPayload
	if err := json.Unmarshal([]byte(jobPayload), &jobData); err != nil {
		fmt.Printf("[Worker %d] CRITICAL: Failed to unmarshal failed job. Dropping: %v\n", workerID, err)
		return
	}

	if jobData.RetryCount < jobData.MaxRetries {
		jobData.RetryCount++
		newPayload, _ := json.Marshal(jobData)
		fmt.Printf("[Worker %d] Retrying job (attempt %d/%d)...\n",
			workerID, jobData.RetryCount, jobData.MaxRetries)

		err := p.RedisClient.LPush(ctx, p.QueueName, newPayload).Err()
		if err != nil {
			fmt.Printf("[Worker %d] CRITICAL: Failed to requeue job: %v\n", workerID, err)
		}
	} else {
		// Max retries hit: Move to Dead-Letter Queue
		fmt.Printf("[Worker %d] Job failed max retries. Moving to DLQ.\n", workerID)
		err := p.RedisClient.LPush(ctx, p.DeadLetterQueueName, jobPayload).Err()
		if err != nil {
			fmt.Printf("[Worker %d] CRITICAL: Failed to move job to DLQ: %v\n", workerID, err)
		}
	}
}

// requeueStaleJobs moves any jobs from "running" back to "pending"
// on startup. This handles jobs that were lost during a crash.
func (p *WorkingPool) requeueStaleJobs(ctx context.Context) {
	for {
		// Atomically move a job from "running" to "pending".
		// LPopLPush is the non-blocking version.
		jobPayload, err := p.RedisClient.RPopLPush(ctx, p.RunningQueueName, p.QueueName).Result()
		if err == redis.Nil {
			// No more stale jobs. We are done.
			fmt.Printf("[Pool %s] No stale jobs found.\n", p.QueueName)
			return
		}
		if err != nil {
			fmt.Printf("[Pool %s] CRITICAL: Could not requeue stale job: %v\n", p.QueueName, err)
			return
		}
		fmt.Printf("[Pool %s] Requeued stale job: %s\n", p.QueueName, jobPayload)
	}
}
