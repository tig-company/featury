package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/service"
	"github.com/tig-company/featury/internal/validation"
	"github.com/tig-company/featury/pkg/errors"
)

// MiddlewareConfig contains configuration for all middleware
type MiddlewareConfig struct {
	Security    *SecurityConfig
	RateLimit   *RateLimiterConfig
	Environment string // development, staging, production
}

// DefaultMiddlewareConfig returns default middleware configuration
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		Security:    DefaultSecurityConfig(),
		RateLimit:   DefaultRateLimiterConfig(),
		Environment: "development",
	}
}

// ProductionMiddlewareConfig returns middleware configuration optimized for production
func ProductionMiddlewareConfig(allowedOrigins []string) *MiddlewareConfig {
	return &MiddlewareConfig{
		Security:    ProductionSecurityConfig(allowedOrigins),
		RateLimit:   DefaultRateLimiterConfig(),
		Environment: "production",
	}
}

// MiddlewareStack manages all middleware components
type MiddlewareStack struct {
	config       *MiddlewareConfig
	security     *SecurityMiddleware
	rateLimiter  *RateLimiter
	auth         *AuthMiddleware
	authService  *service.AuthService
}

// NewMiddlewareStack creates a new middleware stack
func NewMiddlewareStack(config *MiddlewareConfig, authService *service.AuthService) *MiddlewareStack {
	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	return &MiddlewareStack{
		config:      config,
		security:    NewSecurityMiddleware(config.Security),
		rateLimiter: NewRateLimiter(config.RateLimit),
		auth:        NewAuthMiddleware(authService),
		authService: authService,
	}
}

// SetupCore sets up core middleware that should be applied to all routes
func (ms *MiddlewareStack) SetupCore(r *gin.Engine) {
	// Error handling (should be first)
	r.Use(errors.ErrorHandler())
	
	// Recovery middleware (should be early in the chain)
	r.Use(gin.Recovery())
	
	// Request ID generation (should be early for logging)
	r.Use(ms.security.RequestID())
	
	// Logging (after request ID)
	if ms.config.Environment != "test" {
		r.Use(ms.security.Logger())
	}
	
	// Security headers
	r.Use(ms.security.SecurityHeaders())
	
	// CORS (should be before authentication)
	r.Use(ms.security.CORS())
	
	// Rate limiting (should be early to prevent abuse)
	r.Use(ms.rateLimiter.RateLimit())
}

// SetupPublic sets up middleware for public routes (no authentication required)
func (ms *MiddlewareStack) SetupPublic() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		// Optional authentication (allows both authenticated and unauthenticated requests)
		ms.auth.OptionalAuth(),
		
		// Input validation for query parameters
		validation.ValidateQueryParams(),
		
		// Input sanitization
		validation.SanitizeInput(),
	}
}

// SetupProtected sets up middleware for protected routes (authentication required)
func (ms *MiddlewareStack) SetupProtected() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		// Required authentication
		ms.auth.RequireAuth(),
		
		// Input validation for query parameters
		validation.ValidateQueryParams(),
		
		// Input sanitization
		validation.SanitizeInput(),
	}
}

// SetupProtectedWithPermissions sets up middleware for routes requiring specific permissions
func (ms *MiddlewareStack) SetupProtectedWithPermissions(permissions ...string) []gin.HandlerFunc {
	middleware := ms.SetupProtected()
	
	// Add permission-based middleware for each permission
	for _, permission := range permissions {
		// Split permission into resource and action (e.g., "feature_flags:read" -> resource="feature_flags", action="read")
		parts := strings.Split(permission, ":")
		if len(parts) == 2 {
			middleware = append(middleware, ms.auth.RequireResourceAccess(parts[0], parts[1]))
		}
	}
	
	return middleware
}

// RequirePermissions creates middleware that requires specific model permissions
func (ms *MiddlewareStack) RequirePermissions(permissions ...models.Permission) gin.HandlerFunc {
	return ms.auth.RequirePermission(permissions...)
}

// RequireResourceAccess creates middleware that requires access to a specific resource
func (ms *MiddlewareStack) RequireResourceAccess(resource, action string) gin.HandlerFunc {
	return ms.auth.RequireResourceAccess(resource, action)
}

// ValidatePathUUIDs creates middleware to validate UUID path parameters
func (ms *MiddlewareStack) ValidatePathUUIDs(paramNames ...string) gin.HandlerFunc {
	return validation.ValidatePathParams(paramNames...)
}

// ValidateJSONBody creates middleware to validate JSON request body
func (ms *MiddlewareStack) ValidateJSONBody(target interface{}) gin.HandlerFunc {
	return validation.ValidateJSONBody(target)
}

// GetStats returns statistics for all middleware components
func (ms *MiddlewareStack) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"rate_limiter": ms.rateLimiter.GetStats(),
		"environment":  ms.config.Environment,
	}
}

// Convenience methods for common middleware combinations

// APIRouteMiddleware returns middleware chain for API routes with authentication
func (ms *MiddlewareStack) APIRouteMiddleware() []gin.HandlerFunc {
	return ms.SetupProtected()
}

// AdminRouteMiddleware returns middleware chain for admin routes
func (ms *MiddlewareStack) AdminRouteMiddleware() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		ms.auth.RequireAuth(),
		ms.RequirePermissions(models.PermissionReadUsers, models.PermissionWriteUsers),
		validation.ValidateQueryParams(),
		validation.SanitizeInput(),
	}
}

// FeatureFlagRouteMiddleware returns middleware for feature flag routes
func (ms *MiddlewareStack) FeatureFlagRouteMiddleware(action string) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		ms.auth.RequireAuth(),
		ms.RequireResourceAccess("feature_flags", action),
		validation.ValidateQueryParams(),
		validation.SanitizeInput(),
	}
}

// APIKeyRouteMiddleware returns middleware for API key routes
func (ms *MiddlewareStack) APIKeyRouteMiddleware(action string) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		ms.auth.RequireAuth(),
		ms.RequireResourceAccess("api_keys", action),
		validation.ValidateQueryParams(),
		validation.SanitizeInput(),
	}
}

// PublicReadOnlyMiddleware returns middleware for public read-only routes
func (ms *MiddlewareStack) PublicReadOnlyMiddleware() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		ms.auth.OptionalAuth(),
		validation.ValidateQueryParams(),
		validation.SanitizeInput(),
	}
}

// HealthCheckMiddleware returns minimal middleware for health check endpoints
func (ms *MiddlewareStack) HealthCheckMiddleware() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		// Only essential middleware for health checks
	}
}