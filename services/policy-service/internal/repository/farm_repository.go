package repository

import (
	utils "agrisa_utils"
	"fmt"
	"log/slog"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FarmRepository struct {
	db *sqlx.DB
}

func NewFarmRepository(db *sqlx.DB) *FarmRepository {
	return &FarmRepository{db: db}
}

func (r *FarmRepository) Create(farm *models.Farm) error {
	if farm.ID == uuid.Nil {
		farm.ID = uuid.New()
	}
	farm.CreatedAt = time.Now()
	farm.UpdatedAt = time.Now()

	query := `
		INSERT INTO farm (
			id, owner_id, farm_name, farm_code, boundary, center_location, area_sqm,
			province, district, commune, address, crop_type, planting_date, expected_harvest_date,
			crop_type_verified, crop_type_verified_at, crop_type_verified_by, crop_type_confidence,
			land_certificate_number, land_certificate_url, land_ownership_verified, land_ownership_verified_at,
			has_irrigation, irrigation_type, soil_type, status, created_at, updated_at
		) VALUES (
			:id, :owner_id, :farm_name, :farm_code, :boundary, :center_location, :area_sqm,
			:province, :district, :commune, :address, :crop_type, :planting_date, :expected_harvest_date,
			:crop_type_verified, :crop_type_verified_at, :crop_type_verified_by, :crop_type_confidence,
			:land_certificate_number, :land_certificate_url, :land_ownership_verified, :land_ownership_verified_at,
			:has_irrigation, :irrigation_type, :soil_type, :status, :created_at, :updated_at
		)`

	_, err := r.db.NamedExec(query, farm)
	if err != nil {
		return fmt.Errorf("failed to create farm: %w", err)
	}

	return nil
}

func (r *FarmRepository) GetByID(id uuid.UUID) (*models.Farm, error) {
	var farm models.Farm
	query := `SELECT * FROM farm WHERE id = $1`

	err := r.db.Get(&farm, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get farm: %w", err)
	}

	return &farm, nil
}

func (r *FarmRepository) GetAll() ([]models.Farm, error) {
	var farms []models.Farm
	query := `SELECT * FROM farm ORDER BY created_at DESC`

	err := r.db.Select(&farms, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get farms: %w", err)
	}

	return farms, nil
}

func (r *FarmRepository) GetByOwnerID(ownerID string) ([]models.Farm, error) {
	var farms []models.Farm
	query := `SELECT * FROM farm WHERE owner_id = $1 ORDER BY created_at DESC`

	err := r.db.Select(&farms, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get farms by owner: %w", err)
	}

	return farms, nil
}

func (r *FarmRepository) Update(farm *models.Farm) error {
	farm.UpdatedAt = time.Now()

	query := `
		UPDATE farm SET
			farm_name = :farm_name, farm_code = :farm_code, boundary = :boundary,
			center_location = :center_location, area_sqm = :area_sqm, province = :province,
			district = :district, commune = :commune, address = :address, crop_type = :crop_type,
			planting_date = :planting_date, expected_harvest_date = :expected_harvest_date,
			crop_type_verified = :crop_type_verified, crop_type_verified_at = :crop_type_verified_at,
			crop_type_verified_by = :crop_type_verified_by, crop_type_confidence = :crop_type_confidence,
			land_certificate_number = :land_certificate_number, land_certificate_url = :land_certificate_url,
			land_ownership_verified = :land_ownership_verified, land_ownership_verified_at = :land_ownership_verified_at,
			has_irrigation = :has_irrigation, irrigation_type = :irrigation_type, soil_type = :soil_type,
			status = :status, updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExec(query, farm)
	if err != nil {
		return fmt.Errorf("failed to update farm: %w", err)
	}

	return nil
}

func (r *FarmRepository) Delete(id uuid.UUID) error {
	query := `UPDATE farm SET status = $1, updated_at = $2 WHERE id = $3`

	err := utils.ExecWithCheck(r.db, query, utils.ExecUpdate, "deleted", time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete farm: %w", err)
	}
	return nil
}

func (r *FarmRepository) GetFarmByFarmCode(farmCode string) (*models.Farm, error) {
	query := `SELECT * FROM farm WHERE farm_code = $1`
	var farm models.Farm
	err := r.db.Get(&farm, query, farmCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get farm by farm code: %w", err)
	}
	return &farm, nil
}
// ============================================================================
// TRANSACTION SUPPORT
// ============================================================================

// BeginTransaction starts a new database transaction
func (r *FarmRepository) BeginTransaction() (*sqlx.Tx, error) {
	slog.Info("Beginning database transaction for farm")
	tx, err := r.db.Beginx()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// CreateTx creates a farm within a transaction
func (r *FarmRepository) CreateTx(tx *sqlx.Tx, farm *models.Farm) error {
	if farm.ID == uuid.Nil {
		farm.ID = uuid.New()
	}
	farm.CreatedAt = time.Now()
	farm.UpdatedAt = time.Now()

	query := `
		INSERT INTO farm (
			id, owner_id, farm_name, farm_code, boundary, center_location, area_sqm,
			province, district, commune, address, crop_type, planting_date, expected_harvest_date,
			crop_type_verified, crop_type_verified_at, crop_type_verified_by, crop_type_confidence,
			land_certificate_number, land_certificate_url, land_ownership_verified, land_ownership_verified_at,
			has_irrigation, irrigation_type, soil_type, status, created_at, updated_at
		) VALUES (
			:id, :owner_id, :farm_name, :farm_code, :boundary, :center_location, :area_sqm,
			:province, :district, :commune, :address, :crop_type, :planting_date, :expected_harvest_date,
			:crop_type_verified, :crop_type_verified_at, :crop_type_verified_by, :crop_type_confidence,
			:land_certificate_number, :land_certificate_url, :land_ownership_verified, :land_ownership_verified_at,
			:has_irrigation, :irrigation_type, :soil_type, :status, :created_at, :updated_at
		)`

	_, err := tx.NamedExec(query, farm)
	if err != nil {
		return fmt.Errorf("failed to create farm in transaction: %w", err)
	}

	return nil
}

// UpdateTx updates a farm within a transaction
func (r *FarmRepository) UpdateTx(tx *sqlx.Tx, farm *models.Farm) error {
	farm.UpdatedAt = time.Now()

	query := `
		UPDATE farm SET
			farm_name = :farm_name, farm_code = :farm_code, boundary = :boundary,
			center_location = :center_location, area_sqm = :area_sqm, province = :province,
			district = :district, commune = :commune, address = :address, crop_type = :crop_type,
			planting_date = :planting_date, expected_harvest_date = :expected_harvest_date,
			crop_type_verified = :crop_type_verified, crop_type_verified_at = :crop_type_verified_at,
			crop_type_verified_by = :crop_type_verified_by, crop_type_confidence = :crop_type_confidence,
			land_certificate_number = :land_certificate_number, land_certificate_url = :land_certificate_url,
			land_ownership_verified = :land_ownership_verified, land_ownership_verified_at = :land_ownership_verified_at,
			has_irrigation = :has_irrigation, irrigation_type = :irrigation_type, soil_type = :soil_type,
			status = :status, updated_at = :updated_at
		WHERE id = :id`

	_, err := tx.NamedExec(query, farm)
	if err != nil {
		return fmt.Errorf("failed to update farm in transaction: %w", err)
	}

	return nil
}

// DeleteTx deletes a farm within a transaction
func (r *FarmRepository) DeleteTx(tx *sqlx.Tx, id uuid.UUID) error {
	query := `DELETE FROM farm WHERE id = $1`

	_, err := tx.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete farm in transaction: %w", err)
	}

	return nil
}

// GetByIDTx retrieves a farm by ID within a transaction
func (r *FarmRepository) GetByIDTx(tx *sqlx.Tx, id uuid.UUID) (*models.Farm, error) {
	var farm models.Farm
	query := `SELECT * FROM farm WHERE id = $1`

	err := tx.Get(&farm, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get farm in transaction: %w", err)
	}

	return &farm, nil
}

// GetByOwnerIDTx retrieves farms by owner ID within a transaction
func (r *FarmRepository) GetByOwnerIDTx(tx *sqlx.Tx, ownerID string) ([]models.Farm, error) {
	var farms []models.Farm
	query := `SELECT * FROM farm WHERE owner_id = $1 ORDER BY created_at DESC`

	err := tx.Select(&farms, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get farms by owner in transaction: %w", err)
	}

	return farms, nil
}
