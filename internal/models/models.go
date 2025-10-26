package models

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID                 uuid.UUID `json:"id"`
	Key                string    `json:"key"`
	Name               string    `json:"name"`
	RateLimitPerMinute int       `json:"rate_limit_per_minute"`
	RateLimitPerHour   int       `json:"rate_limit_per_hour"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
}

type RequestLog struct {
	ID             uuid.UUID  `json:"id"`
	APIKeyID       *uuid.UUID `json:"api_key_id,omitempty"`
	Method         string     `json:"method"`
	Path           string     `json:"path"`
	StatusCode     int        `json:"status_code"`
	ResponseTimeMs int        `json:"response_time_ms"`
	IPAddress      string     `json:"ip_address"`
	UserAgent      string     `json:"user_agent"`
	CreatedAt      time.Time  `json:"created_at"`
}

type BackendRoute struct {
	ID              uuid.UUID `json:"id"`
	PathPattern     string    `json:"path_pattern"`
	BackendURL      string    `json:"backend_url"`
	Method          string    `json:"method"`
	CacheTTLSeconds *int      `json:"cache_ttl_seconds,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}
