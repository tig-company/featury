package audit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tig-company/featury/internal/models"
)

func TestObjectDiffer_GenerateDiff(t *testing.T) {
	differ := NewObjectDiffer()

	t.Run("No Change", func(t *testing.T) {
		before := map[string]interface{}{
			"name":  "test",
			"value": 100,
		}
		after := map[string]interface{}{
			"name":  "test",
			"value": 100,
		}

		diff, err := differ.GenerateDiff(before, after)
		require.NoError(t, err)
		assert.Equal(t, "no_changes", diff["action"])
	})

	t.Run("Create (Before is nil)", func(t *testing.T) {
		after := map[string]interface{}{
			"name":  "test",
			"value": 100,
		}

		diff, err := differ.GenerateDiff(nil, after)
		require.NoError(t, err)
		assert.Equal(t, "create", diff["action"])
		assert.Equal(t, after, diff["added"])
	})

	t.Run("Delete (After is nil)", func(t *testing.T) {
		before := map[string]interface{}{
			"name":  "test",
			"value": 100,
		}

		diff, err := differ.GenerateDiff(before, nil)
		require.NoError(t, err)
		assert.Equal(t, "delete", diff["action"])
		assert.Equal(t, before, diff["removed"])
	})

	t.Run("Field Changes", func(t *testing.T) {
		before := map[string]interface{}{
			"name":        "old_name",
			"value":       100,
			"description": "old desc",
		}
		after := map[string]interface{}{
			"name":        "new_name",
			"value":       100,
			"description": "new desc",
			"new_field":   "added",
		}

		diff, err := differ.GenerateDiff(before, after)
		require.NoError(t, err)
		assert.Equal(t, "update", diff["action"])

		changes := diff["changes"].(models.JSONB)
		
		// Check changed field
		nameChange := changes["name"].(map[string]interface{})
		assert.Equal(t, "changed", nameChange["action"])
		assert.Equal(t, "old_name", nameChange["before"])
		assert.Equal(t, "new_name", nameChange["after"])
		
		// Check unchanged field (should not be in changes)
		_, valueExists := changes["value"]
		assert.False(t, valueExists)
		
		// Check added field
		newFieldChange := changes["new_field"].(map[string]interface{})
		assert.Equal(t, "added", newFieldChange["action"])
		assert.Equal(t, "added", newFieldChange["value"])
	})
}

func TestObjectDiffer_GenerateFeatureFlagDiff(t *testing.T) {
	differ := NewObjectDiffer()
	flagID := uuid.New()

	t.Run("Basic Field Changes", func(t *testing.T) {
		before := &models.FeatureFlag{
			ID:          flagID,
			Name:        "old_flag",
			ServiceName: "service",
			Description: "old description",
			Environments: make(map[string]models.EnvironmentConfig),
		}

		after := &models.FeatureFlag{
			ID:          flagID,
			Name:        "new_flag",
			ServiceName: "service",
			Description: "new description",
			Environments: make(map[string]models.EnvironmentConfig),
		}

		diff, err := differ.GenerateFeatureFlagDiff(before, after)
		require.NoError(t, err)

		assert.Equal(t, "update", diff["action"])
		assert.Equal(t, flagID, diff["flag_id"])
		assert.Equal(t, "new_flag", diff["flag_name"])
		assert.Equal(t, "service", diff["service_name"])

		changes := diff["changes"].(models.JSONB)
		
		nameChange := changes["name"].(map[string]interface{})
		assert.Equal(t, "old_flag", nameChange["before"])
		assert.Equal(t, "new_flag", nameChange["after"])
		
		descChange := changes["description"].(map[string]interface{})
		assert.Equal(t, "old description", descChange["before"])
		assert.Equal(t, "new description", descChange["after"])
	})

	t.Run("Environment Changes", func(t *testing.T) {
		userID := uuid.New()
		now := time.Now()

		before := &models.FeatureFlag{
			ID:   flagID,
			Name: "test_flag",
			Environments: map[string]models.EnvironmentConfig{
				"prod": {
					Enabled:        false,
					RolloutPercent: 0,
					UpdatedBy:      userID,
					UpdatedAt:      now,
				},
			},
		}

		after := &models.FeatureFlag{
			ID:   flagID,
			Name: "test_flag",
			Environments: map[string]models.EnvironmentConfig{
				"prod": {
					Enabled:        true,
					RolloutPercent: 50,
					UpdatedBy:      userID,
					UpdatedAt:      now,
				},
				"staging": {
					Enabled:        true,
					RolloutPercent: 100,
					UpdatedBy:      userID,
					UpdatedAt:      now,
				},
			},
		}

		diff, err := differ.GenerateFeatureFlagDiff(before, after)
		require.NoError(t, err)

		changes := diff["changes"].(models.JSONB)
		envChanges := changes["environments"].(models.JSONB)
		
		// Production environment should show changes
		prodChanges := envChanges["prod"].(models.JSONB)
		assert.Equal(t, "update", prodChanges["action"])
		
		// Staging environment should show as added
		stagingChanges := envChanges["staging"].(map[string]interface{})
		assert.Equal(t, "added", stagingChanges["action"])
	})

	t.Run("No Changes", func(t *testing.T) {
		flag := &models.FeatureFlag{
			ID:           flagID,
			Name:         "same_flag",
			ServiceName:  "same_service",
			Description:  "same description",
			Environments: make(map[string]models.EnvironmentConfig),
		}

		diff, err := differ.GenerateFeatureFlagDiff(flag, flag)
		require.NoError(t, err)
		assert.Equal(t, "no_changes", diff["action"])
	})
}

func TestObjectDiffer_GenerateEnvironmentDiff(t *testing.T) {
	differ := NewObjectDiffer()
	userID := uuid.New()
	now := time.Now()

	t.Run("Basic Changes", func(t *testing.T) {
		before := models.EnvironmentConfig{
			Enabled:        false,
			RolloutPercent: 0,
			Rules:          []models.ConditionalRule{},
			UpdatedBy:      userID,
			UpdatedAt:      now,
		}

		after := models.EnvironmentConfig{
			Enabled:        true,
			RolloutPercent: 50,
			Rules:          []models.ConditionalRule{},
			UpdatedBy:      userID,
			UpdatedAt:      now,
		}

		diff, err := differ.GenerateEnvironmentDiff(before, after)
		require.NoError(t, err)

		assert.Equal(t, "update", diff["action"])
		changes := diff["changes"].(models.JSONB)

		enabledChange := changes["enabled"].(map[string]interface{})
		assert.Equal(t, false, enabledChange["before"])
		assert.Equal(t, true, enabledChange["after"])

		rolloutChange := changes["rollout_percent"].(map[string]interface{})
		assert.Equal(t, 0, rolloutChange["before"])
		assert.Equal(t, 50, rolloutChange["after"])
	})

	t.Run("Rule Changes", func(t *testing.T) {
		ruleID := uuid.New()
		
		before := models.EnvironmentConfig{
			Enabled:        true,
			RolloutPercent: 50,
			Rules: []models.ConditionalRule{
				{
					ID:        ruleID,
					Attribute: "user_id",
					Operator:  "equals",
					Value:     "123",
					Enabled:   true,
				},
			},
			UpdatedBy: userID,
			UpdatedAt: now,
		}

		after := models.EnvironmentConfig{
			Enabled:        true,
			RolloutPercent: 50,
			Rules: []models.ConditionalRule{
				{
					ID:        ruleID,
					Attribute: "user_id",
					Operator:  "equals",
					Value:     "456", // Changed value
					Enabled:   true,
				},
			},
			UpdatedBy: userID,
			UpdatedAt: now,
		}

		diff, err := differ.GenerateEnvironmentDiff(before, after)
		require.NoError(t, err)

		changes := diff["changes"].(models.JSONB)
		rulesChanges := changes["rules"].(models.JSONB)
		
		ruleChanges := rulesChanges["rules"].(map[string]interface{})
		ruleChange := ruleChanges[ruleID.String()].(map[string]interface{})
		assert.Equal(t, "changed", ruleChange["action"])
	})

	t.Run("No Changes", func(t *testing.T) {
		config := models.EnvironmentConfig{
			Enabled:        true,
			RolloutPercent: 50,
			Rules:          []models.ConditionalRule{},
			UpdatedBy:      userID,
			UpdatedAt:      now,
		}

		diff, err := differ.GenerateEnvironmentDiff(config, config)
		require.NoError(t, err)
		assert.Equal(t, "no_changes", diff["action"])
	})
}