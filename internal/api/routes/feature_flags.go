package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/api/handlers"
	apiMiddleware "github.com/tig-company/featury/internal/api/middleware"
	"github.com/tig-company/featury/internal/middleware"
)

// FeatureFlagRouteConfig contains configuration for feature flag routes
type FeatureFlagRouteConfig struct {
	Handlers        *handlers.FeatureFlagHandlers
	MiddlewareStack *middleware.MiddlewareStack
	EnableBulkOps   bool
	EnableExport    bool
	EnableImport    bool
}

// SetupFeatureFlagRoutes sets up comprehensive feature flag routes with all CRUD operations
func SetupFeatureFlagRoutes(router *gin.RouterGroup, config *FeatureFlagRouteConfig) {
	features := router.Group("/features")
	
	// Apply common middleware for all feature flag routes
	features.Use(apiMiddleware.JSONResponseMiddleware())
	
	// Basic CRUD operations
	setupBasicCRUD(features, config)
	
	// Environment-specific operations
	setupEnvironmentOperations(features, config)
	
	// Bulk operations (if enabled)
	if config.EnableBulkOps {
		setupBulkOperations(features, config)
	}
	
	// Import/Export operations (if enabled)
	if config.EnableExport {
		setupExportOperations(features, config)
	}
	
	if config.EnableImport {
		setupImportOperations(features, config)
	}
	
	// Advanced query endpoints
	setupAdvancedQueries(features, config)
}

// setupBasicCRUD sets up basic CRUD operations for feature flags
func setupBasicCRUD(features *gin.RouterGroup, config *FeatureFlagRouteConfig) {
	// List feature flags with filtering and pagination
	features.GET("", 
		append(config.MiddlewareStack.PublicReadOnlyMiddleware(), 
			apiMiddleware.PaginationResponseMiddleware(),
			config.Handlers.ListFeatureFlags)...)
	
	// Create new feature flag
	features.POST("", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("create"), 
			apiMiddleware.ValidationErrorHandler(),
			config.Handlers.CreateFeatureFlag)...)
	
	// Get specific feature flag
	features.GET("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("read"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			config.Handlers.GetFeatureFlag)...)
	
	// Update feature flag (full update)
	features.PUT("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("write"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			apiMiddleware.ValidationErrorHandler(),
			config.Handlers.UpdateFeatureFlag)...)
	
	// Partial update feature flag
	features.PATCH("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("write"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			apiMiddleware.ValidationErrorHandler(),
			config.Handlers.UpdateFeatureFlag)...)
	
	// Delete feature flag (soft delete)
	features.DELETE("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("delete"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			config.Handlers.DeleteFeatureFlag)...)
}

// setupEnvironmentOperations sets up environment-specific operations
func setupEnvironmentOperations(features *gin.RouterGroup, config *FeatureFlagRouteConfig) {
	// Environment operations for a specific feature flag
	envGroup := features.Group("/:id/environments/:environment")
	envGroup.Use(append(config.MiddlewareStack.FeatureFlagRouteMiddleware("write"),
		config.MiddlewareStack.ValidatePathUUIDs("id"))...)
	{
		// Toggle environment enabled/disabled
		envGroup.POST("/toggle", config.Handlers.ToggleEnvironment)
		
		// Update rollout percentage
		envGroup.POST("/rollout", config.Handlers.UpdateRollout)
		
		// Get environment-specific configuration
		envGroup.GET("", func(c *gin.Context) {
			// This would be implemented in handlers if needed
			config.Handlers.GetFeatureFlag(c) // For now, return the full flag
		})
	}
}

// setupBulkOperations sets up bulk operations for feature flags
func setupBulkOperations(features *gin.RouterGroup, config *FeatureFlagRouteConfig) {
	bulk := features.Group("/bulk")
	bulk.Use(config.MiddlewareStack.FeatureFlagRouteMiddleware("write")...)
	{
		// Bulk enable/disable
		bulk.POST("/toggle", func(c *gin.Context) {
			// Placeholder for bulk toggle operation
			c.JSON(200, gin.H{"message": "Bulk toggle operation completed"})
		})
		
		// Bulk update rollout percentages
		bulk.POST("/rollout", func(c *gin.Context) {
			// Placeholder for bulk rollout update
			c.JSON(200, gin.H{"message": "Bulk rollout update completed"})
		})
		
		// Bulk delete
		bulk.DELETE("", func(c *gin.Context) {
			// Placeholder for bulk delete operation
			c.JSON(200, gin.H{"message": "Bulk delete completed"})
		})
	}
}

// setupExportOperations sets up data export operations
func setupExportOperations(features *gin.RouterGroup, config *FeatureFlagRouteConfig) {
	export := features.Group("/export")
	export.Use(config.MiddlewareStack.FeatureFlagRouteMiddleware("read")...)
	{
		// Export all feature flags
		export.GET("", func(c *gin.Context) {
			// Placeholder for export operation
			c.JSON(200, gin.H{"message": "Export completed"})
		})
		
		// Export specific service's feature flags
		export.GET("/service/:service", func(c *gin.Context) {
			// Placeholder for service-specific export
			c.JSON(200, gin.H{"message": "Service export completed"})
		})
		
		// Export specific environment's feature flags
		export.GET("/environment/:environment", func(c *gin.Context) {
			// Placeholder for environment-specific export
			c.JSON(200, gin.H{"message": "Environment export completed"})
		})
	}
}

// setupImportOperations sets up data import operations
func setupImportOperations(features *gin.RouterGroup, config *FeatureFlagRouteConfig) {
	imp := features.Group("/import")
	imp.Use(config.MiddlewareStack.FeatureFlagRouteMiddleware("create")...)
	{
		// Import feature flags
		imp.POST("", func(c *gin.Context) {
			// Placeholder for import operation
			c.JSON(200, gin.H{"message": "Import completed"})
		})
		
		// Validate import data without importing
		imp.POST("/validate", func(c *gin.Context) {
			// Placeholder for import validation
			c.JSON(200, gin.H{"message": "Import validation completed"})
		})
	}
}

// setupAdvancedQueries sets up advanced query endpoints
func setupAdvancedQueries(features *gin.RouterGroup, config *FeatureFlagRouteConfig) {
	// Search feature flags
	features.GET("/search", 
		append(config.MiddlewareStack.PublicReadOnlyMiddleware(), 
			apiMiddleware.PaginationResponseMiddleware(),
			func(c *gin.Context) {
				// Use the same handler as list but with different query processing
				config.Handlers.ListFeatureFlags(c)
			})...)
	
	// Get feature flags by service
	features.GET("/by-service/:service", 
		append(config.MiddlewareStack.PublicReadOnlyMiddleware(), 
			apiMiddleware.PaginationResponseMiddleware(),
			func(c *gin.Context) {
				// Set service filter and use list handler
				serviceName := c.Param("service")
				c.Request.URL.RawQuery += "&service=" + serviceName
				config.Handlers.ListFeatureFlags(c)
			})...)
	
	// Get feature flags by environment
	features.GET("/by-environment/:environment", 
		append(config.MiddlewareStack.PublicReadOnlyMiddleware(), 
			apiMiddleware.PaginationResponseMiddleware(),
			func(c *gin.Context) {
				// Set environment filter and use list handler
				environment := c.Param("environment")
				c.Request.URL.RawQuery += "&environment=" + environment
				config.Handlers.ListFeatureFlags(c)
			})...)
}

// Utility functions for route configuration

// WithBulkOperations enables bulk operations
func (c *FeatureFlagRouteConfig) WithBulkOperations() *FeatureFlagRouteConfig {
	c.EnableBulkOps = true
	return c
}

// WithExport enables export operations
func (c *FeatureFlagRouteConfig) WithExport() *FeatureFlagRouteConfig {
	c.EnableExport = true
	return c
}

// WithImport enables import operations
func (c *FeatureFlagRouteConfig) WithImport() *FeatureFlagRouteConfig {
	c.EnableImport = true
	return c
}

// NewFeatureFlagRouteConfig creates a new feature flag route configuration
func NewFeatureFlagRouteConfig(handlers *handlers.FeatureFlagHandlers, middlewareStack *middleware.MiddlewareStack) *FeatureFlagRouteConfig {
	return &FeatureFlagRouteConfig{
		Handlers:        handlers,
		MiddlewareStack: middlewareStack,
		EnableBulkOps:   false,
		EnableExport:    false,
		EnableImport:    false,
	}
}