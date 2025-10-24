package worker

import (
	"context"
	"log"
	"time"
)

type JobScheduler struct {
	Name   string
	Ticker *time.Ticker
	Jobs   []Job        // A slice of all jobs to run
	Pool   *WorkingPool // The pool to send work to
}

// NewJobScheduler creates a new scheduler.
func NewJobScheduler(name string, interval time.Duration, pool *WorkingPool) *JobScheduler {
	return &JobScheduler{
		Name:   name,
		Ticker: time.NewTicker(interval),
		Jobs:   make([]Job, 0),
		Pool:   pool,
	}
}

// example: AddJob(func () { Monitor(a,b) })
func (s *JobScheduler) AddJob(job Job) {
	s.Jobs = append(s.Jobs, job)
}

func (s *JobScheduler) Run(ctx context.Context) {
	log.Printf("[Scheduler %s] Running every %v\n", s.Name, s.Ticker)
	defer s.Ticker.Stop()

	for {
		select {
		case <-s.Ticker.C:
			log.Printf("[Scheduler %s] Ticker fired. Submitting %d jobs.\n", s.Name, len(s.Jobs))

			for _, job := range s.Jobs {
				s.Pool.SubmitJob(job)
			}

		case <-ctx.Done():
			// The manager signaled a global shutdown
			log.Printf("[Scheduler %s] Shutting down.\n", s.Name)
			return
		}
	}
}
