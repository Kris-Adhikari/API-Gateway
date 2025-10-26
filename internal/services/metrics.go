package services

import (
	"sync"
	"time"
)

type MetricsCollector struct {
	mu sync.RWMutex

	totalRequests    int64
	successRequests  int64
	errorRequests    int64
	cacheHits        int64
	cacheMisses      int64
	rateLimitHits    int64

	totalResponseTime int64

	startTime time.Time
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

func (mc *MetricsCollector) RecordRequest(responseTimeMs int, statusCode int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.totalRequests++
	mc.totalResponseTime += int64(responseTimeMs)

	if statusCode >= 200 && statusCode < 400 {
		mc.successRequests++
	} else {
		mc.errorRequests++
	}
}

func (mc *MetricsCollector) RecordCacheHit() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.cacheHits++
}

func (mc *MetricsCollector) RecordCacheMiss() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.cacheMisses++
}

func (mc *MetricsCollector) RecordRateLimitHit() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.rateLimitHits++
}

type MetricsSnapshot struct {
	UptimeSeconds       int64   `json:"uptime_seconds"`
	TotalRequests       int64   `json:"total_requests"`
	RequestsPerSecond   float64 `json:"requests_per_second"`
	AvgResponseTimeMs   float64 `json:"avg_response_time_ms"`
	ErrorRate           float64 `json:"error_rate"`
	CacheHitRate        float64 `json:"cache_hit_rate"`
	RateLimitHits       int64   `json:"rate_limit_hits"`
	Timestamp           string  `json:"timestamp"`
}

func (mc *MetricsCollector) GetSnapshot() *MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	uptime := time.Since(mc.startTime)
	uptimeSeconds := int64(uptime.Seconds())

	snapshot := &MetricsSnapshot{
		UptimeSeconds:     uptimeSeconds,
		TotalRequests:     mc.totalRequests,
		RateLimitHits:     mc.rateLimitHits,
		Timestamp:         time.Now().Format(time.RFC3339),
	}

	if uptimeSeconds > 0 {
		snapshot.RequestsPerSecond = float64(mc.totalRequests) / float64(uptimeSeconds)
	}

	if mc.totalRequests > 0 {
		snapshot.AvgResponseTimeMs = float64(mc.totalResponseTime) / float64(mc.totalRequests)
	}

	if mc.totalRequests > 0 {
		snapshot.ErrorRate = float64(mc.errorRequests) / float64(mc.totalRequests)
	}

	totalCacheRequests := mc.cacheHits + mc.cacheMisses
	if totalCacheRequests > 0 {
		snapshot.CacheHitRate = float64(mc.cacheHits) / float64(totalCacheRequests)
	}

	return snapshot
}

func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.totalRequests = 0
	mc.successRequests = 0
	mc.errorRequests = 0
	mc.cacheHits = 0
	mc.cacheMisses = 0
	mc.rateLimitHits = 0
	mc.totalResponseTime = 0
	mc.startTime = time.Now()
}
