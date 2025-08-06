package validation

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/pkg/errors"
)

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid  bool
	Errors map[string]string
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:  true,
		Errors: make(map[string]string),
	}
}

// AddError adds a validation error
func (vr *ValidationResult) AddError(field, message string) {
	vr.Valid = false
	vr.Errors[field] = message
}

// HasErrors returns true if there are validation errors
func (vr *ValidationResult) HasErrors() bool {
	return !vr.Valid
}

// ToAPIError converts validation result to API error
func (vr *ValidationResult) ToAPIError() *errors.APIError {
	if vr.Valid {
		return nil
	}

	err := errors.NewValidationError("Request validation failed")
	return err.WithFields(vr.Errors)
}

// Validator provides common validation functions
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// String validation functions

// ValidateRequired checks if a string field is present and not empty
func (v *Validator) ValidateRequired(value, fieldName string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	return value, nil
}

// ValidateStringLength validates string length
func (v *Validator) ValidateStringLength(value, fieldName string, min, max int) (string, error) {
	length := utf8.RuneCountInString(value)
	
	if min > 0 && length < min {
		return "", fmt.Errorf("%s must be at least %d characters long", fieldName, min)
	}
	
	if max > 0 && length > max {
		return "", fmt.Errorf("%s must not exceed %d characters", fieldName, max)
	}
	
	return value, nil
}

// ValidateAlphanumeric checks if string contains only alphanumeric characters and allowed symbols
func (v *Validator) ValidateAlphanumeric(value, fieldName string, allowedSymbols string) (string, error) {
	if value == "" {
		return value, nil
	}

	pattern := fmt.Sprintf("^[a-zA-Z0-9%s]*$", regexp.QuoteMeta(allowedSymbols))
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return "", fmt.Errorf("invalid pattern for %s validation", fieldName)
	}
	
	if !matched {
		return "", fmt.Errorf("%s contains invalid characters", fieldName)
	}
	
	return value, nil
}

// ValidateEmail validates email format
func (v *Validator) ValidateEmail(email, fieldName string) (string, error) {
	if email == "" {
		return email, nil
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return "", fmt.Errorf("%s must be a valid email address", fieldName)
	}
	
	return email, nil
}

// ValidateURL validates URL format
func (v *Validator) ValidateURL(urlString, fieldName string) (string, error) {
	if urlString == "" {
		return urlString, nil
	}

	_, err := url.ParseRequestURI(urlString)
	if err != nil {
		return "", fmt.Errorf("%s must be a valid URL", fieldName)
	}
	
	return urlString, nil
}

// ValidateUUID validates UUID format
func (v *Validator) ValidateUUID(uuidString, fieldName string) (uuid.UUID, error) {
	if uuidString == "" {
		return uuid.Nil, fmt.Errorf("%s is required", fieldName)
	}

	parsedUUID, err := uuid.Parse(uuidString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s must be a valid UUID", fieldName)
	}
	
	return parsedUUID, nil
}

// ValidateOptionalUUID validates optional UUID format
func (v *Validator) ValidateOptionalUUID(uuidString, fieldName string) (*uuid.UUID, error) {
	if uuidString == "" {
		return nil, nil
	}

	parsedUUID, err := uuid.Parse(uuidString)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", fieldName)
	}
	
	return &parsedUUID, nil
}

// Sanitization functions

// SanitizeString performs basic string sanitization
func (v *Validator) SanitizeString(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Normalize line endings
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	
	return input
}

// SanitizeHTML sanitizes HTML input to prevent XSS
func (v *Validator) SanitizeHTML(input string) string {
	return html.EscapeString(input)
}

// SanitizeForSQL sanitizes input for SQL contexts (basic protection)
// Note: This should be used alongside parameterized queries
func (v *Validator) SanitizeForSQL(input string) string {
	// Remove SQL injection attempt patterns
	input = strings.ReplaceAll(input, "'", "''")
	input = strings.ReplaceAll(input, "--", "")
	input = strings.ReplaceAll(input, "/*", "")
	input = strings.ReplaceAll(input, "*/", "")
	
	return input
}

// Validation middleware

// ValidateJSONBody validates JSON request body against binding and custom validation
func ValidateJSONBody(target interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bind JSON to target struct
		if err := c.ShouldBindJSON(target); err != nil {
			errors.AbortWithValidation(c, "Invalid request body", map[string]string{
				"body": err.Error(),
			})
			return
		}

		// Run custom validation if the target implements Validator interface
		if validator, ok := target.(CustomValidator); ok {
			if err := validator.Validate(); err != nil {
				if ve, ok := err.(*models.ValidationError); ok {
					errors.AbortWithValidation(c, "Validation failed", map[string]string{
						ve.Field: ve.Message,
					})
				} else {
					errors.AbortWithValidation(c, err.Error(), nil)
				}
				return
			}
		}

		c.Next()
	}
}

// ValidateQueryParams validates query parameters
func ValidateQueryParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		validator := NewValidator()
		result := NewValidationResult()

		// Validate common query parameters
		if pageStr := c.Query("page"); pageStr != "" {
			if matched, _ := regexp.MatchString(`^\d+$`, pageStr); !matched {
				result.AddError("page", "page must be a positive integer")
			}
		}

		if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
			if matched, _ := regexp.MatchString(`^\d+$`, pageSizeStr); !matched {
				result.AddError("page_size", "page_size must be a positive integer")
			}
		}

		// Validate UUIDs in query parameters
		for _, param := range []string{"user_id", "api_key_id", "feature_flag_id"} {
			if uuidStr := c.Query(param); uuidStr != "" {
				if _, err := validator.ValidateUUID(uuidStr, param); err != nil {
					result.AddError(param, err.Error())
				}
			}
		}

		if result.HasErrors() {
			errors.AbortWithError(c, result.ToAPIError())
			return
		}

		c.Next()
	}
}

// ValidatePathParams validates path parameters (UUIDs)
func ValidatePathParams(paramNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		validator := NewValidator()
		result := NewValidationResult()

		for _, paramName := range paramNames {
			paramValue := c.Param(paramName)
			if paramValue == "" {
				result.AddError(paramName, fmt.Sprintf("%s is required", paramName))
				continue
			}

			if _, err := validator.ValidateUUID(paramValue, paramName); err != nil {
				result.AddError(paramName, err.Error())
			}
		}

		if result.HasErrors() {
			errors.AbortWithError(c, result.ToAPIError())
			return
		}

		c.Next()
	}
}

// SanitizeInput middleware that sanitizes all input
func SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Note: This is a placeholder for input sanitization
		// In practice, you might want to sanitize query params and form data
		// JSON body sanitization should be done at the struct level

		c.Next()
	}
}

// CustomValidator interface for structs that need custom validation
type CustomValidator interface {
	Validate() error
}

// Common validation patterns
var (
	// FeatureFlagNamePattern validates feature flag names
	FeatureFlagNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	
	// FeatureFlagKeyPattern validates feature flag keys
	FeatureFlagKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`)
	
	// UsernamePattern validates usernames
	UsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	
	// APIKeyNamePattern validates API key names
	APIKeyNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 _-]*$`)
)

// ValidateFeatureFlagName validates feature flag names
func ValidateFeatureFlagName(name string) error {
	if name == "" {
		return fmt.Errorf("feature flag name is required")
	}
	
	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("feature flag name must be between 2 and 100 characters")
	}
	
	if !FeatureFlagNamePattern.MatchString(name) {
		return fmt.Errorf("feature flag name must start with a letter and contain only letters, numbers, hyphens, and underscores")
	}
	
	return nil
}

// ValidateFeatureFlagKey validates feature flag keys
func ValidateFeatureFlagKey(key string) error {
	if key == "" {
		return fmt.Errorf("feature flag key is required")
	}
	
	if len(key) < 2 || len(key) > 100 {
		return fmt.Errorf("feature flag key must be between 2 and 100 characters")
	}
	
	if !FeatureFlagKeyPattern.MatchString(key) {
		return fmt.Errorf("feature flag key must start with a letter and contain only letters, numbers, periods, hyphens, and underscores")
	}
	
	return nil
}

// ValidateAPIKeyName validates API key names
func ValidateAPIKeyName(name string) error {
	if name == "" {
		return fmt.Errorf("API key name is required")
	}
	
	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("API key name must be between 2 and 100 characters")
	}
	
	if !APIKeyNamePattern.MatchString(name) {
		return fmt.Errorf("API key name must start with a letter or number and contain only letters, numbers, spaces, hyphens, and underscores")
	}
	
	return nil
}