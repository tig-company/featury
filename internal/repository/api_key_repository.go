package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

type apiKeyRepository struct {
	db DBExecutor
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db DBExecutor) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	permissionsJSON, err := json.Marshal(apiKey.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	query := `
		INSERT INTO api_keys (id, key_hash, user_id, name, permissions, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err = r.db.ExecContext(ctx, query,
		apiKey.ID, apiKey.KeyHash, apiKey.UserID, apiKey.Name,
		permissionsJSON, apiKey.ExpiresAt, apiKey.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	
	return nil
}

func (r *apiKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	query := `
		SELECT id, key_hash, user_id, name, permissions, expires_at, created_at, last_used_at
		FROM api_keys 
		WHERE id = $1`
	
	return r.scanAPIKey(ctx, query, id)
}

func (r *apiKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	query := `
		SELECT id, key_hash, user_id, name, permissions, expires_at, created_at, last_used_at
		FROM api_keys 
		WHERE key_hash = $1`
	
	return r.scanAPIKey(ctx, query, keyHash)
}

func (r *apiKeyRepository) Update(ctx context.Context, id uuid.UUID, updates *models.UpdateAPIKeyRequest) (*models.APIKey, error) {
	if !updates.HasChanges() {
		return r.GetByID(ctx, id)
	}

	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if updates.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *updates.Name)
		argIndex++
	}

	if updates.ExpiresAt != nil {
		setParts = append(setParts, fmt.Sprintf("expires_at = $%d", argIndex))
		args = append(args, *updates.ExpiresAt)
		argIndex++
	}

	// Add the ID as the last argument
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE api_keys 
		SET %s
		WHERE id = $%d
		RETURNING id, key_hash, user_id, name, permissions, expires_at, created_at, last_used_at`,
		strings.Join(setParts, ", "), argIndex)

	return r.scanAPIKeyFromQuery(ctx, query, args...)
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update last used timestamp: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	
	return nil
}

func (r *apiKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	
	return nil
}

func (r *apiKeyRepository) List(ctx context.Context, filter *models.APIKeyFilter, pagination *models.PaginationParams) ([]*models.APIKey, int64, error) {
	// Normalize pagination
	if pagination != nil {
		pagination.Normalize()
	} else {
		pagination = &models.PaginationParams{}
		pagination.Normalize()
	}

	// Build WHERE clause
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if filter.UserID != uuid.Nil {
			whereConditions = append(whereConditions, fmt.Sprintf("user_id = $%d", argIndex))
			args = append(args, filter.UserID)
			argIndex++
		}

		if filter.Name != "" {
			whereConditions = append(whereConditions, fmt.Sprintf("name ILIKE $%d", argIndex))
			args = append(args, "%"+filter.Name+"%")
			argIndex++
		}

		if filter.IsExpired != nil {
			if *filter.IsExpired {
				whereConditions = append(whereConditions, fmt.Sprintf("expires_at IS NOT NULL AND expires_at < $%d", argIndex))
				args = append(args, time.Now())
				argIndex++
			} else {
				whereConditions = append(whereConditions, fmt.Sprintf("(expires_at IS NULL OR expires_at > $%d)", argIndex))
				args = append(args, time.Now())
				argIndex++
			}
		}

		if filter.HasExpiry != nil {
			if *filter.HasExpiry {
				whereConditions = append(whereConditions, "expires_at IS NOT NULL")
			} else {
				whereConditions = append(whereConditions, "expires_at IS NULL")
			}
		}
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM api_keys %s", whereClause)
	var totalCount int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count API keys: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, key_hash, user_id, name, permissions, expires_at, created_at, last_used_at
		FROM api_keys 
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, pagination.PageSize, pagination.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	apiKeys := make([]*models.APIKey, 0)
	for rows.Next() {
		apiKey, err := r.scanAPIKeyFromRow(rows)
		if err != nil {
			return nil, 0, err
		}
		apiKeys = append(apiKeys, apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate API keys: %w", err)
	}

	return apiKeys, totalCount, nil
}

func (r *apiKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.APIKey, error) {
	query := `
		SELECT id, key_hash, user_id, name, permissions, expires_at, created_at, last_used_at
		FROM api_keys 
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys by user: %w", err)
	}
	defer rows.Close()

	apiKeys := make([]*models.APIKey, 0)
	for rows.Next() {
		apiKey, err := r.scanAPIKeyFromRow(rows)
		if err != nil {
			return nil, err
		}
		apiKeys = append(apiKeys, apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate API keys: %w", err)
	}

	return apiKeys, nil
}

func (r *apiKeyRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = $1)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if API key exists: %w", err)
	}
	
	return exists, nil
}

func (r *apiKeyRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM api_keys WHERE expires_at IS NOT NULL AND expires_at < NOW()`
	
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired API keys: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}

// Helper methods

func (r *apiKeyRepository) scanAPIKey(ctx context.Context, query string, args ...interface{}) (*models.APIKey, error) {
	apiKey := &models.APIKey{}
	var permissionsJSON []byte
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.UserID, &apiKey.Name,
		&permissionsJSON, &expiresAt, &apiKey.CreatedAt, &lastUsedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to scan API key: %w", err)
	}

	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}

	if lastUsedAt.Valid {
		apiKey.LastUsedAt = &lastUsedAt.Time
	}

	// Unmarshal permissions JSON
	if len(permissionsJSON) > 0 {
		if err := json.Unmarshal(permissionsJSON, &apiKey.Permissions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
		}
	} else {
		apiKey.Permissions = make([]models.Permission, 0)
	}

	return apiKey, nil
}

func (r *apiKeyRepository) scanAPIKeyFromQuery(ctx context.Context, query string, args ...interface{}) (*models.APIKey, error) {
	apiKey := &models.APIKey{}
	var permissionsJSON []byte
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.UserID, &apiKey.Name,
		&permissionsJSON, &expiresAt, &apiKey.CreatedAt, &lastUsedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to scan API key: %w", err)
	}

	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}

	if lastUsedAt.Valid {
		apiKey.LastUsedAt = &lastUsedAt.Time
	}

	// Unmarshal permissions JSON
	if len(permissionsJSON) > 0 {
		if err := json.Unmarshal(permissionsJSON, &apiKey.Permissions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
		}
	} else {
		apiKey.Permissions = make([]models.Permission, 0)
	}

	return apiKey, nil
}

func (r *apiKeyRepository) scanAPIKeyFromRow(rows *sql.Rows) (*models.APIKey, error) {
	apiKey := &models.APIKey{}
	var permissionsJSON []byte
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime

	err := rows.Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.UserID, &apiKey.Name,
		&permissionsJSON, &expiresAt, &apiKey.CreatedAt, &lastUsedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to scan API key: %w", err)
	}

	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}

	if lastUsedAt.Valid {
		apiKey.LastUsedAt = &lastUsedAt.Time
	}

	// Unmarshal permissions JSON
	if len(permissionsJSON) > 0 {
		if err := json.Unmarshal(permissionsJSON, &apiKey.Permissions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
		}
	} else {
		apiKey.Permissions = make([]models.Permission, 0)
	}

	return apiKey, nil
}