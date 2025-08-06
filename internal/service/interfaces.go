package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

// FeatureFlagService defines the interface for feature flag business logic
type FeatureFlagService interface {
	// Create creates a new feature flag with business logic validation
	Create(ctx context.Context, req *models.CreateFeatureFlagRequest, createdBy uuid.UUID) (*models.FeatureFlag, error)
	
	// GetByID retrieves a feature flag by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error)
	
	// GetByName retrieves a feature flag by name and service
	GetByName(ctx context.Context, name, serviceName string) (*models.FeatureFlag, error)
	
	// Update updates a feature flag with business logic validation
	Update(ctx context.Context, id uuid.UUID, req *models.UpdateFeatureFlagRequest, updatedBy uuid.UUID) (*models.FeatureFlag, error)
	
	// Delete soft deletes a feature flag
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	
	// List retrieves feature flags with filtering and pagination
	List(ctx context.Context, filter *models.FeatureFlagFilter, pagination *models.PaginationParams) ([]*models.FeatureFlag, int64, error)
	
	// ListByService retrieves all active feature flags for a service
	ListByService(ctx context.Context, serviceName string) ([]*models.FeatureFlag, error)
	
	// UpdateEnvironment updates a specific environment configuration
	UpdateEnvironment(ctx context.Context, id uuid.UUID, environment string, config *models.EnvironmentConfig, updatedBy uuid.UUID) error
	
	// ToggleEnvironment toggles the enabled state for an environment
	ToggleEnvironment(ctx context.Context, id uuid.UUID, environment string, enabled bool, updatedBy uuid.UUID) error
	
	// SetRolloutPercent updates the rollout percentage for an environment
	SetRolloutPercent(ctx context.Context, id uuid.UUID, environment string, percent int, updatedBy uuid.UUID) error
	
	// EvaluateFlag evaluates a feature flag for given context
	EvaluateFlag(ctx context.Context, flagName, serviceName, environment string, evaluationContext map[string]interface{}) (bool, error)
	
	// GetEnvironments retrieves all unique environment names
	GetEnvironments(ctx context.Context) ([]string, error)
	
	// GetServices retrieves all unique service names
	GetServices(ctx context.Context) ([]string, error)
	
	// Exists checks if a feature flag exists by name and service
	Exists(ctx context.Context, name, serviceName string) (bool, error)
}

// ValidationService defines the interface for validation logic
type ValidationService interface {
	// ValidateCreateRequest validates a create feature flag request
	ValidateCreateRequest(ctx context.Context, req *models.CreateFeatureFlagRequest) error
	
	// ValidateUpdateRequest validates an update feature flag request
	ValidateUpdateRequest(ctx context.Context, req *models.UpdateFeatureFlagRequest, existing *models.FeatureFlag) error
	
	// ValidateEnvironmentConfig validates environment configuration
	ValidateEnvironmentConfig(ctx context.Context, config *models.EnvironmentConfig) error
	
	// ValidateConditionalRule validates a conditional rule
	ValidateConditionalRule(ctx context.Context, rule *models.ConditionalRule) error
	
	// ValidateBusinessRules validates business rules for feature flag operations
	ValidateBusinessRules(ctx context.Context, flag *models.FeatureFlag, operation string) error
}

// CacheService defines the interface for caching operations
type CacheService interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) ([]byte, error)
	
	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value []byte, ttl int) error
	
	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error
	
	// DeletePattern removes all keys matching a pattern
	DeletePattern(ctx context.Context, pattern string) error
	
	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Health checks cache health
	Health(ctx context.Context) error
}

// AuditService defines the interface for audit trail operations
type AuditService interface {
	// LogAction logs an audit action
	LogAction(ctx context.Context, entityType string, entityID uuid.UUID, action models.AuditAction, userID uuid.UUID, changes models.JSONB, metadata models.JSONB) error
	
	// LogCreate logs a create operation
	LogCreate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error
	
	// LogUpdate logs an update operation with before/after diff
	LogUpdate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, before, after interface{}, metadata models.JSONB) error
	
	// LogDelete logs a delete operation
	LogDelete(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error
	
	// GetEntityHistory retrieves the complete change history for an entity
	GetEntityHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.AuditLog, error)
	
	// GetUserActivity retrieves recent user activity
	GetUserActivity(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error)
}

// DiffService defines the interface for generating diffs between objects
type DiffService interface {
	// GenerateDiff generates a diff between before and after objects
	GenerateDiff(before, after interface{}) (models.JSONB, error)
	
	// GenerateFeatureFlagDiff generates a specialized diff for feature flags
	GenerateFeatureFlagDiff(before, after *models.FeatureFlag) (models.JSONB, error)
	
	// GenerateEnvironmentDiff generates a diff for environment configurations
	GenerateEnvironmentDiff(before, after models.EnvironmentConfig) (models.JSONB, error)
}

// Services aggregates all service interfaces
type Services struct {
	FeatureFlags FeatureFlagService
	Validation   ValidationService
	Cache        CacheService
	Audit        AuditService
	Diff         DiffService
}