package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/yourusername/api-gateway/internal/models"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// Connect establishes a connection to PostgreSQL
func Connect(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Connected to PostgreSQL")

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// LogRequest inserts a request log into the database
func (db *DB) LogRequest(log *models.RequestLog) error {
	query := `
		INSERT INTO request_logs (api_key_id, method, path, status_code, response_time_ms, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := db.conn.QueryRow(
		query,
		log.APIKeyID,
		log.Method,
		log.Path,
		log.StatusCode,
		log.ResponseTimeMs,
		log.IPAddress,
		log.UserAgent,
		time.Now(),
	).Scan(&log.ID)

	if err != nil {
		return fmt.Errorf("failed to insert request log: %w", err)
	}

	return nil
}

// GetAPIKeyByKey retrieves an API key by its key string
func (db *DB) GetAPIKeyByKey(key string) (*models.APIKey, error) {
	query := `
		SELECT id, key, name, rate_limit_per_minute, rate_limit_per_hour, is_active, created_at
		FROM api_keys
		WHERE key = $1 AND is_active = true
	`

	apiKey := &models.APIKey{}
	err := db.conn.QueryRow(query, key).Scan(
		&apiKey.ID,
		&apiKey.Key,
		&apiKey.Name,
		&apiKey.RateLimitPerMinute,
		&apiKey.RateLimitPerHour,
		&apiKey.IsActive,
		&apiKey.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Key not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return apiKey, nil
}

// CreateAPIKey creates a new API key
func (db *DB) CreateAPIKey(apiKey *models.APIKey) error {
	query := `
		INSERT INTO api_keys (key, name, rate_limit_per_minute, rate_limit_per_hour, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := db.conn.QueryRow(
		query,
		apiKey.Key,
		apiKey.Name,
		apiKey.RateLimitPerMinute,
		apiKey.RateLimitPerHour,
		apiKey.IsActive,
		time.Now(),
	).Scan(&apiKey.ID)

	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// ListAPIKeys retrieves all API keys
func (db *DB) ListAPIKeys() ([]models.APIKey, error) {
	query := `
		SELECT id, key, name, rate_limit_per_minute, rate_limit_per_hour, is_active, created_at
		FROM api_keys
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []models.APIKey
	for rows.Next() {
		var apiKey models.APIKey
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.Key,
			&apiKey.Name,
			&apiKey.RateLimitPerMinute,
			&apiKey.RateLimitPerHour,
			&apiKey.IsActive,
			&apiKey.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

// DeleteAPIKey deletes an API key by ID
func (db *DB) DeleteAPIKey(id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// ToggleAPIKey toggles the is_active status of an API key
func (db *DB) ToggleAPIKey(id uuid.UUID) error {
	query := `UPDATE api_keys SET is_active = NOT is_active WHERE id = $1`

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to toggle API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}
