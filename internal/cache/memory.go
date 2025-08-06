package cache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Cache errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)

type cacheItem struct {
	data      []byte
	expiresAt time.Time
}

type memoryCache struct {
	items   map[string]*cacheItem
	mu      sync.RWMutex
	metrics CacheMetrics
	ticker  *time.Ticker
	done    chan bool
}

// NewMemoryCache creates a new in-memory cache instance
func NewMemoryCache(metrics CacheMetrics) Cache {
	cache := &memoryCache{
		items:   make(map[string]*cacheItem),
		metrics: metrics,
		ticker:  time.NewTicker(time.Minute), // Clean expired items every minute
		done:    make(chan bool),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from cache
func (m *memoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer func() {
		m.metrics.RecordLatency("get", time.Since(start))
	}()

	m.mu.RLock()
	item, exists := m.items[key]
	m.mu.RUnlock()

	if !exists {
		m.metrics.IncMisses(key)
		return nil, ErrCacheMiss
	}

	// Check if item has expired
	if time.Now().After(item.expiresAt) {
		// Item has expired, remove it
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		
		m.metrics.IncMisses(key)
		return nil, ErrCacheMiss
	}

	m.metrics.IncHits(key)
	
	// Return a copy of the data to prevent external modification
	dataCopy := make([]byte, len(item.data))
	copy(dataCopy, item.data)
	return dataCopy, nil
}

// Set stores a value in cache with TTL
func (m *memoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		m.metrics.RecordLatency("set", time.Since(start))
	}()

	// Create a copy of the value to prevent external modification
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	item := &cacheItem{
		data:      valueCopy,
		expiresAt: time.Now().Add(ttl),
	}

	m.mu.Lock()
	m.items[key] = item
	m.mu.Unlock()

	return nil
}

// Delete removes a value from cache
func (m *memoryCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		m.metrics.RecordLatency("delete", time.Since(start))
	}()

	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()

	return nil
}

// DeletePattern removes all keys matching a pattern
func (m *memoryCache) DeletePattern(ctx context.Context, pattern string) error {
	start := time.Now()
	defer func() {
		m.metrics.RecordLatency("delete_pattern", time.Since(start))
	}()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert simple wildcard pattern to Go-compatible matching
	// For now, support only simple prefix patterns with * suffix
	isPrefix := strings.HasSuffix(pattern, "*")
	if isPrefix {
		prefix := strings.TrimSuffix(pattern, "*")
		keysToDelete := make([]string, 0)
		
		for key := range m.items {
			if strings.HasPrefix(key, prefix) {
				keysToDelete = append(keysToDelete, key)
			}
		}
		
		for _, key := range keysToDelete {
			delete(m.items, key)
		}
	} else {
		// Exact match
		delete(m.items, pattern)
	}

	return nil
}

// Exists checks if a key exists in cache
func (m *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		m.metrics.RecordLatency("exists", time.Since(start))
	}()

	m.mu.RLock()
	item, exists := m.items[key]
	m.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// Check if item has expired
	if time.Now().After(item.expiresAt) {
		// Item has expired, remove it
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return false, nil
	}

	return true, nil
}

// Health checks cache health
func (m *memoryCache) Health(ctx context.Context) error {
	// Memory cache is always healthy if it's running
	return nil
}

// Close closes the cache and stops cleanup goroutine
func (m *memoryCache) Close() error {
	m.ticker.Stop()
	close(m.done)
	
	m.mu.Lock()
	m.items = make(map[string]*cacheItem)
	m.mu.Unlock()
	
	return nil
}

// cleanup removes expired items from cache
func (m *memoryCache) cleanup() {
	for {
		select {
		case <-m.ticker.C:
			m.removeExpired()
		case <-m.done:
			return
		}
	}
}

// removeExpired removes all expired items from cache
func (m *memoryCache) removeExpired() {
	now := time.Now()
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	expiredKeys := make([]string, 0)
	for key, item := range m.items {
		if now.After(item.expiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	for _, key := range expiredKeys {
		delete(m.items, key)
	}
}