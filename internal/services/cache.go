package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client     *redis.Client
	defaultTTL time.Duration
}

func NewCacheService(client *redis.Client, defaultTTL time.Duration) *CacheService {
	return &CacheService{
		client:     client,
		defaultTTL: defaultTTL,
	}
}

type CachedResponse struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	ContentType string            `json:"content_type"`
}

func (cs *CacheService) Get(ctx context.Context, key string) (*CachedResponse, error) {
	data, err := cs.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	return &CachedResponse{
		StatusCode: 200,
		Body:       data,
	}, nil
}

func (cs *CacheService) Set(ctx context.Context, key string, response *CachedResponse, ttl time.Duration) error {
	if ttl == 0 {
		ttl = cs.defaultTTL
	}

	err := cs.client.Set(ctx, key, response.Body, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

func (cs *CacheService) GenerateCacheKey(method, path, query string) string {
	data := fmt.Sprintf("%s:%s:%s", method, path, query)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("cache:%x", hash[:16])
}

func (cs *CacheService) Delete(ctx context.Context, key string) error {
	return cs.client.Del(ctx, key).Err()
}

func (cs *CacheService) Clear(ctx context.Context) error {
	iter := cs.client.Scan(ctx, 0, "cache:*", 0).Iterator()
	for iter.Next(ctx) {
		if err := cs.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}
