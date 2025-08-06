package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/auth"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/repository"
	"github.com/tig-company/featury/pkg/errors"
)

// AuthService handles authentication business logic
type AuthService struct {
	apiKeyRepo       repository.APIKeyRepository
	userRepo         repository.UserRepository
	auditRepo        repository.AuditLogRepository
	keyHasher        *auth.KeyHasher
	permissionChecker *auth.PermissionChecker
}

// NewAuthService creates a new authentication service
func NewAuthService(
	apiKeyRepo repository.APIKeyRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
) *AuthService {
	return &AuthService{
		apiKeyRepo:       apiKeyRepo,
		userRepo:         userRepo,
		auditRepo:        auditRepo,
		keyHasher:        auth.NewKeyHasher(nil), // Use default config
		permissionChecker: auth.NewPermissionChecker(),
	}
}

// AuthenticateAPIKey authenticates an API key and returns the associated API key model
func (as *AuthService) AuthenticateAPIKey(ctx context.Context, key string) (*models.APIKey, error) {
	// Validate key format first
	if !auth.IsValidAPIKey(key) {
		return nil, errors.NewInvalidAPIKeyError()
	}

	// Find all API keys and check hashes
	// In a production system, you might want to index by key prefix for better performance
	apiKeys, _, err := as.apiKeyRepo.List(ctx, &models.APIKeyFilter{}, &models.PaginationParams{Page: 1, PageSize: 1000})
	if err != nil {
		return nil, errors.NewInternalError("Failed to retrieve API keys")
	}

	var matchedKey *models.APIKey
	for _, apiKey := range apiKeys {
		if as.keyHasher.CompareKey(key, apiKey.KeyHash) {
			matchedKey = apiKey
			break
		}
	}

	if matchedKey == nil {
		return nil, errors.NewInvalidAPIKeyError()
	}

	// Check if key is expired
	if matchedKey.IsExpired() {
		return nil, errors.NewExpiredAPIKeyError()
	}

	// Update last used timestamp
	now := time.Now()
	matchedKey.LastUsedAt = &now
	
	// Update in background, don't fail request if this fails
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		as.apiKeyRepo.UpdateLastUsed(updateCtx, matchedKey.ID)
	}()

	return matchedKey, nil
}

// CreateAPIKey creates a new API key
func (as *AuthService) CreateAPIKey(ctx context.Context, userID uuid.UUID, req models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		if ve, ok := err.(*models.ValidationError); ok {
			return nil, errors.NewValidationError(ve.Message).WithField(ve.Field, ve.Message)
		}
		return nil, errors.NewValidationError(err.Error())
	}

	// Verify user exists
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, errors.NewNotFoundError("User")
		}
		return nil, errors.NewInternalError("Failed to verify user")
	}

	// Generate new API key
	plainKey, err := auth.GenerateAPIKey()
	if err != nil {
		return nil, errors.NewInternalError("Failed to generate API key")
	}

	// Hash the key
	keyHash, err := as.keyHasher.HashKey(plainKey)
	if err != nil {
		return nil, errors.NewInternalError("Failed to hash API key")
	}

	// Create API key model
	apiKey := &models.APIKey{
		ID:          uuid.New(),
		KeyHash:     keyHash,
		UserID:      userID,
		Name:        req.Name,
		Permissions: req.Permissions,
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   time.Now(),
	}

	// Save to repository
	if err := as.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, errors.NewInternalError("Failed to create API key")
	}

	// Log audit event
	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     models.AuditActionCreate,
		EntityType: "api_key",
		EntityID:   apiKey.ID,
		Metadata: models.JSONB{
			"api_key_name": req.Name,
			"permissions":  req.Permissions,
			"created_by":   user.Email,
		},
		CreatedAt: time.Now(),
	}
	
	// Log audit in background, don't fail request if this fails
	go func() {
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		as.auditRepo.Create(auditCtx, auditLog)
	}()

	return &models.CreateAPIKeyResponse{
		APIKey: apiKey.ToResponse(),
		Key:    plainKey, // Only returned once
	}, nil
}

// ValidatePermissions validates if an API key has required permissions for a resource and action
func (as *AuthService) ValidatePermissions(apiKey *models.APIKey, resource, action string) error {
	resourceType := auth.ResourceType(resource)
	
	if !as.permissionChecker.CheckResourceAccess(apiKey.Permissions, resourceType, action) {
		requiredPerm := as.getRequiredPermission(resourceType, action)
		return errors.NewInsufficientPermissionsError(string(requiredPerm))
	}
	
	return nil
}

// getRequiredPermission determines the required permission for a resource and action
func (as *AuthService) getRequiredPermission(resource auth.ResourceType, action string) models.Permission {
	switch resource {
	case auth.ResourceTypeFeatureFlags:
		switch action {
		case "read", "get", "list":
			return models.PermissionReadFeatureFlags
		case "write", "create", "update", "post", "put", "patch":
			return models.PermissionWriteFeatureFlags
		case "delete":
			return models.PermissionDeleteFeatureFlags
		}
	case auth.ResourceTypeUsers:
		switch action {
		case "read", "get", "list":
			return models.PermissionReadUsers
		case "write", "create", "update", "post", "put", "patch":
			return models.PermissionWriteUsers
		}
	case auth.ResourceTypeAPIKeys:
		switch action {
		case "read", "get", "list":
			return models.PermissionReadAPIKeys
		case "write", "create", "update", "post", "put", "patch", "delete":
			return models.PermissionWriteAPIKeys
		}
	case auth.ResourceTypeAuditLogs:
		return models.PermissionReadAuditLogs
	}
	
	return "" // Unknown permission
}

// ListAPIKeys lists API keys with filtering
func (as *AuthService) ListAPIKeys(ctx context.Context, userID uuid.UUID, filter models.APIKeyFilter, pagination models.PaginationParams) ([]*models.APIKeyResponse, *models.PaginatedResponse, error) {
	// Ensure user can only see their own API keys unless they have read:api_keys permission
	filter.UserID = userID

	apiKeys, totalCount, err := as.apiKeyRepo.List(ctx, &filter, &pagination)
	if err != nil {
		return nil, nil, errors.NewInternalError("Failed to list API keys")
	}

	// Convert to response format
	responses := make([]*models.APIKeyResponse, len(apiKeys))
	for i, apiKey := range apiKeys {
		responses[i] = apiKey.ToResponse()
	}

	paginatedResponse := models.NewPaginatedResponse(responses, pagination, totalCount)

	return responses, paginatedResponse, nil
}

// GetAPIKey retrieves an API key by ID
func (as *AuthService) GetAPIKey(ctx context.Context, userID, apiKeyID uuid.UUID) (*models.APIKeyResponse, error) {
	apiKey, err := as.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, errors.NewNotFoundError("API key")
		}
		return nil, errors.NewInternalError("Failed to retrieve API key")
	}

	// Ensure user can only access their own API keys
	if apiKey.UserID != userID {
		return nil, errors.NewNotFoundError("API key")
	}

	return apiKey.ToResponse(), nil
}

// UpdateAPIKey updates an API key
func (as *AuthService) UpdateAPIKey(ctx context.Context, userID, apiKeyID uuid.UUID, req models.UpdateAPIKeyRequest) (*models.APIKeyResponse, error) {
	if !req.HasChanges() {
		return nil, errors.NewValidationError("No changes provided")
	}

	// Get existing API key
	apiKey, err := as.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, errors.NewNotFoundError("API key")
		}
		return nil, errors.NewInternalError("Failed to retrieve API key")
	}

	// Ensure user can only update their own API keys
	if apiKey.UserID != userID {
		return nil, errors.NewNotFoundError("API key")
	}

	// Validate expiry date if provided
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, errors.NewValidationError("Expiration date must be in the future").
			WithField("expires_at", "must be in the future")
	}

	// Update the API key
	updatedKey, err := as.apiKeyRepo.Update(ctx, apiKeyID, &req)
	if err != nil {
		return nil, errors.NewInternalError("Failed to update API key")
	}

	// Log audit event
	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     models.AuditActionUpdate,
		EntityType: "api_key",
		EntityID:   apiKeyID,
		Metadata: models.JSONB{
			"api_key_name": updatedKey.Name,
			"changes":      req,
		},
		CreatedAt: time.Now(),
	}
	
	// Log audit in background
	go func() {
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		as.auditRepo.Create(auditCtx, auditLog)
	}()

	return updatedKey.ToResponse(), nil
}

// DeleteAPIKey deletes an API key
func (as *AuthService) DeleteAPIKey(ctx context.Context, userID, apiKeyID uuid.UUID) error {
	// Get existing API key
	apiKey, err := as.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return errors.NewNotFoundError("API key")
		}
		return errors.NewInternalError("Failed to retrieve API key")
	}

	// Ensure user can only delete their own API keys
	if apiKey.UserID != userID {
		return errors.NewNotFoundError("API key")
	}

	// Delete the API key
	if err := as.apiKeyRepo.Delete(ctx, apiKeyID); err != nil {
		return errors.NewInternalError("Failed to delete API key")
	}

	// Log audit event
	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     models.AuditActionDelete,
		EntityType: "api_key",
		EntityID:   apiKeyID,
		Metadata: models.JSONB{
			"api_key_name": apiKey.Name,
		},
		CreatedAt: time.Now(),
	}
	
	// Log audit in background
	go func() {
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		as.auditRepo.Create(auditCtx, auditLog)
	}()

	return nil
}

// CheckAPIKeyPermission is a helper method to check if an API key has specific permissions
func (as *AuthService) CheckAPIKeyPermission(apiKey *models.APIKey, permissions ...models.Permission) bool {
	return as.permissionChecker.HasAnyPermission(apiKey.Permissions, permissions...)
}