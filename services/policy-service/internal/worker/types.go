package worker

import (
	"time"

	"github.com/google/uuid"
)

// WorkerPoolStatus represents the lifecycle state of a worker pool
type WorkerPoolStatus string

const (
	PoolStatusCreated  WorkerPoolStatus = "created"  // Pool structure created, not started
	PoolStatusActive   WorkerPoolStatus = "active"   // Pool running and processing jobs
	PoolStatusStopped  WorkerPoolStatus = "stopped"  // Pool stopped gracefully
	PoolStatusArchived WorkerPoolStatus = "archived" // Pool archived, policy expired
)

// WorkerSchedulerStatus represents the lifecycle state of a scheduler
type WorkerSchedulerStatus string

const (
	SchedulerStatusCreated  WorkerSchedulerStatus = "created"
	SchedulerStatusActive   WorkerSchedulerStatus = "active"
	SchedulerStatusStopped  WorkerSchedulerStatus = "stopped"
	SchedulerStatusArchived WorkerSchedulerStatus = "archived"
)

// WorkerJobStatus represents the execution state of a job
type WorkerJobStatus string

const (
	JobStatusPending   WorkerJobStatus = "pending"
	JobStatusRunning   WorkerJobStatus = "running"
	JobStatusCompleted WorkerJobStatus = "completed"
	JobStatusFailed    WorkerJobStatus = "failed"
	JobStatusRetrying  WorkerJobStatus = "retrying"
)

// WorkerPoolState represents the persisted state of a worker pool
type WorkerPoolState struct {
	PolicyID      uuid.UUID        `db:"policy_id" json:"policy_id"`
	PoolName      string           `db:"pool_name" json:"pool_name"`
	QueueNameBase string           `db:"queue_name_base" json:"queue_name_base"`
	NumWorkers    int              `db:"num_workers" json:"num_workers"`
	JobTimeout    time.Duration    `db:"job_timeout" json:"job_timeout"`
	PoolStatus    WorkerPoolStatus `db:"pool_status" json:"pool_status"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	StartedAt     *time.Time       `db:"started_at" json:"started_at,omitempty"`
	StoppedAt     *time.Time       `db:"stopped_at" json:"stopped_at,omitempty"`
	LastJobAt     *time.Time       `db:"last_job_at" json:"last_job_at,omitempty"`
	Metadata      map[string]any   `db:"metadata" json:"metadata,omitempty"`
}

// WorkerSchedulerState represents the persisted state of a scheduler
type WorkerSchedulerState struct {
	PolicyID             uuid.UUID             `db:"policy_id" json:"policy_id"`
	SchedulerName        string                `db:"scheduler_name" json:"scheduler_name"`
	MonitorInterval      time.Duration         `db:"monitor_interval" json:"monitor_interval"`
	MonitorFrequencyUnit string                `db:"monitor_frequency_unit" json:"monitor_frequency_unit"`
	SchedulerStatus      WorkerSchedulerStatus `db:"scheduler_status" json:"scheduler_status"`
	CreatedAt            time.Time             `db:"created_at" json:"created_at"`
	StartedAt            *time.Time            `db:"started_at" json:"started_at,omitempty"`
	StoppedAt            *time.Time            `db:"stopped_at" json:"stopped_at,omitempty"`
	LastRunAt            *time.Time            `db:"last_run_at" json:"last_run_at,omitempty"`
	NextRunAt            *time.Time            `db:"next_run_at" json:"next_run_at,omitempty"`
	RunCount             int64                 `db:"run_count" json:"run_count"`
	Metadata             map[string]any        `db:"metadata" json:"metadata,omitempty"`
}

// WorkerJobExecution represents a job execution record
type WorkerJobExecution struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	PolicyID      uuid.UUID       `db:"policy_id" json:"policy_id"`
	JobID         string          `db:"job_id" json:"job_id"`
	JobType       string          `db:"job_type" json:"job_type"`
	Status        WorkerJobStatus `db:"status" json:"status"`
	RetryCount    int             `db:"retry_count" json:"retry_count"`
	MaxRetries    int             `db:"max_retries" json:"max_retries"`
	StartedAt     *time.Time      `db:"started_at" json:"started_at,omitempty"`
	CompletedAt   *time.Time      `db:"completed_at" json:"completed_at,omitempty"`
	ErrorMessage  *string         `db:"error_message" json:"error_message,omitempty"`
	ResultSummary map[string]any  `db:"result_summary" json:"result_summary,omitempty"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

// WorkerInfrastructureConfig contains configuration for creating worker infrastructure
type WorkerInfrastructureConfig struct {
	PolicyID             uuid.UUID
	BasePolicyID         uuid.UUID
	MonitorInterval      time.Duration
	MonitorFrequencyUnit string
	NumWorkers           int
	JobTimeout           time.Duration
	DataSourceEndpoints  []string
}

// IsValidPoolStatus checks if a pool status is valid
func IsValidPoolStatus(status WorkerPoolStatus) bool {
	switch status {
	case PoolStatusCreated, PoolStatusActive, PoolStatusStopped, PoolStatusArchived:
		return true
	default:
		return false
	}
}

// IsValidSchedulerStatus checks if a scheduler status is valid
func IsValidSchedulerStatus(status WorkerSchedulerStatus) bool {
	switch status {
	case SchedulerStatusCreated, SchedulerStatusActive, SchedulerStatusStopped, SchedulerStatusArchived:
		return true
	default:
		return false
	}
}

// IsValidJobStatus checks if a job status is valid
func IsValidJobStatus(status WorkerJobStatus) bool {
	switch status {
	case JobStatusPending, JobStatusRunning, JobStatusCompleted, JobStatusFailed, JobStatusRetrying:
		return true
	default:
		return false
	}
}
