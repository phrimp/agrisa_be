package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// GeoJSONPolygon represents a GeoJSON Polygon type for API input/output
type GeoJSONPolygon struct {
	Type        string        `json:"type" binding:"required,eq=Polygon"`
	Coordinates [][][]float64 `json:"coordinates" binding:"required"`
}

// Value implements the driver.Valuer interface for GeoJSONPolygon
// Converts GeoJSON to WKT (Well-Known Text) format for PostGIS GEOMETRY(Polygon, 4326)
//
// Flow:
// GeoJSON → geom.Polygon → WKT string
// Example output: "SRID=4326;POLYGON((106.0 10.0, 106.1 10.0, 106.1 10.1, 106.0 10.1, 106.0 10.0))"
func (g *GeoJSONPolygon) Value() (driver.Value, error) {
	if g == nil || g.Type == "" {
		return nil, nil
	}

	// [Bước 1] Convert GeoJSON to go-geom Polygon
	geoJSONBytes, err := json.Marshal(g)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoJSON: %w", err)
	}

	var geometry geom.T
	if err := geojson.Unmarshal(geoJSONBytes, &geometry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GeoJSON: %w", err)
	}

	polygon, ok := geometry.(*geom.Polygon)
	if !ok {
		return nil, fmt.Errorf("geometry is not a Polygon")
	}

	// [Bước 2] Set SRID to 4326 (WGS84)
	polygon.SetSRID(4326)

	// [Bước 3] Convert to WKT (Well-Known Text) format
	// Output example: "POLYGON((106.0 10.0, 106.1 10.0, ...))"
	wktString, err := wkt.Marshal(polygon)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to WKT: %w", err)
	}

	// [Bước 4] Add SRID prefix for PostGIS
	// Final output: "SRID=4326;POLYGON((106.0 10.0, ...))"
	wktWithSRID := fmt.Sprintf("SRID=%d;%s", polygon.SRID(), wktString)

	return wktWithSRID, nil
}

// Scan implements the sql.Scanner interface for GeoJSONPolygon
// Converts PostGIS GEOMETRY to GeoJSON
func (g *GeoJSONPolygon) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan GeoJSONPolygon: expected []byte, got %T", value)
	}

	// Unmarshal EWKB to go-geom
	geometry, err := ewkb.Unmarshal(bytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal EWKB: %w", err)
	}

	polygon, ok := geometry.(*geom.Polygon)
	if !ok {
		return fmt.Errorf("scanned geometry is not a Polygon")
	}

	// Convert to GeoJSON
	geoJSONBytes, err := geojson.Marshal(polygon)
	if err != nil {
		return fmt.Errorf("failed to marshal to GeoJSON: %w", err)
	}

	return json.Unmarshal(geoJSONBytes, g)
}

// GeoJSONPoint represents a GeoJSON Point type for API input/output
type GeoJSONPoint struct {
	Type        string    `json:"type" binding:"required,eq=Point"`
	Coordinates []float64 `json:"coordinates" binding:"required,len=2"`
}

// Value implements the driver.Valuer interface for GeoJSONPoint
// Converts GeoJSON to WKT (Well-Known Text) format for PostGIS GEOGRAPHY(Point, 4326)
//
// Flow:
// GeoJSON → geom.Point → WKT string
// Example output: "SRID=4326;POINT(106.6297 10.8231)"
func (g *GeoJSONPoint) Value() (driver.Value, error) {
	if g == nil || g.Type == "" {
		return nil, nil
	}

	// [Bước 1] Convert GeoJSON to go-geom Point
	geoJSONBytes, err := json.Marshal(g)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GeoJSON: %w", err)
	}

	var geometry geom.T
	if err := geojson.Unmarshal(geoJSONBytes, &geometry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GeoJSON: %w", err)
	}

	point, ok := geometry.(*geom.Point)
	if !ok {
		return nil, fmt.Errorf("geometry is not a Point")
	}

	// [Bước 2] Set SRID to 4326 (WGS84)
	point.SetSRID(4326)

	// [Bước 3] Convert to WKT (Well-Known Text) format
	// Output example: "POINT(106.6297 10.8231)"
	wktString, err := wkt.Marshal(point)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to WKT: %w", err)
	}

	// [Bước 4] Add SRID prefix for PostGIS
	// Final output: "SRID=4326;POINT(106.6297 10.8231)"
	wktWithSRID := fmt.Sprintf("SRID=%d;%s", point.SRID(), wktString)

	return wktWithSRID, nil
}

// Scan implements the sql.Scanner interface for GeoJSONPoint
// Converts PostGIS GEOGRAPHY to GeoJSON
func (g *GeoJSONPoint) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan GeoJSONPoint: expected []byte, got %T", value)
	}

	// Unmarshal EWKB to go-geom
	geometry, err := ewkb.Unmarshal(bytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal EWKB: %w", err)
	}

	point, ok := geometry.(*geom.Point)
	if !ok {
		return fmt.Errorf("scanned geometry is not a Point")
	}

	// Convert to GeoJSON
	geoJSONBytes, err := geojson.Marshal(point)
	if err != nil {
		return fmt.Errorf("failed to marshal to GeoJSON: %w", err)
	}

	return json.Unmarshal(geoJSONBytes, g)
}
