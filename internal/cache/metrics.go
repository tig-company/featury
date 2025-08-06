package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

type cacheMetrics struct {
	hits     int64
	misses   int64
	errors   int64
	
	// Per-operation metrics
	operations map[string]*operationMetrics
	mu         sync.RWMutex
}

type operationMetrics struct {
	count      int64
	totalTime  int64 // nanoseconds
	errorCount int64
}

// NewCacheMetrics creates a new cache metrics instance
func NewCacheMetrics() CacheMetrics {
	return &cacheMetrics{
		operations: make(map[string]*operationMetrics),
	}
}

// IncHits increments cache hit counter
func (m *cacheMetrics) IncHits(key string) {
	atomic.AddInt64(&m.hits, 1)
}

// IncMisses increments cache miss counter
func (m *cacheMetrics) IncMisses(key string) {
	atomic.AddInt64(&m.misses, 1)
}

// IncErrors increments cache error counter
func (m *cacheMetrics) IncErrors(key string, operation string) {
	atomic.AddInt64(&m.errors, 1)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.operations[operation] == nil {
		m.operations[operation] = &operationMetrics{}
	}
	atomic.AddInt64(&m.operations[operation].errorCount, 1)
}

// RecordLatency records cache operation latency
func (m *cacheMetrics) RecordLatency(operation string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.operations[operation] == nil {
		m.operations[operation] = &operationMetrics{}
	}
	
	metrics := m.operations[operation]
	atomic.AddInt64(&metrics.count, 1)
	atomic.AddInt64(&metrics.totalTime, duration.Nanoseconds())
}

// GetHitRate returns current cache hit rate
func (m *cacheMetrics) GetHitRate() float64 {
	hits := atomic.LoadInt64(&m.hits)
	misses := atomic.LoadInt64(&m.misses)
	total := hits + misses
	
	if total == 0 {
		return 0.0
	}
	
	return float64(hits) / float64(total)
}

// Reset resets all metrics
func (m *cacheMetrics) Reset() {
	atomic.StoreInt64(&m.hits, 0)
	atomic.StoreInt64(&m.misses, 0)
	atomic.StoreInt64(&m.errors, 0)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.operations = make(map[string]*operationMetrics)
}

// GetStats returns current cache statistics
func (m *cacheMetrics) GetStats() CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := CacheStats{
		Hits:    atomic.LoadInt64(&m.hits),
		Misses:  atomic.LoadInt64(&m.misses),
		Errors:  atomic.LoadInt64(&m.errors),
		HitRate: m.GetHitRate(),
		Operations: make(map[string]OperationStats),
	}
	
	for operation, metrics := range m.operations {
		count := atomic.LoadInt64(&metrics.count)
		totalTime := atomic.LoadInt64(&metrics.totalTime)
		errorCount := atomic.LoadInt64(&metrics.errorCount)
		
		var avgLatency time.Duration
		if count > 0 {
			avgLatency = time.Duration(totalTime / count)
		}
		
		stats.Operations[operation] = OperationStats{
			Count:      count,
			ErrorCount: errorCount,
			AvgLatency: avgLatency,
		}
	}
	
	return stats
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits       int64                      `json:"hits"`
	Misses     int64                      `json:"misses"`
	Errors     int64                      `json:"errors"`
	HitRate    float64                    `json:"hit_rate"`
	Operations map[string]OperationStats `json:"operations"`
}

// OperationStats represents statistics for a specific cache operation
type OperationStats struct {
	Count      int64         `json:"count"`
	ErrorCount int64         `json:"error_count"`
	AvgLatency time.Duration `json:"avg_latency"`
}