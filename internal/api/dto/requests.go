package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

// CreateFeatureFlagRequest represents a request to create a new feature flag
type CreateFeatureFlagRequest struct {
	Name         string                              `json:"name" binding:"required,min=3,max=100"`
	ServiceName  string                              `json:"service_name" binding:"required,min=3,max=100"`
	Description  string                              `json:"description" binding:"max=500"`
	Environments map[string]EnvironmentConfigRequest `json:"environments,omitempty"`
}

// EnvironmentConfigRequest represents environment configuration in requests
type EnvironmentConfigRequest struct {
	Enabled        bool                      `json:"enabled"`
	RolloutPercent int                       `json:"rollout_percent" binding:"min=0,max=100"`
	Rules          []ConditionalRuleRequest  `json:"rules,omitempty"`
}

// ConditionalRuleRequest represents a conditional rule in requests
type ConditionalRuleRequest struct {
	Attribute string      `json:"attribute" binding:"required"`
	Operator  string      `json:"operator" binding:"required"`
	Value     interface{} `json:"value" binding:"required"`
	Enabled   bool        `json:"enabled"`
}

// Validate validates the create feature flag request
func (r *CreateFeatureFlagRequest) Validate() []string {
	var errors []string

	// Validate environments
	for envName, envConfig := range r.Environments {
		if envName == "" {
			errors = append(errors, "environment name cannot be empty")
			continue
		}
		
		if envErrs := envConfig.Validate(); len(envErrs) > 0 {
			for _, err := range envErrs {
				errors = append(errors, "environment '"+envName+"': "+err)
			}
		}
	}

	return errors
}

// Validate validates the environment configuration request
func (r *EnvironmentConfigRequest) Validate() []string {
	var errors []string

	// Validate rules
	for i, rule := range r.Rules {
		if ruleErrs := rule.Validate(); len(ruleErrs) > 0 {
			for _, err := range ruleErrs {
				errors = append(errors, "rule "+string(rune(i))+": "+err)
			}
		}
	}

	return errors
}

// Validate validates the conditional rule request
func (r *ConditionalRuleRequest) Validate() []string {
	var errors []string

	// Validate operator
	validOperators := []string{"equals", "not_equals", "in", "not_in", "contains", "not_contains", "gt", "gte", "lt", "lte"}
	isValidOperator := false
	for _, op := range validOperators {
		if r.Operator == op {
			isValidOperator = true
			break
		}
	}
	if !isValidOperator {
		errors = append(errors, "invalid operator: "+r.Operator)
	}

	return errors
}

// ToModel converts the request to a model for service layer processing
func (r *CreateFeatureFlagRequest) ToModel(createdBy uuid.UUID) *models.CreateFeatureFlagRequest {
	environments := make(map[string]models.EnvironmentConfig)
	
	for envName, envReq := range r.Environments {
		rules := make([]models.ConditionalRule, len(envReq.Rules))
		for i, ruleReq := range envReq.Rules {
			rules[i] = models.ConditionalRule{
				ID:        uuid.New(),
				Attribute: ruleReq.Attribute,
				Operator:  ruleReq.Operator,
				Value:     ruleReq.Value,
				Enabled:   ruleReq.Enabled,
			}
		}
		
		environments[envName] = models.EnvironmentConfig{
			Enabled:        envReq.Enabled,
			RolloutPercent: envReq.RolloutPercent,
			Rules:          rules,
			UpdatedBy:      createdBy,
			UpdatedAt:      time.Now(),
		}
	}
	
	return &models.CreateFeatureFlagRequest{
		Name:         r.Name,
		ServiceName:  r.ServiceName,
		Description:  r.Description,
		Environments: environments,
	}
}

// UpdateFeatureFlagRequest represents a request to update an existing feature flag
type UpdateFeatureFlagRequest struct {
	Description  *string                              `json:"description,omitempty" binding:"omitempty,max=500"`
	Environments map[string]EnvironmentConfigRequest  `json:"environments,omitempty"`
}

// Validate validates the update feature flag request
func (r *UpdateFeatureFlagRequest) Validate() []string {
	var errors []string

	// Validate environments
	for envName, envConfig := range r.Environments {
		if envName == "" {
			errors = append(errors, "environment name cannot be empty")
			continue
		}
		
		if envErrs := envConfig.Validate(); len(envErrs) > 0 {
			for _, err := range envErrs {
				errors = append(errors, "environment '"+envName+"': "+err)
			}
		}
	}

	return errors
}

// ToModel converts the update request to a model for service layer processing
func (r *UpdateFeatureFlagRequest) ToModel(updatedBy uuid.UUID) *models.UpdateFeatureFlagRequest {
	environments := make(map[string]models.EnvironmentConfig)
	
	for envName, envReq := range r.Environments {
		rules := make([]models.ConditionalRule, len(envReq.Rules))
		for i, ruleReq := range envReq.Rules {
			rules[i] = models.ConditionalRule{
				ID:        uuid.New(),
				Attribute: ruleReq.Attribute,
				Operator:  ruleReq.Operator,
				Value:     ruleReq.Value,
				Enabled:   ruleReq.Enabled,
			}
		}
		
		environments[envName] = models.EnvironmentConfig{
			Enabled:        envReq.Enabled,
			RolloutPercent: envReq.RolloutPercent,
			Rules:          rules,
			UpdatedBy:      updatedBy,
			UpdatedAt:      time.Now(),
		}
	}
	
	return &models.UpdateFeatureFlagRequest{
		Description:  r.Description,
		Environments: environments,
	}
}

// HasChanges returns true if the update request has any changes
func (r *UpdateFeatureFlagRequest) HasChanges() bool {
	return r.Description != nil || len(r.Environments) > 0
}

// ListFeatureFlagsRequest represents a request to list feature flags with filters
type ListFeatureFlagsRequest struct {
	ServiceName string    `json:"service_name,omitempty" form:"service"`
	Environment string    `json:"environment,omitempty" form:"environment"`
	Enabled     *bool     `json:"enabled,omitempty" form:"enabled"`
	Name        string    `json:"name,omitempty" form:"name"`
	CreatedBy   uuid.UUID `json:"created_by,omitempty" form:"created_by"`
	Page        int       `json:"page,omitempty" form:"page" binding:"min=1"`
	Limit       int       `json:"limit,omitempty" form:"limit" binding:"min=1,max=100"`
}

// ToFilterAndPagination converts the request to filter and pagination params
func (r *ListFeatureFlagsRequest) ToFilterAndPagination() (*models.FeatureFlagFilter, *models.PaginationParams) {
	filter := &models.FeatureFlagFilter{
		ServiceName: r.ServiceName,
		Environment: r.Environment,
		Enabled:     r.Enabled,
		Name:        r.Name,
		CreatedBy:   r.CreatedBy,
	}
	
	pagination := &models.PaginationParams{
		Page:     r.Page,
		PageSize: r.Limit,
	}
	
	// Apply defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 20
	}
	if pagination.PageSize > 100 {
		pagination.PageSize = 100
	}
	
	pagination.Normalize()
	
	return filter, pagination
}

// ToggleEnvironmentRequest represents a request to toggle an environment
type ToggleEnvironmentRequest struct {
	Environment string `json:"environment" binding:"required"`
	Enabled     bool   `json:"enabled"`
}

// UpdateRolloutRequest represents a request to update rollout percentage
type UpdateRolloutRequest struct {
	Environment string `json:"environment" binding:"required"`
	Percent     int    `json:"percent" binding:"min=0,max=100"`
}