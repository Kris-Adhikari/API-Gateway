package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	BackendURL  string
	DatabaseURL string
	RedisURL    string
	LogLevel    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		BackendURL:  getEnv("BACKEND_URL", "https://jsonplaceholder.typicode.com"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5433/api_gateway?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
