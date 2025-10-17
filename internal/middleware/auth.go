package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/yourusername/api-gateway/internal/database"
	"github.com/yourusername/api-gateway/internal/models"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const APIKeyContextKey contextKey = "api_key"

// AuthMiddleware validates API keys from the X-API-Key header
type AuthMiddleware struct {
	db *database.DB
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(db *database.DB) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

// Middleware wraps an http.Handler and validates API keys
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from X-API-Key header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			log.Printf("[WARN] Request without API key: %s %s", r.Method, r.URL.Path)
			http.Error(w, `{"error":"Missing API key. Please provide X-API-Key header."}`, http.StatusUnauthorized)
			return
		}

		// Validate API key against database
		key, err := m.db.GetAPIKeyByKey(apiKey)
		if err != nil {
			log.Printf("[ERROR] Database error validating API key: %v", err)
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
			return
		}

		if key == nil {
			log.Printf("[WARN] Invalid API key attempted: %s", apiKey)
			http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		if !key.IsActive {
			log.Printf("[WARN] Inactive API key attempted: %s", apiKey)
			http.Error(w, `{"error":"API key is inactive"}`, http.StatusUnauthorized)
			return
		}

		// Store API key in context for use by handlers
		ctx := context.WithValue(r.Context(), APIKeyContextKey, key)
		r = r.WithContext(ctx)

		// API key is valid, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

// GetAPIKeyFromContext retrieves the API key from the request context
func GetAPIKeyFromContext(ctx context.Context) *models.APIKey {
	if key, ok := ctx.Value(APIKeyContextKey).(*models.APIKey); ok {
		return key
	}
	return nil
}
