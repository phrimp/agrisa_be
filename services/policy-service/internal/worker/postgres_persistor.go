package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PostgresPersistor implements WorkerPersistor for PostgreSQL
type PostgresPersistor struct {
	db *sqlx.DB
}

// NewPostgresPersistor creates a new PostgreSQL persistor
func NewPostgresPersistor(db *sqlx.DB) *PostgresPersistor {
	return &PostgresPersistor{db: db}
}

// Pool State Management

// CreatePoolState creates a new pool state record
func (p *PostgresPersistor) CreatePoolState(ctx context.Context, state *WorkerPoolState) error {
	query := `
		INSERT INTO worker_pool_state (
			policy_id, pool_name, queue_name_base, num_workers, job_timeout, pool_status,
			created_at, started_at, stopped_at, last_job_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, err := json.Marshal(state.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.db.ExecContext(ctx, query,
		state.PolicyID,
		state.PoolName,
		state.QueueNameBase,
		state.NumWorkers,
		state.JobTimeout,
		state.PoolStatus,
		state.CreatedAt,
		state.StartedAt,
		state.StoppedAt,
		state.LastJobAt,
		metadataJSON,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return fmt.Errorf("pool already exists for policy %s: %w", state.PolicyID, err)
		}
		return fmt.Errorf("failed to create pool state: %w", err)
	}

	return nil
}

// UpdatePoolState updates an existing pool state record
func (p *PostgresPersistor) UpdatePoolState(ctx context.Context, state *WorkerPoolState) error {
	query := `
		UPDATE worker_pool_state SET
			pool_status = $2,
			started_at = $3,
			stopped_at = $4,
			last_job_at = $5,
			metadata = $6
		WHERE policy_id = $1
	`

	metadataJSON, err := json.Marshal(state.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := p.db.ExecContext(ctx, query,
		state.PolicyID,
		state.PoolStatus,
		state.StartedAt,
		state.StoppedAt,
		state.LastJobAt,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update pool state: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("pool not found for policy %s", state.PolicyID)
	}

	return nil
}

// GetPoolStateByPolicyID retrieves a pool state by policy ID
func (p *PostgresPersistor) GetPoolStateByPolicyID(ctx context.Context, policyID uuid.UUID) (*WorkerPoolState, error) {
	query := `
		SELECT policy_id, pool_name, queue_name_base, num_workers, job_timeout, pool_status,
		       created_at, started_at, stopped_at, last_job_at, metadata
		FROM worker_pool_state
		WHERE policy_id = $1
	`

	var state WorkerPoolState
	var metadataJSON []byte
	var jobTimeout int64 // PostgreSQL interval stored as nanoseconds

	err := p.db.QueryRowContext(ctx, query, policyID).Scan(
		&state.PolicyID,
		&state.PoolName,
		&state.QueueNameBase,
		&state.NumWorkers,
		&jobTimeout,
		&state.PoolStatus,
		&state.CreatedAt,
		&state.StartedAt,
		&state.StoppedAt,
		&state.LastJobAt,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pool state not found for policy %s", policyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pool state: %w", err)
	}

	// Convert PostgreSQL interval to time.Duration
	state.JobTimeout = nanosecondsToDuration(jobTimeout)

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &state.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &state, nil
}

// SetPoolStatus updates the pool status and related timestamps
func (p *PostgresPersistor) SetPoolStatus(ctx context.Context, policyID uuid.UUID, status WorkerPoolStatus) error {
	query := `
		UPDATE worker_pool_state SET
			pool_status = $2,
			started_at = CASE WHEN $2 = 'active' THEN NOW() ELSE started_at END,
			stopped_at = CASE WHEN $2 IN ('stopped', 'archived') THEN NOW() ELSE stopped_at END
		WHERE policy_id = $1
	`

	result, err := p.db.ExecContext(ctx, query, policyID, status)
	if err != nil {
		return fmt.Errorf("failed to set pool status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("pool not found for policy %s", policyID)
	}

	return nil
}

// Scheduler State Management

// CreateSchedulerState creates a new scheduler state record
func (p *PostgresPersistor) CreateSchedulerState(ctx context.Context, state *WorkerSchedulerState) error {
	query := `
		INSERT INTO worker_scheduler_state (
			policy_id, scheduler_name, monitor_interval, monitor_frequency_unit,
			scheduler_status, created_at, started_at, stopped_at, last_run_at, next_run_at,
			run_count, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	metadataJSON, err := json.Marshal(state.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.db.ExecContext(ctx, query,
		state.PolicyID,
		state.SchedulerName,
		state.MonitorInterval,
		state.MonitorFrequencyUnit,
		state.SchedulerStatus,
		state.CreatedAt,
		state.StartedAt,
		state.StoppedAt,
		state.LastRunAt,
		state.NextRunAt,
		state.RunCount,
		metadataJSON,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("scheduler already exists for policy %s: %w", state.PolicyID, err)
		}
		return fmt.Errorf("failed to create scheduler state: %w", err)
	}

	return nil
}

// UpdateSchedulerState updates an existing scheduler state record
func (p *PostgresPersistor) UpdateSchedulerState(ctx context.Context, state *WorkerSchedulerState) error {
	query := `
		UPDATE worker_scheduler_state SET
			scheduler_status = $2,
			started_at = $3,
			stopped_at = $4,
			last_run_at = $5,
			next_run_at = $6,
			run_count = $7,
			metadata = $8
		WHERE policy_id = $1
	`

	metadataJSON, err := json.Marshal(state.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := p.db.ExecContext(ctx, query,
		state.PolicyID,
		state.SchedulerStatus,
		state.StartedAt,
		state.StoppedAt,
		state.LastRunAt,
		state.NextRunAt,
		state.RunCount,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update scheduler state: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("scheduler not found for policy %s", state.PolicyID)
	}

	return nil
}

// GetSchedulerStateByPolicyID retrieves a scheduler state by policy ID
func (p *PostgresPersistor) GetSchedulerStateByPolicyID(ctx context.Context, policyID uuid.UUID) (*WorkerSchedulerState, error) {
	query := `
		SELECT policy_id, scheduler_name, monitor_interval, monitor_frequency_unit,
		       scheduler_status, created_at, started_at, stopped_at, last_run_at, next_run_at,
		       run_count, metadata
		FROM worker_scheduler_state
		WHERE policy_id = $1
	`

	var state WorkerSchedulerState
	var metadataJSON []byte
	var monitorInterval int64 // PostgreSQL interval stored as nanoseconds

	err := p.db.QueryRowContext(ctx, query, policyID).Scan(
		&state.PolicyID,
		&state.SchedulerName,
		&monitorInterval,
		&state.MonitorFrequencyUnit,
		&state.SchedulerStatus,
		&state.CreatedAt,
		&state.StartedAt,
		&state.StoppedAt,
		&state.LastRunAt,
		&state.NextRunAt,
		&state.RunCount,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scheduler state not found for policy %s", policyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler state: %w", err)
	}

	// Convert PostgreSQL interval to time.Duration
	state.MonitorInterval = nanosecondsToDuration(monitorInterval)

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &state.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &state, nil
}

// SetSchedulerStatus updates the scheduler status and related timestamps
func (p *PostgresPersistor) SetSchedulerStatus(ctx context.Context, policyID uuid.UUID, status WorkerSchedulerStatus) error {
	query := `
		UPDATE worker_scheduler_state SET
			scheduler_status = $2,
			started_at = CASE WHEN $2 = 'active' THEN NOW() ELSE started_at END,
			stopped_at = CASE WHEN $2 IN ('stopped', 'archived') THEN NOW() ELSE stopped_at END
		WHERE policy_id = $1
	`

	result, err := p.db.ExecContext(ctx, query, policyID, status)
	if err != nil {
		return fmt.Errorf("failed to set scheduler status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("scheduler not found for policy %s", policyID)
	}

	return nil
}

// Job Execution Tracking

// CreateJobExecution creates a new job execution record
func (p *PostgresPersistor) CreateJobExecution(ctx context.Context, execution *WorkerJobExecution) error {
	query := `
		INSERT INTO worker_job_execution (
			id, policy_id, job_id, job_type, status, retry_count, max_retries,
			started_at, completed_at, error_message, result_summary, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	resultJSON, err := json.Marshal(execution.ResultSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal result_summary: %w", err)
	}

	_, err = p.db.ExecContext(ctx, query,
		execution.ID,
		execution.PolicyID,
		execution.JobID,
		execution.JobType,
		execution.Status,
		execution.RetryCount,
		execution.MaxRetries,
		execution.StartedAt,
		execution.CompletedAt,
		execution.ErrorMessage,
		resultJSON,
		execution.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create job execution: %w", err)
	}

	return nil
}

// UpdateJobExecution updates an existing job execution record
func (p *PostgresPersistor) UpdateJobExecution(ctx context.Context, execution *WorkerJobExecution) error {
	query := `
		UPDATE worker_job_execution SET
			status = $2,
			retry_count = $3,
			started_at = $4,
			completed_at = $5,
			error_message = $6,
			result_summary = $7
		WHERE id = $1
	`

	resultJSON, err := json.Marshal(execution.ResultSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal result_summary: %w", err)
	}

	result, err := p.db.ExecContext(ctx, query,
		execution.ID,
		execution.Status,
		execution.RetryCount,
		execution.StartedAt,
		execution.CompletedAt,
		execution.ErrorMessage,
		resultJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update job execution: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job execution not found: %s", execution.ID)
	}

	return nil
}

// GetJobExecutionsByPolicyID retrieves job executions for a policy
func (p *PostgresPersistor) GetJobExecutionsByPolicyID(ctx context.Context, policyID uuid.UUID, limit int) ([]*WorkerJobExecution, error) {
	query := `
		SELECT id, policy_id, job_id, job_type, status, retry_count, max_retries,
		       started_at, completed_at, error_message, result_summary, created_at
		FROM worker_job_execution
		WHERE policy_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := p.db.QueryContext(ctx, query, policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query job executions: %w", err)
	}
	defer rows.Close()

	var executions []*WorkerJobExecution
	for rows.Next() {
		var exec WorkerJobExecution
		var resultJSON []byte

		err := rows.Scan(
			&exec.ID,
			&exec.PolicyID,
			&exec.JobID,
			&exec.JobType,
			&exec.Status,
			&exec.RetryCount,
			&exec.MaxRetries,
			&exec.StartedAt,
			&exec.CompletedAt,
			&exec.ErrorMessage,
			&resultJSON,
			&exec.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job execution: %w", err)
		}

		// Parse result summary JSON
		if len(resultJSON) > 0 {
			if err := json.Unmarshal(resultJSON, &exec.ResultSummary); err != nil {
				return nil, fmt.Errorf("failed to unmarshal result_summary: %w", err)
			}
		}

		executions = append(executions, &exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job executions: %w", err)
	}

	return executions, nil
}

// Disaster Recovery

// LoadActiveWorkerInfrastructure loads all active policy IDs
func (p *PostgresPersistor) LoadActiveWorkerInfrastructure(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT p.policy_id
		FROM worker_pool_state p
		INNER JOIN worker_scheduler_state s ON p.policy_id = s.policy_id
		WHERE p.pool_status = 'active' AND s.scheduler_status = 'active'
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load active worker infrastructure: %w", err)
	}
	defer rows.Close()

	var policyIDs []uuid.UUID
	for rows.Next() {
		var policyID uuid.UUID
		if err := rows.Scan(&policyID); err != nil {
			return nil, fmt.Errorf("failed to scan policy ID: %w", err)
		}
		policyIDs = append(policyIDs, policyID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policy IDs: %w", err)
	}

	return policyIDs, nil
}

// Transaction Support

// WithTransaction executes a function within a database transaction
func (p *PostgresPersistor) WithTransaction(ctx context.Context, fn func(context.Context, WorkerPersistor) error) error {
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a persistor that uses the transaction
	// We'll use the *sqlx.Tx directly since it implements the same query methods
	txPersistor := &postgresTxPersistor{tx: tx}

	// Execute the function
	if err := fn(ctx, txPersistor); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction (original error: %w): %w", err, rbErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// postgresTxPersistor is a transaction-aware persistor
type postgresTxPersistor struct {
	tx *sqlx.Tx
}

// Implement all WorkerPersistor methods using tx instead of db
func (p *postgresTxPersistor) CreatePoolState(ctx context.Context, state *WorkerPoolState) error {
	query := `
		INSERT INTO worker_pool_state (
			policy_id, pool_name, queue_name_base, num_workers, job_timeout, pool_status,
			created_at, started_at, stopped_at, last_job_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, err := json.Marshal(state.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.tx.ExecContext(ctx, query,
		state.PolicyID, state.PoolName, state.QueueNameBase, state.NumWorkers,
		state.JobTimeout, state.PoolStatus, state.CreatedAt, state.StartedAt,
		state.StoppedAt, state.LastJobAt, metadataJSON,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("pool already exists for policy %s: %w", state.PolicyID, err)
		}
		return fmt.Errorf("failed to create pool state: %w", err)
	}
	return nil
}

func (p *postgresTxPersistor) UpdatePoolState(ctx context.Context, state *WorkerPoolState) error {
	query := `
		UPDATE worker_pool_state SET
			pool_status = $2, started_at = $3, stopped_at = $4, last_job_at = $5, metadata = $6
		WHERE policy_id = $1
	`

	metadataJSON, _ := json.Marshal(state.Metadata)
	result, err := p.tx.ExecContext(ctx, query, state.PolicyID, state.PoolStatus,
		state.StartedAt, state.StoppedAt, state.LastJobAt, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to update pool state: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("pool not found for policy %s", state.PolicyID)
	}
	return nil
}

func (p *postgresTxPersistor) GetPoolStateByPolicyID(ctx context.Context, policyID uuid.UUID) (*WorkerPoolState, error) {
	// Same implementation as PostgresPersistor but using p.tx
	query := `SELECT policy_id, pool_name, queue_name_base, num_workers, job_timeout, pool_status,
	                 created_at, started_at, stopped_at, last_job_at, metadata
	          FROM worker_pool_state WHERE policy_id = $1`

	var state WorkerPoolState
	var metadataJSON []byte
	var jobTimeout int64

	err := p.tx.QueryRowContext(ctx, query, policyID).Scan(
		&state.PolicyID, &state.PoolName, &state.QueueNameBase, &state.NumWorkers,
		&jobTimeout, &state.PoolStatus, &state.CreatedAt, &state.StartedAt,
		&state.StoppedAt, &state.LastJobAt, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pool state not found for policy %s", policyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pool state: %w", err)
	}

	state.JobTimeout = nanosecondsToDuration(jobTimeout)
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &state.Metadata)
	}
	return &state, nil
}

func (p *postgresTxPersistor) SetPoolStatus(ctx context.Context, policyID uuid.UUID, status WorkerPoolStatus) error {
	query := `
		UPDATE worker_pool_state SET pool_status = $2,
			started_at = CASE WHEN $2 = 'active' THEN NOW() ELSE started_at END,
			stopped_at = CASE WHEN $2 IN ('stopped', 'archived') THEN NOW() ELSE stopped_at END
		WHERE policy_id = $1
	`
	result, err := p.tx.ExecContext(ctx, query, policyID, status)
	if err != nil {
		return fmt.Errorf("failed to set pool status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("pool not found for policy %s", policyID)
	}
	return nil
}

func (p *postgresTxPersistor) CreateSchedulerState(ctx context.Context, state *WorkerSchedulerState) error {
	query := `
		INSERT INTO worker_scheduler_state (
			policy_id, scheduler_name, monitor_interval, monitor_frequency_unit,
			scheduler_status, created_at, started_at, stopped_at, last_run_at, next_run_at,
			run_count, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	metadataJSON, _ := json.Marshal(state.Metadata)
	_, err := p.tx.ExecContext(ctx, query,
		state.PolicyID, state.SchedulerName, state.MonitorInterval, state.MonitorFrequencyUnit,
		state.SchedulerStatus, state.CreatedAt, state.StartedAt, state.StoppedAt,
		state.LastRunAt, state.NextRunAt, state.RunCount, metadataJSON,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("scheduler already exists for policy %s: %w", state.PolicyID, err)
		}
		return fmt.Errorf("failed to create scheduler state: %w", err)
	}
	return nil
}

func (p *postgresTxPersistor) UpdateSchedulerState(ctx context.Context, state *WorkerSchedulerState) error {
	query := `
		UPDATE worker_scheduler_state SET scheduler_status = $2, started_at = $3,
			stopped_at = $4, last_run_at = $5, next_run_at = $6, run_count = $7, metadata = $8
		WHERE policy_id = $1
	`

	metadataJSON, _ := json.Marshal(state.Metadata)
	result, err := p.tx.ExecContext(ctx, query, state.PolicyID, state.SchedulerStatus,
		state.StartedAt, state.StoppedAt, state.LastRunAt, state.NextRunAt,
		state.RunCount, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to update scheduler state: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("scheduler not found for policy %s", state.PolicyID)
	}
	return nil
}

func (p *postgresTxPersistor) GetSchedulerStateByPolicyID(ctx context.Context, policyID uuid.UUID) (*WorkerSchedulerState, error) {
	query := `SELECT policy_id, scheduler_name, monitor_interval, monitor_frequency_unit,
	                 scheduler_status, created_at, started_at, stopped_at, last_run_at, next_run_at,
	                 run_count, metadata
	          FROM worker_scheduler_state WHERE policy_id = $1`

	var state WorkerSchedulerState
	var metadataJSON []byte
	var monitorInterval int64

	err := p.tx.QueryRowContext(ctx, query, policyID).Scan(
		&state.PolicyID, &state.SchedulerName, &monitorInterval, &state.MonitorFrequencyUnit,
		&state.SchedulerStatus, &state.CreatedAt, &state.StartedAt, &state.StoppedAt,
		&state.LastRunAt, &state.NextRunAt, &state.RunCount, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scheduler state not found for policy %s", policyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler state: %w", err)
	}

	state.MonitorInterval = nanosecondsToDuration(monitorInterval)
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &state.Metadata)
	}
	return &state, nil
}

func (p *postgresTxPersistor) SetSchedulerStatus(ctx context.Context, policyID uuid.UUID, status WorkerSchedulerStatus) error {
	query := `
		UPDATE worker_scheduler_state SET scheduler_status = $2,
			started_at = CASE WHEN $2 = 'active' THEN NOW() ELSE started_at END,
			stopped_at = CASE WHEN $2 IN ('stopped', 'archived') THEN NOW() ELSE stopped_at END
		WHERE policy_id = $1
	`
	result, err := p.tx.ExecContext(ctx, query, policyID, status)
	if err != nil {
		return fmt.Errorf("failed to set scheduler status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("scheduler not found for policy %s", policyID)
	}
	return nil
}

func (p *postgresTxPersistor) CreateJobExecution(ctx context.Context, execution *WorkerJobExecution) error {
	query := `
		INSERT INTO worker_job_execution (
			id, policy_id, job_id, job_type, status, retry_count, max_retries,
			started_at, completed_at, error_message, result_summary, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	resultJSON, _ := json.Marshal(execution.ResultSummary)
	_, err := p.tx.ExecContext(ctx, query,
		execution.ID, execution.PolicyID, execution.JobID, execution.JobType,
		execution.Status, execution.RetryCount, execution.MaxRetries,
		execution.StartedAt, execution.CompletedAt, execution.ErrorMessage,
		resultJSON, execution.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create job execution: %w", err)
	}
	return nil
}

func (p *postgresTxPersistor) UpdateJobExecution(ctx context.Context, execution *WorkerJobExecution) error {
	query := `
		UPDATE worker_job_execution SET status = $2, retry_count = $3,
			started_at = $4, completed_at = $5, error_message = $6, result_summary = $7
		WHERE id = $1
	`

	resultJSON, _ := json.Marshal(execution.ResultSummary)
	result, err := p.tx.ExecContext(ctx, query, execution.ID, execution.Status,
		execution.RetryCount, execution.StartedAt, execution.CompletedAt,
		execution.ErrorMessage, resultJSON)
	if err != nil {
		return fmt.Errorf("failed to update job execution: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job execution not found: %s", execution.ID)
	}
	return nil
}

func (p *postgresTxPersistor) GetJobExecutionsByPolicyID(ctx context.Context, policyID uuid.UUID, limit int) ([]*WorkerJobExecution, error) {
	query := `
		SELECT id, policy_id, job_id, job_type, status, retry_count, max_retries,
		       started_at, completed_at, error_message, result_summary, created_at
		FROM worker_job_execution
		WHERE policy_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := p.tx.QueryContext(ctx, query, policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query job executions: %w", err)
	}
	defer rows.Close()

	var executions []*WorkerJobExecution
	for rows.Next() {
		var exec WorkerJobExecution
		var resultJSON []byte

		err := rows.Scan(&exec.ID, &exec.PolicyID, &exec.JobID, &exec.JobType,
			&exec.Status, &exec.RetryCount, &exec.MaxRetries, &exec.StartedAt,
			&exec.CompletedAt, &exec.ErrorMessage, &resultJSON, &exec.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job execution: %w", err)
		}

		if len(resultJSON) > 0 {
			json.Unmarshal(resultJSON, &exec.ResultSummary)
		}
		executions = append(executions, &exec)
	}
	return executions, rows.Err()
}

func (p *postgresTxPersistor) LoadActiveWorkerInfrastructure(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT p.policy_id
		FROM worker_pool_state p
		INNER JOIN worker_scheduler_state s ON p.policy_id = s.policy_id
		WHERE p.pool_status = 'active' AND s.scheduler_status = 'active'
	`

	rows, err := p.tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load active worker infrastructure: %w", err)
	}
	defer rows.Close()

	var policyIDs []uuid.UUID
	for rows.Next() {
		var policyID uuid.UUID
		if err := rows.Scan(&policyID); err != nil {
			return nil, fmt.Errorf("failed to scan policy ID: %w", err)
		}
		policyIDs = append(policyIDs, policyID)
	}
	return policyIDs, rows.Err()
}

func (p *postgresTxPersistor) WithTransaction(ctx context.Context, fn func(context.Context, WorkerPersistor) error) error {
	// Already in a transaction, just execute
	return fn(ctx, p)
}

// Helper functions

// nanosecondsToDuration converts PostgreSQL interval (stored as nanoseconds) to time.Duration
func nanosecondsToDuration(ns int64) time.Duration {
	return time.Duration(ns)
}

// durationToNanoseconds converts time.Duration to nanoseconds for PostgreSQL interval
func durationToNanoseconds(d time.Duration) int64 {
	return int64(d)
}
