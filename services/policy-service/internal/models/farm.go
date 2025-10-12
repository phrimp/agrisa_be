package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

// ============================================================================
// FARM MANAGEMENT
// ============================================================================

type Farm struct {
	ID                      uuid.UUID     `json:"id" db:"id"`
	OwnerID                 string        `json:"owner_id" db:"owner_id"`
	FarmName                *string       `json:"farm_name,omitempty" db:"farm_name"`
	FarmCode                *string       `json:"farm_code,omitempty" db:"farm_code"`
	Boundary                *geom.Polygon `json:"boundary,omitempty" db:"boundary"`
	CenterLocation          *geom.Point   `json:"center_location,omitempty" db:"center_location"`
	AreaSqm                 float64       `json:"area_sqm" db:"area_sqm"`
	Province                *string       `json:"province,omitempty" db:"province"`
	District                *string       `json:"district,omitempty" db:"district"`
	Commune                 *string       `json:"commune,omitempty" db:"commune"`
	Address                 *string       `json:"address,omitempty" db:"address"`
	CropType                string        `json:"crop_type" db:"crop_type"`
	PlantingDate            *int64        `json:"planting_date,omitempty" db:"planting_date"`
	ExpectedHarvestDate     *int64        `json:"expected_harvest_date,omitempty" db:"expected_harvest_date"`
	CropTypeVerified        bool          `json:"crop_type_verified" db:"crop_type_verified"`
	CropTypeVerifiedAt      *int64        `json:"crop_type_verified_at,omitempty" db:"crop_type_verified_at"`
	CropTypeVerifiedBy      *string       `json:"crop_type_verified_by,omitempty" db:"crop_type_verified_by"`
	CropTypeConfidence      *float64      `json:"crop_type_confidence,omitempty" db:"crop_type_confidence"`
	LandCertificateNumber   *string       `json:"land_certificate_number,omitempty" db:"land_certificate_number"`
	LandCertificateURL      *string       `json:"land_certificate_url,omitempty" db:"land_certificate_url"`
	LandOwnershipVerified   bool          `json:"land_ownership_verified" db:"land_ownership_verified"`
	LandOwnershipVerifiedAt *int64        `json:"land_ownership_verified_at,omitempty" db:"land_ownership_verified_at"`
	HasIrrigation           bool          `json:"has_irrigation" db:"has_irrigation"`
	IrrigationType          *string       `json:"irrigation_type,omitempty" db:"irrigation_type"`
	SoilType                *string       `json:"soil_type,omitempty" db:"soil_type"`
	Status                  FarmStatus    `json:"status" db:"status"`
	CreatedAt               time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at" db:"updated_at"`
}

type FarmPhoto struct {
	ID        uuid.UUID `json:"id" db:"id"`
	FarmID    uuid.UUID `json:"farm_id" db:"farm_id"`
	PhotoURL  string    `json:"photo_url" db:"photo_url"`
	PhotoType PhotoType `json:"photo_type" db:"photo_type"`
	TakenAt   *int64    `json:"taken_at,omitempty" db:"taken_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}