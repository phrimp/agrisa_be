package config

import "os"

type AuthServiceConfig struct {
	Port        string
	PostgresCfg PostgresConfig
	RabbitMQCfg RabbitMQConfig
	AuthCfg     AuthConfig
	RedisCfg    RedisConfig
	MinioCfg    MinioConfig
}

type MinioConfig struct {
	MinioUrl         string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioLocation    string
	MinioSecure      string
	MinioResourceUrl string
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

type AuthConfig struct {
	JWTSecret     string
	FptEkycApiKey string
}

func New() *AuthServiceConfig {
	return &AuthServiceConfig{
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
		AuthCfg: AuthConfig{
			JWTSecret:     getEnvOrDefault("JWT_SECRET", "default-secret"),
			FptEkycApiKey: getEnvOrDefault("FPT_EKYC_API_KEY", ""),
		},
		RedisCfg: RedisConfig{
			Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:     getEnvOrDefault("REDIS_PORT", "6379"),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       0,
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
