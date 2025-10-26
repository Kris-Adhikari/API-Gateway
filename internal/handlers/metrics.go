package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"api-gateway/internal/database"
	"api-gateway/internal/services"
)

type MetricsHandler struct {
	metricsCollector *services.MetricsCollector
	db               *database.DB
	rateLimiter      *services.RateLimiter
}

func NewMetricsHandler(metricsCollector *services.MetricsCollector, db *database.DB, rateLimiter *services.RateLimiter) *MetricsHandler {
	return &MetricsHandler{
		metricsCollector: metricsCollector,
		db:               db,
		rateLimiter:      rateLimiter,
	}
}

func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.metricsCollector.GetSnapshot()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

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

	if err := h.checkPostgreSQL(ctx); err != nil {
		health.Services["postgresql"] = "unhealthy: " + err.Error()
		health.Status = "degraded"
		log.Printf("[WARN] PostgreSQL health check failed: %v", err)
	} else {
		health.Services["postgresql"] = "healthy"
	}

	if err := h.checkRedis(ctx); err != nil {
		health.Services["redis"] = "unhealthy: " + err.Error()
		health.Status = "degraded"
		log.Printf("[WARN] Redis health check failed: %v", err)
	} else {
		health.Services["redis"] = "healthy"
	}

	statusCode := http.StatusOK
	if health.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

func (h *MetricsHandler) checkPostgreSQL(_ context.Context) error {
	_, err := h.db.ListAPIKeys()
	return err
}

func (h *MetricsHandler) checkRedis(ctx context.Context) error {
	return h.rateLimiter.GetClient().Ping(ctx).Err()
}
