package services

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ProxyService handles HTTP request proxying to backend services
type ProxyService struct {
	backendURL string
	client     *http.Client
}

// NewProxyService creates a new proxy service
func NewProxyService(backendURL string) *ProxyService {
	return &ProxyService{
		backendURL: backendURL,
		client:     &http.Client{},
	}
}

// ForwardRequest forwards an incoming request to the backend and returns the response
func (p *ProxyService) ForwardRequest(r *http.Request) (*http.Response, error) {
	// Build the target URL
	targetURL, err := url.Parse(p.backendURL)
	if err != nil {
		return nil, err
	}

	// Combine backend URL with request path
	targetURL.Path = strings.TrimSuffix(targetURL.Path, "/") + r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	// Create new request to backend
	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set Host header to backend host
	proxyReq.Host = targetURL.Host

	// Send request to backend
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CopyResponse copies the backend response to the client
func (p *ProxyService) CopyResponse(w http.ResponseWriter, resp *http.Response) error {
	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy body
	_, err := io.Copy(w, resp.Body)
	return err
}
