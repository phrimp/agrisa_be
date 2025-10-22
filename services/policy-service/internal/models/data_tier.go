package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// DATA TIER MANAGEMENT
// ============================================================================

type DataTierCategory struct {
	ID                     uuid.UUID `json:"id" db:"id"`
	CategoryName           string    `json:"category_name" db:"category_name"`
	CategoryDescription    *string   `json:"category_description,omitempty" db:"category_description"`
	CategoryCostMultiplier float64   `json:"category_cost_multiplier" db:"category_cost_multiplier"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

type DataTier struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	DataTierCategoryID uuid.UUID `json:"data_tier_category_id" db:"data_tier_category_id"`
	TierLevel          int       `json:"tier_level" db:"tier_level"`
	TierName           string    `json:"tier_name" db:"tier_name"`
	DataTierMultiplier float64   `json:"data_tier_multiplier" db:"data_tier_multiplier"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
