package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/repository"
)

type featureFlagService struct {
	repo       repository.Repository
	validator  ValidationService
	audit      AuditService
	diff       DiffService
}

// NewFeatureFlagService creates a new feature flag service
func NewFeatureFlagService(repo repository.Repository, validator ValidationService, audit AuditService, diff DiffService) FeatureFlagService {
	return &featureFlagService{
		repo:      repo,
		validator: validator,
		audit:     audit,
		diff:      diff,
	}
}

// Create creates a new feature flag with business logic validation
func (s *featureFlagService) Create(ctx context.Context, req *models.CreateFeatureFlagRequest, createdBy uuid.UUID) (*models.FeatureFlag, error) {
	// Validate the request
	if err := s.validator.ValidateCreateRequest(ctx, req); err != nil {
		return nil, err
	}

	// Create the feature flag model
	now := time.Now()
	flag := &models.FeatureFlag{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(req.Name),
		ServiceName: strings.TrimSpace(req.ServiceName),
		Description: strings.TrimSpace(req.Description),
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
		Environments: make(map[string]models.EnvironmentConfig),
	}

	// Process environments with default values and validation
	for envName, envConfig := range req.Environments {
		processedConfig := s.processEnvironmentConfig(envConfig, createdBy, now)
		flag.Environments[envName] = processedConfig
	}

	// Validate business rules
	if err := s.validator.ValidateBusinessRules(ctx, flag, "create"); err != nil {
		return nil, err
	}

	// Create the feature flag in the repository
	if err := s.repo.FeatureFlags().Create(ctx, flag); err != nil {
		return nil, fmt.Errorf("failed to create feature flag: %w", err)
	}

	// Log the creation in audit trail
	metadata := models.JSONB{
		"request_id": s.generateRequestID(),
		"client_ip":  s.extractClientIP(ctx),
		"user_agent": s.extractUserAgent(ctx),
	}
	
	if err := s.audit.LogCreate(ctx, "feature_flag", flag.ID, createdBy, flag, metadata); err != nil {
		// Log audit error but don't fail the operation
		fmt.Printf("failed to log audit trail for create: %v\n", err)
	}

	return flag, nil
}

// GetByID retrieves a feature flag by ID
func (s *featureFlagService) GetByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error) {
	flag, err := s.repo.FeatureFlags().GetActiveByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}
	return flag, nil
}

// GetByName retrieves a feature flag by name and service
func (s *featureFlagService) GetByName(ctx context.Context, name, serviceName string) (*models.FeatureFlag, error) {
	flag, err := s.repo.FeatureFlags().GetByName(ctx, name, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature flag by name: %w", err)
	}
	return flag, nil
}

// Update updates a feature flag with business logic validation
func (s *featureFlagService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateFeatureFlagRequest, updatedBy uuid.UUID) (*models.FeatureFlag, error) {
	// Get existing feature flag
	existing, err := s.repo.FeatureFlags().GetActiveByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing feature flag: %w", err)
	}

	// Validate the update request
	if err := s.validator.ValidateUpdateRequest(ctx, req, existing); err != nil {
		return nil, err
	}

	// Process environment updates
	if req.Environments != nil {
		processedEnvs := make(map[string]models.EnvironmentConfig)
		now := time.Now()

		// Copy existing environments
		for envName, envConfig := range existing.Environments {
			processedEnvs[envName] = envConfig
		}

		// Apply updates
		for envName, envConfig := range req.Environments {
			processedConfig := s.processEnvironmentConfig(envConfig, updatedBy, now)
			processedEnvs[envName] = processedConfig
		}

		req.Environments = processedEnvs
	}

	// Validate business rules
	tempFlag := *existing
	if req.Description != nil {
		tempFlag.Description = *req.Description
	}
	if req.Environments != nil {
		tempFlag.Environments = req.Environments
	}

	if err := s.validator.ValidateBusinessRules(ctx, &tempFlag, "update"); err != nil {
		return nil, err
	}

	// Update the feature flag
	updated, err := s.repo.FeatureFlags().Update(ctx, id, req, updatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to update feature flag: %w", err)
	}

	// Log the update in audit trail
	metadata := models.JSONB{
		"request_id": s.generateRequestID(),
		"client_ip":  s.extractClientIP(ctx),
		"user_agent": s.extractUserAgent(ctx),
	}

	if err := s.audit.LogUpdate(ctx, "feature_flag", id, updatedBy, existing, updated, metadata); err != nil {
		// Log audit error but don't fail the operation
		fmt.Printf("failed to log audit trail for update: %v\n", err)
	}

	return updated, nil
}

// Delete soft deletes a feature flag
func (s *featureFlagService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// Get existing feature flag
	existing, err := s.repo.FeatureFlags().GetActiveByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get feature flag for deletion: %w", err)
	}

	// Validate business rules for deletion
	if err := s.validator.ValidateBusinessRules(ctx, existing, "delete"); err != nil {
		return err
	}

	// Perform soft delete
	if err := s.repo.FeatureFlags().SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete feature flag: %w", err)
	}

	// Log the deletion in audit trail
	metadata := models.JSONB{
		"request_id": s.generateRequestID(),
		"client_ip":  s.extractClientIP(ctx),
		"user_agent": s.extractUserAgent(ctx),
	}

	if err := s.audit.LogDelete(ctx, "feature_flag", id, deletedBy, existing, metadata); err != nil {
		// Log audit error but don't fail the operation
		fmt.Printf("failed to log audit trail for delete: %v\n", err)
	}

	return nil
}

// List retrieves feature flags with filtering and pagination
func (s *featureFlagService) List(ctx context.Context, filter *models.FeatureFlagFilter, pagination *models.PaginationParams) ([]*models.FeatureFlag, int64, error) {
	flags, total, err := s.repo.FeatureFlags().List(ctx, filter, pagination)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list feature flags: %w", err)
	}
	return flags, total, nil
}

// ListByService retrieves all active feature flags for a service
func (s *featureFlagService) ListByService(ctx context.Context, serviceName string) ([]*models.FeatureFlag, error) {
	flags, err := s.repo.FeatureFlags().ListByService(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to list feature flags by service: %w", err)
	}
	return flags, nil
}

// UpdateEnvironment updates a specific environment configuration
func (s *featureFlagService) UpdateEnvironment(ctx context.Context, id uuid.UUID, environment string, config *models.EnvironmentConfig, updatedBy uuid.UUID) error {
	// Get existing feature flag
	existing, err := s.repo.FeatureFlags().GetActiveByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	// Validate environment configuration
	if err := s.validator.ValidateEnvironmentConfig(ctx, config); err != nil {
		return err
	}

	// Process the configuration
	now := time.Now()
	processedConfig := s.processEnvironmentConfig(*config, updatedBy, now)

	// Update the environment
	if err := s.repo.FeatureFlags().UpdateEnvironment(ctx, id, environment, &processedConfig); err != nil {
		return fmt.Errorf("failed to update environment: %w", err)
	}

	// Log the environment update
	metadata := models.JSONB{
		"environment": environment,
		"request_id":  s.generateRequestID(),
		"client_ip":   s.extractClientIP(ctx),
		"user_agent":  s.extractUserAgent(ctx),
	}

	changes := models.JSONB{
		"environment": environment,
		"before":      existing.Environments[environment],
		"after":       processedConfig,
	}

	if err := s.audit.LogAction(ctx, "feature_flag", id, models.AuditActionUpdate, updatedBy, changes, metadata); err != nil {
		fmt.Printf("failed to log audit trail for environment update: %v\n", err)
	}

	return nil
}

// ToggleEnvironment toggles the enabled state for an environment
func (s *featureFlagService) ToggleEnvironment(ctx context.Context, id uuid.UUID, environment string, enabled bool, updatedBy uuid.UUID) error {
	// Get existing feature flag
	existing, err := s.repo.FeatureFlags().GetActiveByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	// Get current environment config or create new one
	envConfig, exists := existing.Environments[environment]
	if !exists {
		envConfig = models.EnvironmentConfig{
			Enabled:        false,
			RolloutPercent: 0,
			Rules:          []models.ConditionalRule{},
		}
	}

	// Update enabled state
	envConfig.Enabled = enabled
	envConfig.UpdatedBy = updatedBy
	envConfig.UpdatedAt = time.Now()

	// Validate the updated configuration
	if err := s.validator.ValidateEnvironmentConfig(ctx, &envConfig); err != nil {
		return err
	}

	// Update the environment
	if err := s.repo.FeatureFlags().UpdateEnvironment(ctx, id, environment, &envConfig); err != nil {
		return fmt.Errorf("failed to toggle environment: %w", err)
	}

	// Log the toggle action
	action := models.AuditActionEnable
	if !enabled {
		action = models.AuditActionDisable
	}

	metadata := models.JSONB{
		"environment": environment,
		"action":      fmt.Sprintf("toggle_%s", strings.ToLower(string(action))),
		"request_id":  s.generateRequestID(),
		"client_ip":   s.extractClientIP(ctx),
		"user_agent":  s.extractUserAgent(ctx),
	}

	changes := models.JSONB{
		"environment": environment,
		"enabled":     enabled,
	}

	if err := s.audit.LogAction(ctx, "feature_flag", id, action, updatedBy, changes, metadata); err != nil {
		fmt.Printf("failed to log audit trail for environment toggle: %v\n", err)
	}

	return nil
}

// SetRolloutPercent updates the rollout percentage for an environment
func (s *featureFlagService) SetRolloutPercent(ctx context.Context, id uuid.UUID, environment string, percent int, updatedBy uuid.UUID) error {
	// Validate percentage range
	if percent < 0 || percent > 100 {
		return models.NewValidationError("rollout_percent", "rollout percentage must be between 0 and 100")
	}

	// Get existing feature flag
	existing, err := s.repo.FeatureFlags().GetActiveByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get feature flag: %w", err)
	}

	// Get current environment config or create new one
	envConfig, exists := existing.Environments[environment]
	if !exists {
		envConfig = models.EnvironmentConfig{
			Enabled:        false,
			RolloutPercent: 0,
			Rules:          []models.ConditionalRule{},
		}
	}

	// Store old percentage for audit
	oldPercent := envConfig.RolloutPercent

	// Update rollout percentage
	envConfig.RolloutPercent = percent
	envConfig.UpdatedBy = updatedBy
	envConfig.UpdatedAt = time.Now()

	// Validate the updated configuration
	if err := s.validator.ValidateEnvironmentConfig(ctx, &envConfig); err != nil {
		return err
	}

	// Update the environment
	if err := s.repo.FeatureFlags().UpdateEnvironment(ctx, id, environment, &envConfig); err != nil {
		return fmt.Errorf("failed to set rollout percentage: %w", err)
	}

	// Log the rollout percentage change
	metadata := models.JSONB{
		"environment": environment,
		"action":      "set_rollout_percent",
		"request_id":  s.generateRequestID(),
		"client_ip":   s.extractClientIP(ctx),
		"user_agent":  s.extractUserAgent(ctx),
	}

	changes := models.JSONB{
		"environment":        environment,
		"old_rollout_percent": oldPercent,
		"new_rollout_percent": percent,
	}

	if err := s.audit.LogAction(ctx, "feature_flag", id, models.AuditActionUpdate, updatedBy, changes, metadata); err != nil {
		fmt.Printf("failed to log audit trail for rollout percentage update: %v\n", err)
	}

	return nil
}

// EvaluateFlag evaluates a feature flag for given context
func (s *featureFlagService) EvaluateFlag(ctx context.Context, flagName, serviceName, environment string, evaluationContext map[string]interface{}) (bool, error) {
	// Get the feature flag
	flag, err := s.repo.FeatureFlags().GetByName(ctx, flagName, serviceName)
	if err != nil {
		return false, fmt.Errorf("failed to get feature flag for evaluation: %w", err)
	}

	// Check if the flag is deleted
	if flag.IsDeleted() {
		return false, nil
	}

	// Get environment configuration
	envConfig, exists := flag.GetEnvironmentConfig(environment)
	if !exists {
		return false, nil // Environment not configured
	}

	// Check if the flag is enabled for this environment
	if !envConfig.Enabled {
		return false, nil
	}

	// Evaluate conditional rules first
	if len(envConfig.Rules) > 0 {
		ruleResult, err := s.evaluateConditionalRules(envConfig.Rules, evaluationContext)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate conditional rules: %w", err)
		}
		if !ruleResult {
			return false, nil // Rules didn't match
		}
	}

	// Apply rollout percentage
	if envConfig.RolloutPercent == 0 {
		return false, nil
	}
	if envConfig.RolloutPercent == 100 {
		return true, nil
	}

	// Use consistent hashing for rollout
	rolloutResult := s.calculateRollout(flagName, serviceName, environment, evaluationContext, envConfig.RolloutPercent)
	return rolloutResult, nil
}

// GetEnvironments retrieves all unique environment names
func (s *featureFlagService) GetEnvironments(ctx context.Context) ([]string, error) {
	environments, err := s.repo.FeatureFlags().GetEnvironments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get environments: %w", err)
	}
	return environments, nil
}

// GetServices retrieves all unique service names
func (s *featureFlagService) GetServices(ctx context.Context) ([]string, error) {
	services, err := s.repo.FeatureFlags().GetServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}
	return services, nil
}

// Exists checks if a feature flag exists by name and service
func (s *featureFlagService) Exists(ctx context.Context, name, serviceName string) (bool, error) {
	exists, err := s.repo.FeatureFlags().Exists(ctx, name, serviceName)
	if err != nil {
		return false, fmt.Errorf("failed to check if feature flag exists: %w", err)
	}
	return exists, nil
}

// Helper methods

func (s *featureFlagService) processEnvironmentConfig(config models.EnvironmentConfig, updatedBy uuid.UUID, now time.Time) models.EnvironmentConfig {
	processed := config
	processed.UpdatedBy = updatedBy
	processed.UpdatedAt = now

	// Generate IDs for rules that don't have them
	for i := range processed.Rules {
		if processed.Rules[i].ID == uuid.Nil {
			processed.Rules[i].ID = uuid.New()
		}
	}

	return processed
}

func (s *featureFlagService) evaluateConditionalRules(rules []models.ConditionalRule, context map[string]interface{}) (bool, error) {
	// All enabled rules must match (AND logic)
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		contextValue, exists := context[rule.Attribute]
		if !exists {
			return false, nil // Missing attribute means rule fails
		}

		match, err := s.evaluateRule(rule, contextValue)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil // Any rule failure means overall failure
		}
	}

	return true, nil
}

func (s *featureFlagService) evaluateRule(rule models.ConditionalRule, contextValue interface{}) (bool, error) {
	switch rule.Operator {
	case "equals":
		return fmt.Sprintf("%v", contextValue) == fmt.Sprintf("%v", rule.Value), nil

	case "not_equals":
		return fmt.Sprintf("%v", contextValue) != fmt.Sprintf("%v", rule.Value), nil

	case "in":
		return s.evaluateInOperator(contextValue, rule.Value), nil

	case "not_in":
		return !s.evaluateInOperator(contextValue, rule.Value), nil

	case "contains":
		contextStr := fmt.Sprintf("%v", contextValue)
		valueStr := fmt.Sprintf("%v", rule.Value)
		return strings.Contains(contextStr, valueStr), nil

	case "not_contains":
		contextStr := fmt.Sprintf("%v", contextValue)
		valueStr := fmt.Sprintf("%v", rule.Value)
		return !strings.Contains(contextStr, valueStr), nil

	case "starts_with":
		contextStr := fmt.Sprintf("%v", contextValue)
		valueStr := fmt.Sprintf("%v", rule.Value)
		return strings.HasPrefix(contextStr, valueStr), nil

	case "ends_with":
		contextStr := fmt.Sprintf("%v", contextValue)
		valueStr := fmt.Sprintf("%v", rule.Value)
		return strings.HasSuffix(contextStr, valueStr), nil

	case "gt", "gte", "lt", "lte":
		return s.evaluateNumericComparison(rule.Operator, contextValue, rule.Value)

	default:
		return false, fmt.Errorf("unsupported operator: %s", rule.Operator)
	}
}

func (s *featureFlagService) evaluateInOperator(contextValue interface{}, ruleValue interface{}) bool {
	contextStr := fmt.Sprintf("%v", contextValue)

	switch v := ruleValue.(type) {
	case []interface{}:
		for _, item := range v {
			if contextStr == fmt.Sprintf("%v", item) {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if contextStr == item {
				return true
			}
		}
	}

	return false
}

func (s *featureFlagService) evaluateNumericComparison(operator string, contextValue, ruleValue interface{}) (bool, error) {
	contextNum, err := s.toFloat64(contextValue)
	if err != nil {
		return false, fmt.Errorf("context value is not numeric: %v", contextValue)
	}

	ruleNum, err := s.toFloat64(ruleValue)
	if err != nil {
		return false, fmt.Errorf("rule value is not numeric: %v", ruleValue)
	}

	switch operator {
	case "gt":
		return contextNum > ruleNum, nil
	case "gte":
		return contextNum >= ruleNum, nil
	case "lt":
		return contextNum < ruleNum, nil
	case "lte":
		return contextNum <= ruleNum, nil
	default:
		return false, fmt.Errorf("unsupported numeric operator: %s", operator)
	}
}

func (s *featureFlagService) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func (s *featureFlagService) calculateRollout(flagName, serviceName, environment string, context map[string]interface{}, rolloutPercent int) bool {
	// Create a consistent hash key
	hashKey := fmt.Sprintf("%s:%s:%s", serviceName, flagName, environment)
	
	// Add user ID or session ID to the hash for consistent user experience
	if userID, exists := context["user_id"]; exists {
		hashKey = fmt.Sprintf("%s:%v", hashKey, userID)
	} else if sessionID, exists := context["session_id"]; exists {
		hashKey = fmt.Sprintf("%s:%v", hashKey, sessionID)
	}

	// Calculate hash
	hasher := fnv.New32a()
	hasher.Write([]byte(hashKey))
	hash := hasher.Sum32()

	// Convert to percentage (0-99)
	percentage := int(hash % 100)

	// Return true if within rollout percentage
	return percentage < rolloutPercent
}

func (s *featureFlagService) generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func (s *featureFlagService) extractClientIP(ctx context.Context) string {
	// Extract client IP from context (would be set by middleware)
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return ip
	}
	return "unknown"
}

func (s *featureFlagService) extractUserAgent(ctx context.Context) string {
	// Extract user agent from context (would be set by middleware)
	if ua, ok := ctx.Value("user_agent").(string); ok {
		return ua
	}
	return "unknown"
}