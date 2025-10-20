package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/api-gateway/internal/database"
	"github.com/yourusername/api-gateway/internal/services"
)

// MetricsHandler handles metrics endpoints
type MetricsHandler struct {
	metricsCollector *services.MetricsCollector
	db               *database.DB
	rateLimiter      *services.RateLimiter
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(metricsCollector *services.MetricsCollector, db *database.DB, rateLimiter *services.RateLimiter) *MetricsHandler {
	return &MetricsHandler{
		metricsCollector: metricsCollector,
		db:               db,
		rateLimiter:      rateLimiter,
	}
}

// GetMetrics returns current system metrics
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.metricsCollector.GetSnapshot()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// HealthCheck checks the health of the system and its dependencies
func (h *MetricsHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Services:  make(map[string]string),
	}

	// Check PostgreSQL
	if err := h.checkPostgreSQL(ctx); err != nil {
		health.Services["postgresql"] = "unhealthy: " + err.Error()
		health.Status = "degraded"
		log.Printf("[WARN] PostgreSQL health check failed: %v", err)
	} else {
		health.Services["postgresql"] = "healthy"
	}

	// Check Redis
	if err := h.checkRedis(ctx); err != nil {
		health.Services["redis"] = "unhealthy: " + err.Error()
		health.Status = "degraded"
		log.Printf("[WARN] Redis health check failed: %v", err)
	} else {
		health.Services["redis"] = "healthy"
	}

	// Set appropriate status code
	statusCode := http.StatusOK
	if health.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// checkPostgreSQL checks if PostgreSQL is reachable
func (h *MetricsHandler) checkPostgreSQL(_ context.Context) error {
	// Try to list API keys (simple query to test connectivity)
	_, err := h.db.ListAPIKeys()
	return err
}

// checkRedis checks if Redis is reachable
func (h *MetricsHandler) checkRedis(ctx context.Context) error {
	// Try to ping Redis
	return h.rateLimiter.GetClient().Ping(ctx).Err()
}
