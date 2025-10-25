package worker

import (
	"context"
	"log"
	"sync"
)

type WorkingPool struct {
	NumWorkers int
	jobChan    chan Job
}

func NewWorkingPool(numWorkers int, queueSize int) *WorkingPool {
	return &WorkingPool{
		NumWorkers: numWorkers,
		jobChan:    make(chan Job, queueSize),
	}
}

func (p *WorkingPool) SubmitJob(job Job) {
	p.jobChan <- job
}

func (p *WorkingPool) Start(ctx context.Context, managerWg *sync.WaitGroup) {
	defer managerWg.Done() // Tell manager we are done

	var workerWg sync.WaitGroup

	// Start all the workers
	for i := range p.NumWorkers {
		workerWg.Add(1)
		go p.worker(ctx, &workerWg, i+1)
	}

	// Wait for the manager to signal shutdown
	<-ctx.Done()

	// Start graceful shutdown
	log.Println("[WorkingPool] Shutdown signaled. Closing job channel.")
	close(p.jobChan) // Tell workers no more jobs are coming

	// Wait for all workers to finish their current job and exit
	workerWg.Wait()
	log.Println("[WorkingPool] All workers stopped.")
}

// worker is the internal goroutine for a single worker
func (p *WorkingPool) worker(ctx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	log.Printf("[WorkingPool-Worker %d] Started and waiting for jobs.\n", id)

	for {
		select {
		case job, ok := <-p.jobChan:
			if !ok {
				log.Printf("[WorkingPool-Worker %d] Job channel closed. Exiting.\n", id)
				return
			}

			// Got a job, execute it
			log.Printf("[WorkingPool-Worker %d] Picked up a job.\n", id)
			job() // Execute the job
			log.Printf("[WorkingPool-Worker %d] Finished job.\n", id)

		case <-ctx.Done():
			// Exit IMMEDIATELY, even if the job channel is not closed.
			log.Printf("[WorkingPool-Worker %d] Context canceled. Exiting.\n", id)
			return
		}
	}
}
