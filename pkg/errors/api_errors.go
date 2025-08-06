package errors

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorCode represents standardized error codes for the API
type ErrorCode string

const (
	// Authentication errors
	ErrorCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrorCodeInvalidAPIKey       ErrorCode = "INVALID_API_KEY"
	ErrorCodeExpiredAPIKey       ErrorCode = "EXPIRED_API_KEY"
	ErrorCodeMissingAuth         ErrorCode = "MISSING_AUTHORIZATION"
	ErrorCodeInsufficientPerms   ErrorCode = "INSUFFICIENT_PERMISSIONS"

	// Validation errors
	ErrorCodeValidation          ErrorCode = "VALIDATION_ERROR"
	ErrorCodeInvalidInput        ErrorCode = "INVALID_INPUT"
	ErrorCodeMissingRequired     ErrorCode = "MISSING_REQUIRED_FIELD"
	ErrorCodeInvalidFormat       ErrorCode = "INVALID_FORMAT"
	ErrorCodeInvalidUUID         ErrorCode = "INVALID_UUID"

	// Rate limiting errors
	ErrorCodeRateLimited         ErrorCode = "RATE_LIMITED"
	ErrorCodeTooManyRequests     ErrorCode = "TOO_MANY_REQUESTS"

	// Resource errors
	ErrorCodeNotFound            ErrorCode = "NOT_FOUND"
	ErrorCodeConflict            ErrorCode = "CONFLICT"
	ErrorCodeAlreadyExists       ErrorCode = "ALREADY_EXISTS"

	// Server errors
	ErrorCodeInternalError       ErrorCode = "INTERNAL_ERROR"
	ErrorCodeServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeDatabaseError       ErrorCode = "DATABASE_ERROR"

	// Generic errors
	ErrorCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrorCodeForbidden           ErrorCode = "FORBIDDEN"
	ErrorCodeMethodNotAllowed    ErrorCode = "METHOD_NOT_ALLOWED"
)

// APIError represents a structured API error response
type APIError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Fields    map[string]string      `json:"fields,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Error implements the error interface
func (e APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e APIError) HTTPStatus() int {
	switch e.Code {
	case ErrorCodeUnauthorized, ErrorCodeInvalidAPIKey, ErrorCodeExpiredAPIKey, ErrorCodeMissingAuth:
		return http.StatusUnauthorized
	case ErrorCodeInsufficientPerms, ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeValidation, ErrorCodeInvalidInput, ErrorCodeMissingRequired, 
		 ErrorCodeInvalidFormat, ErrorCodeInvalidUUID, ErrorCodeBadRequest:
		return http.StatusBadRequest
	case ErrorCodeRateLimited, ErrorCodeTooManyRequests:
		return http.StatusTooManyRequests
	case ErrorCodeNotFound:
		return http.StatusNotFound
	case ErrorCodeConflict, ErrorCodeAlreadyExists:
		return http.StatusConflict
	case ErrorCodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case ErrorCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrorCodeInternalError, ErrorCodeDatabaseError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewAPIError creates a new API error with timestamp
func NewAPIError(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details string) *APIError {
	e.Details = details
	return e
}

// WithField adds a field-specific error
func (e *APIError) WithField(field, message string) *APIError {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}
	e.Fields[field] = message
	return e
}

// WithFields adds multiple field-specific errors
func (e *APIError) WithFields(fields map[string]string) *APIError {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// WithMetadata adds metadata to the error
func (e *APIError) WithMetadata(key string, value interface{}) *APIError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithRequestID adds a request ID to the error
func (e *APIError) WithRequestID(requestID string) *APIError {
	e.RequestID = requestID
	return e
}

// Common error constructors

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *APIError {
	if message == "" {
		message = "Authentication required"
	}
	return NewAPIError(ErrorCodeUnauthorized, message)
}

// NewInvalidAPIKeyError creates an invalid API key error
func NewInvalidAPIKeyError() *APIError {
	return NewAPIError(ErrorCodeInvalidAPIKey, "Invalid API key provided")
}

// NewExpiredAPIKeyError creates an expired API key error
func NewExpiredAPIKeyError() *APIError {
	return NewAPIError(ErrorCodeExpiredAPIKey, "API key has expired")
}

// NewMissingAuthError creates a missing authorization error
func NewMissingAuthError() *APIError {
	return NewAPIError(ErrorCodeMissingAuth, "Authorization header is required")
}

// NewInsufficientPermissionsError creates an insufficient permissions error
func NewInsufficientPermissionsError(required string) *APIError {
	err := NewAPIError(ErrorCodeInsufficientPerms, "Insufficient permissions")
	if required != "" {
		err.WithDetails(fmt.Sprintf("Required permission: %s", required))
	}
	return err
}

// NewValidationError creates a validation error
func NewValidationError(message string) *APIError {
	if message == "" {
		message = "Request validation failed"
	}
	return NewAPIError(ErrorCodeValidation, message)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(retryAfter time.Duration) *APIError {
	err := NewAPIError(ErrorCodeRateLimited, "Rate limit exceeded")
	if retryAfter > 0 {
		err.WithMetadata("retry_after", int(retryAfter.Seconds()))
	}
	return err
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *APIError {
	message := "Resource not found"
	if resource != "" {
		message = fmt.Sprintf("%s not found", resource)
	}
	return NewAPIError(ErrorCodeNotFound, message)
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *APIError {
	if message == "" {
		message = "Resource conflict"
	}
	return NewAPIError(ErrorCodeConflict, message)
}

// NewInternalError creates an internal server error
func NewInternalError(message string) *APIError {
	if message == "" {
		message = "Internal server error"
	}
	return NewAPIError(ErrorCodeInternalError, message)
}

// ErrorHandler is a Gin middleware for handling errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Process any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			var apiError *APIError
			
			// Check if it's already an APIError
			if ae, ok := err.Err.(*APIError); ok {
				apiError = ae
			} else {
				// Create a generic internal error
				apiError = NewInternalError(err.Error())
			}

			// Add request ID if available
			if requestID, exists := c.Get("RequestID"); exists {
				if id, ok := requestID.(string); ok {
					apiError.WithRequestID(id)
				}
			}

			// Set the HTTP status code and return the error
			c.JSON(apiError.HTTPStatus(), apiError)
			c.Abort()
		}
	}
}

// AbortWithError aborts the request with an API error
func AbortWithError(c *gin.Context, err *APIError) {
	if requestID, exists := c.Get("RequestID"); exists {
		if id, ok := requestID.(string); ok {
			err.WithRequestID(id)
		}
	}
	c.JSON(err.HTTPStatus(), err)
	c.Abort()
}

// AbortWithUnauthorized aborts with unauthorized error
func AbortWithUnauthorized(c *gin.Context, message string) {
	AbortWithError(c, NewUnauthorizedError(message))
}

// AbortWithInvalidAPIKey aborts with invalid API key error
func AbortWithInvalidAPIKey(c *gin.Context) {
	AbortWithError(c, NewInvalidAPIKeyError())
}

// AbortWithExpiredAPIKey aborts with expired API key error
func AbortWithExpiredAPIKey(c *gin.Context) {
	AbortWithError(c, NewExpiredAPIKeyError())
}

// AbortWithInsufficientPermissions aborts with insufficient permissions error
func AbortWithInsufficientPermissions(c *gin.Context, required string) {
	AbortWithError(c, NewInsufficientPermissionsError(required))
}

// AbortWithValidation aborts with validation error
func AbortWithValidation(c *gin.Context, message string, fields map[string]string) {
	err := NewValidationError(message)
	if fields != nil {
		err.WithFields(fields)
	}
	AbortWithError(c, err)
}

// AbortWithRateLimit aborts with rate limit error
func AbortWithRateLimit(c *gin.Context, retryAfter time.Duration) {
	err := NewRateLimitError(retryAfter)
	if retryAfter > 0 {
		c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
	}
	AbortWithError(c, err)
}

// AbortWithNotFound aborts with not found error
func AbortWithNotFound(c *gin.Context, resource string) {
	AbortWithError(c, NewNotFoundError(resource))
}