package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/api/handlers"
	apiMiddleware "github.com/tig-company/featury/internal/api/middleware"
	"github.com/tig-company/featury/internal/middleware"
	"github.com/tig-company/featury/internal/service"
)

// RouterConfig contains all dependencies needed for route setup
type RouterConfig struct {
	DB                  *sql.DB
	MiddlewareStack     *middleware.MiddlewareStack
	AuthService         *service.AuthService
	FeatureFlagService  service.FeatureFlagService
	CacheService        service.CacheService
	AuditService        service.AuditService
}

// SetupRoutes configures all API routes with proper middleware and handlers
func SetupRoutes(r *gin.Engine, config *RouterConfig) {
	// Initialize handlers
	featureFlagHandlers := handlers.NewFeatureFlagHandlers(config.FeatureFlagService, config.AuditService)
	healthHandlers := handlers.NewHealthHandlers(config.DB, config.CacheService)
	metricsHandlers := handlers.NewMetricsHandlers(config.DB, config.CacheService, config.MiddlewareStack, config.FeatureFlagService)
	
	// Setup global middleware
	setupGlobalMiddleware(r, config)
	
	// Setup health and utility endpoints (minimal middleware)
	setupHealthRoutes(r, healthHandlers)
	
	// Setup API v1 routes
	setupAPIV1Routes(r, config, featureFlagHandlers, metricsHandlers)
}

// setupGlobalMiddleware configures middleware that applies to all routes
func setupGlobalMiddleware(r *gin.Engine, config *RouterConfig) {
	// Core middleware from the existing middleware stack
	if config.MiddlewareStack != nil {
		config.MiddlewareStack.SetupCore(r)
	}
	
	// Additional API-specific middleware
	r.Use(apiMiddleware.ResponseMiddleware())
	r.Use(apiMiddleware.SecurityHeadersMiddleware())
	r.Use(apiMiddleware.ErrorHandlerMiddleware())
	
	// Not found handler
	r.NoRoute(apiMiddleware.NotFoundHandler())
	
	// Method not allowed handler
	r.NoMethod(apiMiddleware.MethodNotAllowedHandler())
}

// setupHealthRoutes configures health check and monitoring endpoints
func setupHealthRoutes(r *gin.Engine, healthHandlers *handlers.HealthHandlers) {
	// Basic health endpoints (no authentication required)
	r.GET("/health", healthHandlers.HealthCheck)
	r.GET("/health/live", healthHandlers.LivenessCheck)
	r.GET("/health/ready", healthHandlers.ReadinessCheck)
	r.GET("/health/detailed", healthHandlers.DetailedHealthCheck)
	r.GET("/ping", healthHandlers.PingCheck)
}

// setupAPIV1Routes configures the main API v1 routes
func setupAPIV1Routes(r *gin.Engine, config *RouterConfig, featureFlagHandlers *handlers.FeatureFlagHandlers, metricsHandlers *handlers.MetricsHandlers) {
	v1 := r.Group("/api/v1")
	v1.Use(apiMiddleware.JSONResponseMiddleware())
	v1.Use(apiMiddleware.CacheControlMiddleware())
	
	// Feature flags routes
	setupFeatureFlagRoutes(v1, config, featureFlagHandlers)
	
	// Metrics routes (admin only)
	setupMetricsRoutes(v1, config, metricsHandlers)
	
	// Utility routes
	setupUtilityRoutes(v1, config, featureFlagHandlers)
}

// setupFeatureFlagRoutes configures feature flag specific routes
func setupFeatureFlagRoutes(v1 *gin.RouterGroup, config *RouterConfig, handlers *handlers.FeatureFlagHandlers) {
	features := v1.Group("/features")
	
	// Public read access (with optional auth for filtering)
	features.GET("", 
		append(config.MiddlewareStack.PublicReadOnlyMiddleware(), 
			apiMiddleware.PaginationResponseMiddleware(),
			handlers.ListFeatureFlags)...)
	
	// Protected routes requiring specific permissions
	features.POST("", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("create"), 
			handlers.CreateFeatureFlag)...)
			
	features.GET("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("read"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			handlers.GetFeatureFlag)...)
			
	features.PUT("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("write"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			handlers.UpdateFeatureFlag)...)
			
	features.PATCH("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("write"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			handlers.UpdateFeatureFlag)...)
			
	features.DELETE("/:id", 
		append(config.MiddlewareStack.FeatureFlagRouteMiddleware("delete"), 
			config.MiddlewareStack.ValidatePathUUIDs("id"), 
			handlers.DeleteFeatureFlag)...)
	
	// Environment-specific operations
	envRoutes := features.Group("/:id/environments/:environment")
	envRoutes.Use(append(config.MiddlewareStack.FeatureFlagRouteMiddleware("write"),
		config.MiddlewareStack.ValidatePathUUIDs("id"))...)
	{
		envRoutes.POST("/toggle", handlers.ToggleEnvironment)
		envRoutes.POST("/rollout", handlers.UpdateRollout)
	}
}

// setupMetricsRoutes configures metrics and monitoring routes
func setupMetricsRoutes(v1 *gin.RouterGroup, config *RouterConfig, handlers *handlers.MetricsHandlers) {
	// Metrics routes (admin access required)
	metrics := v1.Group("/metrics")
	metrics.Use(config.MiddlewareStack.AdminRouteMiddleware()...)
	{
		metrics.GET("", handlers.GetMetrics)
		metrics.GET("/database", handlers.GetDatabaseMetrics)
		metrics.GET("/cache", handlers.GetCacheMetrics)
		metrics.GET("/middleware", handlers.GetMiddlewareMetrics)
	}
}

// setupUtilityRoutes configures utility endpoints
func setupUtilityRoutes(v1 *gin.RouterGroup, config *RouterConfig, featureFlagHandlers *handlers.FeatureFlagHandlers) {
	// Utility endpoints for listing available options
	v1.GET("/environments", 
		append(config.MiddlewareStack.PublicReadOnlyMiddleware(), 
			featureFlagHandlers.GetEnvironments)...)
			
	v1.GET("/services", 
		append(config.MiddlewareStack.PublicReadOnlyMiddleware(), 
			featureFlagHandlers.GetServices)...)
}