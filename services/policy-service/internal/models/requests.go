package models

import (
	"errors"

	"github.com/google/uuid"
)

type CreateDataTierCategoryRequest struct {
	CategoryName           string  `json:"category_name" validate:"required,min=1,max=100"`
	CategoryDescription    *string `json:"category_description,omitempty" validate:"omitempty,max=500"`
	CategoryCostMultiplier float64 `json:"category_cost_multiplier" validate:"required,min=0.01,max=100"`
}

func (r CreateDataTierCategoryRequest) Validate() error {
	if r.CategoryName == "" {
		return errors.New("category name is required")
	}
	if len(r.CategoryName) > 100 {
		return errors.New("category name must be 100 characters or less")
	}
	if r.CategoryDescription != nil && len(*r.CategoryDescription) > 500 {
		return errors.New("category description must be 500 characters or less")
	}
	if r.CategoryCostMultiplier <= 0 {
		return errors.New("category cost multiplier must be greater than 0")
	}
	if r.CategoryCostMultiplier > 100 {
		return errors.New("category cost multiplier must be 100 or less")
	}
	return nil
}

type UpdateDataTierCategoryRequest struct {
	CategoryName           *string  `json:"category_name,omitempty" validate:"omitempty,min=1,max=100"`
	CategoryDescription    *string  `json:"category_description,omitempty" validate:"omitempty,max=500"`
	CategoryCostMultiplier *float64 `json:"category_cost_multiplier,omitempty" validate:"omitempty,min=0.01,max=100"`
}

func (r UpdateDataTierCategoryRequest) Validate() error {
	if r.CategoryName != nil {
		if *r.CategoryName == "" {
			return errors.New("category name cannot be empty")
		}
		if len(*r.CategoryName) > 100 {
			return errors.New("category name must be 100 characters or less")
		}
	}
	if r.CategoryDescription != nil && len(*r.CategoryDescription) > 500 {
		return errors.New("category description must be 500 characters or less")
	}
	if r.CategoryCostMultiplier != nil {
		if *r.CategoryCostMultiplier <= 0 {
			return errors.New("category cost multiplier must be greater than 0")
		}
		if *r.CategoryCostMultiplier > 100 {
			return errors.New("category cost multiplier must be 100 or less")
		}
	}
	return nil
}

type CreateDataTierRequest struct {
	DataTierCategoryID uuid.UUID `json:"data_tier_category_id" validate:"required"`
	TierLevel          int       `json:"tier_level" validate:"required,min=1,max=100"`
	TierName           string    `json:"tier_name" validate:"required,min=1,max=100"`
	DataTierMultiplier float64   `json:"data_tier_multiplier" validate:"required,min=0.01,max=100"`
}

func (r CreateDataTierRequest) Validate() error {
	if r.DataTierCategoryID == uuid.Nil {
		return errors.New("data tier category ID is required")
	}
	if r.TierLevel < 1 {
		return errors.New("tier level must be at least 1")
	}
	if r.TierLevel > 100 {
		return errors.New("tier level must be 100 or less")
	}
	if r.TierName == "" {
		return errors.New("tier name is required")
	}
	if len(r.TierName) > 100 {
		return errors.New("tier name must be 100 characters or less")
	}
	if r.DataTierMultiplier <= 0 {
		return errors.New("data tier multiplier must be greater than 0")
	}
	if r.DataTierMultiplier > 100 {
		return errors.New("data tier multiplier must be 100 or less")
	}
	return nil
}

type UpdateDataTierRequest struct {
	DataTierCategoryID *uuid.UUID `json:"data_tier_category_id,omitempty"`
	TierLevel          *int       `json:"tier_level,omitempty" validate:"omitempty,min=1,max=100"`
	TierName           *string    `json:"tier_name,omitempty" validate:"omitempty,min=1,max=100"`
	DataTierMultiplier *float64   `json:"data_tier_multiplier,omitempty" validate:"omitempty,min=0.01,max=100"`
}

func (r UpdateDataTierRequest) Validate() error {
	if r.TierLevel != nil {
		if *r.TierLevel < 1 {
			return errors.New("tier level must be at least 1")
		}
		if *r.TierLevel > 100 {
			return errors.New("tier level must be 100 or less")
		}
	}
	if r.TierName != nil {
		if *r.TierName == "" {
			return errors.New("tier name cannot be empty")
		}
		if len(*r.TierName) > 100 {
			return errors.New("tier name must be 100 characters or less")
		}
	}
	if r.DataTierMultiplier != nil {
		if *r.DataTierMultiplier <= 0 {
			return errors.New("data tier multiplier must be greater than 0")
		}
		if *r.DataTierMultiplier > 100 {
			return errors.New("data tier multiplier must be 100 or less")
		}
	}
	return nil
}

