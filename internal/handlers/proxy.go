package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/api-gateway/internal/database"
	"github.com/yourusername/api-gateway/internal/models"
	"github.com/yourusername/api-gateway/internal/services"
)

// ProxyHandler handles incoming HTTP requests and forwards them to backend
type ProxyHandler struct {
	proxyService *services.ProxyService
	db           *database.DB
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(proxyService *services.ProxyService, db *database.DB) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
		db:           db,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Forward request to backend
	resp, err := h.proxyService.ForwardRequest(r)
	if err != nil {
		h.logRequest(r, http.StatusBadGateway, time.Since(start), "error")
		http.Error(w, "Bad Gateway: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response back to client
	err = h.proxyService.CopyResponse(w, resp)
	if err != nil {
		h.logRequest(r, http.StatusInternalServerError, time.Since(start), "error copying response")
		return
	}

	// Log successful request
	h.logRequest(r, resp.StatusCode, time.Since(start), "")
}

// logRequest logs request details to terminal and database
func (h *ProxyHandler) logRequest(r *http.Request, status int, duration time.Duration, errMsg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	durationMs := duration.Milliseconds()

	var level, symbol string
	if status >= 500 {
		level = "ERROR"
		symbol = "✗"
	} else if status >= 400 {
		level = "WARN"
		symbol = "✗"
	} else {
		level = "INFO"
		symbol = "→"
	}

	logMsg := fmt.Sprintf("[%s] %-5s %s %s %s %d %dms",
		timestamp,
		level,
		symbol,
		r.Method,
		r.URL.Path,
		status,
		durationMs,
	)

	if errMsg != "" {
		logMsg += fmt.Sprintf(" [%s]", errMsg)
	}

	log.Println(logMsg)

	// Log to database
	requestLog := &models.RequestLog{
		APIKeyID:       nil, // Will be set in Phase 3 when we add authentication
		Method:         r.Method,
		Path:           r.URL.Path,
		StatusCode:     status,
		ResponseTimeMs: int(durationMs),
		IPAddress:      getClientIP(r),
		UserAgent:      r.UserAgent(),
	}

	if err := h.db.LogRequest(requestLog); err != nil {
		log.Printf("Failed to log request to database: %v", err)
	}
}

// getClientIP extracts the client's IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	// RemoteAddr includes port, so we need to strip it
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
