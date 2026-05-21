package httpserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RequestLimiter gates expensive POST endpoints (ingest, chat).
type RequestLimiter interface {
	Allow(key string, capacity, perMinute int) bool
}

// NewRequestLimiter uses Redis when redisURL is set; otherwise in-memory (single replica).
func NewRequestLimiter(redisURL string, logger *slog.Logger) RequestLimiter {
	if redisURL == "" {
		if logger != nil {
			logger.Info("rate_limiter_backend", "backend", "memory")
		}
		return NewMemoryRateLimiter()
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		if logger != nil {
			logger.Warn("rate_limiter_redis_invalid_url", "error", err, "fallback", "memory")
		}
		return NewMemoryRateLimiter()
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		if logger != nil {
			logger.Warn("rate_limiter_redis_unavailable", "error", err, "fallback", "memory")
		}
		return NewMemoryRateLimiter()
	}
	if logger != nil {
		logger.Info("rate_limiter_backend", "backend", "redis")
	}
	return &redisRateLimiter{client: client, prefix: "codeatlas:rl:"}
}

// redisRateLimiter uses a fixed per-minute window (shared across API replicas).
type redisRateLimiter struct {
	client *redis.Client
	prefix string
}

func (r *redisRateLimiter) Allow(key string, capacity, perMinute int) bool {
	limit := perMinute
	if capacity > limit {
		limit = capacity
	}
	if limit < 1 {
		limit = 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	fullKey := r.prefix + key
	n, err := r.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return true
	}
	if n == 1 {
		_ = r.client.Expire(ctx, fullKey, time.Minute).Err()
	}
	return n <= int64(limit)
}
