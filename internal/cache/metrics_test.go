package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheMetrics(t *testing.T) {
	metrics := NewCacheMetrics()

	t.Run("Initial State", func(t *testing.T) {
		assert.Equal(t, 0.0, metrics.GetHitRate())
		stats := metrics.(*cacheMetrics).GetStats()
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
		assert.Equal(t, int64(0), stats.Errors)
	})

	t.Run("Hit Rate Calculation", func(t *testing.T) {
		metrics.Reset()
		
		// All misses
		metrics.IncMisses("key1")
		metrics.IncMisses("key2")
		assert.Equal(t, 0.0, metrics.GetHitRate())

		// 50% hit rate
		metrics.IncHits("key3")
		metrics.IncHits("key4")
		assert.Equal(t, 0.5, metrics.GetHitRate())

		// 80% hit rate
		metrics.IncHits("key5")
		metrics.IncHits("key6")
		metrics.IncHits("key7")
		assert.Equal(t, 5.0/7.0, metrics.GetHitRate())
	})

	t.Run("Error Tracking", func(t *testing.T) {
		metrics.Reset()
		
		metrics.IncErrors("key1", "get")
		metrics.IncErrors("key2", "set")
		metrics.IncErrors("key3", "get")

		stats := metrics.(*cacheMetrics).GetStats()
		assert.Equal(t, int64(3), stats.Errors)
		assert.Equal(t, int64(2), stats.Operations["get"].ErrorCount)
		assert.Equal(t, int64(1), stats.Operations["set"].ErrorCount)
	})

	t.Run("Latency Recording", func(t *testing.T) {
		metrics.Reset()
		
		latency1 := 100 * time.Millisecond
		latency2 := 200 * time.Millisecond

		metrics.RecordLatency("get", latency1)
		metrics.RecordLatency("get", latency2)

		stats := metrics.(*cacheMetrics).GetStats()
		getStats := stats.Operations["get"]
		assert.Equal(t, int64(2), getStats.Count)
		
		expectedAvg := (latency1 + latency2) / 2
		assert.Equal(t, expectedAvg, getStats.AvgLatency)
	})

	t.Run("Reset", func(t *testing.T) {
		metrics.IncHits("key1")
		metrics.IncMisses("key2")
		metrics.IncErrors("key3", "get")
		metrics.RecordLatency("set", 50*time.Millisecond)

		metrics.Reset()

		stats := metrics.(*cacheMetrics).GetStats()
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
		assert.Equal(t, int64(0), stats.Errors)
		assert.Equal(t, 0.0, metrics.GetHitRate())
		assert.Empty(t, stats.Operations)
	})

	t.Run("Multiple Operations", func(t *testing.T) {
		metrics.Reset()
		
		// Record different operations
		metrics.RecordLatency("get", 10*time.Millisecond)
		metrics.RecordLatency("set", 20*time.Millisecond)
		metrics.RecordLatency("delete", 5*time.Millisecond)
		metrics.RecordLatency("get", 30*time.Millisecond)

		stats := metrics.(*cacheMetrics).GetStats()
		
		assert.Equal(t, int64(2), stats.Operations["get"].Count)
		assert.Equal(t, int64(1), stats.Operations["set"].Count)
		assert.Equal(t, int64(1), stats.Operations["delete"].Count)
		
		assert.Equal(t, 20*time.Millisecond, stats.Operations["get"].AvgLatency)
		assert.Equal(t, 20*time.Millisecond, stats.Operations["set"].AvgLatency)
		assert.Equal(t, 5*time.Millisecond, stats.Operations["delete"].AvgLatency)
	})
}