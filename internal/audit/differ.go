package audit

import (
	"fmt"
	"reflect"
	"time"

	"github.com/tig-company/featury/internal/models"
)

type objectDiffer struct{}

// NewObjectDiffer creates a new object differ
func NewObjectDiffer() Differ {
	return &objectDiffer{}
}

// GenerateDiff generates a diff between before and after objects
func (d *objectDiffer) GenerateDiff(before, after interface{}) (models.JSONB, error) {
	if before == nil && after == nil {
		return models.JSONB{}, nil
	}

	diff := make(models.JSONB)

	if before == nil {
		diff["action"] = "create"
		diff["added"] = after
		return diff, nil
	}

	if after == nil {
		diff["action"] = "delete"
		diff["removed"] = before
		return diff, nil
	}

	// Convert objects to maps for comparison
	beforeMap, err := d.objectToMap(before)
	if err != nil {
		return nil, fmt.Errorf("failed to convert before object to map: %w", err)
	}

	afterMap, err := d.objectToMap(after)
	if err != nil {
		return nil, fmt.Errorf("failed to convert after object to map: %w", err)
	}

	changes := d.compareObjects(beforeMap, afterMap)
	if len(changes) == 0 {
		return models.JSONB{"action": "no_changes"}, nil
	}

	diff["action"] = "update"
	diff["changes"] = changes

	return diff, nil
}

// GenerateFeatureFlagDiff generates a specialized diff for feature flags
func (d *objectDiffer) GenerateFeatureFlagDiff(before, after *models.FeatureFlag) (models.JSONB, error) {
	if before == nil && after == nil {
		return models.JSONB{}, nil
	}

	diff := make(models.JSONB)

	if before == nil {
		diff["action"] = "create"
		diff["feature_flag"] = after
		return diff, nil
	}

	if after == nil {
		diff["action"] = "delete"
		diff["feature_flag"] = before
		return diff, nil
	}

	changes := make(models.JSONB)

	// Compare basic fields
	if before.Name != after.Name {
		changes["name"] = map[string]interface{}{
			"before": before.Name,
			"after":  after.Name,
		}
	}

	if before.ServiceName != after.ServiceName {
		changes["service_name"] = map[string]interface{}{
			"before": before.ServiceName,
			"after":  after.ServiceName,
		}
	}

	if before.Description != after.Description {
		changes["description"] = map[string]interface{}{
			"before": before.Description,
			"after":  after.Description,
		}
	}

	// Compare environments
	envChanges := d.compareEnvironments(before.Environments, after.Environments)
	if len(envChanges) > 0 {
		changes["environments"] = envChanges
	}

	if len(changes) == 0 {
		return models.JSONB{"action": "no_changes"}, nil
	}

	diff["action"] = "update"
	diff["changes"] = changes
	diff["flag_id"] = after.ID
	diff["flag_name"] = after.Name
	diff["service_name"] = after.ServiceName

	return diff, nil
}

// GenerateEnvironmentDiff generates a diff for environment configurations
func (d *objectDiffer) GenerateEnvironmentDiff(before, after models.EnvironmentConfig) (models.JSONB, error) {
	diff := make(models.JSONB)
	changes := make(models.JSONB)

	if before.Enabled != after.Enabled {
		changes["enabled"] = map[string]interface{}{
			"before": before.Enabled,
			"after":  after.Enabled,
		}
	}

	if before.RolloutPercent != after.RolloutPercent {
		changes["rollout_percent"] = map[string]interface{}{
			"before": before.RolloutPercent,
			"after":  after.RolloutPercent,
		}
	}

	// Compare rules
	rulesChanges := d.compareRules(before.Rules, after.Rules)
	if len(rulesChanges) > 0 {
		changes["rules"] = rulesChanges
	}

	if len(changes) == 0 {
		return models.JSONB{"action": "no_changes"}, nil
	}

	diff["action"] = "update"
	diff["changes"] = changes

	return diff, nil
}

// Helper methods

func (d *objectDiffer) objectToMap(obj interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object is not a struct")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name from json tag if available
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if commaIdx := len(jsonTag); commaIdx > 0 {
				for i, char := range jsonTag {
					if char == ',' {
						commaIdx = i
						break
					}
				}
				fieldName = jsonTag[:commaIdx]
			}
		}

		// Convert value to interface{}
		result[fieldName] = d.convertValue(value)
	}

	return result, nil
}

func (d *objectDiffer) convertValue(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return d.convertValue(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return v.Interface()
	case reflect.Slice, reflect.Array:
		if v.IsNil() {
			return nil
		}
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = d.convertValue(v.Index(i))
		}
		return result
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		result := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			result[keyStr] = d.convertValue(v.MapIndex(key))
		}
		return result
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v.Interface().(time.Time).Format(time.RFC3339)
		}
		// Convert struct to map
		structMap, _ := d.objectToMap(v.Interface())
		return structMap
	default:
		return v.Interface()
	}
}

func (d *objectDiffer) compareObjects(before, after map[string]interface{}) models.JSONB {
	changes := make(models.JSONB)

	// Check for changed and added fields
	for key, afterValue := range after {
		beforeValue, exists := before[key]
		if !exists {
			changes[key] = map[string]interface{}{
				"action": "added",
				"value":  afterValue,
			}
		} else if !d.valuesEqual(beforeValue, afterValue) {
			changes[key] = map[string]interface{}{
				"action": "changed",
				"before": beforeValue,
				"after":  afterValue,
			}
		}
	}

	// Check for removed fields
	for key, beforeValue := range before {
		if _, exists := after[key]; !exists {
			changes[key] = map[string]interface{}{
				"action": "removed",
				"value":  beforeValue,
			}
		}
	}

	return changes
}

func (d *objectDiffer) compareEnvironments(before, after map[string]models.EnvironmentConfig) models.JSONB {
	changes := make(models.JSONB)

	// Check for changed and added environments
	for envName, afterConfig := range after {
		if beforeConfig, exists := before[envName]; exists {
			envDiff, _ := d.GenerateEnvironmentDiff(beforeConfig, afterConfig)
			if envDiff["action"] != "no_changes" {
				changes[envName] = envDiff
			}
		} else {
			changes[envName] = map[string]interface{}{
				"action": "added",
				"config": afterConfig,
			}
		}
	}

	// Check for removed environments
	for envName, beforeConfig := range before {
		if _, exists := after[envName]; !exists {
			changes[envName] = map[string]interface{}{
				"action": "removed",
				"config": beforeConfig,
			}
		}
	}

	return changes
}

func (d *objectDiffer) compareRules(before, after []models.ConditionalRule) models.JSONB {
	changes := make(models.JSONB)

	// Simple comparison - in a production system you might want more sophisticated rule comparison
	if len(before) != len(after) {
		changes["count"] = map[string]interface{}{
			"before": len(before),
			"after":  len(after),
		}
	}

	// Create maps for easier comparison
	beforeRules := make(map[string]models.ConditionalRule)
	afterRules := make(map[string]models.ConditionalRule)

	for _, rule := range before {
		beforeRules[rule.ID.String()] = rule
	}

	for _, rule := range after {
		afterRules[rule.ID.String()] = rule
	}

	// Check for changes, additions, and deletions
	ruleChanges := make(map[string]interface{})

	for id, afterRule := range afterRules {
		if beforeRule, exists := beforeRules[id]; exists {
			if !d.rulesEqual(beforeRule, afterRule) {
				ruleChanges[id] = map[string]interface{}{
					"action": "changed",
					"before": beforeRule,
					"after":  afterRule,
				}
			}
		} else {
			ruleChanges[id] = map[string]interface{}{
				"action": "added",
				"rule":   afterRule,
			}
		}
	}

	for id, beforeRule := range beforeRules {
		if _, exists := afterRules[id]; !exists {
			ruleChanges[id] = map[string]interface{}{
				"action": "removed",
				"rule":   beforeRule,
			}
		}
	}

	if len(ruleChanges) > 0 {
		changes["rules"] = ruleChanges
	}

	return changes
}

func (d *objectDiffer) valuesEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func (d *objectDiffer) rulesEqual(a, b models.ConditionalRule) bool {
	return a.Attribute == b.Attribute &&
		a.Operator == b.Operator &&
		reflect.DeepEqual(a.Value, b.Value) &&
		a.Enabled == b.Enabled
}