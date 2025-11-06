package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
)

// ============================================================================
// FARM MANAGEMENT
// ============================================================================

type Farm struct {
	ID                      uuid.UUID       `json:"id" db:"id"`
	OwnerID                 string          `json:"owner_id" db:"owner_id"`
	FarmName                *string         `json:"farm_name,omitempty" db:"farm_name"`
	FarmCode                *string         `json:"farm_code,omitempty" db:"farm_code"`
	Boundary                *GeoJSONPolygon `json:"boundary,omitempty" db:"boundary"`
	CenterLocation          *GeoJSONPoint   `json:"center_location,omitempty" db:"center_location"`
	AreaSqm                 float64         `json:"area_sqm" db:"area_sqm"`
	Province                *string         `json:"province,omitempty" db:"province"`
	District                *string         `json:"district,omitempty" db:"district"`
	Commune                 *string         `json:"commune,omitempty" db:"commune"`
	Address                 *string         `json:"address,omitempty" db:"address"`
	CropType                string          `json:"crop_type" db:"crop_type"`
	PlantingDate            *int64          `json:"planting_date,omitempty" db:"planting_date"`
	ExpectedHarvestDate     *int64          `json:"expected_harvest_date,omitempty" db:"expected_harvest_date"`
	CropTypeVerified        bool            `json:"crop_type_verified" db:"crop_type_verified"`
	CropTypeVerifiedAt      *int64          `json:"crop_type_verified_at,omitempty" db:"crop_type_verified_at"`
	CropTypeVerifiedBy      *string         `json:"crop_type_verified_by,omitempty" db:"crop_type_verified_by"`
	CropTypeConfidence      *float64        `json:"crop_type_confidence,omitempty" db:"crop_type_confidence"`
	LandCertificateNumber   *string         `json:"land_certificate_number,omitempty" db:"land_certificate_number"`
	LandCertificateURL      *string         `json:"land_certificate_url,omitempty" db:"land_certificate_url"`
	LandOwnershipVerified   bool            `json:"land_ownership_verified" db:"land_ownership_verified"`
	LandOwnershipVerifiedAt *int64          `json:"land_ownership_verified_at,omitempty" db:"land_ownership_verified_at"`
	HasIrrigation           bool            `json:"has_irrigation" db:"has_irrigation"`
	IrrigationType          *string         `json:"irrigation_type,omitempty" db:"irrigation_type"`
	SoilType                *string         `json:"soil_type,omitempty" db:"soil_type"`
	Status                  FarmStatus      `json:"status" db:"status"`
	CreatedAt               time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at" db:"updated_at"`
}

type FarmPhoto struct {
	ID        uuid.UUID `json:"id" db:"id"`
	FarmID    uuid.UUID `json:"farm_id" db:"farm_id"`
	PhotoURL  string    `json:"photo_url" db:"photo_url"`
	PhotoType PhotoType `json:"photo_type" db:"photo_type"`
	TakenAt   *int64    `json:"taken_at,omitempty" db:"taken_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type FarmResponse struct {
	ID                      string       `db:"id"`
	OwnerID                 string       `db:"owner_id"`
	FarmName                *string      `db:"farm_name"`
	FarmCode                *string      `db:"farm_code"`
	Boundary                *orb.Polygon `db:"boundary"`        // PostGIS POLYGON
	CenterLocation          *orb.Point   `db:"center_location"` // PostGIS POINT
	AreaSQM                 float64      `db:"area_sqm"`
	Province                *string      `db:"province"`
	District                *string      `db:"district"`
	Commune                 *string      `db:"commune"`
	Address                 *string      `db:"address"`
	CropType                *string      `db:"crop_type"`
	PlantingDate            *int64       `db:"planting_date"`
	ExpectedHarvestDate     *int64       `db:"expected_harvest_date"`
	CropTypeVerified        bool         `db:"crop_type_verified"`
	CropTypeVerifiedAt      *int64       `db:"crop_type_verified_at"`
	CropTypeVerifiedBy      *string      `db:"crop_type_verified_by"`
	CropTypeConfidence      *float64     `db:"crop_type_confidence"`
	LandCertificateNumber   *string      `db:"land_certificate_number"`
	LandCertificateURL      *string      `db:"land_certificate_url"`
	LandOwnershipVerified   bool         `db:"land_ownership_verified"`
	LandOwnershipVerifiedAt *int64       `db:"land_ownership_verified_at"`
	HasIrrigation           bool         `db:"has_irrigation"`
	IrrigationType          *string      `db:"irrigation_type"`
	SoilType                *string      `db:"soil_type"`
	Status                  FarmStatus   `db:"status"`
	CreatedAt               time.Time    `db:"created_at"`
	UpdatedAt               time.Time    `db:"updated_at"`
}
