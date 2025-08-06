package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/repository"
)

type validationService struct {
	repo repository.Repository
}

// NewValidationService creates a new validation service
func NewValidationService(repo repository.Repository) ValidationService {
	return &validationService{
		repo: repo,
	}
}

// ValidateCreateRequest validates a create feature flag request
func (s *validationService) ValidateCreateRequest(ctx context.Context, req *models.CreateFeatureFlagRequest) error {
	// Basic field validation
	if strings.TrimSpace(req.Name) == "" {
		return models.NewValidationError("name", "name is required and cannot be empty")
	}

	if strings.TrimSpace(req.ServiceName) == "" {
		return models.NewValidationError("service_name", "service name is required and cannot be empty")
	}

	// Validate name format (alphanumeric, underscore, dash)
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !nameRegex.MatchString(req.Name) {
		return models.NewValidationError("name", "name can only contain alphanumeric characters, underscores, and dashes")
	}

	// Validate service name format
	serviceRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !serviceRegex.MatchString(req.ServiceName) {
		return models.NewValidationError("service_name", "service name can only contain alphanumeric characters, underscores, and dashes")
	}

	// Check name length constraints
	if len(req.Name) < 3 || len(req.Name) > 100 {
		return models.NewValidationError("name", "name must be between 3 and 100 characters")
	}

	if len(req.ServiceName) < 3 || len(req.ServiceName) > 50 {
		return models.NewValidationError("service_name", "service name must be between 3 and 50 characters")
	}

	if len(req.Description) > 500 {
		return models.NewValidationError("description", "description cannot exceed 500 characters")
	}

	// Check if flag already exists
	exists, err := s.repo.FeatureFlags().Exists(ctx, req.Name, req.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to check if feature flag exists: %w", err)
	}
	if exists {
		return models.NewValidationError("name", "feature flag with this name already exists for this service")
	}

	// Validate environments
	if len(req.Environments) > 20 {
		return models.NewValidationError("environments", "cannot have more than 20 environments")
	}

	for envName, envConfig := range req.Environments {
		if err := s.validateEnvironmentName(envName); err != nil {
			return models.NewValidationError("environments."+envName, err.Error())
		}
		if err := s.ValidateEnvironmentConfig(ctx, &envConfig); err != nil {
			return models.NewValidationError("environments."+envName, err.Error())
		}
	}

	return nil
}

// ValidateUpdateRequest validates an update feature flag request
func (s *validationService) ValidateUpdateRequest(ctx context.Context, req *models.UpdateFeatureFlagRequest, existing *models.FeatureFlag) error {
	if !req.HasChanges() {
		return models.NewValidationError("request", "no changes provided")
	}

	// Validate description if provided
	if req.Description != nil && len(*req.Description) > 500 {
		return models.NewValidationError("description", "description cannot exceed 500 characters")
	}

	// Validate environments if provided
	if req.Environments != nil {
		// Check total environment count including existing ones
		totalEnvs := len(existing.Environments)
		for envName := range req.Environments {
			if _, exists := existing.Environments[envName]; !exists {
				totalEnvs++
			}
		}
		if totalEnvs > 20 {
			return models.NewValidationError("environments", "cannot have more than 20 environments total")
		}

		for envName, envConfig := range req.Environments {
			if err := s.validateEnvironmentName(envName); err != nil {
				return models.NewValidationError("environments."+envName, err.Error())
			}
			if err := s.ValidateEnvironmentConfig(ctx, &envConfig); err != nil {
				return models.NewValidationError("environments."+envName, err.Error())
			}
		}
	}

	return nil
}

// ValidateEnvironmentConfig validates environment configuration
func (s *validationService) ValidateEnvironmentConfig(ctx context.Context, config *models.EnvironmentConfig) error {
	// Validate rollout percentage
	if config.RolloutPercent < 0 || config.RolloutPercent > 100 {
		return models.NewValidationError("rollout_percent", "rollout percentage must be between 0 and 100")
	}

	// Validate rules
	if len(config.Rules) > 50 {
		return models.NewValidationError("rules", "cannot have more than 50 conditional rules")
	}

	for i, rule := range config.Rules {
		if err := s.ValidateConditionalRule(ctx, &rule); err != nil {
			return models.NewValidationError(fmt.Sprintf("rules[%d]", i), err.Error())
		}
	}

	// Validate updated_by is not nil UUID
	if config.UpdatedBy == uuid.Nil {
		return models.NewValidationError("updated_by", "updated_by is required")
	}

	return nil
}

// ValidateConditionalRule validates a conditional rule
func (s *validationService) ValidateConditionalRule(ctx context.Context, rule *models.ConditionalRule) error {
	// Validate attribute
	if strings.TrimSpace(rule.Attribute) == "" {
		return models.NewValidationError("attribute", "attribute is required")
	}

	if len(rule.Attribute) > 100 {
		return models.NewValidationError("attribute", "attribute cannot exceed 100 characters")
	}

	// Validate attribute format
	attributeRegex := regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	if !attributeRegex.MatchString(rule.Attribute) {
		return models.NewValidationError("attribute", "attribute can only contain alphanumeric characters, underscores, dots, and dashes")
	}

	// Validate operator
	if !s.isValidOperator(rule.Operator) {
		return models.NewValidationError("operator", "invalid operator")
	}

	// Validate value based on operator
	if err := s.validateValueForOperator(rule.Operator, rule.Value); err != nil {
		return models.NewValidationError("value", err.Error())
	}

	// Validate ID if provided (for updates)
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New() // Generate new ID if not provided
	}

	return nil
}

// ValidateBusinessRules validates business rules for feature flag operations
func (s *validationService) ValidateBusinessRules(ctx context.Context, flag *models.FeatureFlag, operation string) error {
	switch operation {
	case "create":
		return s.validateCreateBusinessRules(ctx, flag)
	case "update":
		return s.validateUpdateBusinessRules(ctx, flag)
	case "delete":
		return s.validateDeleteBusinessRules(ctx, flag)
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}

// Helper methods

func (s *validationService) validateEnvironmentName(envName string) error {
	if strings.TrimSpace(envName) == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	if len(envName) < 2 || len(envName) > 50 {
		return fmt.Errorf("environment name must be between 2 and 50 characters")
	}

	// Validate environment name format
	envRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !envRegex.MatchString(envName) {
		return fmt.Errorf("environment name can only contain alphanumeric characters, underscores, and dashes")
	}

	return nil
}

func (s *validationService) isValidOperator(operator string) bool {
	validOperators := []string{
		"equals", "not_equals",
		"in", "not_in",
		"contains", "not_contains",
		"gt", "gte", "lt", "lte",
		"starts_with", "ends_with",
		"regex",
	}

	for _, valid := range validOperators {
		if operator == valid {
			return true
		}
	}
	return false
}

func (s *validationService) validateValueForOperator(operator string, value interface{}) error {
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}

	switch operator {
	case "equals", "not_equals", "contains", "not_contains", "starts_with", "ends_with", "regex":
		// These operators expect string values
		if _, ok := value.(string); !ok {
			return fmt.Errorf("operator %s requires string value", operator)
		}

	case "in", "not_in":
		// These operators expect array values
		switch v := value.(type) {
		case []interface{}:
			if len(v) == 0 {
				return fmt.Errorf("operator %s requires non-empty array", operator)
			}
			if len(v) > 100 {
				return fmt.Errorf("operator %s array cannot have more than 100 items", operator)
			}
		case []string:
			if len(v) == 0 {
				return fmt.Errorf("operator %s requires non-empty array", operator)
			}
			if len(v) > 100 {
				return fmt.Errorf("operator %s array cannot have more than 100 items", operator)
			}
		default:
			return fmt.Errorf("operator %s requires array value", operator)
		}

	case "gt", "gte", "lt", "lte":
		// These operators expect numeric values
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid numeric types
		default:
			return fmt.Errorf("operator %s requires numeric value", operator)
		}
	}

	return nil
}

func (s *validationService) validateCreateBusinessRules(ctx context.Context, flag *models.FeatureFlag) error {
	// Ensure at least one environment is configured if environments are provided
	if len(flag.Environments) == 0 {
		return models.NewValidationError("environments", "at least one environment must be configured")
	}

	// Validate that rollout percentages don't exceed limits for critical environments
	for envName, config := range flag.Environments {
		if s.isCriticalEnvironment(envName) && config.RolloutPercent > 50 && config.Enabled {
			return models.NewValidationError(
				fmt.Sprintf("environments.%s.rollout_percent", envName),
				"critical environments cannot have rollout percentage > 50% on creation",
			)
		}
	}

	return nil
}

func (s *validationService) validateUpdateBusinessRules(ctx context.Context, flag *models.FeatureFlag) error {
	// Check if flag is soft deleted
	if flag.IsDeleted() {
		return models.NewValidationError("flag", "cannot update deleted feature flag")
	}

	return nil
}

func (s *validationService) validateDeleteBusinessRules(ctx context.Context, flag *models.FeatureFlag) error {
	// Check if flag is already deleted
	if flag.IsDeleted() {
		return models.NewValidationError("flag", "feature flag is already deleted")
	}

	// Check if any environment is currently enabled with high rollout
	for envName, config := range flag.Environments {
		if config.Enabled && config.RolloutPercent > 0 {
			if s.isCriticalEnvironment(envName) && config.RolloutPercent > 10 {
				return models.NewValidationError(
					"environments",
					fmt.Sprintf("cannot delete flag with active rollout in critical environment '%s'", envName),
				)
			}
		}
	}

	return nil
}

func (s *validationService) isCriticalEnvironment(envName string) bool {
	criticalEnvs := []string{"production", "prod", "live"}
	envLower := strings.ToLower(envName)
	for _, critical := range criticalEnvs {
		if envLower == critical {
			return true
		}
	}
	return false
}