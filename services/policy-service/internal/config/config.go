package config

import "os"

type PolicyServiceConfig struct {
	Port        string
	PostgresCfg PostgresConfig
	RabbitMQCfg RabbitMQConfig
	RedisCfg    RedisConfig
	MinioCfg    MinioConfig
}

type MinioConfig struct {
	MinioURL         string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioLocation    string
	MinioSecure      string
	MinioResourceURL string
}

type PostgresConfig struct {
	DBname   string
	Username string
	Password string
	Host     string
	Port     string
}

type RabbitMQConfig struct {
	Username string
	Password string
	Port     string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func New() *PolicyServiceConfig {
	return &PolicyServiceConfig{
		Port: getEnvOrDefault("PORT", "8083"),
		PostgresCfg: PostgresConfig{
			DBname:   getEnvOrDefault("POSTGRES_DB", "agrisa"),
			Username: getEnvOrDefault("POSTGRES_USER", "postgres"),
			Password: getEnvOrDefault("POSTGRES_PASSWORD", "postgres"),
			Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
			Port:     getEnvOrDefault("POSTGRES_PORT", "5432"),
		},
		RabbitMQCfg: RabbitMQConfig{
			Username: getEnvOrDefault("RABBITMQ_USER", "admin"),
			Password: getEnvOrDefault("RABBITMQ_PWD", "admin"),
			Port:     getEnvOrDefault("RABBITMQ_PORT", "5672"),
		},
		RedisCfg: RedisConfig{
			Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:     getEnvOrDefault("REDIS_PORT", "6379"),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       0,
		},
		MinioCfg: MinioConfig{
			MinioURL:         getEnvOrDefault("MINIO_ENDPOINT", "http://localhost:9407"),
			MinioAccessKey:   getEnvOrDefault("MINIO_ACCESS_KEY", "minio"),
			MinioSecretKey:   getEnvOrDefault("MINIO_SECRET_KEY", "minio123"),
			MinioLocation:    getEnvOrDefault("MINIO_LOCATION", "us-east-1"),
			MinioSecure:      getEnvOrDefault("MINIO_SECURE", "false"),
			MinioResourceURL: getEnvOrDefault("MINIO_RESOURCE_URL", "http://localhost:9407/"),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
