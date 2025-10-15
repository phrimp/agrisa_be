package config

import "os"

type ProfileServiceConfig struct {
	Port        string
	PostgresCfg PostgresConfig
	MinioCfg    MinioConfig
}

type PostgresConfig struct {
	DBname   string
	Username string
	Password string
	Host     string
	Port     string
}

type MinioConfig struct {
	MinioUrl         string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioLocation    string
	MinioSecure      string
	MinioResourceUrl string
}

func New() *ProfileServiceConfig {
	return &ProfileServiceConfig{
		Port: getEnvOrDefault("PROFILE_SERVICE_PORT", "8087"),
		PostgresCfg: PostgresConfig{
			DBname:   getEnvOrDefault("POSTGRES_DB", ""),
			Username: getEnvOrDefault("POSTGRES_USER", "user"),
			Password: getEnvOrDefault("POSTGRES_PASSWORD", "password"),
			Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
			Port:     getEnvOrDefault("POSTGRES_PORT", "5432"),
		},
		MinioCfg: MinioConfig{
			MinioUrl:         getEnvOrDefault("MINIO_ENDPOINT", "http://localhost:9407"),
			MinioAccessKey:   getEnvOrDefault("MINIO_ACCESS_KEY", "minio"),
			MinioSecretKey:   getEnvOrDefault("MINIO_SECRET_KEY", "minio123"),
			MinioLocation:    getEnvOrDefault("MINIO_LOCATION", "us-east-1"),
			MinioSecure:      getEnvOrDefault("MINIO_SECURE", "false"),
			MinioResourceUrl: getEnvOrDefault("MINIO_RESOURCE_URL", "http://localhost:9407/"),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
