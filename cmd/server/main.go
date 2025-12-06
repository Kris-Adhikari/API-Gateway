package main

import (
	"log"
	"net/http"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/database"
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"
	"api-gateway/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Couldn't load config: %v", err)
	}

	log.Printf("Starting on port %s → %s", cfg.Port, cfg.BackendURL)

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	rateLimiter, err := services.NewRateLimiter(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer rateLimiter.Close()

	cacheService := services.NewCacheService(rateLimiter.GetClient(), 60*time.Second)
	metricsCollector := services.NewMetricsCollector()

	authMiddleware := middleware.NewAuthMiddleware(db)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter, metricsCollector)
	cacheMiddleware := middleware.NewCacheMiddleware(cacheService, 60*time.Second, metricsCollector)

	proxyService := services.NewProxyService(cfg.BackendURL)

	proxyHandler := handlers.NewProxyHandler(proxyService, db, metricsCollector)
	adminHandler := handlers.NewAdminHandler(db)
	metricsHandler := handlers.NewMetricsHandler(metricsCollector, db, rateLimiter)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", metricsHandler.HealthCheck)
	mux.HandleFunc("/metrics", metricsHandler.GetMetrics)

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

	mux.Handle("/", authMiddleware.Middleware(rateLimitMiddleware.Middleware(cacheMiddleware.Middleware(proxyHandler))))

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	log.Printf("Ready")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
