package middleware

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/api-gateway/internal/services"
)

// CacheMiddleware caches GET request responses
type CacheMiddleware struct {
	cacheService *services.CacheService
	cacheTTL     time.Duration
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(cacheService *services.CacheService, cacheTTL time.Duration) *CacheMiddleware {
	return &CacheMiddleware{
		cacheService: cacheService,
		cacheTTL:     cacheTTL,
	}
}

// responseWriter wraps http.ResponseWriter to capture response data
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Write to both the original response and our buffer
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Middleware wraps an http.Handler and caches GET responses
func (m *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[DEBUG] Cache middleware called for: %s %s", r.Method, r.URL.Path)

		// Only cache GET requests
		if r.Method != http.MethodGet {
			log.Printf("[DEBUG] Not a GET request, skipping cache")
			next.ServeHTTP(w, r)
			return
		}

		// Generate cache key
		cacheKey := m.cacheService.GenerateCacheKey(r.Method, r.URL.Path, r.URL.RawQuery)
		log.Printf("[DEBUG] Generated cache key: %s", cacheKey)

		// Try to get from cache (use background context to avoid request timeout issues)
		ctx := context.Background()
		cached, err := m.cacheService.Get(ctx, cacheKey)
		if err != nil {
			log.Printf("[WARN] Cache get error: %v", err)
			// Continue without cache on error
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("[DEBUG] Cache lookup result: cached=%v, err=%v", cached != nil, err)

		// Cache hit
		if cached != nil {
			log.Printf("[INFO] Cache HIT: %s %s", r.Method, r.URL.Path)
			w.Header().Set("X-Cache", "HIT")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(cached.StatusCode)
			w.Write(cached.Body)
			return
		}

		// Cache miss - capture response
		log.Printf("[INFO] Cache MISS: %s %s", r.Method, r.URL.Path)
		w.Header().Set("X-Cache", "MISS")

		// Wrap response writer to capture response
		rw := newResponseWriter(w)

		// Call next handler
		next.ServeHTTP(rw, r)

		// Only cache successful responses (200)
		log.Printf("[DEBUG] Response status: %d, body length: %d", rw.statusCode, rw.body.Len())
		if rw.statusCode == http.StatusOK && rw.body.Len() > 0 {
			cachedResp := &services.CachedResponse{
				StatusCode: rw.statusCode,
				Body:       rw.body.Bytes(),
			}
			log.Printf("[DEBUG] Caching %d bytes for key %s", len(cachedResp.Body), cacheKey)

			if err := m.cacheService.Set(ctx, cacheKey, cachedResp, m.cacheTTL); err != nil {
				log.Printf("[WARN] Failed to cache response: %v", err)
			} else {
				log.Printf("[INFO] Cached response: %s %s (TTL: %v)", r.Method, r.URL.Path, m.cacheTTL)
			}
		} else {
			log.Printf("[DEBUG] Not caching: statusCode=%d, bodyLen=%d", rw.statusCode, rw.body.Len())
		}
	})
}
