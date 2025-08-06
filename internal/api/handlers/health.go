package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/api/dto"
	"github.com/tig-company/featury/internal/service"
)

// HealthHandlers contains all handlers for health check operations
type HealthHandlers struct {
	db    *sql.DB
	cache service.CacheService
}

// NewHealthHandlers creates a new health handlers instance
func NewHealthHandlers(db *sql.DB, cache service.CacheService) *HealthHandlers {
	return &HealthHandlers{
		db:    db,
		cache: cache,
	}
}

// HealthCheck handles GET /health - Basic health check endpoint
func (h *HealthHandlers) HealthCheck(c *gin.Context) {
	checks := make(map[string]string)
	status := "healthy"
	
	// Check database health
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		
		if err := h.db.PingContext(ctx); err != nil {
			checks["database"] = "unhealthy: " + err.Error()
			status = "unhealthy"
		} else {
			checks["database"] = "healthy"
		}
	} else {
		checks["database"] = "not configured"
	}
	
	// Check cache health
	if h.cache != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		
		if err := h.cache.Health(ctx); err != nil {
			checks["cache"] = "unhealthy: " + err.Error()
			// Cache failures are not critical for the service
			if status == "healthy" {
				status = "degraded"
			}
		} else {
			checks["cache"] = "healthy"
		}
	} else {
		checks["cache"] = "not configured"
	}
	
	// Determine overall health
	httpStatus := http.StatusOK
	if status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}
	
	response := dto.NewHealthCheckResponse(status, "featury", "1.0.0", checks)
	c.JSON(httpStatus, response)
}

// ReadinessCheck handles GET /ready - Readiness probe for Kubernetes
func (h *HealthHandlers) ReadinessCheck(c *gin.Context) {
	ready := true
	checks := make(map[string]string)
	
	// Check database readiness
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		
		if err := h.db.PingContext(ctx); err != nil {
			checks["database"] = "not ready: " + err.Error()
			ready = false
		} else {
			// Additional check: can we execute a simple query?
			var result int
			err := h.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				checks["database"] = "not ready: " + err.Error()
				ready = false
			} else {
				checks["database"] = "ready"
			}
		}
	} else {
		checks["database"] = "not configured"
		ready = false
	}
	
	status := "ready"
	httpStatus := http.StatusOK
	if !ready {
		status = "not ready"
		httpStatus = http.StatusServiceUnavailable
	}
	
	response := dto.NewHealthCheckResponse(status, "featury", "1.0.0", checks)
	c.JSON(httpStatus, response)
}

// LivenessCheck handles GET /live - Liveness probe for Kubernetes
func (h *HealthHandlers) LivenessCheck(c *gin.Context) {
	// Liveness check should be very simple and fast
	// It should only fail if the application is completely broken
	response := dto.NewHealthCheckResponse("alive", "featury", "1.0.0", nil)
	c.JSON(http.StatusOK, response)
}

// PingCheck handles GET /ping - Simple ping endpoint
func (h *HealthHandlers) PingCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now(),
		"service":   "featury",
	})
}

// DetailedHealthCheck handles GET /health/detailed - Comprehensive health check
func (h *HealthHandlers) DetailedHealthCheck(c *gin.Context) {
	checks := make(map[string]string)
	status := "healthy"
	
	// Database health with detailed info
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		
		if err := h.db.PingContext(ctx); err != nil {
			checks["database.connection"] = "failed: " + err.Error()
			status = "unhealthy"
		} else {
			checks["database.connection"] = "healthy"
			
			// Check database stats
			stats := h.db.Stats()
			if stats.OpenConnections > 0 {
				checks["database.open_connections"] = string(rune(stats.OpenConnections))
				checks["database.in_use"] = string(rune(stats.InUse))
				checks["database.idle"] = string(rune(stats.Idle))
			}
			
			// Test a simple query
			var count int
			err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
			if err != nil {
				checks["database.query_test"] = "failed: " + err.Error()
				if status == "healthy" {
					status = "degraded"
				}
			} else {
				checks["database.query_test"] = "healthy"
			}
		}
	} else {
		checks["database.connection"] = "not configured"
		status = "degraded"
	}
	
	// Cache health with detailed info
	if h.cache != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		
		if err := h.cache.Health(ctx); err != nil {
			checks["cache.connection"] = "failed: " + err.Error()
			if status == "healthy" {
				status = "degraded"
			}
		} else {
			checks["cache.connection"] = "healthy"
			
			// Test cache operations
			testKey := "health-check-" + time.Now().Format("20060102150405")
			testValue := []byte("test")
			
			if err := h.cache.Set(ctx, testKey, testValue, 60); err != nil {
				checks["cache.write_test"] = "failed: " + err.Error()
				if status == "healthy" {
					status = "degraded"
				}
			} else {
				checks["cache.write_test"] = "healthy"
				
				// Test read
				if _, err := h.cache.Get(ctx, testKey); err != nil {
					checks["cache.read_test"] = "failed: " + err.Error()
					if status == "healthy" {
						status = "degraded"
					}
				} else {
					checks["cache.read_test"] = "healthy"
				}
				
				// Cleanup
				h.cache.Delete(ctx, testKey)
			}
		}
	} else {
		checks["cache.connection"] = "not configured"
	}
	
	// System checks
	checks["memory.goroutines"] = "N/A" // Would need runtime.NumGoroutine() conversion
	checks["system.time"] = time.Now().Format(time.RFC3339)
	
	// Determine overall health
	httpStatus := http.StatusOK
	if status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	} else if status == "degraded" {
		httpStatus = http.StatusOK // Still return 200 for degraded
	}
	
	response := dto.NewHealthCheckResponse(status, "featury", "1.0.0", checks)
	c.JSON(httpStatus, response)
}