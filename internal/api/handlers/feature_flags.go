package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/api/dto"
	"github.com/tig-company/featury/internal/service"
	"github.com/tig-company/featury/pkg/errors"
)

// FeatureFlagHandlers contains all handlers for feature flag operations
type FeatureFlagHandlers struct {
	service service.FeatureFlagService
	audit   service.AuditService
}

// NewFeatureFlagHandlers creates a new feature flag handlers instance
func NewFeatureFlagHandlers(service service.FeatureFlagService, audit service.AuditService) *FeatureFlagHandlers {
	return &FeatureFlagHandlers{
		service: service,
		audit:   audit,
	}
}

// ListFeatureFlags handles GET /features - List feature flags with filtering and pagination
func (h *FeatureFlagHandlers) ListFeatureFlags(c *gin.Context) {
	var req dto.ListFeatureFlagsRequest
	
	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		errors.AbortWithValidation(c, "Invalid query parameters", extractValidationFields(err))
		return
	}
	
	// Convert to filter and pagination
	filter, pagination := req.ToFilterAndPagination()
	
	// Call service
	flags, totalCount, err := h.service.List(c.Request.Context(), filter, pagination)
	if err != nil {
		c.Error(fmt.Errorf("service error: %w", err))
		return
	}
	
	// Convert to response
	response := dto.NewPaginatedFeatureFlagsResponse(flags, pagination, totalCount)
	
	c.JSON(http.StatusOK, response)
}

// CreateFeatureFlag handles POST /features - Create new feature flag
func (h *FeatureFlagHandlers) CreateFeatureFlag(c *gin.Context) {
	var req dto.CreateFeatureFlagRequest
	
	// Bind JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.AbortWithValidation(c, "Invalid request body", extractValidationFields(err))
		return
	}
	
	// Additional validation
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		fields := make(map[string]string)
		for _, err := range validationErrors {
			fields["validation"] = err
		}
		errors.AbortWithValidation(c, "Request validation failed", fields)
		return
	}
	
	// Get authenticated user ID
	userID, exists := c.Get("UserID")
	if !exists {
		errors.AbortWithUnauthorized(c, "User not authenticated")
		return
	}
	
	createdBy, ok := userID.(uuid.UUID)
	if !ok {
		errors.AbortWithError(c, errors.NewInternalError("Invalid user ID format"))
		return
	}
	
	// Convert to service model
	serviceReq := req.ToModel(createdBy)
	
	// Call service
	flag, err := h.service.Create(c.Request.Context(), serviceReq, createdBy)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	
	// Convert to response
	response := dto.NewFeatureFlagResponse(flag)
	
	c.JSON(http.StatusCreated, response)
}

// GetFeatureFlag handles GET /features/:id - Get specific feature flag
func (h *FeatureFlagHandlers) GetFeatureFlag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeInvalidUUID, "Invalid feature flag ID"))
		return
	}
	
	// Call service
	flag, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	
	// Convert to response
	response := dto.NewFeatureFlagResponse(flag)
	
	c.JSON(http.StatusOK, response)
}

// UpdateFeatureFlag handles PUT/PATCH /features/:id - Update feature flag
func (h *FeatureFlagHandlers) UpdateFeatureFlag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeInvalidUUID, "Invalid feature flag ID"))
		return
	}
	
	var req dto.UpdateFeatureFlagRequest
	
	// Bind JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.AbortWithValidation(c, "Invalid request body", extractValidationFields(err))
		return
	}
	
	// Check if there are any changes
	if !req.HasChanges() {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeBadRequest, "No changes provided"))
		return
	}
	
	// Additional validation
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		fields := make(map[string]string)
		for _, err := range validationErrors {
			fields["validation"] = err
		}
		errors.AbortWithValidation(c, "Request validation failed", fields)
		return
	}
	
	// Get authenticated user ID
	userID, exists := c.Get("UserID")
	if !exists {
		errors.AbortWithUnauthorized(c, "User not authenticated")
		return
	}
	
	updatedBy, ok := userID.(uuid.UUID)
	if !ok {
		errors.AbortWithError(c, errors.NewInternalError("Invalid user ID format"))
		return
	}
	
	// Convert to service model
	serviceReq := req.ToModel(updatedBy)
	
	// Call service
	flag, err := h.service.Update(c.Request.Context(), id, serviceReq, updatedBy)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	
	// Convert to response
	response := dto.NewFeatureFlagResponse(flag)
	
	c.JSON(http.StatusOK, response)
}

// DeleteFeatureFlag handles DELETE /features/:id - Delete (soft delete) feature flag
func (h *FeatureFlagHandlers) DeleteFeatureFlag(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeInvalidUUID, "Invalid feature flag ID"))
		return
	}
	
	// Get authenticated user ID
	userID, exists := c.Get("UserID")
	if !exists {
		errors.AbortWithUnauthorized(c, "User not authenticated")
		return
	}
	
	deletedBy, ok := userID.(uuid.UUID)
	if !ok {
		errors.AbortWithError(c, errors.NewInternalError("Invalid user ID format"))
		return
	}
	
	// Call service
	err = h.service.Delete(c.Request.Context(), id, deletedBy)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	
	c.JSON(http.StatusNoContent, gin.H{})
}

// ToggleEnvironment handles POST /features/:id/environments/:environment/toggle
func (h *FeatureFlagHandlers) ToggleEnvironment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeInvalidUUID, "Invalid feature flag ID"))
		return
	}
	
	environment := strings.TrimSpace(c.Param("environment"))
	if environment == "" {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeBadRequest, "Environment name is required"))
		return
	}
	
	var req dto.ToggleEnvironmentRequest
	
	// Bind JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.AbortWithValidation(c, "Invalid request body", extractValidationFields(err))
		return
	}
	
	// Get authenticated user ID
	userID, exists := c.Get("UserID")
	if !exists {
		errors.AbortWithUnauthorized(c, "User not authenticated")
		return
	}
	
	updatedBy, ok := userID.(uuid.UUID)
	if !ok {
		errors.AbortWithError(c, errors.NewInternalError("Invalid user ID format"))
		return
	}
	
	// Call service
	err = h.service.ToggleEnvironment(c.Request.Context(), id, environment, req.Enabled, updatedBy)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	
	response := dto.NewUpdatedResponse(id, "Environment")
	c.JSON(http.StatusOK, response)
}

// UpdateRollout handles POST /features/:id/environments/:environment/rollout
func (h *FeatureFlagHandlers) UpdateRollout(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeInvalidUUID, "Invalid feature flag ID"))
		return
	}
	
	environment := strings.TrimSpace(c.Param("environment"))
	if environment == "" {
		errors.AbortWithError(c, errors.NewAPIError(errors.ErrorCodeBadRequest, "Environment name is required"))
		return
	}
	
	var req dto.UpdateRolloutRequest
	
	// Bind JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.AbortWithValidation(c, "Invalid request body", extractValidationFields(err))
		return
	}
	
	// Get authenticated user ID
	userID, exists := c.Get("UserID")
	if !exists {
		errors.AbortWithUnauthorized(c, "User not authenticated")
		return
	}
	
	updatedBy, ok := userID.(uuid.UUID)
	if !ok {
		errors.AbortWithError(c, errors.NewInternalError("Invalid user ID format"))
		return
	}
	
	// Call service
	err = h.service.SetRolloutPercent(c.Request.Context(), id, environment, req.Percent, updatedBy)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	
	response := dto.NewUpdatedResponse(id, "Rollout percentage")
	c.JSON(http.StatusOK, response)
}

// GetEnvironments handles GET /environments - Get all unique environment names
func (h *FeatureFlagHandlers) GetEnvironments(c *gin.Context) {
	environments, err := h.service.GetEnvironments(c.Request.Context())
	if err != nil {
		c.Error(fmt.Errorf("service error: %w", err))
		return
	}
	
	response := dto.NewEnvironmentListResponse(environments)
	c.JSON(http.StatusOK, response)
}

// GetServices handles GET /services - Get all unique service names
func (h *FeatureFlagHandlers) GetServices(c *gin.Context) {
	services, err := h.service.GetServices(c.Request.Context())
	if err != nil {
		c.Error(fmt.Errorf("service error: %w", err))
		return
	}
	
	response := dto.NewServiceListResponse(services)
	c.JSON(http.StatusOK, response)
}

// Helper functions

// handleServiceError handles service layer errors and converts them to appropriate HTTP responses
func handleServiceError(c *gin.Context, err error) {
	// Check if it's already an APIError
	if apiErr, ok := err.(*errors.APIError); ok {
		errors.AbortWithError(c, apiErr)
		return
	}
	
	// Check for common service errors
	errStr := strings.ToLower(err.Error())
	
	if strings.Contains(errStr, "not found") {
		errors.AbortWithNotFound(c, "Feature flag")
		return
	}
	
	if strings.Contains(errStr, "already exists") {
		errors.AbortWithError(c, errors.NewConflictError("Feature flag already exists"))
		return
	}
	
	if strings.Contains(errStr, "validation") {
		errors.AbortWithValidation(c, err.Error(), nil)
		return
	}
	
	// Default to internal error
	c.Error(fmt.Errorf("service error: %w", err))
}

// extractValidationFields extracts field-specific validation errors from binding errors
func extractValidationFields(err error) map[string]string {
	// This is a simplified implementation
	// In a production environment, you'd want more sophisticated error parsing
	fields := make(map[string]string)
	fields["error"] = err.Error()
	return fields
}