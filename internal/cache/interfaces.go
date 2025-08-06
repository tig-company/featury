package cache

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// DeletePattern removes all keys matching a pattern
	DeletePattern(ctx context.Context, pattern string) error

	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)

	// Health checks cache health
	Health(ctx context.Context) error

	// Close closes the cache connection
	Close() error
}

// CacheMetrics defines the interface for cache metrics
type CacheMetrics interface {
	// IncHits increments cache hit counter
	IncHits(key string)

	// IncMisses increments cache miss counter
	IncMisses(key string)

	// IncErrors increments cache error counter
	IncErrors(key string, operation string)

	// RecordLatency records cache operation latency
	RecordLatency(operation string, duration time.Duration)

	// GetHitRate returns current cache hit rate
	GetHitRate() float64

	// Reset resets all metrics
	Reset()
}