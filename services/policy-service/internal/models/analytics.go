package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ANALYTICS & LOGGING
// ============================================================================

type TriggerEvaluationLog struct {
	ID                   uuid.UUID   `json:"id" db:"id"`
	RegisteredPolicyID   uuid.UUID   `json:"registered_policy_id" db:"registered_policy_id"`
	BasePolicyID         uuid.UUID   `json:"base_policy_id" db:"base_policy_id"`
	FarmID               uuid.UUID   `json:"farm_id" db:"farm_id"`
	BasePolicyTriggerID  uuid.UUID   `json:"base_policy_trigger_id" db:"base_policy_trigger_id"`
	EvaluationTimestamp  int64       `json:"evaluation_timestamp" db:"evaluation_timestamp"`
	EvaluationResult     bool        `json:"evaluation_result" db:"evaluation_result"`
	ConditionsEvaluated  int         `json:"conditions_evaluated" db:"conditions_evaluated"`
	ConditionsMet        int         `json:"conditions_met" db:"conditions_met"`
	ConditionDetails     interface{} `json:"condition_details,omitempty" db:"condition_details"` // JSONB
	ClaimGenerated       bool        `json:"claim_generated" db:"claim_generated"`
	ClaimID              *uuid.UUID  `json:"claim_id,omitempty" db:"claim_id"`
	EvaluationDurationMs *int        `json:"evaluation_duration_ms,omitempty" db:"evaluation_duration_ms"`
	DataSourcesQueried   *int        `json:"data_sources_queried,omitempty" db:"data_sources_queried"`
	CreatedAt            time.Time   `json:"created_at" db:"created_at"`
}