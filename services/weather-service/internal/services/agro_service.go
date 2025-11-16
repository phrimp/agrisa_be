package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"weather-service/internal/config"
	"weather-service/internal/models"
)

type AgroService struct {
	cfg config.WeatherServiceConfig
}

type IAgroService interface {
	CreatePolygon(name string, coordinates [][2]float64) (*models.AgroPolygonResponse, error)
	GetPolygon(polygonID string) (*models.AgroPolygonResponse, error)
	GetAccumulatedPrecipitation(polygonID string, start, end int64) ([]models.PrecipitationDataPoint, error)
	CreatePolygonAndGetPrecipitation(coordinates [][2]float64, start, end int64) (*models.UnifiedAPIResponse, error)
	GetPrecipitationWithPolygonID(polygonID string, coordinates [][2]float64, start, end int64) (*models.UnifiedAPIResponse, error)
}

func NewAgroService(cfg config.WeatherServiceConfig) IAgroService {
	return &AgroService{cfg: cfg}
}

// CreatePolygon creates a polygon in Agro API and returns the polygon ID
func (a *AgroService) CreatePolygon(name string, coordinates [][2]float64) (*models.AgroPolygonResponse, error) {
	if a.cfg.AgroAPIKey == "" {
		log.Println("Agro API key not configured")
		return nil, fmt.Errorf("agro API key not configured")
	}

	// Convert coordinates to GeoJSON format
	// Note: Agro API expects [lon, lat] format
	geoJSONCoords := make([][]float64, len(coordinates))
	for i, coord := range coordinates {
		geoJSONCoords[i] = []float64{coord[0], coord[1]} // [lon, lat]
	}

	// Close the polygon by adding first coordinate at the end if not already closed
	if len(geoJSONCoords) > 0 {
		first := geoJSONCoords[0]
		last := geoJSONCoords[len(geoJSONCoords)-1]
		if first[0] != last[0] || first[1] != last[1] {
			geoJSONCoords = append(geoJSONCoords, first)
		}
	}

	requestBody := models.AgroPolygonRequest{
		Name: name,
		GeoJSON: map[string]any{
			"type":       "Feature",
			"properties": map[string]any{},
			"geometry": map[string]any{
				"type":        "Polygon",
				"coordinates": []any{geoJSONCoords},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error marshaling polygon request: %v", err)
		return nil, fmt.Errorf("failed to create request body")
	}

	url := fmt.Sprintf("%s/polygons?appid=%s", a.cfg.AgroAPIBaseURL, a.cfg.AgroAPIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling Agro API: %v", err)
		return nil, fmt.Errorf("failed to call Agro API")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		log.Printf("Agro API returned non-success status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("agro API error: %s", string(body))
	}

	var polygonResp models.AgroPolygonResponse
	if err := json.Unmarshal(body, &polygonResp); err != nil {
		log.Printf("Error unmarshaling polygon response: %v", err)
		return nil, fmt.Errorf("failed to parse response")
	}

	log.Printf("Successfully created polygon with ID: %s", polygonResp.ID)
	return &polygonResp, nil
}

// GetPolygon retrieves an existing polygon by ID from Agro API
func (a *AgroService) GetPolygon(polygonID string) (*models.AgroPolygonResponse, error) {
	if a.cfg.AgroAPIKey == "" {
		log.Println("Agro API key not configured")
		return nil, fmt.Errorf("agro API key not configured")
	}

	url := fmt.Sprintf("%s/polygons/%s?appid=%s", a.cfg.AgroAPIBaseURL, polygonID, a.cfg.AgroAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Error fetching polygon: %v", err)
		return nil, fmt.Errorf("failed to fetch polygon")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("Polygon not found: %s", polygonID)
		return nil, fmt.Errorf("polygon not found: %s", polygonID)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Agro API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("Agro API error: %s", string(body))
	}

	var polygonResp models.AgroPolygonResponse
	if err := json.Unmarshal(body, &polygonResp); err != nil {
		log.Printf("Error unmarshaling polygon response: %v", err)
		return nil, fmt.Errorf("failed to parse response")
	}

	log.Printf("Successfully retrieved polygon with ID: %s", polygonResp.ID)
	return &polygonResp, nil
}

// GetAccumulatedPrecipitation fetches precipitation data for a polygon
func (a *AgroService) GetAccumulatedPrecipitation(polygonID string, start, end int64) ([]models.PrecipitationDataPoint, error) {
	if a.cfg.AgroAPIKey == "" {
		log.Println("Agro API key not configured")
		return nil, fmt.Errorf("Agro API key not configured")
	}

	url := fmt.Sprintf("%s/weather/history/accumulated_precipitation?polyid=%s&start=%d&end=%d&appid=%s",
		a.cfg.AgroAPIBaseURL, polygonID, start, end, a.cfg.AgroAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Error fetching precipitation data: %v", err)
		return nil, fmt.Errorf("failed to fetch precipitation data")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Agro API returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("Agro API error: %s", string(body))
	}

	var precipData []models.PrecipitationDataPoint
	if err := json.Unmarshal(body, &precipData); err != nil {
		log.Printf("Error unmarshaling precipitation data: %v", err)
		return nil, fmt.Errorf("failed to parse response")
	}

	return precipData, nil
}

// CreatePolygonAndGetPrecipitation combines polygon creation and precipitation fetching
func (a *AgroService) CreatePolygonAndGetPrecipitation(coordinates [][2]float64, start, end int64) (*models.UnifiedAPIResponse, error) {
	// Generate polygon name based on timestamp
	polygonName := fmt.Sprintf("temp_polygon_%d", time.Now().Unix())

	// Create polygon
	polygonResp, err := a.CreatePolygon(polygonName, coordinates)
	if err != nil {
		return nil, err
	}

	// Fetch precipitation data
	precipData, err := a.GetAccumulatedPrecipitation(polygonResp.ID, start, end)
	if err != nil {
		return nil, err
	}

	// Convert PrecipitationDataPoint to DataPoint format
	dataPoints := make([]models.DataPoint, len(precipData))
	totalRainfall := 0.0
	for i, data := range precipData {
		dataPoints[i] = models.DataPoint{
			Dt:    data.Dt,
			Data:  data.Rain,
			Count: data.Count,
			Unit:  "mm",
		}
		totalRainfall += data.Rain
	}

	response := &models.UnifiedAPIResponse{
		PolygonID:         polygonResp.ID,
		PolygonName:       polygonResp.Name,
		PolygonCenter:     polygonResp.Center,
		PolygonArea:       polygonResp.Area,
		PolygonCreatedNew: true,
		PolygonReused:     false,
		TimeRange: models.TimeRange{
			Start: start,
			End:   end,
		},
		Data:           dataPoints,
		TotalDataValue: totalRainfall,
		DataPointCount: len(dataPoints),
	}

	return response, nil
}

// GetPrecipitationWithPolygonID fetches precipitation data, reusing existing polygon if provided,
// or creating a new one if polygon ID is empty or doesn't exist
func (a *AgroService) GetPrecipitationWithPolygonID(polygonID string, coordinates [][2]float64, start, end int64) (*models.UnifiedAPIResponse, error) {
	var polygonResp *models.AgroPolygonResponse
	var err error
	polygonReused := false
	polygonCreatedNew := false

	// Try to use existing polygon if ID is provided
	if polygonID != "" {
		log.Printf("Attempting to reuse existing polygon: %s", polygonID)
		polygonResp, err = a.GetPolygon(polygonID)
		if err != nil {
			log.Printf("Failed to retrieve polygon %s: %v. Will create new polygon.", polygonID, err)
			// Polygon doesn't exist or error occurred, will create new one
			polygonID = ""
		} else {
			log.Printf("Successfully reused existing polygon: %s", polygonID)
			polygonReused = true
		}
	}

	// Create new polygon if no ID provided or existing polygon not found
	if polygonID == "" {
		log.Println("Creating new polygon from coordinates")
		polygonName := fmt.Sprintf("temp_polygon_%d", time.Now().Unix())
		polygonResp, err = a.CreatePolygon(polygonName, coordinates)
		if err != nil {
			return nil, err
		}
		polygonCreatedNew = true
		log.Printf("Created new polygon with ID: %s", polygonResp.ID)
	}

	// Fetch precipitation data using the polygon ID
	precipData, err := a.GetAccumulatedPrecipitation(polygonResp.ID, start, end)
	if err != nil {
		return nil, err
	}

	// Convert PrecipitationDataPoint to DataPoint format
	dataPoints := make([]models.DataPoint, len(precipData))
	totalRainfall := 0.0
	for i, data := range precipData {
		dataPoints[i] = models.DataPoint{
			Dt:    data.Dt,
			Data:  data.Rain,
			Count: data.Count,
			Unit:  "mm",
		}
		totalRainfall += data.Rain
	}

	response := &models.UnifiedAPIResponse{
		PolygonID:         polygonResp.ID,
		PolygonName:       polygonResp.Name,
		PolygonCenter:     polygonResp.Center,
		PolygonArea:       polygonResp.Area,
		PolygonReused:     polygonReused,
		PolygonCreatedNew: polygonCreatedNew,
		TimeRange: models.TimeRange{
			Start: start,
			End:   end,
		},
		Data:           dataPoints,
		TotalDataValue: totalRainfall,
		DataPointCount: len(dataPoints),
	}

	return response, nil
}
