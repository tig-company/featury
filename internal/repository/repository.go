package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Common errors
var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

// repository is the main repository implementation that aggregates all repositories
type repository struct {
	db *sql.DB

	users        UserRepository
	featureFlags FeatureFlagRepository
	apiKeys      APIKeyRepository
	auditLogs    AuditLogRepository
	health       HealthRepository
}

// NewRepository creates a new repository instance
func NewRepository(db *sql.DB) Repository {
	return &repository{
		db:           db,
		users:        NewUserRepository(db),
		featureFlags: NewFeatureFlagRepository(db),
		apiKeys:      NewAPIKeyRepository(db),
		auditLogs:    NewAuditLogRepository(db),
		health:       NewHealthRepository(db),
	}
}

// WithTx executes a function within a database transaction
func (r *repository) WithTx(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a new repository instance using the transaction
	txRepo := &repository{
		db:           r.db,
		users:        NewUserRepository(tx),
		featureFlags: NewFeatureFlagRepository(tx),
		apiKeys:      NewAPIKeyRepository(tx),
		auditLogs:    NewAuditLogRepository(tx),
		health:       NewHealthRepository(tx),
	}

	// Create a context that carries the transaction repository
	txCtx := context.WithValue(ctx, txRepositoryKey, txRepo)

	// Execute the function
	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("failed to rollback transaction after error %v: %w", err, rollbackErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Users returns the user repository
func (r *repository) Users() UserRepository {
	return r.users
}

// FeatureFlags returns the feature flag repository
func (r *repository) FeatureFlags() FeatureFlagRepository {
	return r.featureFlags
}

// APIKeys returns the API key repository
func (r *repository) APIKeys() APIKeyRepository {
	return r.apiKeys
}

// AuditLogs returns the audit log repository
func (r *repository) AuditLogs() AuditLogRepository {
	return r.auditLogs
}

// Health returns the health repository
func (r *repository) Health() HealthRepository {
	return r.health
}

// Close closes the database connection
func (r *repository) Close() error {
	return r.db.Close()
}

// Context key for transaction repository
type contextKey string

const txRepositoryKey contextKey = "tx_repository"

// GetTxRepository retrieves the transaction repository from context
func GetTxRepository(ctx context.Context) Repository {
	if txRepo, ok := ctx.Value(txRepositoryKey).(Repository); ok {
		return txRepo
	}
	return nil
}

// HealthRepository implementation
type healthRepository struct {
	db DBExecutor
}

// NewHealthRepository creates a new health repository
func NewHealthRepository(db DBExecutor) HealthRepository {
	return &healthRepository{db: db}
}

func (r *healthRepository) Ping(ctx context.Context) error {
	// Try to execute a simple query to check connectivity
	var result int
	err := r.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	return err
}

func (r *healthRepository) GetStats(ctx context.Context) (*sql.DBStats, error) {
	// For transaction contexts, we can't get stats directly
	// Return an empty stats object
	return &sql.DBStats{}, nil
}