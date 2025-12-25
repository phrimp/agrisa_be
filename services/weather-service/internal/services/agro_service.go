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
	GetForecastPrecipitation(polygonID string) ([]models.PrecipitationDataPoint, error)
	GetCurrentWeather(polygonID string) (*models.CurrentWeatherResponse, error)
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

// GetForecastPrecipitation fetches forecast precipitation data for a polygon (free tier)
// Returns precipitation data from 5-day forecast (available with free API key)
func (a *AgroService) GetForecastPrecipitation(polygonID string) ([]models.PrecipitationDataPoint, error) {
	if a.cfg.AgroAPIKey == "" {
		log.Println("Agro API key not configured")
		return nil, fmt.Errorf("Agro API key not configured")
	}

	url := fmt.Sprintf("%s/weather/forecast?polyid=%s&appid=%s",
		a.cfg.AgroAPIBaseURL, polygonID, a.cfg.AgroAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Error fetching forecast data: %v", err)
		return nil, fmt.Errorf("failed to fetch forecast data")
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

	var forecastData []models.ForecastWeatherResponse
	if err := json.Unmarshal(body, &forecastData); err != nil {
		log.Printf("Error unmarshaling forecast data: %v", err)
		return nil, fmt.Errorf("failed to parse response")
	}

	// Convert forecast data to precipitation data points
	precipData := make([]models.PrecipitationDataPoint, 0)
	for _, forecast := range forecastData {
		var precipitation float64

		// Check for rain data (in mm for 3-hour period)
		if forecast.Rain != nil && forecast.Rain["3h"] > 0 {
			precipitation += forecast.Rain["3h"]
		}

		// Check for snow data (in mm for 3-hour period)
		if forecast.Snow != nil && forecast.Snow["3h"] > 0 {
			precipitation += forecast.Snow["3h"]
		}

		// Only include data points with actual precipitation
		if precipitation > 0 {
			precipData = append(precipData, models.PrecipitationDataPoint{
				Dt:    forecast.Dt,
				Rain:  precipitation,
				Count: 1, // Single forecast data point
			})
		}
	}

	log.Printf("Retrieved %d precipitation data points from forecast", len(precipData))
	return precipData, nil
}

// GetCurrentWeather fetches current weather data for a polygon (free tier)
func (a *AgroService) GetCurrentWeather(polygonID string) (*models.CurrentWeatherResponse, error) {
	if a.cfg.AgroAPIKey == "" {
		log.Println("Agro API key not configured")
		return nil, fmt.Errorf("Agro API key not configured")
	}

	url := fmt.Sprintf("%s/weather?polyid=%s&appid=%s",
		a.cfg.AgroAPIBaseURL, polygonID, a.cfg.AgroAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Error fetching current weather: %v", err)
		return nil, fmt.Errorf("failed to fetch current weather")
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

	var currentWeather models.CurrentWeatherResponse
	if err := json.Unmarshal(body, &currentWeather); err != nil {
		log.Printf("Error unmarshaling current weather data: %v", err)
		return nil, fmt.Errorf("failed to parse response")
	}

	log.Printf("Successfully retrieved current weather for polygon: %s", polygonID)
	return &currentWeather, nil
}

// CreatePolygonAndGetPrecipitation combines polygon creation and precipitation fetching
// Note: Uses forecast data (free tier) instead of historical data (requires paid plan)
func (a *AgroService) CreatePolygonAndGetPrecipitation(coordinates [][2]float64, start, end int64) (*models.UnifiedAPIResponse, error) {
	// Generate polygon name based on timestamp
	polygonName := fmt.Sprintf("temp_polygon_%d", time.Now().Unix())

	// Create polygon
	polygonResp, err := a.CreatePolygon(polygonName, coordinates)
	if err != nil {
		return nil, err
	}

	// Fetch precipitation data from forecast (free tier compatible)
	precipData, err := a.GetForecastPrecipitation(polygonResp.ID)
	if err != nil {
		return nil, err
	}

	// Filter precipitation data by time range and convert to DataPoint format
	dataPoints := make([]models.DataPoint, 0)
	totalRainfall := 0.0
	for _, data := range precipData {
		// Only include data points within the requested time range
		if data.Dt >= start && data.Dt <= end {
			dataPoints = append(dataPoints, models.DataPoint{
				Dt:    data.Dt,
				Data:  data.Rain,
				Count: data.Count,
				Unit:  "mm",
			})
			totalRainfall += data.Rain
		}
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

	log.Printf("Created polygon %s with %d precipitation data points (total: %.2f mm)",
		polygonResp.ID, len(dataPoints), totalRainfall)
	return response, nil
}

// GetPrecipitationWithPolygonID fetches precipitation data, reusing existing polygon if provided,
// or creating a new one if polygon ID is empty or doesn't exist
// Note: Uses forecast data (free tier) instead of historical data (requires paid plan)
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

	// Fetch precipitation data from forecast (free tier compatible)
	precipData, err := a.GetForecastPrecipitation(polygonResp.ID)
	if err != nil {
		return nil, err
	}

	// Filter precipitation data by time range and convert to DataPoint format
	dataPoints := make([]models.DataPoint, 0)
	totalRainfall := 0.0
	for _, data := range precipData {
		// Only include data points within the requested time range
		if data.Dt >= start && data.Dt <= end {
			dataPoints = append(dataPoints, models.DataPoint{
				Dt:    data.Dt,
				Data:  data.Rain,
				Count: data.Count,
				Unit:  "mm",
			})
			totalRainfall += data.Rain
		}
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

	log.Printf("Retrieved %d precipitation data points for polygon %s (total: %.2f mm)",
		len(dataPoints), polygonResp.ID, totalRainfall)
	return response, nil
}
