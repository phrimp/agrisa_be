package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"weather-service/internal/config"
)

type WeatherService struct {
	cfg config.WeatherServiceConfig
}

type IWeatherService interface {
	// Define service methods here
	FetchWeatherData(lat, lon, exclude, units, lang string) (*WeatherResponse, error)
}

func NewWeatherService(cfg config.WeatherServiceConfig) IWeatherService {
	return &WeatherService{cfg: cfg}
}

type WeatherResponse struct {
	Lat            float64                  `json:"lat"`
	Lon            float64                  `json:"lon"`
	Timezone       string                   `json:"timezone"`
	TimezoneOffset int                      `json:"timezone_offset"`
	Current        map[string]interface{}   `json:"current,omitempty"`
	Minutely       []map[string]interface{} `json:"minutely,omitempty"`
	Hourly         []map[string]interface{} `json:"hourly,omitempty"`
	Daily          []map[string]interface{} `json:"daily,omitempty"`
	Alerts         []map[string]interface{} `json:"alerts,omitempty"`
}

func (w *WeatherService) FetchWeatherData(lat, lon, exclude, units, lang string) (*WeatherResponse, error) {
	var weather WeatherResponse

	appid := w.cfg.APIKey
	if appid == "" {
		log.Println("API key not configured")
		return nil, fmt.Errorf("API key not configured")
	}

	// Build the API URL
	url := fmt.Sprintf("https://api.openweathermap.org/data/3.0/onecall?lat=%s&lon=%s&appid=%s", lat, lon, appid)
	if exclude != "" {
		url += fmt.Sprintf("&exclude=%s", exclude)
	}
	if units != "" {
		url += fmt.Sprintf("&units=%s", units)
	}
	if lang != "" {
		url += fmt.Sprintf("&lang=%s", lang)
	}

	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching weather data: %v", err)
		return nil, fmt.Errorf("failed to call API")
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("failed to read response")
	}

	// Check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("API 3rd party returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API 3rd party error")
	}

	// Unmarshal the JSON response
	if err := json.Unmarshal(body, &weather); err != nil {
		log.Println("Error unmarshaling JSON:", err)
		return nil, fmt.Errorf("failed to parse JSON")
	}

	return &weather, nil
}
