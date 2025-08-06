package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/tig-company/featury/internal/models"
)

// SimpleFeatureFlagValidator provides basic validation for feature flag requests
type SimpleFeatureFlagValidator struct {
	nameRegex        *regexp.Regexp
	serviceNameRegex *regexp.Regexp
}

// NewSimpleFeatureFlagValidator creates a new simple feature flag validator
func NewSimpleFeatureFlagValidator() *SimpleFeatureFlagValidator {
	return &SimpleFeatureFlagValidator{
		nameRegex:        regexp.MustCompile(`^[a-zA-Z0-9_-]{3,100}$`),
		serviceNameRegex: regexp.MustCompile(`^[a-zA-Z0-9_-]{3,100}$`),
	}
}

// ValidateCreateRequest validates a create feature flag request
func (v *SimpleFeatureFlagValidator) ValidateCreateRequest(ctx context.Context, req *models.CreateFeatureFlagRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Validate name
	if err := v.validateName(req.Name); err != nil {
		return fmt.Errorf("name: %w", err)
	}
	
	// Validate service name
	if err := v.validateServiceName(req.ServiceName); err != nil {
		return fmt.Errorf("service_name: %w", err)
	}
	
	// Validate description
	if err := v.validateDescription(req.Description); err != nil {
		return fmt.Errorf("description: %w", err)
	}
	
	// Validate environments
	if err := v.validateEnvironments(req.Environments); err != nil {
		return fmt.Errorf("environments: %w", err)
	}
	
	return nil
}

// ValidateUpdateRequest validates an update feature flag request
func (v *SimpleFeatureFlagValidator) ValidateUpdateRequest(ctx context.Context, req *models.UpdateFeatureFlagRequest, existing *models.FeatureFlag) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	
	if !req.HasChanges() {
		return fmt.Errorf("at least one field must be provided for update")
	}
	
	// Validate description if provided
	if req.Description != nil {
		if err := v.validateDescription(*req.Description); err != nil {
			return fmt.Errorf("description: %w", err)
		}
	}
	
	// Validate environments if provided
	if err := v.validateEnvironments(req.Environments); err != nil {
		return fmt.Errorf("environments: %w", err)
	}
	
	return nil
}

// ValidateEnvironmentConfig validates environment configuration
func (v *SimpleFeatureFlagValidator) ValidateEnvironmentConfig(ctx context.Context, config *models.EnvironmentConfig) error {
	if config == nil {
		return fmt.Errorf("environment config cannot be nil")
	}
	
	// Validate rollout percent
	if config.RolloutPercent < 0 || config.RolloutPercent > 100 {
		return fmt.Errorf("rollout_percent must be between 0 and 100")
	}
	
	// Validate rules
	for i, rule := range config.Rules {
		if err := v.validateConditionalRule(&rule); err != nil {
			return fmt.Errorf("rule[%d]: %w", i, err)
		}
	}
	
	return nil
}

// ValidateConditionalRule validates a conditional rule
func (v *SimpleFeatureFlagValidator) ValidateConditionalRule(ctx context.Context, rule *models.ConditionalRule) error {
	return v.validateConditionalRule(rule)
}

// ValidateBusinessRules validates business rules for feature flag operations
func (v *SimpleFeatureFlagValidator) ValidateBusinessRules(ctx context.Context, flag *models.FeatureFlag, operation string) error {
	if flag == nil {
		return fmt.Errorf("feature flag cannot be nil")
	}
	
	switch operation {
	case "create":
		// Validate that the flag doesn't already exist (would be checked at service level)
		return nil
	case "update":
		// Validate that updates are allowed
		if flag.IsDeleted() {
			return fmt.Errorf("cannot update deleted feature flag")
		}
		return nil
	case "delete":
		// Validate that deletion is allowed
		if flag.IsDeleted() {
			return fmt.Errorf("feature flag is already deleted")
		}
		return nil
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}

// Private validation methods

func (v *SimpleFeatureFlagValidator) validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	
	if len(name) < 3 {
		return fmt.Errorf("name must be at least 3 characters")
	}
	
	if len(name) > 100 {
		return fmt.Errorf("name must not exceed 100 characters")
	}
	
	if !v.nameRegex.MatchString(name) {
		return fmt.Errorf("name can only contain letters, numbers, hyphens, and underscores")
	}
	
	return nil
}

func (v *SimpleFeatureFlagValidator) validateServiceName(serviceName string) error {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return fmt.Errorf("service_name is required")
	}
	
	if len(serviceName) < 3 {
		return fmt.Errorf("service_name must be at least 3 characters")
	}
	
	if len(serviceName) > 100 {
		return fmt.Errorf("service_name must not exceed 100 characters")
	}
	
	if !v.serviceNameRegex.MatchString(serviceName) {
		return fmt.Errorf("service_name can only contain letters, numbers, hyphens, and underscores")
	}
	
	return nil
}

func (v *SimpleFeatureFlagValidator) validateDescription(description string) error {
	if len(description) > 500 {
		return fmt.Errorf("description must not exceed 500 characters")
	}
	
	return nil
}

func (v *SimpleFeatureFlagValidator) validateEnvironments(environments map[string]models.EnvironmentConfig) error {
	for envName, envConfig := range environments {
		if envName == "" {
			return fmt.Errorf("environment name cannot be empty")
		}
		
		if len(envName) > 50 {
			return fmt.Errorf("environment name '%s' must not exceed 50 characters", envName)
		}
		
		if err := v.ValidateEnvironmentConfig(context.Background(), &envConfig); err != nil {
			return fmt.Errorf("environment '%s': %w", envName, err)
		}
	}
	
	return nil
}

func (v *SimpleFeatureFlagValidator) validateConditionalRule(rule *models.ConditionalRule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}
	
	if strings.TrimSpace(rule.Attribute) == "" {
		return fmt.Errorf("attribute is required")
	}
	
	validOperators := []string{
		"equals", "not_equals",
		"in", "not_in",
		"contains", "not_contains",
		"gt", "gte", "lt", "lte",
	}
	
	validOperator := false
	for _, op := range validOperators {
		if rule.Operator == op {
			validOperator = true
			break
		}
	}
	
	if !validOperator {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}
	
	if rule.Value == nil {
		return fmt.Errorf("value is required")
	}
	
	return nil
}