package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/cache"
	"github.com/tig-company/featury/internal/models"
)

type cachedFeatureFlagService struct {
	baseService FeatureFlagService
	cache       cache.Cache
	cacheTTL    time.Duration
}

const (
	// Cache key patterns
	featureFlagByIDKey     = "feature_flag:id:%s"
	featureFlagByNameKey   = "feature_flag:name:%s:%s" // serviceName:flagName
	featureFlagListKey     = "feature_flag:list:%s"    // hash of filter + pagination
	featureFlagServiceKey  = "feature_flag:service:%s"
	featureFlagEnvsKey     = "feature_flag:environments"
	featureFlagServicesKey = "feature_flag:services"

	// Cache patterns for invalidation
	featureFlagPattern    = "feature_flag:*"
	featureFlagIDPattern  = "feature_flag:id:*"
	featureFlagListPattern = "feature_flag:list:*"
)

// NewCachedFeatureFlagService creates a cached feature flag service
func NewCachedFeatureFlagService(baseService FeatureFlagService, cache cache.Cache, cacheTTL time.Duration) FeatureFlagService {
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute // Default 5 minutes
	}

	return &cachedFeatureFlagService{
		baseService: baseService,
		cache:       cache,
		cacheTTL:    cacheTTL,
	}
}

// Create creates a new feature flag and invalidates relevant caches
func (s *cachedFeatureFlagService) Create(ctx context.Context, req *models.CreateFeatureFlagRequest, createdBy uuid.UUID) (*models.FeatureFlag, error) {
	flag, err := s.baseService.Create(ctx, req, createdBy)
	if err != nil {
		return nil, err
	}

	// Invalidate caches after successful creation
	s.invalidateAfterCreate(ctx, flag)

	return flag, nil
}

// GetByID retrieves a feature flag by ID with caching
func (s *cachedFeatureFlagService) GetByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error) {
	cacheKey := fmt.Sprintf(featureFlagByIDKey, id.String())

	// Try to get from cache first
	if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
		var flag models.FeatureFlag
		if err := json.Unmarshal(cachedData, &flag); err == nil {
			return &flag, nil
		}
	}

	// Cache miss or error, get from database
	flag, err := s.baseService.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cacheFeatureFlag(ctx, cacheKey, flag)

	return flag, nil
}

// GetByName retrieves a feature flag by name and service with caching
func (s *cachedFeatureFlagService) GetByName(ctx context.Context, name, serviceName string) (*models.FeatureFlag, error) {
	cacheKey := fmt.Sprintf(featureFlagByNameKey, serviceName, name)

	// Try to get from cache first
	if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
		var flag models.FeatureFlag
		if err := json.Unmarshal(cachedData, &flag); err == nil {
			return &flag, nil
		}
	}

	// Cache miss or error, get from database
	flag, err := s.baseService.GetByName(ctx, name, serviceName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cacheFeatureFlag(ctx, cacheKey, flag)
	
	// Also cache by ID for consistency
	idCacheKey := fmt.Sprintf(featureFlagByIDKey, flag.ID.String())
	s.cacheFeatureFlag(ctx, idCacheKey, flag)

	return flag, nil
}

// Update updates a feature flag and invalidates relevant caches
func (s *cachedFeatureFlagService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateFeatureFlagRequest, updatedBy uuid.UUID) (*models.FeatureFlag, error) {
	// Get existing flag for cache invalidation
	existing, err := s.baseService.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updated, err := s.baseService.Update(ctx, id, req, updatedBy)
	if err != nil {
		return nil, err
	}

	// Invalidate caches after successful update
	s.invalidateAfterUpdate(ctx, existing, updated)

	return updated, nil
}

// Delete soft deletes a feature flag and invalidates relevant caches
func (s *cachedFeatureFlagService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// Get existing flag for cache invalidation
	existing, err := s.baseService.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.baseService.Delete(ctx, id, deletedBy); err != nil {
		return err
	}

	// Invalidate caches after successful deletion
	s.invalidateAfterDelete(ctx, existing)

	return nil
}

// List retrieves feature flags with caching (limited caching for complex queries)
func (s *cachedFeatureFlagService) List(ctx context.Context, filter *models.FeatureFlagFilter, pagination *models.PaginationParams) ([]*models.FeatureFlag, int64, error) {
	// For list operations, we use a shorter TTL and only cache simple queries
	if s.isSimpleListQuery(filter, pagination) {
		cacheKey := s.buildListCacheKey(filter, pagination)
		
		if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
			var result listCacheResult
			if err := json.Unmarshal(cachedData, &result); err == nil {
				return result.Flags, result.Total, nil
			}
		}

		// Get from database
		flags, total, err := s.baseService.List(ctx, filter, pagination)
		if err != nil {
			return nil, 0, err
		}

		// Cache with shorter TTL for list results
		result := listCacheResult{Flags: flags, Total: total}
		s.cacheListResult(ctx, cacheKey, &result, time.Minute) // 1 minute TTL for lists

		return flags, total, nil
	}

	// Complex queries bypass cache
	return s.baseService.List(ctx, filter, pagination)
}

// ListByService retrieves all active feature flags for a service with caching
func (s *cachedFeatureFlagService) ListByService(ctx context.Context, serviceName string) ([]*models.FeatureFlag, error) {
	cacheKey := fmt.Sprintf(featureFlagServiceKey, serviceName)

	// Try to get from cache first
	if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
		var flags []*models.FeatureFlag
		if err := json.Unmarshal(cachedData, &flags); err == nil {
			return flags, nil
		}
	}

	// Cache miss or error, get from database
	flags, err := s.baseService.ListByService(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if flagsData, err := json.Marshal(flags); err == nil {
		s.cache.Set(ctx, cacheKey, flagsData, s.cacheTTL)
	}

	return flags, nil
}

// UpdateEnvironment updates environment config and invalidates caches
func (s *cachedFeatureFlagService) UpdateEnvironment(ctx context.Context, id uuid.UUID, environment string, config *models.EnvironmentConfig, updatedBy uuid.UUID) error {
	// Get existing flag for cache invalidation
	existing, err := s.baseService.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.baseService.UpdateEnvironment(ctx, id, environment, config, updatedBy); err != nil {
		return err
	}

	// Invalidate caches after successful update
	s.invalidateAfterEnvironmentUpdate(ctx, existing)

	return nil
}

// ToggleEnvironment toggles environment state and invalidates caches
func (s *cachedFeatureFlagService) ToggleEnvironment(ctx context.Context, id uuid.UUID, environment string, enabled bool, updatedBy uuid.UUID) error {
	// Get existing flag for cache invalidation
	existing, err := s.baseService.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.baseService.ToggleEnvironment(ctx, id, environment, enabled, updatedBy); err != nil {
		return err
	}

	// Invalidate caches after successful toggle
	s.invalidateAfterEnvironmentUpdate(ctx, existing)

	return nil
}

// SetRolloutPercent sets rollout percentage and invalidates caches
func (s *cachedFeatureFlagService) SetRolloutPercent(ctx context.Context, id uuid.UUID, environment string, percent int, updatedBy uuid.UUID) error {
	// Get existing flag for cache invalidation
	existing, err := s.baseService.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.baseService.SetRolloutPercent(ctx, id, environment, percent, updatedBy); err != nil {
		return err
	}

	// Invalidate caches after successful update
	s.invalidateAfterEnvironmentUpdate(ctx, existing)

	return nil
}

// EvaluateFlag evaluates a feature flag (uses caching for flag retrieval)
func (s *cachedFeatureFlagService) EvaluateFlag(ctx context.Context, flagName, serviceName, environment string, evaluationContext map[string]interface{}) (bool, error) {
	// This method benefits from caching the flag retrieval part
	return s.baseService.EvaluateFlag(ctx, flagName, serviceName, environment, evaluationContext)
}

// GetEnvironments retrieves all unique environment names with caching
func (s *cachedFeatureFlagService) GetEnvironments(ctx context.Context) ([]string, error) {
	cacheKey := featureFlagEnvsKey

	// Try to get from cache first
	if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
		var environments []string
		if err := json.Unmarshal(cachedData, &environments); err == nil {
			return environments, nil
		}
	}

	// Cache miss or error, get from database
	environments, err := s.baseService.GetEnvironments(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result with longer TTL since environments change less frequently
	if envData, err := json.Marshal(environments); err == nil {
		s.cache.Set(ctx, cacheKey, envData, s.cacheTTL*2) // 2x TTL for environments
	}

	return environments, nil
}

// GetServices retrieves all unique service names with caching
func (s *cachedFeatureFlagService) GetServices(ctx context.Context) ([]string, error) {
	cacheKey := featureFlagServicesKey

	// Try to get from cache first
	if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
		var services []string
		if err := json.Unmarshal(cachedData, &services); err == nil {
			return services, nil
		}
	}

	// Cache miss or error, get from database
	services, err := s.baseService.GetServices(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result with longer TTL since services change less frequently
	if serviceData, err := json.Marshal(services); err == nil {
		s.cache.Set(ctx, cacheKey, serviceData, s.cacheTTL*2) // 2x TTL for services
	}

	return services, nil
}

// Exists checks if a feature flag exists (no caching for simple existence checks)
func (s *cachedFeatureFlagService) Exists(ctx context.Context, name, serviceName string) (bool, error) {
	return s.baseService.Exists(ctx, name, serviceName)
}

// Helper methods

type listCacheResult struct {
	Flags []*models.FeatureFlag `json:"flags"`
	Total int64                 `json:"total"`
}

func (s *cachedFeatureFlagService) cacheFeatureFlag(ctx context.Context, key string, flag *models.FeatureFlag) {
	if flagData, err := json.Marshal(flag); err == nil {
		s.cache.Set(ctx, key, flagData, s.cacheTTL)
	}
}

func (s *cachedFeatureFlagService) cacheListResult(ctx context.Context, key string, result *listCacheResult, ttl time.Duration) {
	if resultData, err := json.Marshal(result); err == nil {
		s.cache.Set(ctx, key, resultData, ttl)
	}
}

func (s *cachedFeatureFlagService) isSimpleListQuery(filter *models.FeatureFlagFilter, pagination *models.PaginationParams) bool {
	// Only cache simple queries with basic filters
	if filter == nil {
		return true
	}
	
	// Count non-empty filter fields
	filterCount := 0
	if filter.ServiceName != "" {
		filterCount++
	}
	if filter.Environment != "" {
		filterCount++
	}
	if filter.Enabled != nil {
		filterCount++
	}
	if filter.Name != "" {
		filterCount++
	}
	if filter.CreatedBy != uuid.Nil {
		filterCount++
	}

	// Only cache queries with 0-2 simple filters and reasonable pagination
	return filterCount <= 2 && pagination != nil && pagination.PageSize <= 50
}

func (s *cachedFeatureFlagService) buildListCacheKey(filter *models.FeatureFlagFilter, pagination *models.PaginationParams) string {
	// Create a simple hash-like key for caching list results
	key := "feature_flag:list:"
	if filter != nil {
		if filter.ServiceName != "" {
			key += fmt.Sprintf("svc:%s:", filter.ServiceName)
		}
		if filter.Environment != "" {
			key += fmt.Sprintf("env:%s:", filter.Environment)
		}
		if filter.Enabled != nil {
			key += fmt.Sprintf("enabled:%t:", *filter.Enabled)
		}
	}
	if pagination != nil {
		key += fmt.Sprintf("page:%d:limit:%d", pagination.Page, pagination.PageSize)
	}
	return key
}

func (s *cachedFeatureFlagService) invalidateAfterCreate(ctx context.Context, flag *models.FeatureFlag) {
	// Invalidate list caches and aggregates
	s.cache.DeletePattern(ctx, featureFlagListPattern)
	s.cache.Delete(ctx, featureFlagEnvsKey)
	s.cache.Delete(ctx, featureFlagServicesKey)
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagServiceKey, flag.ServiceName))
}

func (s *cachedFeatureFlagService) invalidateAfterUpdate(ctx context.Context, existing, updated *models.FeatureFlag) {
	// Invalidate specific flag caches
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagByIDKey, updated.ID.String()))
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagByNameKey, updated.ServiceName, updated.Name))
	
	// If service name changed, invalidate old name cache too
	if existing.ServiceName != updated.ServiceName {
		s.cache.Delete(ctx, fmt.Sprintf(featureFlagByNameKey, existing.ServiceName, existing.Name))
		s.cache.Delete(ctx, fmt.Sprintf(featureFlagServiceKey, existing.ServiceName))
	}
	
	// Invalidate aggregates and lists
	s.cache.DeletePattern(ctx, featureFlagListPattern)
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagServiceKey, updated.ServiceName))
	s.cache.Delete(ctx, featureFlagEnvsKey)
	s.cache.Delete(ctx, featureFlagServicesKey)
}

func (s *cachedFeatureFlagService) invalidateAfterDelete(ctx context.Context, flag *models.FeatureFlag) {
	// Invalidate specific flag caches
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagByIDKey, flag.ID.String()))
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagByNameKey, flag.ServiceName, flag.Name))
	
	// Invalidate aggregates and lists
	s.cache.DeletePattern(ctx, featureFlagListPattern)
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagServiceKey, flag.ServiceName))
	s.cache.Delete(ctx, featureFlagEnvsKey)
	s.cache.Delete(ctx, featureFlagServicesKey)
}

func (s *cachedFeatureFlagService) invalidateAfterEnvironmentUpdate(ctx context.Context, flag *models.FeatureFlag) {
	// Invalidate specific flag caches (environments changed)
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagByIDKey, flag.ID.String()))
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagByNameKey, flag.ServiceName, flag.Name))
	
	// Invalidate lists that might be filtered by environment state
	s.cache.DeletePattern(ctx, featureFlagListPattern)
	s.cache.Delete(ctx, fmt.Sprintf(featureFlagServiceKey, flag.ServiceName))
}