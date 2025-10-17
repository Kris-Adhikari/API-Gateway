package main

import (
	"log"
	"net/http"

	"github.com/yourusername/api-gateway/internal/config"
	"github.com/yourusername/api-gateway/internal/database"
	"github.com/yourusername/api-gateway/internal/handlers"
	"github.com/yourusername/api-gateway/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print startup info
	log.Printf("Starting API Gateway (Phase 2: Database + Logging)")
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

	// Initialize proxy service
	proxyService := services.NewProxyService(cfg.BackendURL)

	// Initialize proxy handler with database
	proxyHandler := handlers.NewProxyHandler(proxyService, db)

	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle("/", proxyHandler)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Start server
	log.Printf("API Gateway started on port %s", cfg.Port)
	log.Printf("Forwarding requests to %s", cfg.BackendURL)
	log.Printf("Ready to accept requests")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
