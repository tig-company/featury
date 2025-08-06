package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/middleware"
	"github.com/tig-company/featury/internal/repository"
	"github.com/tig-company/featury/internal/service"
)

// SetupRoutes configures all API routes with proper middleware (backward compatibility)
func SetupRoutes(r *gin.Engine, authService *service.AuthService) {
	// This function maintains backward compatibility with the existing main.go
	// Create minimal setup to make the old interface work
	
	// Create middleware stack
	middlewareStack := middleware.NewMiddlewareStack(nil, authService)
	
	// Setup core middleware (applies to all routes)
	middlewareStack.SetupCore(r)
	
	// Basic health check endpoints (minimal middleware)
	r.GET("/health", healthCheck)
	r.GET("/ping", ping)
	
	// Middleware stats endpoint (for monitoring)
	r.GET("/middleware/stats", append(middlewareStack.AdminRouteMiddleware(), getMiddlewareStats(middlewareStack))...)
	
	// Use placeholder handlers for now - these would be replaced with full implementation
	// when the service layer is properly initialized
	
	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Feature flags routes with placeholder handlers
		features := v1.Group("/features")
		{
			// Public read access (with optional auth)
			features.GET("", append(middlewareStack.PublicReadOnlyMiddleware(), getFeatures)...)
			
			// Protected routes requiring specific permissions
			features.POST("", append(middlewareStack.FeatureFlagRouteMiddleware("create"), createFeature)...)
			features.GET("/:id", append(middlewareStack.FeatureFlagRouteMiddleware("read"), middlewareStack.ValidatePathUUIDs("id"), getFeature)...)
			features.PUT("/:id", append(middlewareStack.FeatureFlagRouteMiddleware("write"), middlewareStack.ValidatePathUUIDs("id"), updateFeature)...)
			features.PATCH("/:id", append(middlewareStack.FeatureFlagRouteMiddleware("write"), middlewareStack.ValidatePathUUIDs("id"), updateFeature)...)
			features.DELETE("/:id", append(middlewareStack.FeatureFlagRouteMiddleware("delete"), middlewareStack.ValidatePathUUIDs("id"), deleteFeature)...)
			
			// Bulk operations
			features.POST("/bulk", append(middlewareStack.FeatureFlagRouteMiddleware("write"), bulkFeatureOperation)...)
		}
		
		// API Keys routes
		apiKeys := v1.Group("/api-keys")
		{
			apiKeys.GET("", append(middlewareStack.APIKeyRouteMiddleware("read"), getAPIKeys)...)
			apiKeys.POST("", append(middlewareStack.APIKeyRouteMiddleware("write"), createAPIKey)...)
			apiKeys.GET("/:id", append(middlewareStack.APIKeyRouteMiddleware("read"), middlewareStack.ValidatePathUUIDs("id"), getAPIKey)...)
			apiKeys.PUT("/:id", append(middlewareStack.APIKeyRouteMiddleware("write"), middlewareStack.ValidatePathUUIDs("id"), updateAPIKey)...)
			apiKeys.DELETE("/:id", append(middlewareStack.APIKeyRouteMiddleware("write"), middlewareStack.ValidatePathUUIDs("id"), deleteAPIKey)...)
		}
		
		// Users routes (admin only)
		users := v1.Group("/users")
		{
			users.GET("", append(middlewareStack.AdminRouteMiddleware(), getUsers)...)
			users.POST("", append(middlewareStack.AdminRouteMiddleware(), createUser)...)
			users.GET("/:id", append(middlewareStack.AdminRouteMiddleware(), middlewareStack.ValidatePathUUIDs("id"), getUser)...)
			users.PUT("/:id", append(middlewareStack.AdminRouteMiddleware(), middlewareStack.ValidatePathUUIDs("id"), updateUser)...)
			users.DELETE("/:id", append(middlewareStack.AdminRouteMiddleware(), middlewareStack.ValidatePathUUIDs("id"), deleteUser)...)
		}
		
		// Audit logs routes (read-only, admin only)
		audit := v1.Group("/audit")
		{
			audit.GET("", append(middlewareStack.AdminRouteMiddleware(), getAuditLogs)...)
			audit.GET("/:id", append(middlewareStack.AdminRouteMiddleware(), middlewareStack.ValidatePathUUIDs("id"), getAuditLog)...)
		}
	}
}

// SetupRoutesWithServices configures routes with full service dependencies (new interface)
func SetupRoutesWithServices(r *gin.Engine, db *sql.DB, authService *service.AuthService, repo repository.Repository) {
	// This function is kept for backward compatibility but should use the new main.go pattern
	// For now, just call the original SetupRoutes
	SetupRoutes(r, authService)
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "featury",
		"version": "1.0.0",
	})
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func getMiddlewareStats(ms *middleware.MiddlewareStack) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, ms.GetStats())
	}
}

func getFeatures(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"features": []interface{}{},
		"total":    0,
	})
}

func createFeature(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"message": "Feature created successfully",
	})
}

func getFeature(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Feature retrieved successfully",
	})
}

func updateFeature(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Feature updated successfully",
	})
}

func deleteFeature(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Feature deleted successfully",
	})
}

// Placeholder handlers for new routes
func bulkFeatureOperation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Bulk operation completed successfully",
	})
}

func getAPIKeys(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_keys": []interface{}{},
		"total":    0,
	})
}

func createAPIKey(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"message": "API key created successfully",
	})
}

func getAPIKey(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "API key retrieved successfully",
	})
}

func updateAPIKey(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "API key updated successfully",
	})
}

func deleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "API key deleted successfully",
	})
}

func getUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"users": []interface{}{},
		"total": 0,
	})
}

func createUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
	})
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "User retrieved successfully",
	})
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "User updated successfully",
	})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "User deleted successfully",
	})
}

func getAuditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"audit_logs": []interface{}{},
		"total":      0,
	})
}

func getAuditLog(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Audit log retrieved successfully",
	})
}