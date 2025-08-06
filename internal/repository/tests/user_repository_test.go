package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/repository"
)

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Name:      "Test User",
		Role:      models.RoleEditor,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user was created
	retrieved, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve created user: %v", err)
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}
	if retrieved.Name != user.Name {
		t.Errorf("Expected name %s, got %s", user.Name, retrieved.Name)
	}
	if retrieved.Role != user.Role {
		t.Errorf("Expected role %s, got %s", user.Role, retrieved.Role)
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := createTestUser(t, repo)

	retrieved, err := repo.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if retrieved.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrieved.ID)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := createTestUser(t, repo)

	newName := "Updated Name"
	newRole := models.RoleAdmin

	updates := &models.UpdateUserRequest{
		Name: &newName,
		Role: &newRole,
	}

	updated, err := repo.Update(ctx, user.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, updated.Name)
	}
	if updated.Role != newRole {
		t.Errorf("Expected role %s, got %s", newRole, updated.Role)
	}
	if updated.UpdatedAt.Before(user.UpdatedAt) {
		t.Error("UpdatedAt should have been updated")
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := createTestUser(t, repo)

	err := repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user was deleted
	_, err = repo.GetByID(ctx, user.ID)
	if err == nil {
		t.Error("Expected error when getting deleted user")
	}
}

func TestUserRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	// Create test users
	users := []*models.User{
		{
			ID:        uuid.New(),
			Email:     "user1@example.com",
			Name:      "User One",
			Role:      models.RoleEditor,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "user2@example.com",
			Name:      "User Two",
			Role:      models.RoleViewer,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "admin@example.com",
			Name:      "Admin User",
			Role:      models.RoleAdmin,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, user := range users {
		if err := repo.Create(ctx, user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	// Test listing all users
	pagination := &models.PaginationParams{Page: 1, PageSize: 10}
	retrieved, total, err := repo.List(ctx, nil, pagination)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if total != int64(len(users)) {
		t.Errorf("Expected %d users, got %d", len(users), total)
	}
	if len(retrieved) != len(users) {
		t.Errorf("Expected %d retrieved users, got %d", len(users), len(retrieved))
	}

	// Test filtering by role
	filter := &models.UserFilter{Role: models.RoleEditor}
	retrieved, total, err = repo.List(ctx, filter, pagination)
	if err != nil {
		t.Fatalf("Failed to list users with filter: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected 1 editor user, got %d", total)
	}
	if len(retrieved) != 1 || retrieved[0].Role != models.RoleEditor {
		t.Error("Filter did not work correctly")
	}

	// Test pagination
	pagination = &models.PaginationParams{Page: 1, PageSize: 2}
	retrieved, total, err = repo.List(ctx, nil, pagination)
	if err != nil {
		t.Fatalf("Failed to list users with pagination: %v", err)
	}

	if total != int64(len(users)) {
		t.Errorf("Expected total %d users, got %d", len(users), total)
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 retrieved users, got %d", len(retrieved))
	}
}

func TestUserRepository_Exists(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := createTestUser(t, repo)

	exists, err := repo.Exists(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to check if user exists: %v", err)
	}
	if !exists {
		t.Error("Expected user to exist")
	}

	exists, err = repo.Exists(ctx, uuid.New())
	if err != nil {
		t.Fatalf("Failed to check if user exists: %v", err)
	}
	if exists {
		t.Error("Expected user to not exist")
	}
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := createTestUser(t, repo)

	exists, err := repo.ExistsByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("Failed to check if user exists by email: %v", err)
	}
	if !exists {
		t.Error("Expected user to exist by email")
	}

	exists, err = repo.ExistsByEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("Failed to check if user exists by email: %v", err)
	}
	if exists {
		t.Error("Expected user to not exist by email")
	}
}

func createTestUser(t *testing.T, repo repository.UserRepository) *models.User {
	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Name:      "Test User",
		Role:      models.RoleEditor,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}