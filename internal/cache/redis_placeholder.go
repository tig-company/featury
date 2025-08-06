package cache

import (
	"context"
	"fmt"
	"time"
)

// This is a placeholder for Redis implementation
// TODO: Implement proper Redis client integration

type redisPlaceholder struct {
	fallback Cache
	metrics  CacheMetrics
}

// NewRedisPlaceholder creates a placeholder that falls back to memory cache
func NewRedisPlaceholder(metrics CacheMetrics) Cache {
	return &redisPlaceholder{
		fallback: NewMemoryCache(metrics),
		metrics:  metrics,
	}
}

func (r *redisPlaceholder) Get(ctx context.Context, key string) ([]byte, error) {
	return r.fallback.Get(ctx, key)
}

func (r *redisPlaceholder) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.fallback.Set(ctx, key, value, ttl)
}

func (r *redisPlaceholder) Delete(ctx context.Context, key string) error {
	return r.fallback.Delete(ctx, key)
}

func (r *redisPlaceholder) DeletePattern(ctx context.Context, pattern string) error {
	return r.fallback.DeletePattern(ctx, pattern)
}

func (r *redisPlaceholder) Exists(ctx context.Context, key string) (bool, error) {
	return r.fallback.Exists(ctx, key)
}

func (r *redisPlaceholder) Health(ctx context.Context) error {
	return fmt.Errorf("redis not implemented - using memory cache fallback")
}

func (r *redisPlaceholder) Close() error {
	return r.fallback.Close()
}