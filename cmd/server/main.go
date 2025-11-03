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
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting API Gateway")
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Backend: %s", cfg.BackendURL)
	log.Println()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Printf("Connected to PostgreSQL")

	rateLimiter, err := services.NewRateLimiter(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rateLimiter.Close()
	log.Printf("Connected to Redis")

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

	log.Printf("API Gateway started on port %s", cfg.Port)
	log.Printf("Forwarding requests to %s", cfg.BackendURL)
	log.Printf("Admin endpoints available at /admin/keys")
	log.Printf("Ready to accept requests")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
