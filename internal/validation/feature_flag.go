package validation

import (
	"fmt"
	"strings"

	"github.com/tig-company/featury/internal/models"
)

// FeatureFlagValidator provides validation for feature flag related requests
type FeatureFlagValidator struct {
	*Validator
}

// NewFeatureFlagValidator creates a new feature flag validator
func NewFeatureFlagValidator() *FeatureFlagValidator {
	return &FeatureFlagValidator{
		Validator: NewValidator(),
	}
}

// ValidateCreateFeatureFlagRequest validates a create feature flag request
func (ffv *FeatureFlagValidator) ValidateCreateFeatureFlagRequest(req *models.CreateFeatureFlagRequest) *ValidationResult {
	result := NewValidationResult()

	// Validate name
	if err := ValidateFeatureFlagName(req.Name); err != nil {
		result.AddError("name", err.Error())
	}

	// Validate key
	if err := ValidateFeatureFlagKey(req.Key); err != nil {
		result.AddError("key", err.Error())
	}

	// Validate description
	if req.Description != "" {
		if len(req.Description) > 1000 {
			result.AddError("description", "description must not exceed 1000 characters")
		}
		// Sanitize description
		req.Description = ffv.SanitizeString(req.Description)
	}

	// Validate tags
	if len(req.Tags) > 0 {
		if err := ffv.validateTags(req.Tags); err != nil {
			result.AddError("tags", err.Error())
		}
	}

	// Validate configuration if provided
	if req.Config != nil {
		if err := ffv.validateFeatureConfig(req.Config); err != nil {
			result.AddError("config", err.Error())
		}
	}

	return result
}

// ValidateUpdateFeatureFlagRequest validates an update feature flag request
func (ffv *FeatureFlagValidator) ValidateUpdateFeatureFlagRequest(req *models.UpdateFeatureFlagRequest) *ValidationResult {
	result := NewValidationResult()

	// Check if any changes are provided
	if !req.HasChanges() {
		result.AddError("request", "at least one field must be provided for update")
		return result
	}

	// Validate name if provided
	if req.Name != nil {
		if err := ValidateFeatureFlagName(*req.Name); err != nil {
			result.AddError("name", err.Error())
		}
	}

	// Validate description if provided
	if req.Description != nil {
		if len(*req.Description) > 1000 {
			result.AddError("description", "description must not exceed 1000 characters")
		}
		// Sanitize description
		sanitized := ffv.SanitizeString(*req.Description)
		req.Description = &sanitized
	}

	// Validate enabled flag
	if req.Enabled != nil {
		// Enabled is a boolean, no additional validation needed
	}

	// Validate tags if provided
	if req.Tags != nil {
		if err := ffv.validateTags(*req.Tags); err != nil {
			result.AddError("tags", err.Error())
		}
	}

	// Validate configuration if provided
	if req.Config != nil {
		if err := ffv.validateFeatureConfig(req.Config); err != nil {
			result.AddError("config", err.Error())
		}
	}

	return result
}

// ValidateFeatureFlagFilter validates feature flag filter parameters
func (ffv *FeatureFlagValidator) ValidateFeatureFlagFilter(filter *models.FeatureFlagFilter) *ValidationResult {
	result := NewValidationResult()

	// Validate name filter
	if filter.Name != "" {
		if len(filter.Name) > 100 {
			result.AddError("name", "name filter must not exceed 100 characters")
		}
		filter.Name = ffv.SanitizeString(filter.Name)
	}

	// Validate key filter
	if filter.Key != "" {
		if len(filter.Key) > 100 {
			result.AddError("key", "key filter must not exceed 100 characters")
		}
		filter.Key = ffv.SanitizeString(filter.Key)
	}

	// Validate tags filter
	if len(filter.Tags) > 0 {
		if err := ffv.validateTags(filter.Tags); err != nil {
			result.AddError("tags", err.Error())
		}
	}

	// Validate user ID filter
	if filter.UserID != nil {
		// User ID validation is handled elsewhere, just ensure it's not nil UUID
		if *filter.UserID == (models.BaseModel{}.ID) {
			result.AddError("user_id", "user_id cannot be nil UUID")
		}
	}

	return result
}

// validateTags validates feature flag tags
func (ffv *FeatureFlagValidator) validateTags(tags []string) error {
	if len(tags) > 20 {
		return fmt.Errorf("maximum of 20 tags allowed")
	}

	tagMap := make(map[string]bool)
	for i, tag := range tags {
		// Sanitize tag
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return fmt.Errorf("tag at index %d cannot be empty", i)
		}

		// Check tag length
		if len(tag) > 50 {
			return fmt.Errorf("tag at index %d exceeds maximum length of 50 characters", i)
		}

		// Check for duplicate tags
		if tagMap[tag] {
			return fmt.Errorf("duplicate tag found: %s", tag)
		}
		tagMap[tag] = true

		// Validate tag format (alphanumeric, hyphens, underscores)
		if !strings.ContainsAny(tag, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_") {
			return fmt.Errorf("tag '%s' contains invalid characters", tag)
		}

		// Update the sanitized tag back to the slice
		tags[i] = tag
	}

	return nil
}

// validateFeatureConfig validates feature flag configuration
func (ffv *FeatureFlagValidator) validateFeatureConfig(config *models.FeatureConfig) error {
	if config == nil {
		return nil
	}

	// Validate config type
	if config.Type == "" {
		return fmt.Errorf("config type is required")
	}

	validTypes := []string{"boolean", "string", "number", "json"}
	typeValid := false
	for _, validType := range validTypes {
		if config.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("config type must be one of: %s", strings.Join(validTypes, ", "))
	}

	// Validate default value based on type
	if config.DefaultValue != nil {
		if err := ffv.validateConfigValue(config.Type, config.DefaultValue); err != nil {
			return fmt.Errorf("invalid default value: %w", err)
		}
	}

	// Validate rules if provided
	if len(config.Rules) > 0 {
		if err := ffv.validateFeatureRules(config.Rules); err != nil {
			return fmt.Errorf("invalid rules: %w", err)
		}
	}

	return nil
}

// validateFeatureRules validates feature flag rules
func (ffv *FeatureFlagValidator) validateFeatureRules(rules []models.FeatureRule) error {
	if len(rules) > 50 {
		return fmt.Errorf("maximum of 50 rules allowed")
	}

	for i, rule := range rules {
		if err := ffv.validateFeatureRule(rule); err != nil {
			return fmt.Errorf("rule at index %d: %w", i, err)
		}
	}

	return nil
}

// validateFeatureRule validates a single feature flag rule
func (ffv *FeatureFlagValidator) validateFeatureRule(rule models.FeatureRule) error {
	// Validate condition
	if rule.Condition == "" {
		return fmt.Errorf("rule condition is required")
	}

	// Validate condition format (basic validation)
	if len(rule.Condition) > 500 {
		return fmt.Errorf("rule condition exceeds maximum length of 500 characters")
	}

	// Validate value
	if rule.Value == nil {
		return fmt.Errorf("rule value is required")
	}

	// Validate percentage if provided
	if rule.Percentage != nil {
		if *rule.Percentage < 0 || *rule.Percentage > 100 {
			return fmt.Errorf("rule percentage must be between 0 and 100")
		}
	}

	return nil
}

// validateConfigValue validates a configuration value based on its type
func (ffv *FeatureFlagValidator) validateConfigValue(configType string, value interface{}) error {
	switch configType {
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("value must be a boolean")
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("value must be a string")
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid number types
		default:
			return fmt.Errorf("value must be a number")
		}
	case "json":
		// For JSON, we accept any valid interface{} value
		// The JSON marshaling will validate if it's serializable
		if value == nil {
			return fmt.Errorf("JSON value cannot be null")
		}
	default:
		return fmt.Errorf("unsupported config type: %s", configType)
	}

	return nil
}

// ValidateBulkOperation validates bulk operation requests
func (ffv *FeatureFlagValidator) ValidateBulkOperation(req *models.BulkOperationRequest) *ValidationResult {
	result := NewValidationResult()

	// Validate operation type
	validOperations := []string{"enable", "disable", "delete"}
	operationValid := false
	for _, op := range validOperations {
		if req.Operation == op {
			operationValid = true
			break
		}
	}
	if !operationValid {
		result.AddError("operation", fmt.Sprintf("operation must be one of: %s", strings.Join(validOperations, ", ")))
	}

	// Validate feature flag IDs
	if len(req.FeatureFlagIDs) == 0 {
		result.AddError("feature_flag_ids", "at least one feature flag ID is required")
	} else if len(req.FeatureFlagIDs) > 100 {
		result.AddError("feature_flag_ids", "maximum of 100 feature flags allowed per bulk operation")
	}

	// Validate each UUID
	for i, id := range req.FeatureFlagIDs {
		if _, err := ffv.ValidateUUID(id.String(), fmt.Sprintf("feature_flag_ids[%d]", i)); err != nil {
			result.AddError(fmt.Sprintf("feature_flag_ids[%d]", i), err.Error())
		}
	}

	return result
}