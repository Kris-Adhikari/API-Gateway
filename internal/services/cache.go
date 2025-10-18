package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService handles response caching using Redis
type CacheService struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewCacheService creates a new cache service
func NewCacheService(client *redis.Client, defaultTTL time.Duration) *CacheService {
	return &CacheService{
		client:     client,
		defaultTTL: defaultTTL,
	}
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	ContentType string            `json:"content_type"`
}

// Get retrieves a cached response
func (cs *CacheService) Get(ctx context.Context, key string) (*CachedResponse, error) {
	data, err := cs.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		// Key doesn't exist - cache miss
		return nil, nil
	}
	if err != nil {
		// Some other error occurred
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	// For simplicity, we'll store the body directly
	// In production, you might want to use JSON or msgpack
	return &CachedResponse{
		StatusCode: 200,
		Body:       data,
	}, nil
}

// Set stores a response in the cache
func (cs *CacheService) Set(ctx context.Context, key string, response *CachedResponse, ttl time.Duration) error {
	if ttl == 0 {
		ttl = cs.defaultTTL
	}

	// Store the body directly
	err := cs.client.Set(ctx, key, response.Body, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// GenerateCacheKey creates a cache key from request details
func (cs *CacheService) GenerateCacheKey(method, path, query string) string {
	// Create a hash of method + path + query to handle long URLs
	data := fmt.Sprintf("%s:%s:%s", method, path, query)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("cache:%x", hash[:16]) // Use first 16 bytes for shorter keys
}

// Delete removes a cached response
func (cs *CacheService) Delete(ctx context.Context, key string) error {
	return cs.client.Del(ctx, key).Err()
}

// Clear removes all cached responses (useful for debugging)
func (cs *CacheService) Clear(ctx context.Context) error {
	// Delete all keys matching cache:*
	iter := cs.client.Scan(ctx, 0, "cache:*", 0).Iterator()
	for iter.Next(ctx) {
		if err := cs.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}
