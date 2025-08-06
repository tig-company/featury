package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator handles database migrations
type Migrator struct {
	db            *sql.DB
	migrationsDir string
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	migrator, err := m.createMigrator()
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("Migrations completed successfully")
	return nil
}

// Down runs one migration down
func (m *Migrator) Down() error {
	migrator, err := m.createMigrator()
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	fmt.Println("Migration rolled back successfully")
	return nil
}

// Reset rolls back all migrations
func (m *Migrator) Reset() error {
	migrator, err := m.createMigrator()
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	fmt.Println("All migrations reset successfully")
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	migrator, err := m.createMigrator()
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	version, dirty, err := migrator.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	migrator, err := m.createMigrator()
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}

	fmt.Printf("Migration version forced to %d\n", version)
	return nil
}

// Drop drops the entire database schema
func (m *Migrator) Drop() error {
	migrator, err := m.createMigrator()
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	if err := migrator.Drop(); err != nil {
		return fmt.Errorf("failed to drop database schema: %w", err)
	}

	fmt.Println("Database schema dropped successfully")
	return nil
}

// createMigrator creates a new migrate instance
func (m *Migrator) createMigrator() (*migrate.Migrate, error) {
	// Create file source URL
	sourceURL := fmt.Sprintf("file://%s", filepath.Clean(m.migrationsDir))

	// Create database driver
	dbDriver, err := postgres.WithInstance(m.db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator
	migrator, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return migrator, nil
}

// MigrateUp is a convenience function to run migrations up
func MigrateUp(db *sql.DB, migrationsDir string) error {
	migrator := NewMigrator(db, migrationsDir)
	return migrator.Up()
}

// MigrateDown is a convenience function to run one migration down
func MigrateDown(db *sql.DB, migrationsDir string) error {
	migrator := NewMigrator(db, migrationsDir)
	return migrator.Down()
}

// MigrateReset is a convenience function to reset all migrations
func MigrateReset(db *sql.DB, migrationsDir string) error {
	migrator := NewMigrator(db, migrationsDir)
	return migrator.Reset()
}