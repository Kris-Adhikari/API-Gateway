package main

import (
	"log"
	"net/http"
	"time"

	"github.com/yourusername/api-gateway/internal/config"
	"github.com/yourusername/api-gateway/internal/database"
	"github.com/yourusername/api-gateway/internal/handlers"
	"github.com/yourusername/api-gateway/internal/middleware"
	"github.com/yourusername/api-gateway/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print startup info
	log.Printf("Starting API Gateway (Phase 6: Metrics & Observability)")
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Backend: %s", cfg.BackendURL)
	log.Println()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Printf("Connected to PostgreSQL")

	// Connect to Redis
	rateLimiter, err := services.NewRateLimiter(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rateLimiter.Close()
	log.Printf("Connected to Redis")

	// Initialize cache service (using same Redis client)
	cacheService := services.NewCacheService(rateLimiter.GetClient(), 60*time.Second)

	// Initialize metrics collector
	metricsCollector := services.NewMetricsCollector()

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(db)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter, metricsCollector)
	cacheMiddleware := middleware.NewCacheMiddleware(cacheService, 60*time.Second, metricsCollector)

	// Initialize proxy service
	proxyService := services.NewProxyService(cfg.BackendURL)

	// Initialize handlers
	proxyHandler := handlers.NewProxyHandler(proxyService, db, metricsCollector)
	adminHandler := handlers.NewAdminHandler(db)
	metricsHandler := handlers.NewMetricsHandler(metricsCollector, db, rateLimiter)

	// Create HTTP server with routes
	mux := http.NewServeMux()

	// Monitoring endpoints (no auth required)
	mux.HandleFunc("/health", metricsHandler.HealthCheck)
	mux.HandleFunc("/metrics", metricsHandler.GetMetrics)

	// Admin routes (no auth required for managing keys)
	mux.HandleFunc("/admin/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			adminHandler.CreateAPIKey(w, r)
		case http.MethodGet:
			adminHandler.ListAPIKeys(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/keys/delete", adminHandler.DeleteAPIKey)
	mux.HandleFunc("/admin/keys/toggle", adminHandler.ToggleAPIKey)

	// Gateway routes (require authentication, rate limiting, and caching)
	// Order: auth -> rate limit -> cache -> proxy
	mux.Handle("/", authMiddleware.Middleware(rateLimitMiddleware.Middleware(cacheMiddleware.Middleware(proxyHandler))))

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Start server
	log.Printf("API Gateway started on port %s", cfg.Port)
	log.Printf("Forwarding requests to %s", cfg.BackendURL)
	log.Printf("Admin endpoints available at /admin/keys")
	log.Printf("Ready to accept requests")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
