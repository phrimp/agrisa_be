package config

import "os"

type WeatherServiceConfig struct {
	APIKey               string
	XweatherClientID     string
	XweatherClientSecret string
	AgroAPIKey           string
	AgroAPIBaseURL       string
}

func New() *WeatherServiceConfig {
	return &WeatherServiceConfig{
		APIKey:               getEnvOrDefault("WEATHER_API_KEY", ""),
		XweatherClientID:     getEnvOrDefault("XWEATHER_CLIENT_ID", ""),
		XweatherClientSecret: getEnvOrDefault("XWEATHER_CLIENT_SECRET", ""),
		AgroAPIKey:           getEnvOrDefault("AGRO_API_KEY", ""),
		AgroAPIBaseURL:       getEnvOrDefault("AGRO_API_BASE_URL", "http://api.agromonitoring.com/agro/1.0"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
