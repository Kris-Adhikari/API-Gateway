package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/yourusername/api-gateway/internal/services"
)

// RateLimitMiddleware enforces rate limiting per API key
type RateLimitMiddleware struct {
	rateLimiter      *services.RateLimiter
	metricsCollector *services.MetricsCollector
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(rateLimiter *services.RateLimiter, metricsCollector *services.MetricsCollector) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		rateLimiter:      rateLimiter,
		metricsCollector: metricsCollector,
	}
}

// Middleware wraps an http.Handler and enforces rate limits
func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from context (set by auth middleware)
		apiKey := GetAPIKeyFromContext(r.Context())
		if apiKey == nil {
			// No API key in context - this shouldn't happen after auth middleware
			// but we'll let it pass through as a safety measure
			next.ServeHTTP(w, r)
			return
		}

		// Check rate limit
		allowed, remainingMinute, remainingHour, err := m.rateLimiter.AllowRequest(
			r.Context(),
			apiKey.Key,
			apiKey.RateLimitPerMinute,
			apiKey.RateLimitPerHour,
		)

		if err != nil {
			log.Printf("[ERROR] Rate limiter error: %v", err)
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit-Minute", fmt.Sprintf("%d", apiKey.RateLimitPerMinute))
		w.Header().Set("X-RateLimit-Limit-Hour", fmt.Sprintf("%d", apiKey.RateLimitPerHour))
		w.Header().Set("X-RateLimit-Remaining-Minute", fmt.Sprintf("%d", remainingMinute))
		w.Header().Set("X-RateLimit-Remaining-Hour", fmt.Sprintf("%d", remainingHour))

		if !allowed {
			log.Printf("[WARN] Rate limit exceeded for API key: %s", apiKey.Name)
			m.metricsCollector.RecordRateLimitHit()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"Rate limit exceeded. Try again later."}`)
			return
		}

		// Rate limit not exceeded, proceed
		next.ServeHTTP(w, r)
	})
}
