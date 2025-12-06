package middleware

import (
	"context"
	"log"
	"net/http"

	"api-gateway/internal/database"
	"api-gateway/internal/models"
)

type contextKey string

const APIKeyContextKey contextKey = "api_key"

type AuthMiddleware struct {
	db *database.DB
}

func NewAuthMiddleware(db *database.DB) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, `{"error":"Missing API key"}`, http.StatusUnauthorized)
			return
		}

		key, err := m.db.GetAPIKeyByKey(apiKey)
		if err != nil {
			log.Printf("Auth DB error: %v", err)
			http.Error(w, `{"error":"Internal error"}`, http.StatusInternalServerError)
			return
		}

		if key == nil {
			http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}

		if !key.IsActive {
			http.Error(w, `{"error":"API key inactive"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), APIKeyContextKey, key)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func GetAPIKeyFromContext(ctx context.Context) *models.APIKey {
	if key, ok := ctx.Value(APIKeyContextKey).(*models.APIKey); ok {
		return key
	}
	return nil
}
