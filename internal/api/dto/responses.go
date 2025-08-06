package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

// FeatureFlagResponse represents a feature flag in API responses
type FeatureFlagResponse struct {
	ID           uuid.UUID                     `json:"id"`
	Name         string                        `json:"name"`
	ServiceName  string                        `json:"service_name"`
	Description  string                        `json:"description"`
	CreatedBy    uuid.UUID                     `json:"created_by"`
	CreatedAt    time.Time                     `json:"created_at"`
	UpdatedAt    time.Time                     `json:"updated_at"`
	Environments map[string]EnvironmentResponse `json:"environments"`
}

// EnvironmentResponse represents environment configuration in responses
type EnvironmentResponse struct {
	Enabled        bool                    `json:"enabled"`
	RolloutPercent int                     `json:"rollout_percent"`
	Rules          []ConditionalRuleResponse `json:"rules,omitempty"`
	UpdatedBy      uuid.UUID               `json:"updated_by"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

// ConditionalRuleResponse represents a conditional rule in responses
type ConditionalRuleResponse struct {
	ID        uuid.UUID   `json:"id"`
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
	Enabled   bool        `json:"enabled"`
}

// FromModel creates a FeatureFlagResponse from a model
func NewFeatureFlagResponse(ff *models.FeatureFlag) *FeatureFlagResponse {
	environments := make(map[string]EnvironmentResponse)
	
	for envName, envConfig := range ff.Environments {
		rules := make([]ConditionalRuleResponse, len(envConfig.Rules))
		for i, rule := range envConfig.Rules {
			rules[i] = ConditionalRuleResponse{
				ID:        rule.ID,
				Attribute: rule.Attribute,
				Operator:  rule.Operator,
				Value:     rule.Value,
				Enabled:   rule.Enabled,
			}
		}
		
		environments[envName] = EnvironmentResponse{
			Enabled:        envConfig.Enabled,
			RolloutPercent: envConfig.RolloutPercent,
			Rules:          rules,
			UpdatedBy:      envConfig.UpdatedBy,
			UpdatedAt:      envConfig.UpdatedAt,
		}
	}
	
	return &FeatureFlagResponse{
		ID:           ff.ID,
		Name:         ff.Name,
		ServiceName:  ff.ServiceName,
		Description:  ff.Description,
		CreatedBy:    ff.CreatedBy,
		CreatedAt:    ff.CreatedAt,
		UpdatedAt:    ff.UpdatedAt,
		Environments: environments,
	}
}

// NewFeatureFlagResponseList creates a list of FeatureFlagResponse from models
func NewFeatureFlagResponseList(flags []*models.FeatureFlag) []*FeatureFlagResponse {
	responses := make([]*FeatureFlagResponse, len(flags))
	for i, flag := range flags {
		responses[i] = NewFeatureFlagResponse(flag)
	}
	return responses
}

// PaginatedFeatureFlagsResponse represents a paginated list of feature flags
type PaginatedFeatureFlagsResponse struct {
	Data       []*FeatureFlagResponse `json:"data"`
	Pagination PaginationResponse     `json:"pagination"`
}

// PaginationResponse represents pagination metadata in responses
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPaginatedFeatureFlagsResponse creates a paginated response
func NewPaginatedFeatureFlagsResponse(flags []*models.FeatureFlag, params *models.PaginationParams, totalCount int64) *PaginatedFeatureFlagsResponse {
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}
	
	return &PaginatedFeatureFlagsResponse{
		Data: NewFeatureFlagResponseList(flags),
		Pagination: PaginationResponse{
			Page:       params.Page,
			Limit:      params.PageSize,
			Total:      totalCount,
			TotalPages: totalPages,
			HasNext:    params.Page < totalPages,
			HasPrev:    params.Page > 1,
		},
	}
}

// CreatedResponse represents a creation success response
type CreatedResponse struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"message"`
}

// NewCreatedResponse creates a new creation response
func NewCreatedResponse(id uuid.UUID, resourceType string) *CreatedResponse {
	return &CreatedResponse{
		ID:      id,
		Message: resourceType + " created successfully",
	}
}

// UpdatedResponse represents an update success response
type UpdatedResponse struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"message"`
}

// NewUpdatedResponse creates a new update response
func NewUpdatedResponse(id uuid.UUID, resourceType string) *UpdatedResponse {
	return &UpdatedResponse{
		ID:      id,
		Message: resourceType + " updated successfully",
	}
}

// DeletedResponse represents a deletion success response
type DeletedResponse struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"message"`
}

// NewDeletedResponse creates a new deletion response
func NewDeletedResponse(id uuid.UUID, resourceType string) *DeletedResponse {
	return &DeletedResponse{
		ID:      id,
		Message: resourceType + " deleted successfully",
	}
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// NewHealthCheckResponse creates a new health check response
func NewHealthCheckResponse(status, service, version string, checks map[string]string) *HealthCheckResponse {
	return &HealthCheckResponse{
		Status:    status,
		Service:   service,
		Version:   version,
		Timestamp: time.Now(),
		Checks:    checks,
	}
}

// MetricsResponse represents a metrics response
type MetricsResponse struct {
	Metrics   map[string]interface{} `json:"metrics"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewMetricsResponse creates a new metrics response
func NewMetricsResponse(metrics map[string]interface{}) *MetricsResponse {
	return &MetricsResponse{
		Metrics:   metrics,
		Timestamp: time.Now(),
	}
}

// ErrorResponse represents an error response (already handled by pkg/errors but included for completeness)
type ErrorResponse struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Fields    map[string]string      `json:"fields,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// EnvironmentListResponse represents a list of environments
type EnvironmentListResponse struct {
	Environments []string `json:"environments"`
}

// NewEnvironmentListResponse creates a new environment list response
func NewEnvironmentListResponse(environments []string) *EnvironmentListResponse {
	return &EnvironmentListResponse{
		Environments: environments,
	}
}

// ServiceListResponse represents a list of services
type ServiceListResponse struct {
	Services []string `json:"services"`
}

// NewServiceListResponse creates a new service list response
func NewServiceListResponse(services []string) *ServiceListResponse {
	return &ServiceListResponse{
		Services: services,
	}
}