package repository

import (
	"database/sql"
	"fmt"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DataTierRepository struct {
	db *sqlx.DB
}

func NewDataTierRepository(db *sqlx.DB) *DataTierRepository {
	return &DataTierRepository{db: db}
}

func (r *DataTierRepository) CreateDataTierCategory(category *models.DataTierCategory) error {
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	query := `
		INSERT INTO data_tier_category (id, category_name, category_description, category_cost_multiplier, created_at, updated_at)
		VALUES (:id, :category_name, :category_description, :category_cost_multiplier, :created_at, :updated_at)`

	_, err := r.db.NamedExec(query, category)
	if err != nil {
		return fmt.Errorf("failed to create data tier category: %w", err)
	}

	return nil
}

func (r *DataTierRepository) GetDataTierCategoryByID(id uuid.UUID) (*models.DataTierCategory, error) {
	var category models.DataTierCategory
	query := `
		SELECT id, category_name, category_description, category_cost_multiplier, created_at, updated_at
		FROM data_tier_category
		WHERE id = $1`

	err := r.db.Get(&category, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data tier category not found")
		}
		return nil, fmt.Errorf("failed to get data tier category: %w", err)
	}

	return &category, nil
}

func (r *DataTierRepository) GetAllDataTierCategories() ([]models.DataTierCategory, error) {
	var categories []models.DataTierCategory
	query := `
		SELECT id, category_name, category_description, category_cost_multiplier, created_at, updated_at
		FROM data_tier_category
		ORDER BY category_name`

	err := r.db.Select(&categories, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get data tier categories: %w", err)
	}

	return categories, nil
}

func (r *DataTierRepository) UpdateDataTierCategory(category *models.DataTierCategory) error {
	category.UpdatedAt = time.Now()

	query := `
		UPDATE data_tier_category
		SET category_name = :category_name,
			category_description = :category_description,
			category_cost_multiplier = :category_cost_multiplier,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExec(query, category)
	if err != nil {
		return fmt.Errorf("failed to update data tier category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("data tier category not found")
	}

	return nil
}

func (r *DataTierRepository) DeleteDataTierCategory(id uuid.UUID) error {
	query := `DELETE FROM data_tier_category WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete data tier category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("data tier category not found")
	}

	return nil
}

func (r *DataTierRepository) CreateDataTier(tier *models.DataTier) error {
	tier.CreatedAt = time.Now()
	tier.UpdatedAt = time.Now()

	query := `
		INSERT INTO data_tier (id, data_tier_category_id, tier_level, tier_name, data_tier_multiplier, created_at, updated_at)
		VALUES (:id, :data_tier_category_id, :tier_level, :tier_name, :data_tier_multiplier, :created_at, :updated_at)`

	_, err := r.db.NamedExec(query, tier)
	if err != nil {
		return fmt.Errorf("failed to create data tier: %w", err)
	}

	return nil
}

func (r *DataTierRepository) GetDataTierByID(id uuid.UUID) (*models.DataTier, error) {
	var tier models.DataTier
	query := `
		SELECT id, data_tier_category_id, tier_level, tier_name, data_tier_multiplier, created_at, updated_at
		FROM data_tier
		WHERE id = $1`

	err := r.db.Get(&tier, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data tier not found")
		}
		return nil, fmt.Errorf("failed to get data tier: %w", err)
	}

	return &tier, nil
}

func (r *DataTierRepository) GetDataTiersByCategoryID(categoryID uuid.UUID) ([]models.DataTier, error) {
	var tiers []models.DataTier
	query := `
		SELECT id, data_tier_category_id, tier_level, tier_name, data_tier_multiplier, created_at, updated_at
		FROM data_tier
		WHERE data_tier_category_id = $1
		ORDER BY tier_level`

	err := r.db.Select(&tiers, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data tiers by category: %w", err)
	}

	return tiers, nil
}

func (r *DataTierRepository) GetAllDataTiers() ([]models.DataTier, error) {
	var tiers []models.DataTier
	query := `
		SELECT id, data_tier_category_id, tier_level, tier_name, data_tier_multiplier, created_at, updated_at
		FROM data_tier
		ORDER BY data_tier_category_id, tier_level`

	err := r.db.Select(&tiers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get data tiers: %w", err)
	}

	return tiers, nil
}

func (r *DataTierRepository) UpdateDataTier(tier *models.DataTier) error {
	tier.UpdatedAt = time.Now()

	query := `
		UPDATE data_tier
		SET data_tier_category_id = :data_tier_category_id,
			tier_level = :tier_level,
			tier_name = :tier_name,
			data_tier_multiplier = :data_tier_multiplier,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExec(query, tier)
	if err != nil {
		return fmt.Errorf("failed to update data tier: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("data tier not found")
	}

	return nil
}

func (r *DataTierRepository) DeleteDataTier(id uuid.UUID) error {
	query := `DELETE FROM data_tier WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete data tier: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("data tier not found")
	}

	return nil
}

func (r *DataTierRepository) GetDataTierWithCategory(tierID uuid.UUID) (*models.DataTier, *models.DataTierCategory, error) {
	tier, err := r.GetDataTierByID(tierID)
	if err != nil {
		return nil, nil, err
	}

	category, err := r.GetDataTierCategoryByID(tier.DataTierCategoryID)
	if err != nil {
		return nil, nil, err
	}

	return tier, category, nil
}

func (r *DataTierRepository) CheckCategoryExists(categoryID uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM data_tier_category WHERE id = $1`

	err := r.db.Get(&count, query, categoryID)
	if err != nil {
		return false, fmt.Errorf("failed to check category existence: %w", err)
	}

	return count > 0, nil
}

func (r *DataTierRepository) CheckTierLevelExists(categoryID uuid.UUID, tierLevel int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM data_tier WHERE data_tier_category_id = $1 AND tier_level = $2`

	err := r.db.Get(&count, query, categoryID, tierLevel)
	if err != nil {
		return false, fmt.Errorf("failed to check tier level existence: %w", err)
	}

	return count > 0, nil
}
