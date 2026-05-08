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
	AmqpURL     string // пока не используем, но добавим для будущих событий
}

// NewConfig читает переменные окружения, задаёт sane‑defaults.
func NewConfig() *Config {
	ttl, _ := strconv.Atoi(getEnv("JWT_TTL_MINUTES", "60"))
	return &Config{
		Port:        getEnv("APP_PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me"),
		JWTTTL:      time.Duration(ttl) * time.Minute,
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://vk:vk@localhost:5432/vk?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		AmqpURL:     getEnv("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
