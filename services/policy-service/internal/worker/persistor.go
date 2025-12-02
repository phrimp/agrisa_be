package worker

import (
	"context"

	"github.com/google/uuid"
)

// WorkerPersistor handles persistence of worker state to database
type WorkerPersistor interface {
	// Pool State Management
	CreatePoolState(ctx context.Context, state *WorkerPoolState) error
	UpdatePoolState(ctx context.Context, state *WorkerPoolState) error
	GetPoolStateByPolicyID(ctx context.Context, policyID uuid.UUID) (*WorkerPoolState, error)
	SetPoolStatus(ctx context.Context, policyID uuid.UUID, status WorkerPoolStatus) error

	// Scheduler State Management
	CreateSchedulerState(ctx context.Context, state *WorkerSchedulerState) error
	UpdateSchedulerState(ctx context.Context, state *WorkerSchedulerState) error
	GetSchedulerStateByPolicyID(ctx context.Context, policyID uuid.UUID) (*WorkerSchedulerState, error)
	SetSchedulerStatus(ctx context.Context, policyID uuid.UUID, status WorkerSchedulerStatus) error

	// Job Execution Tracking
	CreateJobExecution(ctx context.Context, execution *WorkerJobExecution) error
	UpdateJobExecution(ctx context.Context, execution *WorkerJobExecution) error
	GetJobExecutionsByPolicyID(ctx context.Context, policyID uuid.UUID, limit int) ([]*WorkerJobExecution, error)

	// Disaster Recovery
	LoadActiveWorkerInfrastructure(ctx context.Context) ([]uuid.UUID, error)

	// Cleanup
	DeleteWorkerInfrastructure(ctx context.Context, policyID uuid.UUID) error

	// Transaction Support
	WithTransaction(ctx context.Context, fn func(context.Context, WorkerPersistor) error) error
}
