package services

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

type ProxyService struct {
	backendURL string
	client     *http.Client
}

func NewProxyService(backendURL string) *ProxyService {
	return &ProxyService{
		backendURL: backendURL,
		client:     &http.Client{},
	}
}

func (p *ProxyService) ForwardRequest(r *http.Request) (*http.Response, error) {
	targetURL, err := url.Parse(p.backendURL)
	if err != nil {
		return nil, err
	}

	targetURL.Path = strings.TrimSuffix(targetURL.Path, "/") + r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		return nil, err
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	proxyReq.Host = targetURL.Host

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (p *ProxyService) CopyResponse(w http.ResponseWriter, resp *http.Response) error {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err := io.Copy(w, resp.Body)
	return err
}
