package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"api-gateway/internal/database"
	"api-gateway/internal/middleware"
	"api-gateway/internal/models"
	"api-gateway/internal/services"
)

type ProxyHandler struct {
	proxyService     *services.ProxyService
	db               *database.DB
	metricsCollector *services.MetricsCollector
}

func NewProxyHandler(proxyService *services.ProxyService, db *database.DB, metricsCollector *services.MetricsCollector) *ProxyHandler {
	return &ProxyHandler{
		proxyService:     proxyService,
		db:               db,
		metricsCollector: metricsCollector,
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	resp, err := h.proxyService.ForwardRequest(r)
	if err != nil {
		h.logRequest(r, http.StatusBadGateway, time.Since(start), "error")
		http.Error(w, "Bad Gateway: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	err = h.proxyService.CopyResponse(w, resp)
	if err != nil {
		h.logRequest(r, http.StatusInternalServerError, time.Since(start), "error copying response")
		return
	}

	h.logRequest(r, resp.StatusCode, time.Since(start), "")
}

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

	// Get API key from context if available
	apiKey := middleware.GetAPIKeyFromContext(r.Context())
	if apiKey != nil {
		logMsg += fmt.Sprintf(" [key:%s]", apiKey.Name)
	}

	if errMsg != "" {
		logMsg += fmt.Sprintf(" [%s]", errMsg)
	}

	log.Println(logMsg)

	requestLog := &models.RequestLog{
		APIKeyID:       nil,
		Method:         r.Method,
		Path:           r.URL.Path,
		StatusCode:     status,
		ResponseTimeMs: int(durationMs),
		IPAddress:      getClientIP(r),
		UserAgent:      r.UserAgent(),
	}

	if apiKey != nil {
		requestLog.APIKeyID = &apiKey.ID
	}

	if err := h.db.LogRequest(requestLog); err != nil {
		log.Printf("Failed to log request to database: %v", err)
	}

	h.metricsCollector.RecordRequest(int(durationMs), status)
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
