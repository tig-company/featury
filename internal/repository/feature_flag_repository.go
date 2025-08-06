package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

type featureFlagRepository struct {
	db DBExecutor
}

// NewFeatureFlagRepository creates a new feature flag repository
func NewFeatureFlagRepository(db DBExecutor) FeatureFlagRepository {
	return &featureFlagRepository{db: db}
}

func (r *featureFlagRepository) Create(ctx context.Context, flag *models.FeatureFlag) error {
	environmentsJSON, err := json.Marshal(flag.Environments)
	if err != nil {
		return fmt.Errorf("failed to marshal environments: %w", err)
	}

	query := `
		INSERT INTO feature_flags (id, name, service_name, description, created_by, 
			created_at, updated_at, environments)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err = r.db.ExecContext(ctx, query,
		flag.ID, flag.Name, flag.ServiceName, flag.Description, flag.CreatedBy,
		flag.CreatedAt, flag.UpdatedAt, environmentsJSON)
	
	if err != nil {
		return fmt.Errorf("failed to create feature flag: %w", err)
	}
	
	return nil
}

func (r *featureFlagRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error) {
	query := `
		SELECT id, name, service_name, description, created_by, created_at, 
			updated_at, deleted_at, environments
		FROM feature_flags 
		WHERE id = $1`
	
	return r.scanFeatureFlag(ctx, query, id)
}

func (r *featureFlagRepository) GetActiveByID(ctx context.Context, id uuid.UUID) (*models.FeatureFlag, error) {
	query := `
		SELECT id, name, service_name, description, created_by, created_at, 
			updated_at, deleted_at, environments
		FROM feature_flags 
		WHERE id = $1 AND deleted_at IS NULL`
	
	return r.scanFeatureFlag(ctx, query, id)
}

func (r *featureFlagRepository) GetByName(ctx context.Context, name, serviceName string) (*models.FeatureFlag, error) {
	query := `
		SELECT id, name, service_name, description, created_by, created_at, 
			updated_at, deleted_at, environments
		FROM feature_flags 
		WHERE name = $1 AND service_name = $2 AND deleted_at IS NULL`
	
	return r.scanFeatureFlag(ctx, query, name, serviceName)
}

func (r *featureFlagRepository) Update(ctx context.Context, id uuid.UUID, updates *models.UpdateFeatureFlagRequest, updatedBy uuid.UUID) (*models.FeatureFlag, error) {
	if !updates.HasChanges() {
		return r.GetActiveByID(ctx, id)
	}

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if updates.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *updates.Description)
		argIndex++
	}

	if updates.Environments != nil {
		environmentsJSON, err := json.Marshal(updates.Environments)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environments: %w", err)
		}
		setParts = append(setParts, fmt.Sprintf("environments = $%d", argIndex))
		args = append(args, environmentsJSON)
		argIndex++
	}

	// Always update the updated_at timestamp
	setParts = append(setParts, fmt.Sprintf("updated_at = NOW()"))

	// Add the ID as the last argument
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE feature_flags 
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, name, service_name, description, created_by, created_at, 
			updated_at, deleted_at, environments`,
		strings.Join(setParts, ", "), argIndex)

	return r.scanFeatureFlagFromQuery(ctx, query, args...)
}

func (r *featureFlagRepository) UpdateEnvironment(ctx context.Context, id uuid.UUID, environment string, config *models.EnvironmentConfig) error {
	// First get the current feature flag to get existing environments
	flag, err := r.GetActiveByID(ctx, id)
	if err != nil {
		return err
	}

	// Update the specific environment
	flag.Environments[environment] = *config

	// Marshal the updated environments
	environmentsJSON, err := json.Marshal(flag.Environments)
	if err != nil {
		return fmt.Errorf("failed to marshal environments: %w", err)
	}

	query := `
		UPDATE feature_flags 
		SET environments = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, environmentsJSON, id)
	if err != nil {
		return fmt.Errorf("failed to update feature flag environment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feature flag not found or already deleted")
	}

	return nil
}

func (r *featureFlagRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE feature_flags 
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete feature flag: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("feature flag not found or already deleted")
	}
	
	return nil
}

func (r *featureFlagRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM feature_flags WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to hard delete feature flag: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("feature flag not found")
	}
	
	return nil
}

func (r *featureFlagRepository) Restore(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE feature_flags 
		SET deleted_at = NULL, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NOT NULL`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to restore feature flag: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("feature flag not found or not deleted")
	}
	
	return nil
}

func (r *featureFlagRepository) List(ctx context.Context, filter *models.FeatureFlagFilter, pagination *models.PaginationParams) ([]*models.FeatureFlag, int64, error) {
	// Normalize pagination
	if pagination != nil {
		pagination.Normalize()
	} else {
		pagination = &models.PaginationParams{}
		pagination.Normalize()
	}

	// Build WHERE clause
	whereConditions := []string{"deleted_at IS NULL"} // Always exclude soft deleted
	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if filter.ServiceName != "" {
			whereConditions = append(whereConditions, fmt.Sprintf("service_name = $%d", argIndex))
			args = append(args, filter.ServiceName)
			argIndex++
		}

		if filter.CreatedBy != uuid.Nil {
			whereConditions = append(whereConditions, fmt.Sprintf("created_by = $%d", argIndex))
			args = append(args, filter.CreatedBy)
			argIndex++
		}

		if filter.Name != "" {
			whereConditions = append(whereConditions, fmt.Sprintf("name ILIKE $%d", argIndex))
			args = append(args, "%"+filter.Name+"%")
			argIndex++
		}

		if filter.Environment != "" && filter.Enabled != nil {
			// Query environments JSONB for specific environment and enabled status
			whereConditions = append(whereConditions, fmt.Sprintf("environments->$%d->>'enabled' = $%d", argIndex, argIndex+1))
			args = append(args, filter.Environment, fmt.Sprintf("%t", *filter.Enabled))
			argIndex += 2
		} else if filter.Environment != "" {
			// Just check if environment exists
			whereConditions = append(whereConditions, fmt.Sprintf("environments ? $%d", argIndex))
			args = append(args, filter.Environment)
			argIndex++
		}
	}

	whereClause := "WHERE " + strings.Join(whereConditions, " AND ")

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM feature_flags %s", whereClause)
	var totalCount int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count feature flags: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, name, service_name, description, created_by, created_at, 
			updated_at, deleted_at, environments
		FROM feature_flags 
		%s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, pagination.PageSize, pagination.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query feature flags: %w", err)
	}
	defer rows.Close()

	flags := make([]*models.FeatureFlag, 0)
	for rows.Next() {
		flag, err := r.scanFeatureFlagFromRow(rows)
		if err != nil {
			return nil, 0, err
		}
		flags = append(flags, flag)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate feature flags: %w", err)
	}

	return flags, totalCount, nil
}

func (r *featureFlagRepository) ListByService(ctx context.Context, serviceName string) ([]*models.FeatureFlag, error) {
	query := `
		SELECT id, name, service_name, description, created_by, created_at, 
			updated_at, deleted_at, environments
		FROM feature_flags 
		WHERE service_name = $1 AND deleted_at IS NULL
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to query feature flags by service: %w", err)
	}
	defer rows.Close()

	flags := make([]*models.FeatureFlag, 0)
	for rows.Next() {
		flag, err := r.scanFeatureFlagFromRow(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate feature flags: %w", err)
	}

	return flags, nil
}

func (r *featureFlagRepository) Exists(ctx context.Context, name, serviceName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM feature_flags WHERE name = $1 AND service_name = $2 AND deleted_at IS NULL)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, name, serviceName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if feature flag exists: %w", err)
	}
	
	return exists, nil
}

func (r *featureFlagRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM feature_flags WHERE id = $1)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if feature flag exists: %w", err)
	}
	
	return exists, nil
}

func (r *featureFlagRepository) GetEnvironments(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT jsonb_object_keys(environments) as environment
		FROM feature_flags 
		WHERE deleted_at IS NULL AND environments != '{}'::jsonb
		ORDER BY environment`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get environments: %w", err)
	}
	defer rows.Close()

	environments := make([]string, 0)
	for rows.Next() {
		var env string
		if err := rows.Scan(&env); err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}
		environments = append(environments, env)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate environments: %w", err)
	}

	return environments, nil
}

func (r *featureFlagRepository) GetServices(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT service_name
		FROM feature_flags 
		WHERE deleted_at IS NULL
		ORDER BY service_name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}
	defer rows.Close()

	services := make([]string, 0)
	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate services: %w", err)
	}

	return services, nil
}

// Helper methods

func (r *featureFlagRepository) scanFeatureFlag(ctx context.Context, query string, args ...interface{}) (*models.FeatureFlag, error) {
	flag := &models.FeatureFlag{}
	var environmentsJSON []byte
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&flag.ID, &flag.Name, &flag.ServiceName, &flag.Description, &flag.CreatedBy,
		&flag.CreatedAt, &flag.UpdatedAt, &deletedAt, &environmentsJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feature flag not found")
		}
		return nil, fmt.Errorf("failed to scan feature flag: %w", err)
	}

	if deletedAt.Valid {
		flag.DeletedAt = &deletedAt.Time
	}

	// Unmarshal environments JSON
	if len(environmentsJSON) > 0 {
		if err := json.Unmarshal(environmentsJSON, &flag.Environments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal environments: %w", err)
		}
	} else {
		flag.Environments = make(map[string]models.EnvironmentConfig)
	}

	return flag, nil
}

func (r *featureFlagRepository) scanFeatureFlagFromQuery(ctx context.Context, query string, args ...interface{}) (*models.FeatureFlag, error) {
	flag := &models.FeatureFlag{}
	var environmentsJSON []byte
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&flag.ID, &flag.Name, &flag.ServiceName, &flag.Description, &flag.CreatedBy,
		&flag.CreatedAt, &flag.UpdatedAt, &deletedAt, &environmentsJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feature flag not found")
		}
		return nil, fmt.Errorf("failed to scan feature flag: %w", err)
	}

	if deletedAt.Valid {
		flag.DeletedAt = &deletedAt.Time
	}

	// Unmarshal environments JSON
	if len(environmentsJSON) > 0 {
		if err := json.Unmarshal(environmentsJSON, &flag.Environments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal environments: %w", err)
		}
	} else {
		flag.Environments = make(map[string]models.EnvironmentConfig)
	}

	return flag, nil
}

func (r *featureFlagRepository) scanFeatureFlagFromRow(rows *sql.Rows) (*models.FeatureFlag, error) {
	flag := &models.FeatureFlag{}
	var environmentsJSON []byte
	var deletedAt sql.NullTime

	err := rows.Scan(
		&flag.ID, &flag.Name, &flag.ServiceName, &flag.Description, &flag.CreatedBy,
		&flag.CreatedAt, &flag.UpdatedAt, &deletedAt, &environmentsJSON)

	if err != nil {
		return nil, fmt.Errorf("failed to scan feature flag: %w", err)
	}

	if deletedAt.Valid {
		flag.DeletedAt = &deletedAt.Time
	}

	// Unmarshal environments JSON
	if len(environmentsJSON) > 0 {
		if err := json.Unmarshal(environmentsJSON, &flag.Environments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal environments: %w", err)
		}
	} else {
		flag.Environments = make(map[string]models.EnvironmentConfig)
	}

	return flag, nil
}