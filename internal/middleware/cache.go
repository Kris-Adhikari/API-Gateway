package middleware

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"time"

	"api-gateway/internal/services"
)

type CacheMiddleware struct {
	cacheService     *services.CacheService
	cacheTTL         time.Duration
	metricsCollector *services.MetricsCollector
}

func NewCacheMiddleware(cacheService *services.CacheService, cacheTTL time.Duration, metricsCollector *services.MetricsCollector) *CacheMiddleware {
	return &CacheMiddleware{
		cacheService:     cacheService,
		cacheTTL:         cacheTTL,
		metricsCollector: metricsCollector,
	}
}

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
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (m *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		cacheKey := m.cacheService.GenerateCacheKey(r.Method, r.URL.Path, r.URL.RawQuery)
		ctx := context.Background()
		cached, err := m.cacheService.Get(ctx, cacheKey)

		if err != nil {
			log.Printf("[WARN] Cache get error: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		if cached != nil {
			log.Printf("[INFO] Cache HIT: %s %s", r.Method, r.URL.Path)
			m.metricsCollector.RecordCacheHit()
			w.Header().Set("X-Cache", "HIT")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(cached.StatusCode)
			w.Write(cached.Body)
			return
		}

		log.Printf("[INFO] Cache MISS: %s %s", r.Method, r.URL.Path)
		m.metricsCollector.RecordCacheMiss()
		w.Header().Set("X-Cache", "MISS")

		rw := newResponseWriter(w)
		next.ServeHTTP(rw, r)

		if rw.statusCode == http.StatusOK && rw.body.Len() > 0 {
			cachedResp := &services.CachedResponse{
				StatusCode: rw.statusCode,
				Body:       rw.body.Bytes(),
			}

			if err := m.cacheService.Set(ctx, cacheKey, cachedResp, m.cacheTTL); err != nil {
				log.Printf("[WARN] Failed to cache response: %v", err)
			} else {
				log.Printf("[INFO] Cached response: %s %s (TTL: %v)", r.Method, r.URL.Path, m.cacheTTL)
			}
		}
	})
}
