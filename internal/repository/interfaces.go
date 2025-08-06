package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

// DBExecutor interface for *sql.DB and *sql.Tx
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// Transactor defines the interface for database transaction management
type Transactor interface {
	// WithTx executes a function within a database transaction
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *models.User) error
	
	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	
	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	
	// Update updates an existing user
	Update(ctx context.Context, id uuid.UUID, updates *models.UpdateUserRequest) (*models.User, error)
	
	// Delete soft deletes a user
	Delete(ctx context.Context, id uuid.UUID) error
	
	// List retrieves users with filtering and pagination
	List(ctx context.Context, filter *models.UserFilter, pagination *models.PaginationParams) ([]*models.User, int64, error)
	
	// Exists checks if a user exists by ID
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	
	// ExistsByEmail checks if a user exists by email
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// FeatureFlagRepository defines the interface for feature flag data access
type FeatureFlagRepository interface {
	// Create creates a new feature flag
	Create(ctx context.Context, flag *models.FeatureFlag) error
	
	// GetByID retrieves a feature flag by ID (including soft deleted)
	GetByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error)
	
	// GetActiveByID retrieves a non-deleted feature flag by ID
	GetActiveByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error)
	
	// GetByName retrieves a feature flag by name and service name
	GetByName(ctx context.Context, name, serviceName string) (*models.FeatureFlag, error)
	
	// Update updates an existing feature flag
	Update(ctx context.Context, id uuid.UUID, updates *models.UpdateFeatureFlagRequest, updatedBy uuid.UUID) (*models.FeatureFlag, error)
	
	// UpdateEnvironment updates a specific environment configuration
	UpdateEnvironment(ctx context.Context, id uuid.UUID, environment string, config *models.EnvironmentConfig) error
	
	// SoftDelete soft deletes a feature flag
	SoftDelete(ctx context.Context, id uuid.UUID) error
	
	// HardDelete permanently deletes a feature flag
	HardDelete(ctx context.Context, id uuid.UUID) error
	
	// Restore restores a soft deleted feature flag
	Restore(ctx context.Context, id uuid.UUID) error
	
	// List retrieves feature flags with filtering and pagination
	List(ctx context.Context, filter *models.FeatureFlagFilter, pagination *models.PaginationParams) ([]*models.FeatureFlag, int64, error)
	
	// ListByService retrieves all active feature flags for a service
	ListByService(ctx context.Context, serviceName string) ([]*models.FeatureFlag, error)
	
	// Exists checks if a feature flag exists by name and service
	Exists(ctx context.Context, name, serviceName string) (bool, error)
	
	// ExistsByID checks if a feature flag exists by ID
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
	
	// GetEnvironments retrieves all unique environment names across feature flags
	GetEnvironments(ctx context.Context) ([]string, error)
	
	// GetServices retrieves all unique service names
	GetServices(ctx context.Context) ([]string, error)
}

// APIKeyRepository defines the interface for API key data access
type APIKeyRepository interface {
	// Create creates a new API key
	Create(ctx context.Context, apiKey *models.APIKey) error
	
	// GetByID retrieves an API key by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
	
	// GetByKeyHash retrieves an API key by key hash
	GetByKeyHash(ctx context.Context, keyHash string) (*models.APIKey, error)
	
	// Update updates an existing API key
	Update(ctx context.Context, id uuid.UUID, updates *models.UpdateAPIKeyRequest) (*models.APIKey, error)
	
	// UpdateLastUsed updates the last used timestamp
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	
	// Delete deletes an API key
	Delete(ctx context.Context, id uuid.UUID) error
	
	// List retrieves API keys with filtering and pagination
	List(ctx context.Context, filter *models.APIKeyFilter, pagination *models.PaginationParams) ([]*models.APIKey, int64, error)
	
	// ListByUser retrieves all API keys for a user
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.APIKey, error)
	
	// Exists checks if an API key exists by ID
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	
	// CleanupExpired removes expired API keys older than the specified duration
	CleanupExpired(ctx context.Context) (int64, error)
}

// AuditLogRepository defines the interface for audit log data access
type AuditLogRepository interface {
	// Create creates a new audit log entry
	Create(ctx context.Context, auditLog *models.AuditLog) error
	
	// GetByID retrieves an audit log entry by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error)
	
	// List retrieves audit logs with filtering and pagination
	List(ctx context.Context, filter *models.AuditLogFilter, pagination *models.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// ListByEntity retrieves audit logs for a specific entity
	ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID, pagination *models.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// ListByUser retrieves audit logs for a specific user
	ListByUser(ctx context.Context, userID uuid.UUID, pagination *models.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// DeleteOlderThan deletes audit logs older than the specified time
	DeleteOlderThan(ctx context.Context, olderThanDays int) (int64, error)
	
	// GetEntityHistory retrieves the complete change history for an entity
	GetEntityHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.AuditLog, error)
	
	// GetUserActivity retrieves recent user activity
	GetUserActivity(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error)
}

// HealthRepository defines the interface for database health checks
type HealthRepository interface {
	// Ping checks if the database is reachable
	Ping(ctx context.Context) error
	
	// GetStats returns database connection statistics
	GetStats(ctx context.Context) (*sql.DBStats, error)
}

// Repository aggregates all repository interfaces
type Repository interface {
	// Database transaction management
	Transactor
	
	// Entity repositories
	Users() UserRepository
	FeatureFlags() FeatureFlagRepository
	APIKeys() APIKeyRepository
	AuditLogs() AuditLogRepository
	Health() HealthRepository
	
	// Close closes the database connection
	Close() error
}