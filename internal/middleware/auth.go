package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/auth"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/service"
	"github.com/tig-company/featury/pkg/errors"
)

// AuthContext keys for storing authentication data in Gin context
const (
	ContextKeyAPIKey = "api_key"
	ContextKeyUser   = "user"
	ContextKeyUserID = "user_id"
)

// AuthMiddleware handles API key authentication
type AuthMiddleware struct {
	authService *service.AuthService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth middleware that requires API key authentication
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract API key from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.AbortWithError(c, errors.NewMissingAuthError())
			return
		}

		// Extract key from header
		apiKey, err := auth.ExtractKeyFromAuthHeader(authHeader)
		if err != nil {
			errors.AbortWithError(c, errors.NewInvalidAPIKeyError().WithDetails(err.Error()))
			return
		}

		// Authenticate the API key
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		authenticatedKey, err := am.authService.AuthenticateAPIKey(ctx, apiKey)
		if err != nil {
			if apiErr, ok := err.(*errors.APIError); ok {
				errors.AbortWithError(c, apiErr)
			} else {
				errors.AbortWithError(c, errors.NewUnauthorizedError(err.Error()))
			}
			return
		}

		// Store authentication data in context
		c.Set(ContextKeyAPIKey, authenticatedKey)
		c.Set(ContextKeyUserID, authenticatedKey.UserID)

		c.Next()
	}
}

// RequirePermission middleware that requires specific permissions
func (am *AuthMiddleware) RequirePermission(permissions ...models.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from context (should be set by RequireAuth middleware)
		apiKeyInterface, exists := c.Get(ContextKeyAPIKey)
		if !exists {
			errors.AbortWithError(c, errors.NewUnauthorizedError("Authentication required"))
			return
		}

		apiKey, ok := apiKeyInterface.(*models.APIKey)
		if !ok {
			errors.AbortWithError(c, errors.NewInternalError("Invalid authentication context"))
			return
		}

		// Check if API key has any of the required permissions
		hasPermission := false
		for _, permission := range permissions {
			if apiKey.HasPermission(permission) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			permissionStrings := make([]string, len(permissions))
			for i, p := range permissions {
				permissionStrings[i] = string(p)
			}
			errors.AbortWithInsufficientPermissions(c, fmt.Sprintf("Required permissions: %v", permissionStrings))
			return
		}

		c.Next()
	}
}

// RequireResourceAccess middleware that requires access to a specific resource with an action
func (am *AuthMiddleware) RequireResourceAccess(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from context
		apiKeyInterface, exists := c.Get(ContextKeyAPIKey)
		if !exists {
			errors.AbortWithError(c, errors.NewUnauthorizedError("Authentication required"))
			return
		}

		apiKey, ok := apiKeyInterface.(*models.APIKey)
		if !ok {
			errors.AbortWithError(c, errors.NewInternalError("Invalid authentication context"))
			return
		}

		// Check resource access
		if err := am.authService.ValidatePermissions(apiKey, resource, action); err != nil {
			if apiErr, ok := err.(*errors.APIError); ok {
				errors.AbortWithError(c, apiErr)
			} else {
				errors.AbortWithInsufficientPermissions(c, err.Error())
			}
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware that allows both authenticated and unauthenticated requests
// but sets authentication context if valid credentials are provided
func (am *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header provided, continue without authentication
			c.Next()
			return
		}

		// Try to authenticate if header is provided
		apiKey, err := auth.ExtractKeyFromAuthHeader(authHeader)
		if err != nil {
			// Invalid auth header format, continue without authentication
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		authenticatedKey, err := am.authService.AuthenticateAPIKey(ctx, apiKey)
		if err != nil {
			// Invalid credentials, continue without authentication
			c.Next()
			return
		}

		// Store authentication data in context
		c.Set(ContextKeyAPIKey, authenticatedKey)
		c.Set(ContextKeyUserID, authenticatedKey.UserID)

		c.Next()
	}
}

// Helper functions to get authentication data from Gin context

// GetAPIKeyFromContext retrieves the authenticated API key from Gin context
func GetAPIKeyFromContext(c *gin.Context) (*models.APIKey, bool) {
	apiKeyInterface, exists := c.Get(ContextKeyAPIKey)
	if !exists {
		return nil, false
	}

	apiKey, ok := apiKeyInterface.(*models.APIKey)
	return apiKey, ok
}

// GetUserIDFromContext retrieves the user ID from Gin context
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userIDInterface, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, false
	}

	userID, ok := userIDInterface.(uuid.UUID)
	return userID, ok
}

// MustGetAPIKeyFromContext retrieves the API key from context or panics
// Should only be used after RequireAuth middleware
func MustGetAPIKeyFromContext(c *gin.Context) *models.APIKey {
	apiKey, exists := GetAPIKeyFromContext(c)
	if !exists {
		panic("API key not found in context - ensure RequireAuth middleware is used")
	}
	return apiKey
}

// MustGetUserIDFromContext retrieves the user ID from context or panics
// Should only be used after RequireAuth middleware
func MustGetUserIDFromContext(c *gin.Context) uuid.UUID {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		panic("User ID not found in context - ensure RequireAuth middleware is used")
	}
	return userID
}