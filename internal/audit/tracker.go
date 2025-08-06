package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
	"github.com/tig-company/featury/internal/repository"
)

type auditTracker struct {
	repo              repository.Repository
	differ            Differ
	metadataExtractor MetadataExtractor
}

// NewAuditTracker creates a new audit tracker
func NewAuditTracker(repo repository.Repository, differ Differ, metadataExtractor MetadataExtractor) Tracker {
	return &auditTracker{
		repo:              repo,
		differ:            differ,
		metadataExtractor: metadataExtractor,
	}
}

// TrackCreate tracks the creation of an entity
func (a *auditTracker) TrackCreate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error {
	changes := models.JSONB{
		"action": "create",
		"entity": entity,
	}

	// Merge provided metadata with extracted metadata
	finalMetadata := a.mergeMetadata(ctx, metadata)

	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     models.AuditActionCreate,
		UserID:     userID,
		Changes:    changes,
		Metadata:   finalMetadata,
	}

	if err := a.repo.AuditLogs().Create(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to create audit log for create action: %w", err)
	}

	return nil
}

// TrackUpdate tracks the update of an entity with before/after diff
func (a *auditTracker) TrackUpdate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, before, after interface{}, metadata models.JSONB) error {
	// Generate diff between before and after
	diff, err := a.differ.GenerateDiff(before, after)
	if err != nil {
		return fmt.Errorf("failed to generate diff for update tracking: %w", err)
	}

	changes := models.JSONB{
		"action": "update",
		"diff":   diff,
		"before": before,
		"after":  after,
	}

	// Merge provided metadata with extracted metadata
	finalMetadata := a.mergeMetadata(ctx, metadata)

	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     models.AuditActionUpdate,
		UserID:     userID,
		Changes:    changes,
		Metadata:   finalMetadata,
	}

	if err := a.repo.AuditLogs().Create(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to create audit log for update action: %w", err)
	}

	return nil
}

// TrackDelete tracks the deletion of an entity
func (a *auditTracker) TrackDelete(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error {
	changes := models.JSONB{
		"action": "delete",
		"entity": entity,
	}

	// Merge provided metadata with extracted metadata
	finalMetadata := a.mergeMetadata(ctx, metadata)

	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     models.AuditActionDelete,
		UserID:     userID,
		Changes:    changes,
		Metadata:   finalMetadata,
	}

	if err := a.repo.AuditLogs().Create(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to create audit log for delete action: %w", err)
	}

	return nil
}

// TrackAction tracks a custom action on an entity
func (a *auditTracker) TrackAction(ctx context.Context, entityType string, entityID uuid.UUID, action models.AuditAction, userID uuid.UUID, changes models.JSONB, metadata models.JSONB) error {
	// Merge provided metadata with extracted metadata
	finalMetadata := a.mergeMetadata(ctx, metadata)

	auditLog := &models.AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		UserID:     userID,
		Changes:    changes,
		Metadata:   finalMetadata,
	}

	if err := a.repo.AuditLogs().Create(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to create audit log for action %s: %w", action, err)
	}

	return nil
}

// mergeMetadata merges provided metadata with extracted metadata from context
func (a *auditTracker) mergeMetadata(ctx context.Context, providedMetadata models.JSONB) models.JSONB {
	finalMetadata := make(models.JSONB)

	// Start with extracted metadata from context
	if a.metadataExtractor != nil {
		extractedMetadata := a.metadataExtractor.ExtractMetadata(ctx)
		for key, value := range extractedMetadata {
			finalMetadata[key] = value
		}
	}

	// Override with provided metadata
	for key, value := range providedMetadata {
		finalMetadata[key] = value
	}

	return finalMetadata
}