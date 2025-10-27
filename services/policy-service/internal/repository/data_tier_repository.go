package repository

import (
	utils "agrisa_utils"
	"database/sql"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"strings"
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
	if category.ID == uuid.Nil {
		category.ID = uuid.New()
	}

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
	slog.Info("Retrieving data tier category by ID", "category_id", id)
	start := time.Now()

	var category models.DataTierCategory
	query := `
		SELECT id, category_name, category_description, category_cost_multiplier, created_at, updated_at
		FROM data_tier_category
		WHERE id = $1`

	err := r.db.Get(&category, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Data tier category not found", "category_id", id)
			return nil, fmt.Errorf("data tier category not found")
		}
		slog.Error("Failed to get data tier category",
			"category_id", id,
			"error", err)
		return nil, fmt.Errorf("failed to get data tier category: %w", err)
	}

	slog.Info("Successfully retrieved data tier category",
		"category_id", id,
		"category_name", category.CategoryName,
		"cost_multiplier", category.CategoryCostMultiplier,
		"duration", time.Since(start))
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
	if tier.ID == uuid.Nil {
		tier.ID = uuid.New()
	}

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
	slog.Info("Retrieving data tier by ID", "data_tier_id", id)
	start := time.Now()

	var tier models.DataTier
	query := `
		SELECT id, data_tier_category_id, tier_level, tier_name, data_tier_multiplier, created_at, updated_at
		FROM data_tier
		WHERE id = $1`

	err := r.db.Get(&tier, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Data tier not found", "data_tier_id", id)
			return nil, fmt.Errorf("data tier not found")
		}
		slog.Error("Failed to get data tier",
			"data_tier_id", id,
			"error", err)
		return nil, fmt.Errorf("failed to get data tier: %w", err)
	}

	slog.Info("Successfully retrieved data tier",
		"data_tier_id", id,
		"tier_name", tier.TierName,
		"tier_level", tier.TierLevel,
		"duration", time.Since(start))
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

// ============================================================================
// DATATIER WITH CATEGORY READ OPERATIONS WITH MULTIPLE OPTIONS
// ============================================================================

// DataTierWCategoryQueryOptions defines query options for flexible data retrieval
type DataTierWCategoryQueryOptions struct {
	// Filtering options
	CategoryID   *uuid.UUID // Filter by specific category
	CategoryName *string    // Filter by category name (partial match)
	TierLevel    *int       // Filter by specific tier level
	TierLevelMin *int       // Minimum tier level
	TierLevelMax *int       // Maximum tier level
	TierName     *string    // Filter by tier name (partial match)
	IsActive     *bool      // Filter by active status (if you add this field later)

	// Multiplier range filtering
	DataTierMultiplierMin *float64 // Minimum data tier multiplier
	DataTierMultiplierMax *float64 // Maximum data tier multiplier
	CategoryMultiplierMin *float64 // Minimum category cost multiplier
	CategoryMultiplierMax *float64 // Maximum category cost multiplier

	// Sorting options
	SortBy    string // Field to sort by: "tier_level", "tier_name", "category_name", "multiplier", "created_at"
	SortOrder string // "ASC" or "DESC"

	// Pagination options
	Limit  *int // Maximum number of results
	Offset *int // Number of results to skip

	// Include options
	IncludeInactive bool // Include inactive tiers (if you have this field)
}

// GetDataTierWCategoryByID retrieves a single data tier with category by ID
func (r *DataTierRepository) GetDataTierWCategoryByID(id uuid.UUID) (*models.DataTierWCategory, error) {
	slog.Info("Retrieving data tier with category by ID", "data_tier_id", id)
	start := time.Now()

	// Query result struct with prefixed category fields
	var queryResult struct {
		// DataTier fields
		ID                 uuid.UUID `db:"id"`
		TierLevel          int       `db:"tier_level"`
		TierName           string    `db:"tier_name"`
		DataTierMultiplier float64   `db:"data_tier_multiplier"`
		CreatedAt          time.Time `db:"created_at"`
		UpdatedAt          time.Time `db:"updated_at"`

		// DataTierCategory fields with cat_ prefix
		CatID                     uuid.UUID `db:"cat_id"`
		CatCategoryName           string    `db:"cat_category_name"`
		CatCategoryDescription    *string   `db:"cat_category_description"`
		CatCategoryCostMultiplier float64   `db:"cat_category_cost_multiplier"`
		CatCreatedAt              time.Time `db:"cat_created_at"`
		CatUpdatedAt              time.Time `db:"cat_updated_at"`
	}

	query := `
		SELECT 
			dt.id, dt.tier_level, dt.tier_name, dt.data_tier_multiplier,
			dt.created_at, dt.updated_at,
			dtc.id as cat_id,
			dtc.category_name as cat_category_name,
			dtc.category_description as cat_category_description,
			dtc.category_cost_multiplier as cat_category_cost_multiplier,
			dtc.created_at as cat_created_at,
			dtc.updated_at as cat_updated_at
		FROM data_tier dt
		JOIN data_tier_category dtc ON dt.data_tier_category_id = dtc.id
		WHERE dt.id = $1`

	err := r.db.Get(&queryResult, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			slog.Warn("Data tier with category not found", "data_tier_id", id)
			return nil, fmt.Errorf("data tier not found")
		}
		slog.Error("Failed to get data tier with category",
			"data_tier_id", id,
			"error", err)
		return nil, fmt.Errorf("failed to get data tier with category: %w", err)
	}

	// Assemble the result using FastAssembleWithPrefix
	var result models.DataTierWCategory
	err = utils.FastAssembleWithPrefix(&result, queryResult, "cat_")
	if err != nil {
		return nil, fmt.Errorf("failed to assemble data tier with category: %w", err)
	}

	slog.Info("Successfully retrieved data tier with category",
		"data_tier_id", id,
		"tier_name", result.TierName,
		"category_name", result.DataTierCategory.CategoryName,
		"duration", time.Since(start))

	return &result, nil
}

// GetDataTiersWCategory retrieves data tiers with categories using flexible query options
func (r *DataTierRepository) GetDataTiersWCategory(options DataTierWCategoryQueryOptions) ([]models.DataTierWCategory, error) {
	slog.Info("Retrieving data tiers with categories using options",
		"category_id", options.CategoryID,
		"tier_level", options.TierLevel,
		"sort_by", options.SortBy,
		"limit", options.Limit)
	start := time.Now()

	// Build the query dynamically
	query, args := r.buildDataTierWCategoryQuery(options)

	// Query result struct with prefixed category fields
	var queryResults []struct {
		// DataTier fields
		ID                 uuid.UUID `db:"id"`
		TierLevel          int       `db:"tier_level"`
		TierName           string    `db:"tier_name"`
		DataTierMultiplier float64   `db:"data_tier_multiplier"`
		CreatedAt          time.Time `db:"created_at"`
		UpdatedAt          time.Time `db:"updated_at"`

		// DataTierCategory fields with cat_ prefix
		CatID                     uuid.UUID `db:"cat_id"`
		CatCategoryName           string    `db:"cat_category_name"`
		CatCategoryDescription    *string   `db:"cat_category_description"`
		CatCategoryCostMultiplier float64   `db:"cat_category_cost_multiplier"`
		CatCreatedAt              time.Time `db:"cat_created_at"`
		CatUpdatedAt              time.Time `db:"cat_updated_at"`
	}

	err := r.db.Select(&queryResults, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get data tiers with categories: %w", err)
	}

	// Assemble results using FastAssembleWithPrefix
	results := make([]models.DataTierWCategory, len(queryResults))
	for i, queryResult := range queryResults {
		err = utils.FastAssembleWithPrefix(&results[i], queryResult, "cat_")
		if err != nil {
			return nil, fmt.Errorf("failed to assemble data tier %d with category: %w", i, err)
		}
	}

	slog.Info("Successfully retrieved data tiers with categories",
		"count", len(results),
		"duration", time.Since(start))

	return results, nil
}

// GetAllDataTiersWCategory retrieves all data tiers with categories (convenience method)
func (r *DataTierRepository) GetAllDataTiersWCategory() ([]models.DataTierWCategory, error) {
	return r.GetDataTiersWCategory(DataTierWCategoryQueryOptions{
		SortBy:    "category_name",
		SortOrder: "ASC",
	})
}

// GetDataTiersWCategoryByCategoryID retrieves data tiers by category ID (convenience method)
func (r *DataTierRepository) GetDataTiersWCategoryByCategoryID(categoryID uuid.UUID) ([]models.DataTierWCategory, error) {
	return r.GetDataTiersWCategory(DataTierWCategoryQueryOptions{
		CategoryID: &categoryID,
		SortBy:     "tier_level",
		SortOrder:  "ASC",
	})
}

// GetDataTiersWCategoryByTierLevel retrieves data tiers by tier level (convenience method)
func (r *DataTierRepository) GetDataTiersWCategoryByTierLevel(tierLevel int) ([]models.DataTierWCategory, error) {
	return r.GetDataTiersWCategory(DataTierWCategoryQueryOptions{
		TierLevel: &tierLevel,
		SortBy:    "category_name",
		SortOrder: "ASC",
	})
}

// GetDataTiersWCategoryByTierLevelRange retrieves data tiers within tier level range
func (r *DataTierRepository) GetDataTiersWCategoryByTierLevelRange(minLevel, maxLevel int) ([]models.DataTierWCategory, error) {
	return r.GetDataTiersWCategory(DataTierWCategoryQueryOptions{
		TierLevelMin: &minLevel,
		TierLevelMax: &maxLevel,
		SortBy:       "tier_level",
		SortOrder:    "ASC",
	})
}

// GetDataTiersWCategoryByMultiplierRange retrieves data tiers within multiplier range
func (r *DataTierRepository) GetDataTiersWCategoryByMultiplierRange(minMultiplier, maxMultiplier float64) ([]models.DataTierWCategory, error) {
	return r.GetDataTiersWCategory(DataTierWCategoryQueryOptions{
		DataTierMultiplierMin: &minMultiplier,
		DataTierMultiplierMax: &maxMultiplier,
		SortBy:                "data_tier_multiplier",
		SortOrder:             "ASC",
	})
}

// SearchDataTiersWCategoryByName searches data tiers by tier name or category name
func (r *DataTierRepository) SearchDataTiersWCategoryByName(searchTerm string) ([]models.DataTierWCategory, error) {
	return r.GetDataTiersWCategory(DataTierWCategoryQueryOptions{
		TierName:     &searchTerm,
		CategoryName: &searchTerm,
		SortBy:       "tier_name",
		SortOrder:    "ASC",
	})
}

// GetDataTiersWCategoryPaginated retrieves data tiers with pagination
func (r *DataTierRepository) GetDataTiersWCategoryPaginated(limit, offset int, sortBy, sortOrder string) ([]models.DataTierWCategory, error) {
	return r.GetDataTiersWCategory(DataTierWCategoryQueryOptions{
		Limit:     &limit,
		Offset:    &offset,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
}

// CountDataTiersWCategory counts data tiers matching the given options (without pagination)
func (r *DataTierRepository) CountDataTiersWCategory(options DataTierWCategoryQueryOptions) (int, error) {
	// Remove pagination for counting
	countOptions := options
	countOptions.Limit = nil
	countOptions.Offset = nil
	countOptions.SortBy = ""
	countOptions.SortOrder = ""

	query, args := r.buildDataTierWCategoryCountQuery(countOptions)

	var count int
	err := r.db.Get(&count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count data tiers with categories: %w", err)
	}

	return count, nil
}

// buildDataTierWCategoryQuery builds the dynamic query based on options
func (r *DataTierRepository) buildDataTierWCategoryQuery(options DataTierWCategoryQueryOptions) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT 
			dt.id, dt.tier_level, dt.tier_name, dt.data_tier_multiplier,
			dt.created_at, dt.updated_at,
			dtc.id as cat_id,
			dtc.category_name as cat_category_name,
			dtc.category_description as cat_category_description,
			dtc.category_cost_multiplier as cat_category_cost_multiplier,
			dtc.created_at as cat_created_at,
			dtc.updated_at as cat_updated_at
		FROM data_tier dt
		JOIN data_tier_category dtc ON dt.data_tier_category_id = dtc.id`

	// Add WHERE conditions
	if options.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.id = $%d", argIndex))
		args = append(args, *options.CategoryID)
		argIndex++
	}

	if options.CategoryName != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.category_name ILIKE $%d", argIndex))
		args = append(args, "%"+*options.CategoryName+"%")
		argIndex++
	}

	if options.TierLevel != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_level = $%d", argIndex))
		args = append(args, *options.TierLevel)
		argIndex++
	}

	if options.TierLevelMin != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_level >= $%d", argIndex))
		args = append(args, *options.TierLevelMin)
		argIndex++
	}

	if options.TierLevelMax != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_level <= $%d", argIndex))
		args = append(args, *options.TierLevelMax)
		argIndex++
	}

	if options.TierName != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_name ILIKE $%d", argIndex))
		args = append(args, "%"+*options.TierName+"%")
		argIndex++
	}

	if options.DataTierMultiplierMin != nil {
		conditions = append(conditions, fmt.Sprintf("dt.data_tier_multiplier >= $%d", argIndex))
		args = append(args, *options.DataTierMultiplierMin)
		argIndex++
	}

	if options.DataTierMultiplierMax != nil {
		conditions = append(conditions, fmt.Sprintf("dt.data_tier_multiplier <= $%d", argIndex))
		args = append(args, *options.DataTierMultiplierMax)
		argIndex++
	}

	if options.CategoryMultiplierMin != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.category_cost_multiplier >= $%d", argIndex))
		args = append(args, *options.CategoryMultiplierMin)
		argIndex++
	}

	if options.CategoryMultiplierMax != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.category_cost_multiplier <= $%d", argIndex))
		args = append(args, *options.CategoryMultiplierMax)
		argIndex++
	}

	// Build final query
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	if options.SortBy != "" {
		orderBy := r.mapSortField(options.SortBy)
		sortOrder := "ASC"
		if strings.ToUpper(options.SortOrder) == "DESC" {
			sortOrder = "DESC"
		}
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", orderBy, sortOrder)
	} else {
		baseQuery += " ORDER BY dtc.category_name ASC, dt.tier_level ASC"
	}

	// Add LIMIT and OFFSET
	if options.Limit != nil {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, *options.Limit)
		argIndex++
	}

	if options.Offset != nil {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, *options.Offset)
		argIndex++
	}

	return baseQuery, args
}

// buildDataTierWCategoryCountQuery builds count query
func (r *DataTierRepository) buildDataTierWCategoryCountQuery(options DataTierWCategoryQueryOptions) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT COUNT(*)
		FROM data_tier dt
		JOIN data_tier_category dtc ON dt.data_tier_category_id = dtc.id`

	// Add same WHERE conditions as main query (without ORDER BY, LIMIT, OFFSET)
	if options.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.id = $%d", argIndex))
		args = append(args, *options.CategoryID)
		argIndex++
	}

	if options.CategoryName != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.category_name ILIKE $%d", argIndex))
		args = append(args, "%"+*options.CategoryName+"%")
		argIndex++
	}

	if options.TierLevel != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_level = $%d", argIndex))
		args = append(args, *options.TierLevel)
		argIndex++
	}

	if options.TierLevelMin != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_level >= $%d", argIndex))
		args = append(args, *options.TierLevelMin)
		argIndex++
	}

	if options.TierLevelMax != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_level <= $%d", argIndex))
		args = append(args, *options.TierLevelMax)
		argIndex++
	}

	if options.TierName != nil {
		conditions = append(conditions, fmt.Sprintf("dt.tier_name ILIKE $%d", argIndex))
		args = append(args, "%"+*options.TierName+"%")
		argIndex++
	}

	if options.DataTierMultiplierMin != nil {
		conditions = append(conditions, fmt.Sprintf("dt.data_tier_multiplier >= $%d", argIndex))
		args = append(args, *options.DataTierMultiplierMin)
		argIndex++
	}

	if options.DataTierMultiplierMax != nil {
		conditions = append(conditions, fmt.Sprintf("dt.data_tier_multiplier <= $%d", argIndex))
		args = append(args, *options.DataTierMultiplierMax)
		argIndex++
	}

	if options.CategoryMultiplierMin != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.category_cost_multiplier >= $%d", argIndex))
		args = append(args, *options.CategoryMultiplierMin)
		argIndex++
	}

	if options.CategoryMultiplierMax != nil {
		conditions = append(conditions, fmt.Sprintf("dtc.category_cost_multiplier <= $%d", argIndex))
		args = append(args, *options.CategoryMultiplierMax)
		argIndex++
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	return baseQuery, args
}

// mapSortField maps sort field names to actual database column names
func (r *DataTierRepository) mapSortField(sortBy string) string {
	switch strings.ToLower(sortBy) {
	case "tier_level":
		return "dt.tier_level"
	case "tier_name":
		return "dt.tier_name"
	case "category_name":
		return "dtc.category_name"
	case "data_tier_multiplier", "multiplier":
		return "dt.data_tier_multiplier"
	case "category_multiplier":
		return "dtc.category_cost_multiplier"
	case "created_at":
		return "dt.created_at"
	case "updated_at":
		return "dt.updated_at"
	default:
		return "dtc.category_name"
	}
}
