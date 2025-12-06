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
		return nil, fmt.Errorf("Redis not responding: %w", err)
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
	minuteKey := fmt.Sprintf("rate_limit:%s:minute", apiKey)
	hourKey := fmt.Sprintf("rate_limit:%s:hour", apiKey)

	minuteAllowed, minuteRemaining, err := rl.tokenBucket(ctx, minuteKey, limitPerMinute, 60)
	if err != nil {
		return false, 0, 0, fmt.Errorf("minute rate check failed: %w", err)
	}

	if !minuteAllowed {
		hourRemaining, _ := rl.getTokens(ctx, hourKey, limitPerHour, 3600)
		return false, minuteRemaining, hourRemaining, nil
	}

	hourAllowed, hourRemaining, err := rl.tokenBucket(ctx, hourKey, limitPerHour, 3600)
	if err != nil {
		return false, 0, 0, fmt.Errorf("hour rate check failed: %w", err)
	}

	if !hourAllowed {
		rl.refundToken(ctx, minuteKey)
		return false, minuteRemaining, hourRemaining, nil
	}

	return true, minuteRemaining, hourRemaining, nil
}

func (rl *RateLimiter) tokenBucket(ctx context.Context, key string, capacity int, refillSeconds int) (bool, int, error) {
	now := time.Now().Unix()
	tokensKey := key + ":tokens"
	timestampKey := key + ":timestamp"

	tokensVal, err1 := rl.client.Get(ctx, tokensKey).Int()
	timestampVal, err2 := rl.client.Get(ctx, timestampKey).Int64()

	var tokens int
	var lastRefill int64

	if err1 != nil || err2 != nil {
		tokens = capacity - 1
		lastRefill = now
	} else {
		tokens = tokensVal
		lastRefill = timestampVal

		elapsed := now - lastRefill
		refillRate := float64(capacity) / float64(refillSeconds)
		tokensToAdd := int(float64(elapsed) * refillRate)

		if tokensToAdd > 0 {
			tokens = tokens + tokensToAdd
			if tokens > capacity {
				tokens = capacity
			}
			lastRefill = now
		}

		if tokens <= 0 {
			return false, 0, nil
		}

		tokens--
	}

	pipe := rl.client.Pipeline()
	pipe.Set(ctx, tokensKey, tokens, time.Duration(refillSeconds*2)*time.Second)
	pipe.Set(ctx, timestampKey, lastRefill, time.Duration(refillSeconds*2)*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	return true, tokens, nil
}

func (rl *RateLimiter) getTokens(ctx context.Context, key string, capacity int, refillSeconds int) (int, error) {
	now := time.Now().Unix()
	tokensKey := key + ":tokens"
	timestampKey := key + ":timestamp"

	pipe := rl.client.Pipeline()
	getTokens := pipe.Get(ctx, tokensKey)
	getTimestamp := pipe.Get(ctx, timestampKey)
	_, err := pipe.Exec(ctx)

	tokens := capacity
	lastRefill := now

	if err == nil {
		if tokensVal, err := getTokens.Int(); err == nil {
			tokens = tokensVal
		}
		if timestampVal, err := getTimestamp.Int64(); err == nil {
			lastRefill = timestampVal
		}
	}

	elapsed := now - lastRefill
	refillRate := float64(capacity) / float64(refillSeconds)
	tokensToAdd := int(float64(elapsed) * refillRate)

	if tokensToAdd > 0 {
		tokens = tokens + tokensToAdd
		if tokens > capacity {
			tokens = capacity
		}
	}

	return tokens, nil
}

func (rl *RateLimiter) refundToken(ctx context.Context, key string) {
	tokensKey := key + ":tokens"
	rl.client.Incr(ctx, tokensKey)
}
