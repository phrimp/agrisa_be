package models

// AgroPolygonRequest represents the request to create a polygon in Agro API
type AgroPolygonRequest struct {
	Name    string                 `json:"name"`
	GeoJSON map[string]interface{} `json:"geo_json"`
}

// AgroPolygonResponse represents the response from Agro API polygon creation
type AgroPolygonResponse struct {
	ID      string                 `json:"id"`
	GeoJSON map[string]interface{} `json:"geo_json"`
	Name    string                 `json:"name"`
	Center  []float64              `json:"center"`
	Area    float64                `json:"area"`
}

// PrecipitationDataPoint represents a single precipitation measurement
type PrecipitationDataPoint struct {
	Dt    int64   `json:"dt"`    // Unix timestamp
	Rain  float64 `json:"rain"`  // Precipitation in mm
	Count int     `json:"count"` // Number of measurements
}

// PrecipitationRequest represents the query parameters for precipitation endpoint
type PrecipitationRequest struct {
	PolygonID string  `form:"polygon_id"` // Optional: if provided, reuse existing polygon
	Lat1      float64 `form:"lat1" binding:"required_without=PolygonID,omitempty,min=-90,max=90"`
	Lon1      float64 `form:"lon1" binding:"required_without=PolygonID,omitempty,min=-180,max=180"`
	Lat2      float64 `form:"lat2" binding:"required_without=PolygonID,omitempty,min=-90,max=90"`
	Lon2      float64 `form:"lon2" binding:"required_without=PolygonID,omitempty,min=-180,max=180"`
	Lat3      float64 `form:"lat3" binding:"required_without=PolygonID,omitempty,min=-90,max=90"`
	Lon3      float64 `form:"lon3" binding:"required_without=PolygonID,omitempty,min=-180,max=180"`
	Lat4      float64 `form:"lat4" binding:"required_without=PolygonID,omitempty,min=-90,max=90"`
	Lon4      float64 `form:"lon4" binding:"required_without=PolygonID,omitempty,min=-180,max=180"`
	Start     int64   `form:"start" binding:"required,min=0"`
	End       int64   `form:"end" binding:"required,min=0"`
}

// PrecipitationResponse represents the complete response with polygon and precipitation data
type PrecipitationResponse struct {
	PolygonID         string                   `json:"polygon_id"`
	PolygonName       string                   `json:"polygon_name"`
	PolygonCenter     []float64                `json:"polygon_center"`
	PolygonArea       float64                  `json:"polygon_area"`
	PolygonReused     bool                     `json:"polygon_reused"`      // True if existing polygon was reused
	PolygonCreatedNew bool                     `json:"polygon_created_new"` // True if new polygon was created
	TimeRange         TimeRange                `json:"time_range"`
	PrecipitationData []PrecipitationDataPoint `json:"precipitation_data"`
	TotalRainfall     float64                  `json:"total_rainfall_mm"`
	DataPointCount    int                      `json:"data_point_count"`
}

type DataPoint struct {
	Dt    int64   `json:"dt"`    // Unix timestamp
	Data  float64 `json:"data"`  // Precipitation in mm
	Count int     `json:"count"` // Number of measurements
	Unit  string  `json:"unit"`
}
type UnifiedAPIResponse struct {
	PolygonID         string      `json:"polygon_id"`
	PolygonName       string      `json:"polygon_name"`
	PolygonCenter     []float64   `json:"polygon_center"`
	PolygonArea       float64     `json:"polygon_area"`
	PolygonReused     bool        `json:"polygon_reused"`      // True if existing polygon was reused
	PolygonCreatedNew bool        `json:"polygon_created_new"` // True if new polygon was created
	TimeRange         TimeRange   `json:"time_range"`
	Data              []DataPoint `json:"data"`
	TotalDataValue    float64     `json:"total_data_value"`
	DataPointCount    int         `json:"data_point_count"`
}

// TimeRange represents the start and end time of the query
type TimeRange struct {
	Start int64 `json:"start"` // Unix timestamp
	End   int64 `json:"end"`   // Unix timestamp
}

// AgroErrorResponse represents an error response from Agro API
type AgroErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
