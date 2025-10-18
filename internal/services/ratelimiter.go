package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements token bucket rate limiting using Redis
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisURL string) (*RateLimiter, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RateLimiter{client: client}, nil
}

// Close closes the Redis connection
func (rl *RateLimiter) Close() error {
	return rl.client.Close()
}

// AllowRequest checks if a request should be allowed based on rate limits
// Returns (allowed bool, remainingMinute int, remainingHour int, error)
func (rl *RateLimiter) AllowRequest(ctx context.Context, apiKey string, limitPerMinute, limitPerHour int) (bool, int, int, error) {
	now := time.Now()

	// Keys for minute and hour buckets
	minuteKey := fmt.Sprintf("rate_limit:%s:minute:%d", apiKey, now.Unix()/60)
	hourKey := fmt.Sprintf("rate_limit:%s:hour:%d", apiKey, now.Unix()/3600)

	// Check and increment minute counter
	minuteCount, err := rl.incrementAndGet(ctx, minuteKey, 60*time.Second)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to check minute rate limit: %w", err)
	}

	// Check and increment hour counter
	hourCount, err := rl.incrementAndGet(ctx, hourKey, 3600*time.Second)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to check hour rate limit: %w", err)
	}

	// Calculate remaining tokens
	remainingMinute := limitPerMinute - minuteCount
	remainingHour := limitPerHour - hourCount

	// If either limit is exceeded, deny the request
	if minuteCount > limitPerMinute {
		return false, remainingMinute, remainingHour, nil
	}
	if hourCount > limitPerHour {
		return false, remainingMinute, remainingHour, nil
	}

	// Request is allowed
	return true, remainingMinute, remainingHour, nil
}

// incrementAndGet atomically increments a counter and returns the new value
// Sets TTL on first increment
func (rl *RateLimiter) incrementAndGet(ctx context.Context, key string, ttl time.Duration) (int, error) {
	pipe := rl.client.Pipeline()

	// Increment the counter
	incr := pipe.Incr(ctx, key)

	// Set expiration (only takes effect if key doesn't have one)
	pipe.Expire(ctx, key, ttl)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return int(incr.Val()), nil
}
