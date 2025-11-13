package repository

import (
	utils "agrisa_utils"
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"policy-service/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkb"
)

type FarmRepository struct {
	db *sqlx.DB
}

func NewFarmRepository(db *sqlx.DB) *FarmRepository {
	return &FarmRepository{db: db}
}

type farmRow struct {
	models.Farm
	BoundaryWKB []byte     `db:"boundary_wkb"`
	CenterWKB   []byte     `db:"center_wkb"`
	FarmPhotoID *uuid.UUID `db:"farm_photo_id"`
	FarmID      *uuid.UUID `db:"farm_id"`
	PhotoURL    *string    `db:"photo_url"`
	PhotoType   *string    `db:"photo_type"`
	TakenAt     *int64     `db:"taken_at"`
}

func (r *FarmRepository) Create(farm *models.Farm) error {
	if farm.ID == uuid.Nil {
		farm.ID = uuid.New()
	}
	farm.CreatedAt = time.Now()
	farm.UpdatedAt = time.Now()

	// Query sử dụng PostGIS functions để convert WKT string
	// ST_GeomFromText: convert WKT → GEOMETRY
	// ST_GeogFromText: convert WKT → GEOGRAPHY
	query := `
		INSERT INTO farm (
			id, owner_id, farm_name, farm_code, 
			boundary, 
			center_location, 
			area_sqm,
			province, district, commune, address, 
			crop_type, planting_date, expected_harvest_date,
			crop_type_verified, crop_type_verified_at, crop_type_verified_by, crop_type_confidence,
			land_certificate_number, land_certificate_url, 
			land_ownership_verified, land_ownership_verified_at,
			has_irrigation, irrigation_type, soil_type, 
			status, created_at, updated_at
		) VALUES (
			:id, :owner_id, :farm_name, :farm_code, 
			ST_GeomFromText(:boundary), 
			ST_GeogFromText(:center_location), 
			:area_sqm,
			:province, :district, :commune, :address, 
			:crop_type, :planting_date, :expected_harvest_date,
			:crop_type_verified, :crop_type_verified_at, :crop_type_verified_by, :crop_type_confidence,
			:land_certificate_number, :land_certificate_url, 
			:land_ownership_verified, :land_ownership_verified_at,
			:has_irrigation, :irrigation_type, :soil_type, 
			:status, :created_at, :updated_at
		)`

	_, err := r.db.NamedExec(query, farm)
	if err != nil {
		return fmt.Errorf("failed to create farm: %w", err)
	}

	return nil
}

func (r *FarmRepository) GetFarmByID(ctx context.Context, id string) (*models.Farm, error) {
	query := `
		SELECT 
			f.id, owner_id, farm_name, farm_code,
			area_sqm, province, district, commune, address,
			crop_type, planting_date, expected_harvest_date,
			crop_type_verified, crop_type_verified_at,
			crop_type_verified_by, crop_type_confidence,
			land_certificate_number, land_certificate_url,
			land_ownership_verified, land_ownership_verified_at,
			has_irrigation, irrigation_type, soil_type,
			status, f.created_at, f.updated_at,
			ST_AsBinary(boundary) as boundary_wkb,
			ST_AsBinary(center_location) as center_wkb,
			fp.id as farm_photo_id,
			farm_id,
			photo_url,
			photo_type,
			taken_at
		FROM farm f left join farm_photo fp on f.id = fp.farm_id 
		WHERE f.id = $1
	`

	var rows []farmRow
	err := r.db.SelectContext(ctx, &rows, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No farm found with ID: %s", id)
			return nil, fmt.Errorf("not_found farm not found: %s", id)
		}
		log.Printf("Error querying farm by ID: %v", err)
		return nil, fmt.Errorf("internal_error: query failed: %w", err)
	}

	farm := models.Farm{
		ID:                      rows[0].ID,
		OwnerID:                 rows[0].OwnerID,
		FarmName:                rows[0].FarmName,
		FarmCode:                rows[0].FarmCode,
		AreaSqm:                 rows[0].AreaSqm,
		Province:                rows[0].Province,
		District:                rows[0].District,
		Commune:                 rows[0].Commune,
		Address:                 rows[0].Address,
		CropType:                rows[0].CropType,
		PlantingDate:            rows[0].PlantingDate,
		ExpectedHarvestDate:     rows[0].ExpectedHarvestDate,
		CropTypeVerified:        rows[0].CropTypeVerified,
		CropTypeVerifiedAt:      rows[0].CropTypeVerifiedAt,
		CropTypeVerifiedBy:      rows[0].CropTypeVerifiedBy,
		CropTypeConfidence:      rows[0].CropTypeConfidence,
		LandCertificateNumber:   rows[0].LandCertificateNumber,
		LandCertificateURL:      rows[0].LandCertificateURL,
		LandOwnershipVerified:   rows[0].LandOwnershipVerified,
		LandOwnershipVerifiedAt: rows[0].LandOwnershipVerifiedAt,
		HasIrrigation:           rows[0].HasIrrigation,
		IrrigationType:          rows[0].IrrigationType,
		SoilType:                rows[0].SoilType,
		Status:                  rows[0].Status,
		CreatedAt:               rows[0].CreatedAt,
		UpdatedAt:               rows[0].UpdatedAt,
	}

	if err := r.unmarshalGeometry(&rows[0], &farm); err != nil {
		log.Println("Error unmarshaling geometry:", err)
		return nil, err
	}

	for _, row := range rows {
		if row.FarmPhotoID != nil {
			farm.FarmPhotos = append(farm.FarmPhotos, models.FarmPhoto{
				ID:        *row.FarmPhotoID,
				FarmID:    *row.FarmID,
				PhotoURL:  *row.PhotoURL,
				PhotoType: models.PhotoType(*row.PhotoType),
				TakenAt:   row.TakenAt,
			})
		}

	}

	return &farm, nil
}

func (r *FarmRepository) GetAll(ctx context.Context) ([]models.Farm, error) {
	query := `
		SELECT 
			id, owner_id, farm_name, farm_code,
			area_sqm, province, district, commune, address,
			crop_type, planting_date, expected_harvest_date,
			crop_type_verified, crop_type_verified_at,
			crop_type_verified_by, crop_type_confidence,
			land_certificate_number, land_certificate_url,
			land_ownership_verified, land_ownership_verified_at,
			has_irrigation, irrigation_type, soil_type,
			status, created_at, updated_at,
			ST_AsBinary(boundary) as boundary_wkb,
			ST_AsBinary(center_location) as center_wkb
		FROM farm 
		ORDER BY created_at DESC
	`

	var rows []farmRow
	err := r.db.SelectContext(ctx, &rows, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get farms: %w", err)
	}

	// Convert sang FarmResponse và unmarshal geometry
	farms := make([]models.Farm, 0, len(rows))
	for _, row := range rows {
		farm := row.Farm
		if err := r.unmarshalGeometry(&row, &farm); err != nil {
			log.Println("Error unmarshaling geometry:", err)
			return nil, err
		}

		farms = append(farms, farm)
	}

	return farms, nil
}

func (r *FarmRepository) GetByOwnerID(ctx context.Context, ownerID string) ([]models.Farm, error) {
	var rows []farmRow
	query := `
		SELECT 
			f.id, owner_id, farm_name, farm_code,
			area_sqm, province, district, commune, address,
			crop_type, planting_date, expected_harvest_date,
			crop_type_verified, crop_type_verified_at,
			crop_type_verified_by, crop_type_confidence,
			land_certificate_number, land_certificate_url,
			land_ownership_verified, land_ownership_verified_at,
			has_irrigation, irrigation_type, soil_type,
			status, f.created_at, f.updated_at,
			ST_AsBinary(boundary) as boundary_wkb,
			ST_AsBinary(center_location) as center_wkb,
			fp.id as farm_photo_id,
			farm_id,
			photo_url,
			photo_type,
			taken_at
		FROM farm f left join farm_photo fp on f.id = fp.farm_id 
		WHERE f.owner_id = $1
	`

	err := r.db.Select(&rows, query, ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("farm not found: %s", ownerID)
		}
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var results []models.Farm
	farmMap := make(map[string]*models.Farm)

	for _, row := range rows {
		farm, exists := farmMap[row.ID.String()]
		if !exists {
			farm = &models.Farm{
				ID:                      row.ID,
				OwnerID:                 row.OwnerID,
				FarmName:                row.FarmName,
				FarmCode:                row.FarmCode,
				AreaSqm:                 row.AreaSqm,
				Province:                row.Province,
				District:                row.District,
				Commune:                 row.Commune,
				Address:                 row.Address,
				CropType:                row.CropType,
				PlantingDate:            row.PlantingDate,
				ExpectedHarvestDate:     row.ExpectedHarvestDate,
				CropTypeVerified:        row.CropTypeVerified,
				CropTypeVerifiedAt:      row.CropTypeVerifiedAt,
				CropTypeVerifiedBy:      row.CropTypeVerifiedBy,
				CropTypeConfidence:      row.CropTypeConfidence,
				LandCertificateNumber:   row.LandCertificateNumber,
				LandCertificateURL:      row.LandCertificateURL,
				LandOwnershipVerified:   row.LandOwnershipVerified,
				LandOwnershipVerifiedAt: row.LandOwnershipVerifiedAt,
				HasIrrigation:           row.HasIrrigation,
				IrrigationType:          row.IrrigationType,
				SoilType:                row.SoilType,
				Status:                  row.Status,
				CreatedAt:               row.CreatedAt,
				UpdatedAt:               row.UpdatedAt,
			}

			if err := r.unmarshalGeometry(&row, farm); err != nil {
				log.Println("Error unmarshaling geometry:", err)
				return nil, err
			}
			farmMap[row.ID.String()] = farm
		}

		if row.FarmPhotoID != nil {
			photo := models.FarmPhoto{
				ID:        *row.FarmPhotoID,
				FarmID:    *row.FarmID,
				PhotoURL:  *row.PhotoURL,
				PhotoType: models.PhotoType(*row.PhotoType),
				TakenAt:   row.TakenAt,
			}
			farm.FarmPhotos = append(farm.FarmPhotos, photo)
		}

	}

	for _, farm := range farmMap {
		results = append(results, *farm)
	}
	return results, nil
}

func (r *FarmRepository) Update(farm *models.Farm) error {
	farm.UpdatedAt = time.Now()

	query := `
		UPDATE farm SET
			farm_name = :farm_name, farm_code = :farm_code, boundary = ST_GeomFromText(:boundary),
			center_location = ST_GeomFromText(:center_location), area_sqm = :area_sqm, province = :province,
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

	err := utils.ExecWithCheck(r.db, query, utils.ExecUpdate, "inactive", time.Now(), id)
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
			:id, :owner_id, :farm_name, :farm_code, ST_GeomFromText(:boundary), ST_GeomFromText(:center_location), :area_sqm,
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
			farm_name = :farm_name, farm_code = :farm_code, boundary = ST_GeomFromText(:boundary),
			center_location = ST_GeomFromText(:center_location), area_sqm = :area_sqm, province = :province,
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

func (r *FarmRepository) unmarshalGeometry(row *farmRow, farm *models.Farm) error {
	if len(row.BoundaryWKB) > 0 {
		boundaryGeom, err := wkb.Unmarshal(row.BoundaryWKB)
		if err != nil {
			return fmt.Errorf("unmarshal boundary: %w", err)
		}
		poly, ok := boundaryGeom.(*geom.Polygon)
		if !ok {
			log.Printf("error boundary is not a Polygon: %+v", boundaryGeom)
			return fmt.Errorf("boundary is not a Polygon")
		}

		coords := make([][][]float64, poly.NumLinearRings())
		for i := 0; i < poly.NumLinearRings(); i++ {
			ring := poly.LinearRing(i)
			ringCoords := make([][]float64, ring.NumCoords())
			for j := 0; j < ring.NumCoords(); j++ {
				coord := ring.Coord(j)
				ringCoords[j] = []float64{coord.X(), coord.Y()}
			}
			coords[i] = ringCoords
		}

		farm.Boundary = &models.GeoJSONPolygon{
			Type:        "Polygon",
			Coordinates: coords,
		}
	}

	if len(row.CenterWKB) > 0 {
		centerGeom, err := wkb.Unmarshal(row.CenterWKB)
		if err != nil {
			log.Printf("Error decoding center WKB: %v", err)
			return fmt.Errorf("unmarshal center: %w", err)
		}
		point, ok := centerGeom.(*geom.Point)
		if !ok {
			log.Printf("Error asserting center to Point")
			return fmt.Errorf("center is not a Point")
		}

		pointCoords := point.Coords()
		geoJSONPoint := models.GeoJSONPoint{
			Type:        "Point",
			Coordinates: []float64{pointCoords.X(), pointCoords.Y()},
		}
		farm.CenterLocation = &geoJSONPoint
	}

	return nil
}
