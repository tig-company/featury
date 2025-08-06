package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tig-company/featury/internal/cache"
	"github.com/tig-company/featury/internal/models"
)

// Mock implementations
type MockFeatureFlagService struct {
	mock.Mock
}

func (m *MockFeatureFlagService) Create(ctx context.Context, req *models.CreateFeatureFlagRequest, createdBy uuid.UUID) (*models.FeatureFlag, error) {
	args := m.Called(ctx, req, createdBy)
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagService) GetByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagService) GetByName(ctx context.Context, name, serviceName string) (*models.FeatureFlag, error) {
	args := m.Called(ctx, name, serviceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateFeatureFlagRequest, updatedBy uuid.UUID) (*models.FeatureFlag, error) {
	args := m.Called(ctx, id, req, updatedBy)
	return args.Get(0).(*models.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	args := m.Called(ctx, id, deletedBy)
	return args.Error(0)
}

func (m *MockFeatureFlagService) List(ctx context.Context, filter *models.FeatureFlagFilter, pagination *models.PaginationParams) ([]*models.FeatureFlag, int64, error) {
	args := m.Called(ctx, filter, pagination)
	return args.Get(0).([]*models.FeatureFlag), args.Get(1).(int64), args.Error(2)
}

func (m *MockFeatureFlagService) ListByService(ctx context.Context, serviceName string) ([]*models.FeatureFlag, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).([]*models.FeatureFlag), args.Error(1)
}

func (m *MockFeatureFlagService) UpdateEnvironment(ctx context.Context, id uuid.UUID, environment string, config *models.EnvironmentConfig, updatedBy uuid.UUID) error {
	args := m.Called(ctx, id, environment, config, updatedBy)
	return args.Error(0)
}

func (m *MockFeatureFlagService) ToggleEnvironment(ctx context.Context, id uuid.UUID, environment string, enabled bool, updatedBy uuid.UUID) error {
	args := m.Called(ctx, id, environment, enabled, updatedBy)
	return args.Error(0)
}

func (m *MockFeatureFlagService) SetRolloutPercent(ctx context.Context, id uuid.UUID, environment string, percent int, updatedBy uuid.UUID) error {
	args := m.Called(ctx, id, environment, percent, updatedBy)
	return args.Error(0)
}

func (m *MockFeatureFlagService) EvaluateFlag(ctx context.Context, flagName, serviceName, environment string, evaluationContext map[string]interface{}) (bool, error) {
	args := m.Called(ctx, flagName, serviceName, environment, evaluationContext)
	return args.Bool(0), args.Error(1)
}

func (m *MockFeatureFlagService) GetEnvironments(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFeatureFlagService) GetServices(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockFeatureFlagService) Exists(ctx context.Context, name, serviceName string) (bool, error) {
	args := m.Called(ctx, name, serviceName)
	return args.Bool(0), args.Error(1)
}

func createTestFeatureFlag() *models.FeatureFlag {
	return &models.FeatureFlag{
		ID:          uuid.New(),
		Name:        "test_flag",
		ServiceName: "test_service",
		Description: "Test feature flag",
		CreatedBy:   uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Environments: map[string]models.EnvironmentConfig{
			"prod": {
				Enabled:        true,
				RolloutPercent: 50,
				UpdatedBy:      uuid.New(),
				UpdatedAt:      time.Now(),
			},
		},
	}
}

func TestCachedFeatureFlagService_GetByID(t *testing.T) {
	mockService := new(MockFeatureFlagService)
	cacheMetrics := cache.NewCacheMetrics()
	memCache := cache.NewMemoryCache(cacheMetrics)
	defer memCache.Close()

	cachedService := NewCachedFeatureFlagService(mockService, memCache, 5*time.Minute)
	ctx := context.Background()

	flag := createTestFeatureFlag()
	flagID := flag.ID

	t.Run("Cache Miss - Fetch from Service", func(t *testing.T) {
		mockService.On("GetByID", ctx, flagID).Return(flag, nil).Once()

		result, err := cachedService.GetByID(ctx, flagID)
		
		require.NoError(t, err)
		assert.Equal(t, flag.ID, result.ID)
		assert.Equal(t, flag.Name, result.Name)
		
		mockService.AssertExpectations(t)
	})

	t.Run("Cache Hit - Return from Cache", func(t *testing.T) {
		// First call should populate cache
		mockService.On("GetByID", ctx, flagID).Return(flag, nil).Once()
		_, err := cachedService.GetByID(ctx, flagID)
		require.NoError(t, err)

		// Second call should use cache (no mock call expected)
		result, err := cachedService.GetByID(ctx, flagID)
		
		require.NoError(t, err)
		assert.Equal(t, flag.ID, result.ID)
		assert.Equal(t, flag.Name, result.Name)
		
		mockService.AssertExpectations(t)
	})
}

func TestCachedFeatureFlagService_GetByName(t *testing.T) {
	mockService := new(MockFeatureFlagService)
	cacheMetrics := cache.NewCacheMetrics()
	memCache := cache.NewMemoryCache(cacheMetrics)
	defer memCache.Close()

	cachedService := NewCachedFeatureFlagService(mockService, memCache, 5*time.Minute)
	ctx := context.Background()

	flag := createTestFeatureFlag()

	t.Run("Cache Miss - Fetch from Service", func(t *testing.T) {
		mockService.On("GetByName", ctx, flag.Name, flag.ServiceName).Return(flag, nil).Once()

		result, err := cachedService.GetByName(ctx, flag.Name, flag.ServiceName)
		
		require.NoError(t, err)
		assert.Equal(t, flag.ID, result.ID)
		assert.Equal(t, flag.Name, result.Name)
		
		mockService.AssertExpectations(t)
	})

	t.Run("Cache Hit - Both Name and ID Cached", func(t *testing.T) {
		// First call populates both name and ID caches
		mockService.On("GetByName", ctx, flag.Name, flag.ServiceName).Return(flag, nil).Once()
		_, err := cachedService.GetByName(ctx, flag.Name, flag.ServiceName)
		require.NoError(t, err)

		// Second call by name should use cache
		result, err := cachedService.GetByName(ctx, flag.Name, flag.ServiceName)
		require.NoError(t, err)
		assert.Equal(t, flag.ID, result.ID)

		// Call by ID should also use cache (from name cache population)
		result, err = cachedService.GetByID(ctx, flag.ID)
		require.NoError(t, err)
		assert.Equal(t, flag.Name, result.Name)
		
		mockService.AssertExpectations(t)
	})
}

func TestCachedFeatureFlagService_Create(t *testing.T) {
	mockService := new(MockFeatureFlagService)
	cacheMetrics := cache.NewCacheMetrics()
	memCache := cache.NewMemoryCache(cacheMetrics)
	defer memCache.Close()

	cachedService := NewCachedFeatureFlagService(mockService, memCache, 5*time.Minute)
	ctx := context.Background()

	flag := createTestFeatureFlag()
	userID := uuid.New()
	
	req := &models.CreateFeatureFlagRequest{
		Name:         flag.Name,
		ServiceName:  flag.ServiceName,
		Description:  flag.Description,
		Environments: flag.Environments,
	}

	t.Run("Create Flag and Invalidate Caches", func(t *testing.T) {
		// Pre-populate some cache entries that should be invalidated
		listCacheKey := "feature_flag:list:page:1:limit:20"
		cacheData, _ := json.Marshal([]*models.FeatureFlag{flag})
		memCache.Set(ctx, listCacheKey, cacheData, time.Minute)

		mockService.On("Create", ctx, req, userID).Return(flag, nil).Once()

		result, err := cachedService.Create(ctx, req, userID)
		
		require.NoError(t, err)
		assert.Equal(t, flag.ID, result.ID)
		
		// Verify list cache was invalidated
		_, err = memCache.Get(ctx, listCacheKey)
		assert.Error(t, err) // Should be cache miss after invalidation
		
		mockService.AssertExpectations(t)
	})
}

func TestCachedFeatureFlagService_Update(t *testing.T) {
	mockService := new(MockFeatureFlagService)
	cacheMetrics := cache.NewCacheMetrics()
	memCache := cache.NewMemoryCache(cacheMetrics)
	defer memCache.Close()

	cachedService := NewCachedFeatureFlagService(mockService, memCache, 5*time.Minute)
	ctx := context.Background()

	originalFlag := createTestFeatureFlag()
	updatedFlag := createTestFeatureFlag()
	updatedFlag.ID = originalFlag.ID
	updatedFlag.Description = "Updated description"
	
	userID := uuid.New()
	updateReq := &models.UpdateFeatureFlagRequest{
		Description: &updatedFlag.Description,
	}

	t.Run("Update Flag and Invalidate Caches", func(t *testing.T) {
		// Pre-populate cache with original flag
		idCacheKey := fmt.Sprintf(featureFlagByIDKey, originalFlag.ID.String())
		nameCacheKey := fmt.Sprintf(featureFlagByNameKey, originalFlag.ServiceName, originalFlag.Name)
		flagData, _ := json.Marshal(originalFlag)
		memCache.Set(ctx, idCacheKey, flagData, time.Minute)
		memCache.Set(ctx, nameCacheKey, flagData, time.Minute)

		mockService.On("GetByID", ctx, originalFlag.ID).Return(originalFlag, nil).Once()
		mockService.On("Update", ctx, originalFlag.ID, updateReq, userID).Return(updatedFlag, nil).Once()

		result, err := cachedService.Update(ctx, originalFlag.ID, updateReq, userID)
		
		require.NoError(t, err)
		assert.Equal(t, updatedFlag.Description, result.Description)
		
		// Verify caches were invalidated
		_, err = memCache.Get(ctx, idCacheKey)
		assert.Error(t, err) // Should be cache miss after invalidation
		_, err = memCache.Get(ctx, nameCacheKey)
		assert.Error(t, err) // Should be cache miss after invalidation
		
		mockService.AssertExpectations(t)
	})
}

func TestCachedFeatureFlagService_ListByService(t *testing.T) {
	mockService := new(MockFeatureFlagService)
	cacheMetrics := cache.NewCacheMetrics()
	memCache := cache.NewMemoryCache(cacheMetrics)
	defer memCache.Close()

	cachedService := NewCachedFeatureFlagService(mockService, memCache, 5*time.Minute)
	ctx := context.Background()

	flags := []*models.FeatureFlag{createTestFeatureFlag()}
	serviceName := "test_service"

	t.Run("Cache Miss - Fetch from Service", func(t *testing.T) {
		mockService.On("ListByService", ctx, serviceName).Return(flags, nil).Once()

		result, err := cachedService.ListByService(ctx, serviceName)
		
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, flags[0].ID, result[0].ID)
		
		mockService.AssertExpectations(t)
	})

	t.Run("Cache Hit - Return from Cache", func(t *testing.T) {
		// First call populates cache
		mockService.On("ListByService", ctx, serviceName).Return(flags, nil).Once()
		_, err := cachedService.ListByService(ctx, serviceName)
		require.NoError(t, err)

		// Second call should use cache
		result, err := cachedService.ListByService(ctx, serviceName)
		
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, flags[0].ID, result[0].ID)
		
		mockService.AssertExpectations(t)
	})
}

func TestCachedFeatureFlagService_IsSimpleListQuery(t *testing.T) {
	mockService := new(MockFeatureFlagService)
	cacheMetrics := cache.NewCacheMetrics()
	memCache := cache.NewMemoryCache(cacheMetrics)
	defer memCache.Close()

	cachedService := NewCachedFeatureFlagService(mockService, memCache, 5*time.Minute).(*cachedFeatureFlagService)

	t.Run("Simple Query - Should Cache", func(t *testing.T) {
		filter := &models.FeatureFlagFilter{
			ServiceName: "test_service",
		}
		pagination := &models.PaginationParams{
			Page:     1,
			PageSize: 20,
		}

		isSimple := cachedService.isSimpleListQuery(filter, pagination)
		assert.True(t, isSimple)
	})

	t.Run("Complex Query - Should Not Cache", func(t *testing.T) {
		enabled := true
		filter := &models.FeatureFlagFilter{
			ServiceName: "test_service",
			Environment: "prod",
			Enabled:     &enabled,
			Name:        "flag_name",
			CreatedBy:   uuid.New(),
		}
		pagination := &models.PaginationParams{
			Page:     1,
			PageSize: 20,
		}

		isSimple := cachedService.isSimpleListQuery(filter, pagination)
		assert.False(t, isSimple) // Too many filters (5 > 2)
	})

	t.Run("Large Page Size - Should Not Cache", func(t *testing.T) {
		filter := &models.FeatureFlagFilter{
			ServiceName: "test_service",
		}
		pagination := &models.PaginationParams{
			Page:     1,
			PageSize: 100, // Too large
		}

		isSimple := cachedService.isSimpleListQuery(filter, pagination)
		assert.False(t, isSimple)
	})
}