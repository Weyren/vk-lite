package utils

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	JWTSecret   string
	JWTTTL      time.Duration
	PostgresDSN string
	RedisAddr   string
	AMQPURL     string
}

func NewConfig() *Config {
	ttlMinutes := getEnvInt("JWT_TTL_MINUTES", 60)

	return &Config{
		Port:        getEnv("APP_PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret"),
		JWTTTL:      time.Duration(ttlMinutes) * time.Minute,
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://vk:vk@localhost:5432/vk?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		AMQPURL:     getEnv("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
