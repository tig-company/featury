# Phase 1 Implementation: Data Layer and Persistence

This document describes the implementation of Phase 1 of the Featury Feature Flags CRUD API, focusing on establishing the foundational data layer and persistence mechanisms.

## Overview

Phase 1 implements the core data infrastructure needed for the Featury feature flag management system, including:
- PostgreSQL database schema and migrations
- Enhanced data models with proper Go structs
- Repository layer with comprehensive CRUD operations
- Database connection management and migration system
- Comprehensive unit tests

## Architecture

### Database Schema

The system uses PostgreSQL with the following core tables:

- **users**: User management with role-based access
- **feature_flags**: Feature flags with multi-environment configurations stored as JSONB
- **api_keys**: API authentication with permissions and expiration
- **audit_logs**: Complete audit trail for all operations
- **Indexes**: Optimized for common query patterns

### Data Models

Enhanced data models located in `/internal/models/`:

- `User`: User management with roles (admin, editor, viewer)
- `FeatureFlag`: Multi-environment feature flags with conditional rules
- `APIKey`: API authentication with granular permissions
- `AuditLog`: Comprehensive audit logging
- `Common`: Shared types, validation, and pagination utilities

### Repository Layer

Repository pattern implementation in `/internal/repository/`:

- Interface-based design for testability
- Transaction support with proper rollback handling  
- Advanced filtering and pagination
- Connection pooling and error handling
- Soft deletion support for feature flags

## Getting Started

### Prerequisites

- Go 1.24.2+
- PostgreSQL 12+
- Make (optional but recommended)

### Database Setup

1. **Start PostgreSQL** and create a user:
   ```sql
   CREATE USER featury WITH PASSWORD 'featury';
   CREATE DATABASE featury OWNER featury;
   GRANT ALL PRIVILEGES ON DATABASE featury TO featury;
   ```

2. **Copy configuration**:
   ```bash
   cp config.yaml.example config.yaml
   # Edit config.yaml with your database settings
   ```

3. **Build and run migrations**:
   ```bash
   # Build all binaries
   make build
   
   # Create database (if needed)
   make db-create
   
   # Run migrations
   make migrate-up
   
   # Check migration status
   make migrate-version
   ```

### Build and Test

```bash
# Build all components
make build

# Run unit tests (non-database tests)
make test

# Run integration tests (requires database)
make test-integration

# Run specific repository tests
make test-repo

# Generate test coverage report
make test-coverage
```

## Data Models Details

### User Model

```go
type User struct {
    ID        uuid.UUID `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Role      UserRole  `json:"role"` // admin, editor, viewer
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

**Features:**
- Email-based identification
- Role-based access control
- Audit timestamps

### Feature Flag Model

```go
type FeatureFlag struct {
    ID          uuid.UUID                      `json:"id"`
    Name        string                         `json:"name"`
    ServiceName string                         `json:"service_name"`
    Description string                         `json:"description"`
    CreatedBy   uuid.UUID                      `json:"created_by"`
    CreatedAt   time.Time                      `json:"created_at"`
    UpdatedAt   time.Time                      `json:"updated_at"`
    DeletedAt   *time.Time                     `json:"deleted_at,omitempty"`
    
    // Environment-specific configurations
    Environments map[string]EnvironmentConfig `json:"environments"`
}
```

**Features:**
- Multi-environment support (dev, staging, prod, etc.)
- Service-scoped feature flags
- Soft deletion capability
- Per-environment rollout percentages and conditional rules

### Environment Configuration

```go
type EnvironmentConfig struct {
    Enabled        bool               `json:"enabled"`
    RolloutPercent int                `json:"rollout_percent"` // 0-100
    Rules          []ConditionalRule  `json:"rules,omitempty"`
    UpdatedBy      uuid.UUID          `json:"updated_by"`
    UpdatedAt      time.Time          `json:"updated_at"`
}

type ConditionalRule struct {
    ID        uuid.UUID   `json:"id"`
    Attribute string      `json:"attribute"` // user_id, country, etc.
    Operator  string      `json:"operator"`  // equals, in, contains
    Value     interface{} `json:"value"`
    Enabled   bool        `json:"enabled"`
}
```

**Features:**
- Gradual rollout support with percentage-based targeting
- Conditional rules for advanced targeting
- Environment-specific configuration tracking

### API Key Model

```go
type APIKey struct {
    ID          uuid.UUID    `json:"id"`
    KeyHash     string       `json:"-"` // Hashed for security
    UserID      uuid.UUID    `json:"user_id"`
    Name        string       `json:"name"`
    Permissions []Permission `json:"permissions"`
    ExpiresAt   *time.Time   `json:"expires_at"`
    CreatedAt   time.Time    `json:"created_at"`
    LastUsedAt  *time.Time   `json:"last_used_at"`
}
```

**Features:**
- Secure key hashing
- Granular permission system
- Optional expiration
- Usage tracking

## Repository Operations

### Basic CRUD Operations

All repositories support:
- `Create()`: Insert new records
- `GetByID()`: Retrieve by primary key  
- `Update()`: Partial updates with optimistic concurrency
- `Delete()`: Hard or soft deletion
- `List()`: Paginated listing with filtering
- `Exists()`: Existence checks

### Advanced Features

- **Transaction Support**: `WithTx()` for atomic operations
- **Pagination**: Built-in pagination with configurable page sizes
- **Filtering**: Type-safe filtering with multiple criteria
- **Soft Deletion**: Feature flags support soft deletion with restore capability
- **Audit Trail**: Automatic audit logging for all operations

### Example Usage

```go
// Create a repository
repo := repository.NewRepository(db)

// Simple operation
user, err := repo.Users().GetByEmail(ctx, "user@example.com")

// Transaction example
err = repo.WithTx(ctx, func(txCtx context.Context) error {
    txRepo := repository.GetTxRepository(txCtx)
    
    // Create user
    user := &models.User{...}
    if err := txRepo.Users().Create(txCtx, user); err != nil {
        return err
    }
    
    // Create feature flag
    flag := &models.FeatureFlag{CreatedBy: user.ID, ...}
    return txRepo.FeatureFlags().Create(txCtx, flag)
})

// Paginated listing with filters
filter := &models.FeatureFlagFilter{ServiceName: "user-service"}
pagination := &models.PaginationParams{Page: 1, PageSize: 20}
flags, total, err := repo.FeatureFlags().List(ctx, filter, pagination)
```

## Database Migration System

### Migration CLI

The migration system provides:

```bash
# Run all pending migrations
./bin/featury-migrate up

# Rollback one migration  
./bin/featury-migrate down

# Show current version
./bin/featury-migrate version

# Reset all migrations (dangerous!)
./bin/featury-migrate reset

# Database management
./bin/featury-migrate create-db
./bin/featury-migrate drop-db
```

### Migration Files Structure

```
internal/database/migrations/
├── 001_create_users_table.up.sql
├── 001_create_users_table.down.sql
├── 002_create_feature_flags_table.up.sql
├── 002_create_feature_flags_table.down.sql
├── 003_create_api_keys_table.up.sql
├── 003_create_api_keys_table.down.sql
├── 004_create_audit_logs_table.up.sql
├── 004_create_audit_logs_table.down.sql
├── 005_create_indexes.up.sql
└── 005_create_indexes.down.sql
```

## Testing Strategy

### Unit Tests

Located in `/internal/repository/tests/`:
- Complete repository functionality testing
- Transaction rollback verification  
- Error condition handling
- Edge case validation

### Integration Tests

- Database-backed testing with real PostgreSQL
- Migration verification
- Performance testing for large datasets
- Connection pooling validation

### Running Tests

```bash
# Unit tests only
make test

# Integration tests (requires database)  
TEST_INTEGRATION=1 make test-integration

# Repository-specific tests
make test-repo

# Coverage report
make test-coverage
```

## Performance Considerations

### Database Optimizations

- **Indexes**: Comprehensive indexing strategy for common queries
- **JSONB**: Efficient storage and querying of environment configurations  
- **Partial Indexes**: Optimized indexes for filtered queries
- **Connection Pooling**: Configurable connection management

### Query Patterns

- **Pagination**: Offset/limit with total count caching
- **Filtering**: Indexed column filtering with JSONB support
- **Soft Deletion**: Efficient exclusion of deleted records
- **Audit Queries**: Time-based indexing for audit log retrieval

## Security Features

### Data Protection

- **API Key Hashing**: Secure storage of authentication tokens
- **Input Validation**: Comprehensive request validation
- **SQL Injection Protection**: Parameterized queries throughout
- **Audit Logging**: Complete operation tracking

### Access Control Foundation

- **Role-Based Permissions**: User role management
- **API Key Permissions**: Granular operation permissions  
- **Resource Scoping**: Service-based feature flag isolation

## Next Steps

Phase 1 provides the foundation for:

**Phase 2**: HTTP API Implementation
- RESTful API endpoints
- Authentication middleware  
- Input validation and error handling
- OpenAPI documentation

**Phase 3**: Advanced Features
- Real-time feature flag updates
- A/B testing framework
- Analytics and metrics
- Web dashboard

## Troubleshooting

### Common Issues

1. **Migration Errors**: Check database permissions and connection string
2. **Test Failures**: Ensure PostgreSQL is running and accessible
3. **Connection Issues**: Verify database configuration in config.yaml
4. **Build Errors**: Run `go mod tidy` to resolve dependencies

### Debug Commands

```bash
# Check database connectivity
./bin/featury-migrate version

# Verify migrations
./bin/featury-migrate version

# Test repository functions
TEST_INTEGRATION=1 go test -v ./internal/repository/tests/
```

## Configuration Reference

See `config.yaml.example` for complete configuration options including:
- Database connection settings
- Server configuration
- Logging levels
- Feature toggles
- Authentication settings (for future phases)

---

This Phase 1 implementation provides a robust, scalable foundation for the Featury feature flag system, with comprehensive data persistence, transaction support, and extensive testing coverage.