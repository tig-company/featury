package tests

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/tig-company/featury/internal/database"
)

// setupTestDB creates a test database and runs migrations
func setupTestDB(t *testing.T) *sql.DB {
	// Use a unique test database name
	dbName := fmt.Sprintf("featury_test_%s", t.Name())
	
	config := &database.Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("TEST_DB_USER", "featury"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "featury"),
		Database: dbName,
		SSLMode:  "disable",
	}

	// Create the test database
	if err := database.CreateDatabase(config); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to the test database
	db, err := database.Connect(config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	migrationsDir := getMigrationsDir()
	migrator := database.NewMigrator(db, migrationsDir)
	if err := migrator.Up(); err != nil {
		db.Close()
		database.DropDatabase(config)
		t.Fatalf("Failed to run test migrations: %v", err)
	}

	return db
}

// teardownTestDB cleans up the test database
func teardownTestDB(t *testing.T, db *sql.DB) {
	dbName := fmt.Sprintf("featury_test_%s", t.Name())
	
	config := &database.Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("TEST_DB_USER", "featury"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "featury"),
		Database: dbName,
		SSLMode:  "disable",
	}

	// Close the database connection
	db.Close()

	// Drop the test database
	if err := database.DropDatabase(config); err != nil {
		t.Logf("Failed to drop test database (non-fatal): %v", err)
	}
}

// getEnvOrDefault returns the environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getMigrationsDir returns the path to the migrations directory
func getMigrationsDir() string {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "../../../internal/database/migrations"
	}

	// Navigate up to find the migrations directory
	migrationsDir := filepath.Join(cwd, "..", "..", "database", "migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try alternative path
		migrationsDir = filepath.Join(cwd, "..", "..", "..", "internal", "database", "migrations")
	}

	return migrationsDir
}

// Integration test setup (only runs when TEST_INTEGRATION is set)
func skipIfNoIntegration(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set TEST_INTEGRATION=1 to run")
	}
}