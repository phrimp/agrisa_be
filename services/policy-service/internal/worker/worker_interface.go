package worker

import (
	"context"
	"sync"
)

type (
	Job    func(ctx context.Context) error
	Worker interface {
		Run(ctx context.Context, wg *sync.WaitGroup)
	}
)

type JobPayload struct {
	JobID      string         `json:"job_id"`
	Type       string         `json:"type"`
	Params     map[string]any `json:"params"`
	MaxRetries int            `json:"max_retries"`
	RetryCount int            `json:"retry_count"`
	OneTime    bool           `json:"one_time"`
	RunNow     bool           `json:"run_now"`
}

type Pool interface {
	Start(ctx context.Context, managerWg *sync.WaitGroup)

	SubmitJob(ctx context.Context, job JobPayload) error

	RegisterJob(
		jobType string,
		jobFunc func(params map[string]any) error,
	)

	GetName() string
}
