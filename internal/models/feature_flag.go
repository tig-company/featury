package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FeatureFlag represents a feature flag in the system
type FeatureFlag struct {
	ID          uuid.UUID                      `json:"id" db:"id"`
	Name        string                         `json:"name" db:"name"`
	ServiceName string                         `json:"service_name" db:"service_name"`
	Description string                         `json:"description" db:"description"`
	CreatedBy   uuid.UUID                      `json:"created_by" db:"created_by"`
	CreatedAt   time.Time                      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time                     `json:"deleted_at,omitempty" db:"deleted_at"`
	
	// Environment-specific configurations stored in JSONB
	Environments map[string]EnvironmentConfig `json:"environments" db:"environments"`
}

// EnvironmentConfig represents the configuration for a specific environment
type EnvironmentConfig struct {
	Enabled        bool               `json:"enabled"`
	RolloutPercent int                `json:"rollout_percent"` // 0-100
	Rules          []ConditionalRule  `json:"rules,omitempty"`
	UpdatedBy      uuid.UUID          `json:"updated_by"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// ConditionalRule represents a conditional rule for feature flag evaluation
type ConditionalRule struct {
	ID        uuid.UUID   `json:"id"`
	Attribute string      `json:"attribute"` // user_id, country, etc.
	Operator  string      `json:"operator"`  // equals, in, contains, not_equals, not_in
	Value     interface{} `json:"value"`
	Enabled   bool        `json:"enabled"`
}

// Valid returns true if the conditional rule is valid
func (cr *ConditionalRule) Valid() bool {
	if cr.Attribute == "" {
		return false
	}
	
	switch cr.Operator {
	case "equals", "not_equals", "in", "not_in", "contains", "not_contains", "gt", "gte", "lt", "lte":
		return true
	default:
		return false
	}
}

// CreateFeatureFlagRequest represents a request to create a new feature flag
type CreateFeatureFlagRequest struct {
	Name         string                         `json:"name" binding:"required"`
	ServiceName  string                         `json:"service_name" binding:"required"`
	Description  string                         `json:"description"`
	Environments map[string]EnvironmentConfig `json:"environments"`
}

// Validate validates the create feature flag request
func (cfr *CreateFeatureFlagRequest) Validate() error {
	// Validate environment configurations
	for envName, envConfig := range cfr.Environments {
		if envName == "" {
			return NewValidationError("environments", "environment name cannot be empty")
		}
		if err := envConfig.Validate(); err != nil {
			return NewValidationError("environments."+envName, err.Error())
		}
	}
	return nil
}

// UpdateFeatureFlagRequest represents a request to update an existing feature flag
type UpdateFeatureFlagRequest struct {
	Description  *string                        `json:"description,omitempty"`
	Environments map[string]EnvironmentConfig `json:"environments,omitempty"`
}

// Validate validates the update feature flag request
func (ufr *UpdateFeatureFlagRequest) Validate() error {
	// Validate environment configurations if provided
	for envName, envConfig := range ufr.Environments {
		if envName == "" {
			return NewValidationError("environments", "environment name cannot be empty")
		}
		if err := envConfig.Validate(); err != nil {
			return NewValidationError("environments."+envName, err.Error())
		}
	}
	return nil
}

// HasChanges returns true if the update request has any changes
func (ufr *UpdateFeatureFlagRequest) HasChanges() bool {
	return ufr.Description != nil || len(ufr.Environments) > 0
}

// Validate validates the environment configuration
func (ec *EnvironmentConfig) Validate() error {
	if ec.RolloutPercent < 0 || ec.RolloutPercent > 100 {
		return NewValidationError("rollout_percent", "must be between 0 and 100")
	}
	
	for i, rule := range ec.Rules {
		if !rule.Valid() {
			return NewValidationError("rules["+string(rune(i))+"]", "invalid conditional rule")
		}
	}
	
	return nil
}

// FeatureFlagFilter represents filters for feature flag queries
type FeatureFlagFilter struct {
	ServiceName string    `json:"service_name,omitempty" form:"service_name"`
	CreatedBy   uuid.UUID `json:"created_by,omitempty" form:"created_by"`
	Name        string    `json:"name,omitempty" form:"name"`
	Environment string    `json:"environment,omitempty" form:"environment"`
	Enabled     *bool     `json:"enabled,omitempty" form:"enabled"`
}

// FeatureFlagResponse represents a feature flag in API responses
type FeatureFlagResponse struct {
	ID          uuid.UUID                      `json:"id"`
	Name        string                         `json:"name"`
	ServiceName string                         `json:"service_name"`
	Description string                         `json:"description"`
	CreatedBy   uuid.UUID                      `json:"created_by"`
	CreatedAt   time.Time                      `json:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at"`
	Environments map[string]EnvironmentConfig `json:"environments"`
}

// ToResponse converts a FeatureFlag to FeatureFlagResponse
func (ff *FeatureFlag) ToResponse() *FeatureFlagResponse {
	return &FeatureFlagResponse{
		ID:           ff.ID,
		Name:         ff.Name,
		ServiceName:  ff.ServiceName,
		Description:  ff.Description,
		CreatedBy:    ff.CreatedBy,
		CreatedAt:    ff.CreatedAt,
		UpdatedAt:    ff.UpdatedAt,
		Environments: ff.Environments,
	}
}

// IsDeleted returns true if the feature flag is soft deleted
func (ff *FeatureFlag) IsDeleted() bool {
	return ff.DeletedAt != nil
}

// IsEnabledForEnvironment checks if the feature flag is enabled for a specific environment
func (ff *FeatureFlag) IsEnabledForEnvironment(environment string) bool {
	if envConfig, exists := ff.Environments[environment]; exists {
		return envConfig.Enabled
	}
	return false
}

// GetEnvironmentConfig returns the environment configuration for a specific environment
func (ff *FeatureFlag) GetEnvironmentConfig(environment string) (EnvironmentConfig, bool) {
	config, exists := ff.Environments[environment]
	return config, exists
}

// MarshalEnvironments converts environments map to JSON for database storage
func (ff *FeatureFlag) MarshalEnvironments() ([]byte, error) {
	return json.Marshal(ff.Environments)
}

// UnmarshalEnvironments converts JSON from database to environments map
func (ff *FeatureFlag) UnmarshalEnvironments(data []byte) error {
	if len(data) == 0 {
		ff.Environments = make(map[string]EnvironmentConfig)
		return nil
	}
	return json.Unmarshal(data, &ff.Environments)
}