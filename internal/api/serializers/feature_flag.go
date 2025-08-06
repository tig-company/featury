package serializers

import (
	"encoding/json"
	"fmt"

	"github.com/tig-company/featury/internal/api/dto"
	"github.com/tig-company/featury/internal/models"
)

// FeatureFlagSerializer handles serialization of feature flag data
type FeatureFlagSerializer struct {
	includeDeleted      bool
	includeEnvironments []string
	excludeFields       []string
	includeAuditInfo    bool
}

// NewFeatureFlagSerializer creates a new feature flag serializer with default options
func NewFeatureFlagSerializer() *FeatureFlagSerializer {
	return &FeatureFlagSerializer{
		includeDeleted:   false,
		includeAuditInfo: true,
	}
}

// WithDeleted configures the serializer to include deleted items
func (s *FeatureFlagSerializer) WithDeleted(include bool) *FeatureFlagSerializer {
	s.includeDeleted = include
	return s
}

// WithEnvironments configures the serializer to only include specific environments
func (s *FeatureFlagSerializer) WithEnvironments(environments []string) *FeatureFlagSerializer {
	s.includeEnvironments = environments
	return s
}

// WithoutFields configures the serializer to exclude specific fields
func (s *FeatureFlagSerializer) WithoutFields(fields []string) *FeatureFlagSerializer {
	s.excludeFields = fields
	return s
}

// WithoutAuditInfo configures the serializer to exclude audit information
func (s *FeatureFlagSerializer) WithoutAuditInfo() *FeatureFlagSerializer {
	s.includeAuditInfo = false
	return s
}

// Serialize converts a model to response DTO with serializer options
func (s *FeatureFlagSerializer) Serialize(flag *models.FeatureFlag) *dto.FeatureFlagResponse {
	if flag == nil {
		return nil
	}
	
	// Skip deleted items if not configured to include them
	if !s.includeDeleted && flag.IsDeleted() {
		return nil
	}
	
	response := &dto.FeatureFlagResponse{
		ID:          flag.ID,
		Name:        flag.Name,
		ServiceName: flag.ServiceName,
		Description: flag.Description,
	}
	
	// Include audit information if configured
	if s.includeAuditInfo {
		response.CreatedBy = flag.CreatedBy
		response.CreatedAt = flag.CreatedAt
		response.UpdatedAt = flag.UpdatedAt
	}
	
	// Process environments
	response.Environments = s.serializeEnvironments(flag.Environments)
	
	// Apply field exclusions
	s.applyFieldExclusions(response)
	
	return response
}

// SerializeList converts a list of models to response DTOs
func (s *FeatureFlagSerializer) SerializeList(flags []*models.FeatureFlag) []*dto.FeatureFlagResponse {
	responses := make([]*dto.FeatureFlagResponse, 0, len(flags))
	
	for _, flag := range flags {
		if serialized := s.Serialize(flag); serialized != nil {
			responses = append(responses, serialized)
		}
	}
	
	return responses
}

// SerializePaginated creates a paginated response with serialized data
func (s *FeatureFlagSerializer) SerializePaginated(flags []*models.FeatureFlag, params *models.PaginationParams, totalCount int64) *dto.PaginatedFeatureFlagsResponse {
	return dto.NewPaginatedFeatureFlagsResponse(flags, params, totalCount)
}

// SerializeToJSON converts a feature flag to JSON with serializer options
func (s *FeatureFlagSerializer) SerializeToJSON(flag *models.FeatureFlag) ([]byte, error) {
	response := s.Serialize(flag)
	if response == nil {
		return nil, fmt.Errorf("flag could not be serialized")
	}
	
	return json.Marshal(response)
}

// SerializeListToJSON converts a list of feature flags to JSON
func (s *FeatureFlagSerializer) SerializeListToJSON(flags []*models.FeatureFlag) ([]byte, error) {
	responses := s.SerializeList(flags)
	return json.Marshal(responses)
}

// Private helper methods

// serializeEnvironments processes environment configurations based on serializer options
func (s *FeatureFlagSerializer) serializeEnvironments(environments map[string]models.EnvironmentConfig) map[string]dto.EnvironmentResponse {
	if environments == nil {
		return make(map[string]dto.EnvironmentResponse)
	}
	
	result := make(map[string]dto.EnvironmentResponse)
	
	for envName, envConfig := range environments {
		// Filter environments if specific ones are requested
		if len(s.includeEnvironments) > 0 {
			shouldInclude := false
			for _, includedEnv := range s.includeEnvironments {
				if envName == includedEnv {
					shouldInclude = true
					break
				}
			}
			if !shouldInclude {
				continue
			}
		}
		
		envResponse := dto.EnvironmentResponse{
			Enabled:        envConfig.Enabled,
			RolloutPercent: envConfig.RolloutPercent,
		}
		
		// Include audit info if configured
		if s.includeAuditInfo {
			envResponse.UpdatedBy = envConfig.UpdatedBy
			envResponse.UpdatedAt = envConfig.UpdatedAt
		}
		
		// Serialize rules
		envResponse.Rules = s.serializeRules(envConfig.Rules)
		
		result[envName] = envResponse
	}
	
	return result
}

// serializeRules processes conditional rules
func (s *FeatureFlagSerializer) serializeRules(rules []models.ConditionalRule) []dto.ConditionalRuleResponse {
	if len(rules) == 0 {
		return nil // Return nil for cleaner JSON (omitempty)
	}
	
	result := make([]dto.ConditionalRuleResponse, len(rules))
	
	for i, rule := range rules {
		result[i] = dto.ConditionalRuleResponse{
			ID:        rule.ID,
			Attribute: rule.Attribute,
			Operator:  rule.Operator,
			Value:     rule.Value,
			Enabled:   rule.Enabled,
		}
	}
	
	return result
}

// applyFieldExclusions removes specified fields from the response
func (s *FeatureFlagSerializer) applyFieldExclusions(response *dto.FeatureFlagResponse) {
	for _, field := range s.excludeFields {
		switch field {
		case "description":
			response.Description = ""
		case "created_by":
			response.CreatedBy = [16]byte{} // Zero UUID
		case "environments":
			response.Environments = make(map[string]dto.EnvironmentResponse)
		// Add more field exclusions as needed
		}
	}
}

// Specialized serializers for different contexts

// PublicSerializer returns a serializer configured for public API responses
func PublicSerializer() *FeatureFlagSerializer {
	return NewFeatureFlagSerializer().
		WithDeleted(false).
		WithoutFields([]string{"created_by"})
}

// AdminSerializer returns a serializer configured for admin API responses
func AdminSerializer() *FeatureFlagSerializer {
	return NewFeatureFlagSerializer().
		WithDeleted(true)
}

// MinimalSerializer returns a serializer with minimal information
func MinimalSerializer() *FeatureFlagSerializer {
	return NewFeatureFlagSerializer().
		WithDeleted(false).
		WithoutAuditInfo().
		WithoutFields([]string{"description"})
}

// EnvironmentSpecificSerializer returns a serializer for specific environments
func EnvironmentSpecificSerializer(environments []string) *FeatureFlagSerializer {
	return NewFeatureFlagSerializer().
		WithEnvironments(environments).
		WithDeleted(false)
}

// Utility functions

// SerializeForCache creates a minimal representation suitable for caching
func SerializeForCache(flag *models.FeatureFlag) ([]byte, error) {
	serializer := MinimalSerializer()
	return serializer.SerializeToJSON(flag)
}

// DeserializeFromCache reconstructs a feature flag from cached data
func DeserializeFromCache(data []byte) (*dto.FeatureFlagResponse, error) {
	var response dto.FeatureFlagResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize cached flag: %w", err)
	}
	return &response, nil
}

// SerializeForExport creates a comprehensive representation for data export
func SerializeForExport(flags []*models.FeatureFlag) ([]byte, error) {
	serializer := AdminSerializer()
	return serializer.SerializeListToJSON(flags)
}

// FilterSerializableFields returns only the fields that can be serialized safely
func FilterSerializableFields(flag *models.FeatureFlag) map[string]interface{} {
	return map[string]interface{}{
		"id":           flag.ID,
		"name":         flag.Name,
		"service_name": flag.ServiceName,
		"description":  flag.Description,
		"created_at":   flag.CreatedAt,
		"updated_at":   flag.UpdatedAt,
		"environments": flag.Environments,
	}
}