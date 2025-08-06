package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/audit"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/repository"
)

type auditService struct {
	repo    repository.Repository
	tracker audit.Tracker
	differ  audit.Differ
}

// NewAuditService creates a new audit service
func NewAuditService(repo repository.Repository, tracker audit.Tracker, differ audit.Differ) AuditService {
	return &auditService{
		repo:    repo,
		tracker: tracker,
		differ:  differ,
	}
}

// LogAction logs an audit action
func (s *auditService) LogAction(ctx context.Context, entityType string, entityID uuid.UUID, action models.AuditAction, userID uuid.UUID, changes models.JSONB, metadata models.JSONB) error {
	if err := s.tracker.TrackAction(ctx, entityType, entityID, action, userID, changes, metadata); err != nil {
		return fmt.Errorf("failed to track audit action: %w", err)
	}
	return nil
}

// LogCreate logs a create operation
func (s *auditService) LogCreate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error {
	if err := s.tracker.TrackCreate(ctx, entityType, entityID, userID, entity, metadata); err != nil {
		return fmt.Errorf("failed to track create operation: %w", err)
	}
	return nil
}

// LogUpdate logs an update operation with before/after diff
func (s *auditService) LogUpdate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, before, after interface{}, metadata models.JSONB) error {
	if err := s.tracker.TrackUpdate(ctx, entityType, entityID, userID, before, after, metadata); err != nil {
		return fmt.Errorf("failed to track update operation: %w", err)
	}
	return nil
}

// LogDelete logs a delete operation
func (s *auditService) LogDelete(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error {
	if err := s.tracker.TrackDelete(ctx, entityType, entityID, userID, entity, metadata); err != nil {
		return fmt.Errorf("failed to track delete operation: %w", err)
	}
	return nil
}

// GetEntityHistory retrieves the complete change history for an entity
func (s *auditService) GetEntityHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.AuditLog, error) {
	auditLogs, err := s.repo.AuditLogs().GetEntityHistory(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity history: %w", err)
	}
	return auditLogs, nil
}

// GetUserActivity retrieves recent user activity
func (s *auditService) GetUserActivity(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error) {
	auditLogs, err := s.repo.AuditLogs().GetUserActivity(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activity: %w", err)
	}
	return auditLogs, nil
}

// GetRecentActivity retrieves recent audit activity across all entities
func (s *auditService) GetRecentActivity(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	// Use List with empty filter and pagination to get recent activity
	pagination := &models.PaginationParams{
		Page:     1,
		PageSize: limit,
	}
	pagination.Normalize()

	auditLogs, _, err := s.repo.AuditLogs().List(ctx, nil, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	return auditLogs, nil
}

// GetActivityByAction retrieves audit logs filtered by action type
func (s *auditService) GetActivityByAction(ctx context.Context, action models.AuditAction, limit int) ([]*models.AuditLog, error) {
	// Create filter for specific action
	filter := &models.AuditLogFilter{
		Action: action,
	}
	
	pagination := &models.PaginationParams{
		Page:     1,
		PageSize: limit,
	}
	pagination.Normalize()

	auditLogs, _, err := s.repo.AuditLogs().List(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity by action: %w", err)
	}
	return auditLogs, nil
}

// GetActivitySummary retrieves a summary of audit activity (simplified implementation)
func (s *auditService) GetActivitySummary(ctx context.Context, entityType string, days int) (*AuditSummary, error) {
	// For now, provide a basic implementation using List
	// In production, this should be implemented as a specialized repository method
	filter := &models.AuditLogFilter{
		EntityType: entityType,
	}
	
	pagination := &models.PaginationParams{
		Page:     1,
		PageSize: 1000, // Get a reasonable sample
	}
	pagination.Normalize()

	auditLogs, totalCount, err := s.repo.AuditLogs().List(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity summary: %w", err)
	}

	// Build summary from results
	actionCounts := make(map[models.AuditAction]int64)
	userCounts := make(map[string]int64)

	for _, log := range auditLogs {
		actionCounts[log.Action]++
		userCounts[log.UserID.String()]++
	}

	return &AuditSummary{
		EntityType:   entityType,
		Days:         days,
		TotalActions: totalCount,
		ActionCounts: actionCounts,
		UserCounts:   userCounts,
	}, nil
}

// AuditSummary represents a summary of audit activity
type AuditSummary struct {
	EntityType   string                    `json:"entity_type"`
	Days         int                       `json:"days"`
	TotalActions int64                     `json:"total_actions"`
	ActionCounts map[models.AuditAction]int64 `json:"action_counts"`
	UserCounts   map[string]int64          `json:"user_counts"`
}