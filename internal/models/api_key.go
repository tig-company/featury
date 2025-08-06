package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// APIKey represents an API key in the system
type APIKey struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	KeyHash     string       `json:"-" db:"key_hash"` // Never expose in JSON
	UserID      uuid.UUID    `json:"user_id" db:"user_id"`
	Name        string       `json:"name" db:"name"`
	Permissions []Permission `json:"permissions" db:"permissions"`
	ExpiresAt   *time.Time   `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	LastUsedAt  *time.Time   `json:"last_used_at" db:"last_used_at"`
}

// CreateAPIKeyRequest represents a request to create a new API key
type CreateAPIKeyRequest struct {
	Name        string       `json:"name" binding:"required"`
	Permissions []Permission `json:"permissions" binding:"required"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
}

// Validate validates the create API key request
func (car *CreateAPIKeyRequest) Validate() error {
	if len(car.Permissions) == 0 {
		return NewValidationError("permissions", "at least one permission is required")
	}
	
	for _, permission := range car.Permissions {
		if !permission.Valid() {
			return NewValidationError("permissions", "invalid permission: "+string(permission))
		}
	}
	
	if car.ExpiresAt != nil && car.ExpiresAt.Before(time.Now()) {
		return NewValidationError("expires_at", "expiration date must be in the future")
	}
	
	return nil
}

// UpdateAPIKeyRequest represents a request to update an existing API key
type UpdateAPIKeyRequest struct {
	Name        *string    `json:"name,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// HasChanges returns true if the update request has any changes
func (uar *UpdateAPIKeyRequest) HasChanges() bool {
	return uar.Name != nil || uar.ExpiresAt != nil
}

// APIKeyFilter represents filters for API key queries
type APIKeyFilter struct {
	UserID     uuid.UUID  `json:"user_id,omitempty" form:"user_id"`
	Name       string     `json:"name,omitempty" form:"name"`
	IsExpired  *bool      `json:"is_expired,omitempty" form:"is_expired"`
	HasExpiry  *bool      `json:"has_expiry,omitempty" form:"has_expiry"`
}

// APIKeyResponse represents an API key in API responses
type APIKeyResponse struct {
	ID          uuid.UUID    `json:"id"`
	UserID      uuid.UUID    `json:"user_id"`
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
	ExpiresAt   *time.Time   `json:"expires_at"`
	CreatedAt   time.Time    `json:"created_at"`
	LastUsedAt  *time.Time   `json:"last_used_at"`
	IsExpired   bool         `json:"is_expired"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	APIKey *APIKeyResponse `json:"api_key"`
	Key    string          `json:"key"` // Raw key is only returned once
}

// ToResponse converts an APIKey to APIKeyResponse
func (ak *APIKey) ToResponse() *APIKeyResponse {
	return &APIKeyResponse{
		ID:          ak.ID,
		UserID:      ak.UserID,
		Name:        ak.Name,
		Permissions: ak.Permissions,
		ExpiresAt:   ak.ExpiresAt,
		CreatedAt:   ak.CreatedAt,
		LastUsedAt:  ak.LastUsedAt,
		IsExpired:   ak.IsExpired(),
	}
}

// IsExpired returns true if the API key is expired
func (ak *APIKey) IsExpired() bool {
	return ak.ExpiresAt != nil && ak.ExpiresAt.Before(time.Now())
}

// HasPermission checks if the API key has a specific permission
func (ak *APIKey) HasPermission(permission Permission) bool {
	for _, p := range ak.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the API key has any of the specified permissions
func (ak *APIKey) HasAnyPermission(permissions ...Permission) bool {
	for _, permission := range permissions {
		if ak.HasPermission(permission) {
			return true
		}
	}
	return false
}

// GenerateAPIKey generates a new random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "ftr_" + hex.EncodeToString(bytes), nil
}