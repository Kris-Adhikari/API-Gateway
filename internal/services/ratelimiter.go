package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(redisURL string) (*RateLimiter, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RateLimiter{client: client}, nil
}

func (rl *RateLimiter) Close() error {
	return rl.client.Close()
}

func (rl *RateLimiter) GetClient() *redis.Client {
	return rl.client
}

func (rl *RateLimiter) AllowRequest(ctx context.Context, apiKey string, limitPerMinute, limitPerHour int) (bool, int, int, error) {
	now := time.Now()

	minuteKey := fmt.Sprintf("rate_limit:%s:minute:%d", apiKey, now.Unix()/60)
	hourKey := fmt.Sprintf("rate_limit:%s:hour:%d", apiKey, now.Unix()/3600)

	minuteCount, err := rl.incrementAndGet(ctx, minuteKey, 60*time.Second)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to check minute rate limit: %w", err)
	}

	hourCount, err := rl.incrementAndGet(ctx, hourKey, 3600*time.Second)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to check hour rate limit: %w", err)
	}

	remainingMinute := limitPerMinute - minuteCount
	remainingHour := limitPerHour - hourCount

	if minuteCount > limitPerMinute {
		return false, remainingMinute, remainingHour, nil
	}
	if hourCount > limitPerHour {
		return false, remainingMinute, remainingHour, nil
	}

	return true, remainingMinute, remainingHour, nil
}

func (rl *RateLimiter) incrementAndGet(ctx context.Context, key string, ttl time.Duration) (int, error) {
	pipe := rl.client.Pipeline()

	incr := pipe.Incr(ctx, key)

	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return int(incr.Val()), nil
}
