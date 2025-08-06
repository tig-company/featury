package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/pkg/errors"
)

// ErrorHandlerMiddleware provides centralized error handling for all API endpoints
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		defer func() {
			// Handle panics
			if err := recover(); err != nil {
				handlePanic(c, err)
			}
		}()
		
		c.Next()
		
		// Process any errors that occurred during request processing
		if len(c.Errors) > 0 {
			handleErrors(c)
		}
	})
}

// handleErrors processes Gin errors and converts them to appropriate API responses
func handleErrors(c *gin.Context) {
	// Get the last (most recent) error
	err := c.Errors.Last()
	
	// Determine if response has already been sent
	if c.Writer.Written() {
		return
	}
	
	var apiError *errors.APIError
	
	// Check if it's already an APIError
	if ae, ok := err.Err.(*errors.APIError); ok {
		apiError = ae
	} else {
		// Convert other error types to APIError
		apiError = convertToAPIError(err.Err, c)
	}
	
	// Add request context
	enrichErrorWithContext(c, apiError)
	
	// Log error details
	logError(c, apiError, err)
	
	// Send error response
	c.JSON(apiError.HTTPStatus(), apiError)
	c.Abort()
}

// handlePanic handles panics and converts them to 500 errors
func handlePanic(c *gin.Context, recovered interface{}) {
	// Create internal server error
	apiError := errors.NewInternalError("Internal server error")
	
	// Add panic details to metadata for debugging
	apiError.WithMetadata("panic", recovered)
	apiError.WithMetadata("stack_trace", string(debug.Stack()))
	
	// Add request context
	enrichErrorWithContext(c, apiError)
	
	// Log panic details
	log.Printf("PANIC in request %s %s: %v\nStack trace:\n%s",
		c.Request.Method, c.Request.URL.Path, recovered, debug.Stack())
	
	// Send error response if not already sent
	if !c.Writer.Written() {
		c.JSON(http.StatusInternalServerError, apiError)
	}
	
	c.Abort()
}

// convertToAPIError converts various error types to APIError
func convertToAPIError(err error, c *gin.Context) *errors.APIError {
	if err == nil {
		return errors.NewInternalError("Unknown error occurred")
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Check for common error patterns
	switch {
	case strings.Contains(errStr, "not found"):
		return errors.NewNotFoundError("")
		
	case strings.Contains(errStr, "unauthorized"):
		return errors.NewUnauthorizedError("")
		
	case strings.Contains(errStr, "forbidden"):
		return errors.NewAPIError(errors.ErrorCodeForbidden, "Access forbidden")
		
	case strings.Contains(errStr, "validation"):
		return errors.NewValidationError(err.Error())
		
	case strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "already exists"):
		return errors.NewConflictError("Resource already exists")
		
	case strings.Contains(errStr, "timeout"):
		return errors.NewAPIError(errors.ErrorCodeServiceUnavailable, "Request timeout")
		
	case strings.Contains(errStr, "database"):
		return errors.NewAPIError(errors.ErrorCodeDatabaseError, "Database error")
		
	case strings.Contains(errStr, "service error:"):
		// Extract the actual error from service error wrapper
		actualError := strings.TrimPrefix(errStr, "service error:")
		actualError = strings.TrimSpace(actualError)
		return convertToAPIError(&wrappedError{actualError}, c)
		
	default:
		// Default to internal server error
		return errors.NewInternalError(err.Error())
	}
}

// wrappedError is a simple error wrapper for recursion
type wrappedError struct {
	message string
}

func (e *wrappedError) Error() string {
	return e.message
}

// enrichErrorWithContext adds request-specific context to the error
func enrichErrorWithContext(c *gin.Context, apiError *errors.APIError) {
	// Add request ID if available
	if requestID, exists := c.Get("RequestID"); exists {
		if id, ok := requestID.(string); ok {
			apiError.WithRequestID(id)
		}
	}
	
	// Add user context if available
	if userID, exists := c.Get("UserID"); exists {
		apiError.WithMetadata("user_id", userID)
	}
	
	// Add request metadata for debugging
	apiError.WithMetadata("method", c.Request.Method)
	apiError.WithMetadata("path", c.Request.URL.Path)
	apiError.WithMetadata("user_agent", c.Request.UserAgent())
	apiError.WithMetadata("client_ip", c.ClientIP())
}

// logError logs error details for monitoring and debugging
func logError(c *gin.Context, apiError *errors.APIError, ginError *gin.Error) {
	// Determine log level based on error type
	logLevel := getLogLevel(apiError)
	
	// Create log entry
	logEntry := map[string]interface{}{
		"timestamp":    time.Now().Format(time.RFC3339),
		"level":        logLevel,
		"error_code":   apiError.Code,
		"message":      apiError.Message,
		"details":      apiError.Details,
		"request_id":   apiError.RequestID,
		"method":       c.Request.Method,
		"path":         c.Request.URL.Path,
		"status_code":  apiError.HTTPStatus(),
		"user_agent":   c.Request.UserAgent(),
		"client_ip":    c.ClientIP(),
	}
	
	// Add original error if different
	if ginError != nil && ginError.Err.Error() != apiError.Message {
		logEntry["original_error"] = ginError.Err.Error()
	}
	
	// Add user context if available
	if userID, exists := c.Get("UserID"); exists {
		logEntry["user_id"] = userID
	}
	
	// Add metadata if present
	if len(apiError.Metadata) > 0 {
		logEntry["metadata"] = apiError.Metadata
	}
	
	// Log based on severity
	switch logLevel {
	case "error":
		log.Printf("ERROR: %+v", logEntry)
	case "warn":
		log.Printf("WARN: %+v", logEntry)
	case "info":
		log.Printf("INFO: %+v", logEntry)
	default:
		log.Printf("DEBUG: %+v", logEntry)
	}
}

// getLogLevel determines the appropriate log level for an error
func getLogLevel(apiError *errors.APIError) string {
	switch apiError.Code {
	case errors.ErrorCodeInternalError, errors.ErrorCodeDatabaseError, errors.ErrorCodeServiceUnavailable:
		return "error"
	case errors.ErrorCodeUnauthorized, errors.ErrorCodeForbidden, errors.ErrorCodeConflict:
		return "warn"
	case errors.ErrorCodeValidation, errors.ErrorCodeNotFound, errors.ErrorCodeBadRequest:
		return "info"
	default:
		return "debug"
	}
}

// Custom error handlers for specific scenarios

// ValidationErrorHandler creates a specialized handler for validation errors
func ValidationErrorHandler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()
		
		// Only handle validation errors
		if len(c.Errors) > 0 {
			for _, ginError := range c.Errors {
				if strings.Contains(strings.ToLower(ginError.Error()), "validation") {
					// Handle validation error specifically
					handleValidationError(c, ginError)
					return
				}
			}
		}
	})
}

// handleValidationError handles validation errors with detailed field information
func handleValidationError(c *gin.Context, ginError *gin.Error) {
	if c.Writer.Written() {
		return
	}
	
	// Create validation error
	validationError := errors.NewValidationError("Request validation failed")
	
	// Try to extract field-specific errors
	// This is a simplified implementation - in production, you'd want more sophisticated parsing
	errorStr := ginError.Error()
	if strings.Contains(errorStr, "required") {
		validationError.WithField("required", "One or more required fields are missing")
	}
	if strings.Contains(errorStr, "binding") {
		validationError.WithField("binding", "Request body format is invalid")
	}
	
	// Add context
	enrichErrorWithContext(c, validationError)
	
	// Log and respond
	logError(c, validationError, ginError)
	c.JSON(validationError.HTTPStatus(), validationError)
	c.Abort()
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// This handler is called when no route matches
		apiError := errors.NewNotFoundError("Endpoint not found")
		apiError.WithDetails("The requested endpoint " + c.Request.Method + " " + c.Request.URL.Path + " does not exist")
		
		enrichErrorWithContext(c, apiError)
		
		c.JSON(http.StatusNotFound, apiError)
		c.Abort()
	})
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		apiError := errors.NewAPIError(errors.ErrorCodeMethodNotAllowed, "Method not allowed")
		apiError.WithDetails("The " + c.Request.Method + " method is not allowed for this endpoint")
		
		enrichErrorWithContext(c, apiError)
		
		c.JSON(http.StatusMethodNotAllowed, apiError)
		c.Abort()
	})
}

// TimeoutHandler handles request timeout errors
func TimeoutHandler(timeout time.Duration) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Set timeout context
		// Note: This is a simplified implementation
		// In production, you'd want more sophisticated timeout handling
		
		c.Next()
		
		// Check if request took too long (this would be implemented differently in practice)
		// This is just a placeholder to show the concept
	})
}

// RateLimitErrorHandler handles rate limit exceeded errors
func RateLimitErrorHandler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()
		
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			if strings.Contains(strings.ToLower(err.Error()), "rate limit") {
				handleRateLimitError(c, err)
				return
			}
		}
	})
}

// handleRateLimitError handles rate limit errors with retry information
func handleRateLimitError(c *gin.Context, ginError *gin.Error) {
	if c.Writer.Written() {
		return
	}
	
	rateLimitError := errors.NewRateLimitError(time.Minute) // Default 1 minute retry
	enrichErrorWithContext(c, rateLimitError)
	
	// Set Retry-After header
	c.Header("Retry-After", "60")
	
	logError(c, rateLimitError, ginError)
	c.JSON(rateLimitError.HTTPStatus(), rateLimitError)
	c.Abort()
}