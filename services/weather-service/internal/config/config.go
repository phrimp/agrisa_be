package config

import "os"

type WeatherServiceConfig struct {
	APIKey               string
	XweatherClientID     string
	XweatherClientSecret string
}

func New() *WeatherServiceConfig {
	return &WeatherServiceConfig{
		APIKey:               getEnvOrDefault("WEATHER_API_KEY", ""),
		XweatherClientID:     getEnvOrDefault("XWEATHER_CLIENT_ID", ""),
		XweatherClientSecret: getEnvOrDefault("XWEATHER_CLIENT_SECRET", ""),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
