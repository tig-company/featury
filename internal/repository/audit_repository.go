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

type auditLogRepository struct {
	db DBExecutor
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db DBExecutor) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, auditLog *models.AuditLog) error {
	var changesJSON, metadataJSON []byte
	var err error

	if auditLog.Changes != nil {
		changesJSON, err = json.Marshal(auditLog.Changes)
		if err != nil {
			return fmt.Errorf("failed to marshal changes: %w", err)
		}
	}

	if auditLog.Metadata != nil {
		metadataJSON, err = json.Marshal(auditLog.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO audit_logs (id, entity_type, entity_id, action, user_id, changes, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err = r.db.ExecContext(ctx, query,
		auditLog.ID, auditLog.EntityType, auditLog.EntityID, auditLog.Action,
		auditLog.UserID, changesJSON, metadataJSON, auditLog.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	
	return nil
}

func (r *auditLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, user_id, changes, metadata, created_at
		FROM audit_logs 
		WHERE id = $1`
	
	return r.scanAuditLog(ctx, query, id)
}

func (r *auditLogRepository) List(ctx context.Context, filter *models.AuditLogFilter, pagination *models.PaginationParams) ([]*models.AuditLog, int64, error) {
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
		if filter.EntityType != "" {
			whereConditions = append(whereConditions, fmt.Sprintf("entity_type = $%d", argIndex))
			args = append(args, filter.EntityType)
			argIndex++
		}

		if filter.EntityID != nil {
			whereConditions = append(whereConditions, fmt.Sprintf("entity_id = $%d", argIndex))
			args = append(args, *filter.EntityID)
			argIndex++
		}

		if filter.Action != "" {
			whereConditions = append(whereConditions, fmt.Sprintf("action = $%d", argIndex))
			args = append(args, filter.Action)
			argIndex++
		}

		if filter.UserID != nil {
			whereConditions = append(whereConditions, fmt.Sprintf("user_id = $%d", argIndex))
			args = append(args, *filter.UserID)
			argIndex++
		}

		if filter.FromDate != nil {
			whereConditions = append(whereConditions, fmt.Sprintf("created_at >= $%d", argIndex))
			args = append(args, *filter.FromDate)
			argIndex++
		}

		if filter.ToDate != nil {
			whereConditions = append(whereConditions, fmt.Sprintf("created_at <= $%d", argIndex))
			args = append(args, *filter.ToDate)
			argIndex++
		}
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var totalCount int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, entity_type, entity_id, action, user_id, changes, metadata, created_at
		FROM audit_logs 
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, pagination.PageSize, pagination.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	auditLogs := make([]*models.AuditLog, 0)
	for rows.Next() {
		auditLog, err := r.scanAuditLogFromRow(rows)
		if err != nil {
			return nil, 0, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate audit logs: %w", err)
	}

	return auditLogs, totalCount, nil
}

func (r *auditLogRepository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID, pagination *models.PaginationParams) ([]*models.AuditLog, int64, error) {
	// Normalize pagination
	if pagination != nil {
		pagination.Normalize()
	} else {
		pagination = &models.PaginationParams{}
		pagination.Normalize()
	}

	// Count total records
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE entity_type = $1 AND entity_id = $2`
	var totalCount int64
	err := r.db.QueryRowContext(ctx, countQuery, entityType, entityID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs by entity: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, entity_type, entity_id, action, user_id, changes, metadata, created_at
		FROM audit_logs 
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID, pagination.PageSize, pagination.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs by entity: %w", err)
	}
	defer rows.Close()

	auditLogs := make([]*models.AuditLog, 0)
	for rows.Next() {
		auditLog, err := r.scanAuditLogFromRow(rows)
		if err != nil {
			return nil, 0, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate audit logs: %w", err)
	}

	return auditLogs, totalCount, nil
}

func (r *auditLogRepository) ListByUser(ctx context.Context, userID uuid.UUID, pagination *models.PaginationParams) ([]*models.AuditLog, int64, error) {
	// Normalize pagination
	if pagination != nil {
		pagination.Normalize()
	} else {
		pagination = &models.PaginationParams{}
		pagination.Normalize()
	}

	// Count total records
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE user_id = $1`
	var totalCount int64
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs by user: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, entity_type, entity_id, action, user_id, changes, metadata, created_at
		FROM audit_logs 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, pagination.PageSize, pagination.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs by user: %w", err)
	}
	defer rows.Close()

	auditLogs := make([]*models.AuditLog, 0)
	for rows.Next() {
		auditLog, err := r.scanAuditLogFromRow(rows)
		if err != nil {
			return nil, 0, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate audit logs: %w", err)
	}

	return auditLogs, totalCount, nil
}

func (r *auditLogRepository) DeleteOlderThan(ctx context.Context, olderThanDays int) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '%d days'`
	
	result, err := r.db.ExecContext(ctx, fmt.Sprintf(query, olderThanDays))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}

func (r *auditLogRepository) GetEntityHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, user_id, changes, metadata, created_at
		FROM audit_logs 
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity history: %w", err)
	}
	defer rows.Close()

	auditLogs := make([]*models.AuditLog, 0)
	for rows.Next() {
		auditLog, err := r.scanAuditLogFromRow(rows)
		if err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate entity history: %w", err)
	}

	return auditLogs, nil
}

func (r *auditLogRepository) GetUserActivity(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, entity_type, entity_id, action, user_id, changes, metadata, created_at
		FROM audit_logs 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query user activity: %w", err)
	}
	defer rows.Close()

	auditLogs := make([]*models.AuditLog, 0)
	for rows.Next() {
		auditLog, err := r.scanAuditLogFromRow(rows)
		if err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate user activity: %w", err)
	}

	return auditLogs, nil
}

// Helper methods

func (r *auditLogRepository) scanAuditLog(ctx context.Context, query string, args ...interface{}) (*models.AuditLog, error) {
	auditLog := &models.AuditLog{}
	var changesJSON, metadataJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&auditLog.ID, &auditLog.EntityType, &auditLog.EntityID, &auditLog.Action,
		&auditLog.UserID, &changesJSON, &metadataJSON, &auditLog.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to scan audit log: %w", err)
	}

	// Unmarshal changes JSON
	if changesJSON.Valid && changesJSON.String != "" {
		changes := make(models.JSONB)
		if err := json.Unmarshal([]byte(changesJSON.String), &changes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal changes: %w", err)
		}
		auditLog.Changes = changes
	}

	// Unmarshal metadata JSON
	if metadataJSON.Valid && metadataJSON.String != "" {
		metadata := make(models.JSONB)
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		auditLog.Metadata = metadata
	}

	return auditLog, nil
}

func (r *auditLogRepository) scanAuditLogFromRow(rows *sql.Rows) (*models.AuditLog, error) {
	auditLog := &models.AuditLog{}
	var changesJSON, metadataJSON sql.NullString

	err := rows.Scan(
		&auditLog.ID, &auditLog.EntityType, &auditLog.EntityID, &auditLog.Action,
		&auditLog.UserID, &changesJSON, &metadataJSON, &auditLog.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to scan audit log: %w", err)
	}

	// Unmarshal changes JSON
	if changesJSON.Valid && changesJSON.String != "" {
		changes := make(models.JSONB)
		if err := json.Unmarshal([]byte(changesJSON.String), &changes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal changes: %w", err)
		}
		auditLog.Changes = changes
	}

	// Unmarshal metadata JSON
	if metadataJSON.Valid && metadataJSON.String != "" {
		metadata := make(models.JSONB)
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		auditLog.Metadata = metadata
	}

	return auditLog, nil
}