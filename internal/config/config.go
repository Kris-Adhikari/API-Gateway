package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port        string
	BackendURL  string
	DatabaseURL string
	LogLevel    string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		BackendURL:  getEnv("BACKEND_URL", "https://jsonplaceholder.typicode.com"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/api_gateway?sslmode=disable"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
