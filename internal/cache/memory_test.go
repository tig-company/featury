package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache(t *testing.T) {
	metrics := NewCacheMetrics()
	cache := NewMemoryCache(metrics)
	defer cache.Close()

	ctx := context.Background()
	key := "test_key"
	value := []byte("test_value")
	ttl := 100 * time.Millisecond

	t.Run("Set and Get", func(t *testing.T) {
		err := cache.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		retrieved, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	t.Run("Get Non-existent Key", func(t *testing.T) {
		_, err := cache.Get(ctx, "non_existent_key")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("Exists", func(t *testing.T) {
		err := cache.Set(ctx, "exists_key", value, time.Minute)
		require.NoError(t, err)

		exists, err := cache.Exists(ctx, "exists_key")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = cache.Exists(ctx, "non_existent_key")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Delete", func(t *testing.T) {
		err := cache.Set(ctx, "delete_key", value, time.Minute)
		require.NoError(t, err)

		err = cache.Delete(ctx, "delete_key")
		require.NoError(t, err)

		_, err = cache.Get(ctx, "delete_key")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("Expiration", func(t *testing.T) {
		shortTTL := 50 * time.Millisecond
		err := cache.Set(ctx, "expire_key", value, shortTTL)
		require.NoError(t, err)

		// Should exist immediately
		retrieved, err := cache.Get(ctx, "expire_key")
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Should be expired
		_, err = cache.Get(ctx, "expire_key")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("Delete Pattern", func(t *testing.T) {
		// Set multiple keys with same prefix
		err := cache.Set(ctx, "prefix:key1", []byte("value1"), time.Minute)
		require.NoError(t, err)
		err = cache.Set(ctx, "prefix:key2", []byte("value2"), time.Minute)
		require.NoError(t, err)
		err = cache.Set(ctx, "other:key", []byte("value3"), time.Minute)
		require.NoError(t, err)

		// Delete keys with pattern
		err = cache.DeletePattern(ctx, "prefix:*")
		require.NoError(t, err)

		// Prefix keys should be gone
		_, err = cache.Get(ctx, "prefix:key1")
		assert.ErrorIs(t, err, ErrCacheMiss)
		_, err = cache.Get(ctx, "prefix:key2")
		assert.ErrorIs(t, err, ErrCacheMiss)

		// Other key should still exist
		retrieved, err := cache.Get(ctx, "other:key")
		require.NoError(t, err)
		assert.Equal(t, []byte("value3"), retrieved)
	})

	t.Run("Health Check", func(t *testing.T) {
		err := cache.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("Data Isolation", func(t *testing.T) {
		originalValue := []byte("original")
		err := cache.Set(ctx, "isolation_key", originalValue, time.Minute)
		require.NoError(t, err)

		retrieved, err := cache.Get(ctx, "isolation_key")
		require.NoError(t, err)

		// Modify retrieved data
		retrieved[0] = 'X'

		// Original data in cache should be unchanged
		retrieved2, err := cache.Get(ctx, "isolation_key")
		require.NoError(t, err)
		assert.Equal(t, originalValue, retrieved2)
	})
}