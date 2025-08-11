package config

import "os"

type AuthServiceConfig struct {
	Port        string
	PostgresCfg PostgresConfig
	RabbitMQCfg RabbitMQConfig
	AuthCfg     AuthConfig
	RedisCfg    RedisConfig
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
	Address  string
	Password string
	DB       int
}

type AuthConfig struct {
	JWTSecret string
}

func New() *AuthServiceConfig {
	return &AuthServiceConfig{
		Port: os.Getenv("PORT"),
		PostgresCfg: PostgresConfig{
			DBname:   os.Getenv("DB_NAME"),
			Username: os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PWD"),
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
		},
		RabbitMQCfg: RabbitMQConfig{
			Username: os.Getenv("RABBITMQ_USER"),
			Password: os.Getenv("RABBITMQ_PWD"),
			Port:     os.Getenv("RABBITMQ_PORT"),
		},
		AuthCfg: AuthConfig{
			JWTSecret: os.Getenv("JWT_SECRECT"),
		},
	}
}
