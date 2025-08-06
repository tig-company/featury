package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/api/dto"
	"github.com/tig-company/featury/internal/middleware"
	"github.com/tig-company/featury/internal/service"
)

// MetricsHandlers contains all handlers for metrics operations
type MetricsHandlers struct {
	db               *sql.DB
	cache            service.CacheService
	middlewareStack  *middleware.MiddlewareStack
	featureFlagService service.FeatureFlagService
}

// NewMetricsHandlers creates a new metrics handlers instance
func NewMetricsHandlers(db *sql.DB, cache service.CacheService, middlewareStack *middleware.MiddlewareStack, featureFlagService service.FeatureFlagService) *MetricsHandlers {
	return &MetricsHandlers{
		db:                 db,
		cache:              cache,
		middlewareStack:    middlewareStack,
		featureFlagService: featureFlagService,
	}
}

// GetMetrics handles GET /metrics - Get system metrics
func (h *MetricsHandlers) GetMetrics(c *gin.Context) {
	metrics := make(map[string]interface{})
	
	// Runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	metrics["runtime"] = map[string]interface{}{
		"goroutines":           runtime.NumGoroutine(),
		"memory_alloc":         memStats.Alloc,
		"memory_total_alloc":   memStats.TotalAlloc,
		"memory_sys":           memStats.Sys,
		"memory_heap_alloc":    memStats.HeapAlloc,
		"memory_heap_sys":      memStats.HeapSys,
		"memory_heap_idle":     memStats.HeapIdle,
		"memory_heap_in_use":   memStats.HeapInuse,
		"gc_cycles":            memStats.NumGC,
		"last_gc_time":         time.Unix(0, int64(memStats.LastGC)).Format(time.RFC3339),
	}
	
	// Database metrics
	if h.db != nil {
		stats := h.db.Stats()
		metrics["database"] = map[string]interface{}{
			"open_connections":    stats.OpenConnections,
			"in_use":             stats.InUse,
			"idle":               stats.Idle,
			"wait_count":         stats.WaitCount,
			"wait_duration":      stats.WaitDuration.String(),
			"max_idle_closed":    stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		}
	}
	
	// Middleware metrics
	if h.middlewareStack != nil {
		middlewareStats := h.middlewareStack.GetStats()
		metrics["middleware"] = middlewareStats
	}
	
	// Cache metrics (if available)
	if h.cache != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		
		// Basic cache health check
		if err := h.cache.Health(ctx); err != nil {
			metrics["cache"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			metrics["cache"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}
	
	// Feature flag metrics
	if h.featureFlagService != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		
		// Get environment and service counts
		environments, err := h.featureFlagService.GetEnvironments(ctx)
		if err == nil {
			metrics["feature_flags"] = map[string]interface{}{
				"environment_count": len(environments),
				"environments":      environments,
			}
			
			services, err := h.featureFlagService.GetServices(ctx)
			if err == nil {
				metrics["feature_flags"].(map[string]interface{})["service_count"] = len(services)
				metrics["feature_flags"].(map[string]interface{})["services"] = services
			}
		}
	}
	
	// System metrics
	metrics["system"] = map[string]interface{}{
		"timestamp":    time.Now(),
		"uptime_since": time.Now().Add(-time.Since(time.Now())), // This would be calculated from application start time in real implementation
		"version":      "1.0.0",
		"service":      "featury",
	}
	
	response := dto.NewMetricsResponse(metrics)
	c.JSON(http.StatusOK, response)
}

// GetDatabaseMetrics handles GET /metrics/database - Get detailed database metrics
func (h *MetricsHandlers) GetDatabaseMetrics(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Database not configured",
		})
		return
	}
	
	metrics := make(map[string]interface{})
	
	// Database connection stats
	stats := h.db.Stats()
	metrics["connection_stats"] = map[string]interface{}{
		"max_open_connections":    stats.MaxOpenConnections,
		"open_connections":        stats.OpenConnections,
		"in_use":                  stats.InUse,
		"idle":                    stats.Idle,
		"wait_count":              stats.WaitCount,
		"wait_duration_ms":        stats.WaitDuration.Milliseconds(),
		"max_idle_closed":         stats.MaxIdleClosed,
		"max_idle_time_closed":    stats.MaxIdleTimeClosed,
		"max_lifetime_closed":     stats.MaxLifetimeClosed,
	}
	
	// Database performance test
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	start := time.Now()
	err := h.db.PingContext(ctx)
	pingDuration := time.Since(start)
	
	if err != nil {
		metrics["health"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		metrics["health"] = map[string]interface{}{
			"status":       "healthy",
			"ping_time_ms": pingDuration.Milliseconds(),
		}
		
		// Test query performance
		start = time.Now()
		var count int
		err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM feature_flags WHERE deleted_at IS NULL").Scan(&count)
		queryDuration := time.Since(start)
		
		if err != nil {
			metrics["query_performance"] = map[string]interface{}{
				"status": "failed",
				"error":  err.Error(),
			}
		} else {
			metrics["query_performance"] = map[string]interface{}{
				"status":           "success",
				"query_time_ms":    queryDuration.Milliseconds(),
				"feature_flag_count": count,
			}
		}
	}
	
	response := dto.NewMetricsResponse(metrics)
	c.JSON(http.StatusOK, response)
}

// GetCacheMetrics handles GET /metrics/cache - Get detailed cache metrics
func (h *MetricsHandlers) GetCacheMetrics(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache not configured",
		})
		return
	}
	
	metrics := make(map[string]interface{})
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	// Cache health check
	start := time.Now()
	err := h.cache.Health(ctx)
	healthDuration := time.Since(start)
	
	if err != nil {
		metrics["health"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		metrics["health"] = map[string]interface{}{
			"status":         "healthy",
			"health_check_ms": healthDuration.Milliseconds(),
		}
		
		// Test cache operations
		testKey := "metrics-test-" + time.Now().Format("20060102150405")
		testValue := []byte("test-value")
		
		// Write test
		start = time.Now()
		writeErr := h.cache.Set(ctx, testKey, testValue, 60)
		writeDuration := time.Since(start)
		
		metrics["write_performance"] = map[string]interface{}{
			"write_time_ms": writeDuration.Milliseconds(),
			"status":        "success",
		}
		
		if writeErr != nil {
			metrics["write_performance"].(map[string]interface{})["status"] = "failed"
			metrics["write_performance"].(map[string]interface{})["error"] = writeErr.Error()
		} else {
			// Read test
			start = time.Now()
			_, readErr := h.cache.Get(ctx, testKey)
			readDuration := time.Since(start)
			
			metrics["read_performance"] = map[string]interface{}{
				"read_time_ms": readDuration.Milliseconds(),
				"status":       "success",
			}
			
			if readErr != nil {
				metrics["read_performance"].(map[string]interface{})["status"] = "failed"
				metrics["read_performance"].(map[string]interface{})["error"] = readErr.Error()
			}
			
			// Cleanup
			h.cache.Delete(ctx, testKey)
		}
	}
	
	response := dto.NewMetricsResponse(metrics)
	c.JSON(http.StatusOK, response)
}

// GetMiddlewareMetrics handles GET /metrics/middleware - Get detailed middleware metrics
func (h *MetricsHandlers) GetMiddlewareMetrics(c *gin.Context) {
	if h.middlewareStack == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Middleware stack not configured",
		})
		return
	}
	
	stats := h.middlewareStack.GetStats()
	metrics := map[string]interface{}{
		"middleware_stats": stats,
		"timestamp":        time.Now(),
	}
	
	response := dto.NewMetricsResponse(metrics)
	c.JSON(http.StatusOK, response)
}