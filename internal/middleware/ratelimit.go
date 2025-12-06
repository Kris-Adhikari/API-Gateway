package middleware

import (
	"fmt"
	"log"
	"net/http"

	"api-gateway/internal/services"
)

type RateLimitMiddleware struct {
	rateLimiter      *services.RateLimiter
	metricsCollector *services.MetricsCollector
}

func NewRateLimitMiddleware(rateLimiter *services.RateLimiter, metricsCollector *services.MetricsCollector) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		rateLimiter:      rateLimiter,
		metricsCollector: metricsCollector,
	}
}

func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := GetAPIKeyFromContext(r.Context())
		if apiKey == nil {
			next.ServeHTTP(w, r)
			return
		}

		allowed, remainingMinute, remainingHour, err := m.rateLimiter.AllowRequest(
			r.Context(),
			apiKey.Key,
			apiKey.RateLimitPerMinute,
			apiKey.RateLimitPerHour,
		)

		if err != nil {
			log.Printf("Rate limiter error: %v", err)
			http.Error(w, `{"error":"Internal error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-RateLimit-Limit-Minute", fmt.Sprintf("%d", apiKey.RateLimitPerMinute))
		w.Header().Set("X-RateLimit-Limit-Hour", fmt.Sprintf("%d", apiKey.RateLimitPerHour))
		w.Header().Set("X-RateLimit-Remaining-Minute", fmt.Sprintf("%d", remainingMinute))
		w.Header().Set("X-RateLimit-Remaining-Hour", fmt.Sprintf("%d", remainingHour))

		if !allowed {
			log.Printf("Rate limited: %s", apiKey.Name)
			m.metricsCollector.RecordRateLimitHit()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"Rate limit exceeded"}`)
			return
		}

		next.ServeHTTP(w, r)
	})
}
