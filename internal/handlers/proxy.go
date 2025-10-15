package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/api-gateway/internal/services"
)

// ProxyHandler handles incoming HTTP requests and forwards them to backend
type ProxyHandler struct {
	proxyService *services.ProxyService
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(proxyService *services.ProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
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

// logRequest logs request details to terminal
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
}
