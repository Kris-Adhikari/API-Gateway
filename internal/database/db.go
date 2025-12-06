package database

import (
	"database/sql"
	"fmt"
	"time"

	"api-gateway/internal/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func Connect(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("couldn't open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("database not responding: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

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
		return fmt.Errorf("couldn't log request: %w", err)
	}

	return nil
}

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
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return apiKey, nil
}

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
		return fmt.Errorf("couldn't create API key: %w", err)
	}

	return nil
}

func (db *DB) ListAPIKeys() ([]models.APIKey, error) {
	query := `
		SELECT id, key, name, rate_limit_per_minute, rate_limit_per_hour, is_active, created_at
		FROM api_keys
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("couldn't list API keys: %w", err)
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
			return nil, fmt.Errorf("scan error: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

func (db *DB) DeleteAPIKey(id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("couldn't delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

func (db *DB) ToggleAPIKey(id uuid.UUID) error {
	query := `UPDATE api_keys SET is_active = NOT is_active WHERE id = $1`

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("couldn't toggle API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}
