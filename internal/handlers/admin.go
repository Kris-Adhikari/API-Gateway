package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"api-gateway/internal/database"
	"api-gateway/internal/models"
)

type AdminHandler struct {
	db *database.DB
}

func NewAdminHandler(db *database.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

type CreateAPIKeyRequest struct {
	Name               string `json:"name"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	RateLimitPerHour   int    `json:"rate_limit_per_hour"`
}

func (h *AdminHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"Name is required"}`, http.StatusBadRequest)
		return
	}
	if req.RateLimitPerMinute <= 0 {
		req.RateLimitPerMinute = 100
	}
	if req.RateLimitPerHour <= 0 {
		req.RateLimitPerHour = 5000
	}

	apiKey := &models.APIKey{
		Key:                uuid.New().String(),
		Name:               req.Name,
		RateLimitPerMinute: req.RateLimitPerMinute,
		RateLimitPerHour:   req.RateLimitPerHour,
		IsActive:           true,
		CreatedAt:          time.Now(),
	}

	if err := h.db.CreateAPIKey(apiKey); err != nil {
		log.Printf("[ERROR] Failed to create API key: %v", err)
		http.Error(w, `{"error":"Failed to create API key"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Created new API key: %s (%s)", apiKey.Name, apiKey.Key)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apiKey)
}

func (h *AdminHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	keys, err := h.db.ListAPIKeys()
	if err != nil {
		log.Printf("[ERROR] Failed to list API keys: %v", err)
		http.Error(w, `{"error":"Failed to list API keys"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (h *AdminHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, `{"error":"ID parameter is required"}`, http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"Invalid UUID"}`, http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteAPIKey(id); err != nil {
		log.Printf("[ERROR] Failed to delete API key: %v", err)
		http.Error(w, `{"error":"Failed to delete API key"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Deleted API key: %s", id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "API key deleted successfully"})
}

func (h *AdminHandler) ToggleAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, `{"error":"ID parameter is required"}`, http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"Invalid UUID"}`, http.StatusBadRequest)
		return
	}

	if err := h.db.ToggleAPIKey(id); err != nil {
		log.Printf("[ERROR] Failed to toggle API key: %v", err)
		http.Error(w, `{"error":"Failed to toggle API key"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] Toggled API key: %s", id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "API key toggled successfully"})
}
