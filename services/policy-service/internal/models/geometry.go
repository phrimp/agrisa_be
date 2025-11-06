package models

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
func (g *GeoJSONPolygon) Value() (driver.Value, error) {
	if g == nil || g.Type == "" {
		return nil, nil
	}

	// Convert GeoJSON to go-geom Polygon
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

	// Set SRID to 4326 (WGS84)
	polygon.SetSRID(4326)

	// Convert to WKT format
	wktString, err := wkt.Marshal(polygon)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to WKT: %w", err)
	}

	// Add SRID prefix
	wktWithSRID := fmt.Sprintf("SRID=%d;%s", polygon.SRID(), wktString)

	return wktWithSRID, nil
}

// Scan implements the sql.Scanner interface for GeoJSONPolygon
// Converts PostGIS GEOMETRY to GeoJSON
func (g *GeoJSONPolygon) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var ewkbData []byte

	switch v := value.(type) {
	case []byte:
		// Case 1: Already binary EWKB data
		ewkbData = v
		log.Printf("DEBUG Polygon: Received []byte, length=%d", len(v))

	case string:
		// Case 2: HEX-encoded string from PostgreSQL
		log.Printf("DEBUG Polygon: Received string, length=%d", len(v))

		hexStr := v

		// Strip \x prefix if exists (some drivers add this)
		if strings.HasPrefix(v, "\\x") {
			hexStr = v[2:]
			log.Printf("DEBUG Polygon: Stripped \\x prefix")
		}

		// Normalize to lowercase (hex.DecodeString accepts both but let's be consistent)
		hexStr = strings.ToLower(hexStr)

		// Validate it looks like hex
		if len(hexStr)%2 != 0 {
			return fmt.Errorf("invalid hex string length: %d (must be even)", len(hexStr))
		}

		// Decode HEX string to binary
		decoded, err := hex.DecodeString(hexStr)
		if err != nil {
			log.Printf("ERROR Polygon: Failed to decode hex string")
			log.Printf("ERROR Polygon: First 100 chars: %s", hexStr[:min(100, len(hexStr))])
			return fmt.Errorf("failed to decode hex string for Polygon: %w", err)
		}

		ewkbData = decoded
		log.Printf("DEBUG Polygon: Decoded to %d bytes, first byte: 0x%02x", len(decoded), decoded[0])

	default:
		return fmt.Errorf("unsupported type for Polygon scan: %T", value)
	}

	// Unmarshal EWKB binary to go-geom Polygon
	geometry, err := ewkb.Unmarshal(ewkbData)
	if err != nil {
		log.Printf("ERROR Polygon: EWKB unmarshal failed")
		log.Printf("ERROR Polygon: Data length: %d bytes", len(ewkbData))
		log.Printf("ERROR Polygon: First 20 bytes: %v", ewkbData[:min(20, len(ewkbData))])
		return fmt.Errorf("failed to unmarshal EWKB for Polygon: %w", err)
	}

	polygon, ok := geometry.(*geom.Polygon)
	if !ok {
		return fmt.Errorf("scanned geometry is not a Polygon, got %T", geometry)
	}

	// Convert to GeoJSON
	geoJSONBytes, err := geojson.Marshal(polygon)
	if err != nil {
		return fmt.Errorf("failed to marshal Polygon to GeoJSON: %w", err)
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
func (g *GeoJSONPoint) Value() (driver.Value, error) {
	if g == nil || g.Type == "" {
		return nil, nil
	}

	// Convert GeoJSON to go-geom Point
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

	// Set SRID to 4326 (WGS84)
	point.SetSRID(4326)

	// Convert to WKT format
	wktString, err := wkt.Marshal(point)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to WKT: %w", err)
	}

	// Add SRID prefix
	wktWithSRID := fmt.Sprintf("SRID=%d;%s", point.SRID(), wktString)

	return wktWithSRID, nil
}

// Scan implements the sql.Scanner interface for GeoJSONPoint
// Converts PostGIS GEOGRAPHY to GeoJSON
func (g *GeoJSONPoint) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var ewkbData []byte

	switch v := value.(type) {
	case []byte:
		// Case 1: Already binary EWKB data
		ewkbData = v
		log.Printf("DEBUG Point: Received []byte, length=%d", len(v))

	case string:
		// Case 2: HEX-encoded string from PostgreSQL
		log.Printf("DEBUG Point: Received string, length=%d", len(v))

		hexStr := v

		// Strip \x prefix if exists
		if strings.HasPrefix(v, "\\x") {
			hexStr = v[2:]
			log.Printf("DEBUG Point: Stripped \\x prefix")
		}

		// Normalize to lowercase
		hexStr = strings.ToLower(hexStr)

		// Validate hex format
		if len(hexStr)%2 != 0 {
			return fmt.Errorf("invalid hex string length: %d (must be even)", len(hexStr))
		}

		// Decode HEX string to binary
		decoded, err := hex.DecodeString(hexStr)
		if err != nil {
			log.Printf("ERROR Point: Failed to decode hex string")
			log.Printf("ERROR Point: First 100 chars: %s", hexStr[:min(100, len(hexStr))])
			return fmt.Errorf("failed to decode hex string for Point: %w", err)
		}

		ewkbData = decoded
		log.Printf("DEBUG Point: Decoded to %d bytes, first byte: 0x%02x", len(decoded), decoded[0])

	default:
		return fmt.Errorf("unsupported type for Point scan: %T", value)
	}

	// Unmarshal EWKB binary to go-geom Point
	geometry, err := ewkb.Unmarshal(ewkbData)
	if err != nil {
		log.Printf("ERROR Point: EWKB unmarshal failed")
		log.Printf("ERROR Point: Data length: %d bytes", len(ewkbData))
		log.Printf("ERROR Point: First 20 bytes: %v", ewkbData[:min(20, len(ewkbData))])
		return fmt.Errorf("failed to unmarshal EWKB for Point: %w", err)
	}

	point, ok := geometry.(*geom.Point)
	if !ok {
		return fmt.Errorf("scanned geometry is not a Point, got %T", geometry)
	}

	// Convert to GeoJSON
	geoJSONBytes, err := geojson.Marshal(point)
	if err != nil {
		return fmt.Errorf("failed to marshal Point to GeoJSON: %w", err)
	}

	return json.Unmarshal(geoJSONBytes, g)
}

// Helper function for safe min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
