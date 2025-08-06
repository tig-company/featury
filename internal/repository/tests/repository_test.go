package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/tig-company/featury/internal/repository"
)

func TestRepository_WithTx(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewRepository(db)
	ctx := context.Background()

	// Test successful transaction
	err := repo.WithTx(ctx, func(txCtx context.Context) error {
		txRepo := repository.GetTxRepository(txCtx)
		if txRepo == nil {
			t.Error("Expected transaction repository to be available in context")
			return nil
		}

		// Create a user within the transaction
		user := createTestUser(t, txRepo.Users())
		
		// Verify it exists within the same transaction
		exists, err := txRepo.Users().Exists(txCtx, user.ID)
		if err != nil {
			return err
		}
		if !exists {
			t.Error("User should exist within transaction")
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}
}

func TestRepository_WithTx_Rollback(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewRepository(db)
	ctx := context.Background()

	// Create a user outside the transaction first
	existingUser := createTestUser(t, repo.Users())

	// Test transaction rollback
	err := repo.WithTx(ctx, func(txCtx context.Context) error {
		txRepo := repository.GetTxRepository(txCtx)
		
		// Create another user within the transaction
		newUser := createTestUser(t, txRepo.Users())
		
		// Verify both users exist within the transaction
		exists, err := txRepo.Users().Exists(txCtx, existingUser.ID)
		if err != nil {
			return err
		}
		if !exists {
			t.Error("Existing user should exist within transaction")
		}

		exists, err = txRepo.Users().Exists(txCtx, newUser.ID)
		if err != nil {
			return err
		}
		if !exists {
			t.Error("New user should exist within transaction")
		}

		// Force a rollback by returning an error
		return fmt.Errorf("intentional rollback")
	})

	if err == nil {
		t.Fatal("Expected transaction to fail and rollback")
	}

	// Verify only the existing user remains (new user should be rolled back)
	exists, err := repo.Users().Exists(ctx, existingUser.ID)
	if err != nil {
		t.Fatalf("Failed to check if existing user exists: %v", err)
	}
	if !exists {
		t.Error("Existing user should still exist after rollback")
	}
}

func TestRepository_Health(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewRepository(db)
	ctx := context.Background()

	// Test ping
	err := repo.Health().Ping(ctx)
	if err != nil {
		t.Fatalf("Health ping failed: %v", err)
	}

	// Test stats
	stats, err := repo.Health().GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get database stats: %v", err)
	}

	if stats == nil {
		t.Error("Expected database stats to be returned")
	}
}

func TestRepository_Close(t *testing.T) {
	db := setupTestDB(t)
	// Don't use defer teardownTestDB here since we're testing Close()

	repo := repository.NewRepository(db)

	err := repo.Close()
	if err != nil {
		t.Fatalf("Failed to close repository: %v", err)
	}

	// Verify connection is closed by trying to ping
	ctx := context.Background()
	err = repo.Health().Ping(ctx)
	if err == nil {
		t.Error("Expected ping to fail after closing connection")
	}
}