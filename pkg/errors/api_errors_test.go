package errors

import (
	"net/http"
	"testing"
	"time"
)

func TestAPIError_HTTPStatus(t *testing.T) {
	tests := []struct {
		name         string
		errorCode    ErrorCode
		expectedCode int
	}{
		{"Unauthorized", ErrorCodeUnauthorized, http.StatusUnauthorized},
		{"Invalid API Key", ErrorCodeInvalidAPIKey, http.StatusUnauthorized},
		{"Expired API Key", ErrorCodeExpiredAPIKey, http.StatusUnauthorized},
		{"Insufficient Permissions", ErrorCodeInsufficientPerms, http.StatusForbidden},
		{"Validation Error", ErrorCodeValidation, http.StatusBadRequest},
		{"Rate Limited", ErrorCodeRateLimited, http.StatusTooManyRequests},
		{"Not Found", ErrorCodeNotFound, http.StatusNotFound},
		{"Conflict", ErrorCodeConflict, http.StatusConflict},
		{"Internal Error", ErrorCodeInternalError, http.StatusInternalServerError},
		{"Unknown Error", ErrorCode("UNKNOWN"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.errorCode, "test message")
			if status := err.HTTPStatus(); status != tt.expectedCode {
				t.Errorf("HTTPStatus() = %d, expected %d", status, tt.expectedCode)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name           string
		errorCode      ErrorCode
		message        string
		details        string
		expectedString string
	}{
		{
			name:           "Basic error",
			errorCode:      ErrorCodeValidation,
			message:        "validation failed",
			expectedString: "VALIDATION_ERROR: validation failed",
		},
		{
			name:           "Error with details",
			errorCode:      ErrorCodeInvalidInput,
			message:        "invalid field",
			details:        "field must be non-empty",
			expectedString: "INVALID_INPUT: invalid field (field must be non-empty)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.errorCode, tt.message)
			if tt.details != "" {
				err.WithDetails(tt.details)
			}

			if errStr := err.Error(); errStr != tt.expectedString {
				t.Errorf("Error() = %q, expected %q", errStr, tt.expectedString)
			}
		})
	}
}

func TestAPIError_WithMethods(t *testing.T) {
	err := NewAPIError(ErrorCodeValidation, "test message")

	// Test WithDetails
	err.WithDetails("test details")
	if err.Details != "test details" {
		t.Errorf("WithDetails() failed, got %q, expected %q", err.Details, "test details")
	}

	// Test WithField
	err.WithField("test_field", "test field error")
	if err.Fields["test_field"] != "test field error" {
		t.Errorf("WithField() failed, got %q, expected %q", err.Fields["test_field"], "test field error")
	}

	// Test WithFields
	fields := map[string]string{
		"field1": "error1",
		"field2": "error2",
	}
	err.WithFields(fields)
	for k, v := range fields {
		if err.Fields[k] != v {
			t.Errorf("WithFields() failed for %q, got %q, expected %q", k, err.Fields[k], v)
		}
	}

	// Test WithMetadata
	err.WithMetadata("key1", "value1")
	if err.Metadata["key1"] != "value1" {
		t.Errorf("WithMetadata() failed, got %v, expected %q", err.Metadata["key1"], "value1")
	}

	// Test WithRequestID
	err.WithRequestID("test-request-id")
	if err.RequestID != "test-request-id" {
		t.Errorf("WithRequestID() failed, got %q, expected %q", err.RequestID, "test-request-id")
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name      string
		create    func() *APIError
		checkCode ErrorCode
	}{
		{
			name:      "NewUnauthorizedError",
			create:    func() *APIError { return NewUnauthorizedError("test") },
			checkCode: ErrorCodeUnauthorized,
		},
		{
			name:      "NewInvalidAPIKeyError",
			create:    func() *APIError { return NewInvalidAPIKeyError() },
			checkCode: ErrorCodeInvalidAPIKey,
		},
		{
			name:      "NewExpiredAPIKeyError",
			create:    func() *APIError { return NewExpiredAPIKeyError() },
			checkCode: ErrorCodeExpiredAPIKey,
		},
		{
			name:      "NewMissingAuthError",
			create:    func() *APIError { return NewMissingAuthError() },
			checkCode: ErrorCodeMissingAuth,
		},
		{
			name:      "NewInsufficientPermissionsError",
			create:    func() *APIError { return NewInsufficientPermissionsError("read:users") },
			checkCode: ErrorCodeInsufficientPerms,
		},
		{
			name:      "NewValidationError",
			create:    func() *APIError { return NewValidationError("test") },
			checkCode: ErrorCodeValidation,
		},
		{
			name:      "NewRateLimitError",
			create:    func() *APIError { return NewRateLimitError(time.Second * 30) },
			checkCode: ErrorCodeRateLimited,
		},
		{
			name:      "NewNotFoundError",
			create:    func() *APIError { return NewNotFoundError("User") },
			checkCode: ErrorCodeNotFound,
		},
		{
			name:      "NewConflictError",
			create:    func() *APIError { return NewConflictError("test") },
			checkCode: ErrorCodeConflict,
		},
		{
			name:      "NewInternalError",
			create:    func() *APIError { return NewInternalError("test") },
			checkCode: ErrorCodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.create()
			if err.Code != tt.checkCode {
				t.Errorf("Error code mismatch, got %q, expected %q", err.Code, tt.checkCode)
			}
			if err.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}
		})
	}
}

func TestNewRateLimitError_WithRetryAfter(t *testing.T) {
	retryAfter := time.Second * 30
	err := NewRateLimitError(retryAfter)

	if err.Code != ErrorCodeRateLimited {
		t.Errorf("Expected error code %q, got %q", ErrorCodeRateLimited, err.Code)
	}

	if retry, ok := err.Metadata["retry_after"].(int); ok {
		expectedRetry := int(retryAfter.Seconds())
		if retry != expectedRetry {
			t.Errorf("Expected retry_after %d, got %d", expectedRetry, retry)
		}
	} else {
		t.Error("retry_after metadata should be set")
	}
}

func TestNewInsufficientPermissionsError_WithDetails(t *testing.T) {
	required := "read:users"
	err := NewInsufficientPermissionsError(required)

	expectedDetails := "Required permission: " + required
	if err.Details != expectedDetails {
		t.Errorf("Expected details %q, got %q", expectedDetails, err.Details)
	}

	// Test with empty required permission
	err2 := NewInsufficientPermissionsError("")
	if err2.Details != "" {
		t.Errorf("Expected empty details, got %q", err2.Details)
	}
}