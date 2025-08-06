package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/repository"
)

func TestFeatureFlagRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	// Create a test user first (required for foreign key)
	user := createTestUser(t, userRepo)

	flag := &models.FeatureFlag{
		ID:          uuid.New(),
		Name:        "test-flag",
		ServiceName: "test-service",
		Description: "Test feature flag",
		CreatedBy:   user.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Environments: map[string]models.EnvironmentConfig{
			"development": {
				Enabled:        true,
				RolloutPercent: 100,
				Rules:          []models.ConditionalRule{},
				UpdatedBy:      user.ID,
				UpdatedAt:      time.Now(),
			},
		},
	}

	err := flagRepo.Create(ctx, flag)
	if err != nil {
		t.Fatalf("Failed to create feature flag: %v", err)
	}

	// Verify feature flag was created
	retrieved, err := flagRepo.GetByID(ctx, flag.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve created feature flag: %v", err)
	}

	if retrieved.Name != flag.Name {
		t.Errorf("Expected name %s, got %s", flag.Name, retrieved.Name)
	}
	if retrieved.ServiceName != flag.ServiceName {
		t.Errorf("Expected service name %s, got %s", flag.ServiceName, retrieved.ServiceName)
	}
	if len(retrieved.Environments) != len(flag.Environments) {
		t.Errorf("Expected %d environments, got %d", len(flag.Environments), len(retrieved.Environments))
	}
}

func TestFeatureFlagRepository_GetByName(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	flag := createTestFeatureFlag(t, flagRepo, user.ID)

	retrieved, err := flagRepo.GetByName(ctx, flag.Name, flag.ServiceName)
	if err != nil {
		t.Fatalf("Failed to get feature flag by name: %v", err)
	}

	if retrieved.ID != flag.ID {
		t.Errorf("Expected flag ID %s, got %s", flag.ID, retrieved.ID)
	}
}

func TestFeatureFlagRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	flag := createTestFeatureFlag(t, flagRepo, user.ID)

	newDescription := "Updated description"
	newEnvironments := map[string]models.EnvironmentConfig{
		"production": {
			Enabled:        false,
			RolloutPercent: 0,
			Rules:          []models.ConditionalRule{},
			UpdatedBy:      user.ID,
			UpdatedAt:      time.Now(),
		},
	}

	updates := &models.UpdateFeatureFlagRequest{
		Description:  &newDescription,
		Environments: newEnvironments,
	}

	updated, err := flagRepo.Update(ctx, flag.ID, updates, user.ID)
	if err != nil {
		t.Fatalf("Failed to update feature flag: %v", err)
	}

	if updated.Description != newDescription {
		t.Errorf("Expected description %s, got %s", newDescription, updated.Description)
	}
	if len(updated.Environments) != len(newEnvironments) {
		t.Errorf("Expected %d environments, got %d", len(newEnvironments), len(updated.Environments))
	}
	if updated.UpdatedAt.Before(flag.UpdatedAt) {
		t.Error("UpdatedAt should have been updated")
	}
}

func TestFeatureFlagRepository_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	flag := createTestFeatureFlag(t, flagRepo, user.ID)

	err := flagRepo.SoftDelete(ctx, flag.ID)
	if err != nil {
		t.Fatalf("Failed to soft delete feature flag: %v", err)
	}

	// Should still be accessible via GetByID
	retrieved, err := flagRepo.GetByID(ctx, flag.ID)
	if err != nil {
		t.Fatalf("Failed to get soft deleted feature flag: %v", err)
	}
	if retrieved.DeletedAt == nil {
		t.Error("Expected DeletedAt to be set after soft delete")
	}

	// Should not be accessible via GetActiveByID
	_, err = flagRepo.GetActiveByID(ctx, flag.ID)
	if err == nil {
		t.Error("Expected error when getting active soft deleted feature flag")
	}
}

func TestFeatureFlagRepository_Restore(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	flag := createTestFeatureFlag(t, flagRepo, user.ID)

	// Soft delete
	err := flagRepo.SoftDelete(ctx, flag.ID)
	if err != nil {
		t.Fatalf("Failed to soft delete feature flag: %v", err)
	}

	// Restore
	err = flagRepo.Restore(ctx, flag.ID)
	if err != nil {
		t.Fatalf("Failed to restore feature flag: %v", err)
	}

	// Should now be accessible via GetActiveByID
	retrieved, err := flagRepo.GetActiveByID(ctx, flag.ID)
	if err != nil {
		t.Fatalf("Failed to get restored feature flag: %v", err)
	}
	if retrieved.DeletedAt != nil {
		t.Error("Expected DeletedAt to be nil after restore")
	}
}

func TestFeatureFlagRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	// Create test feature flags
	flags := []*models.FeatureFlag{
		{
			ID:          uuid.New(),
			Name:        "flag-1",
			ServiceName: "service-a",
			Description: "Flag 1",
			CreatedBy:   user.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environments: map[string]models.EnvironmentConfig{
				"development": {
					Enabled:        true,
					RolloutPercent: 100,
					UpdatedBy:      user.ID,
					UpdatedAt:      time.Now(),
				},
			},
		},
		{
			ID:          uuid.New(),
			Name:        "flag-2",
			ServiceName: "service-b",
			Description: "Flag 2",
			CreatedBy:   user.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environments: map[string]models.EnvironmentConfig{
				"production": {
					Enabled:        false,
					RolloutPercent: 0,
					UpdatedBy:      user.ID,
					UpdatedAt:      time.Now(),
				},
			},
		},
	}

	for _, flag := range flags {
		if err := flagRepo.Create(ctx, flag); err != nil {
			t.Fatalf("Failed to create test feature flag: %v", err)
		}
	}

	// Test listing all flags
	pagination := &models.PaginationParams{Page: 1, PageSize: 10}
	retrieved, total, err := flagRepo.List(ctx, nil, pagination)
	if err != nil {
		t.Fatalf("Failed to list feature flags: %v", err)
	}

	if total != int64(len(flags)) {
		t.Errorf("Expected %d flags, got %d", len(flags), total)
	}
	if len(retrieved) != len(flags) {
		t.Errorf("Expected %d retrieved flags, got %d", len(flags), len(retrieved))
	}

	// Test filtering by service
	filter := &models.FeatureFlagFilter{ServiceName: "service-a"}
	retrieved, total, err = flagRepo.List(ctx, filter, pagination)
	if err != nil {
		t.Fatalf("Failed to list feature flags with filter: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected 1 flag for service-a, got %d", total)
	}
	if len(retrieved) != 1 || retrieved[0].ServiceName != "service-a" {
		t.Error("Service filter did not work correctly")
	}
}

func TestFeatureFlagRepository_ListByService(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	serviceName := "test-service"

	// Create multiple flags for the same service
	for i := 0; i < 3; i++ {
		flag := &models.FeatureFlag{
			ID:          uuid.New(),
			Name:        fmt.Sprintf("flag-%d", i),
			ServiceName: serviceName,
			Description: fmt.Sprintf("Flag %d", i),
			CreatedBy:   user.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environments: map[string]models.EnvironmentConfig{},
		}
		if err := flagRepo.Create(ctx, flag); err != nil {
			t.Fatalf("Failed to create test feature flag: %v", err)
		}
	}

	retrieved, err := flagRepo.ListByService(ctx, serviceName)
	if err != nil {
		t.Fatalf("Failed to list feature flags by service: %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("Expected 3 flags for service, got %d", len(retrieved))
	}

	for _, flag := range retrieved {
		if flag.ServiceName != serviceName {
			t.Errorf("Expected service name %s, got %s", serviceName, flag.ServiceName)
		}
	}
}

func TestFeatureFlagRepository_UpdateEnvironment(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := repository.NewUserRepository(db)
	flagRepo := repository.NewFeatureFlagRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	flag := createTestFeatureFlag(t, flagRepo, user.ID)

	newConfig := &models.EnvironmentConfig{
		Enabled:        false,
		RolloutPercent: 50,
		Rules: []models.ConditionalRule{
			{
				ID:        uuid.New(),
				Attribute: "user_id",
				Operator:  "equals",
				Value:     "test-user",
				Enabled:   true,
			},
		},
		UpdatedBy: user.ID,
		UpdatedAt: time.Now(),
	}

	err := flagRepo.UpdateEnvironment(ctx, flag.ID, "production", newConfig)
	if err != nil {
		t.Fatalf("Failed to update environment: %v", err)
	}

	// Verify the environment was updated
	updated, err := flagRepo.GetByID(ctx, flag.ID)
	if err != nil {
		t.Fatalf("Failed to get updated feature flag: %v", err)
	}

	prodConfig, exists := updated.Environments["production"]
	if !exists {
		t.Fatal("Production environment should exist after update")
	}

	if prodConfig.Enabled != newConfig.Enabled {
		t.Errorf("Expected enabled %v, got %v", newConfig.Enabled, prodConfig.Enabled)
	}
	if prodConfig.RolloutPercent != newConfig.RolloutPercent {
		t.Errorf("Expected rollout percent %d, got %d", newConfig.RolloutPercent, prodConfig.RolloutPercent)
	}
	if len(prodConfig.Rules) != len(newConfig.Rules) {
		t.Errorf("Expected %d rules, got %d", len(newConfig.Rules), len(prodConfig.Rules))
	}
}

func createTestFeatureFlag(t *testing.T, repo repository.FeatureFlagRepository, userID uuid.UUID) *models.FeatureFlag {
	flag := &models.FeatureFlag{
		ID:          uuid.New(),
		Name:        "test-flag",
		ServiceName: "test-service",
		Description: "Test feature flag",
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Environments: map[string]models.EnvironmentConfig{
			"development": {
				Enabled:        true,
				RolloutPercent: 100,
				Rules:          []models.ConditionalRule{},
				UpdatedBy:      userID,
				UpdatedAt:      time.Now(),
			},
		},
	}

	err := repo.Create(context.Background(), flag)
	if err != nil {
		t.Fatalf("Failed to create test feature flag: %v", err)
	}

	return flag
}