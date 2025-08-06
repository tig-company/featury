package audit

import (
	"context"

	"github.com/google/uuid"
	"github.com/tig-company/featury/internal/models"
)

// Tracker defines the interface for tracking changes
type Tracker interface {
	// TrackCreate tracks the creation of an entity
	TrackCreate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error

	// TrackUpdate tracks the update of an entity with before/after diff
	TrackUpdate(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, before, after interface{}, metadata models.JSONB) error

	// TrackDelete tracks the deletion of an entity
	TrackDelete(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID, entity interface{}, metadata models.JSONB) error

	// TrackAction tracks a custom action on an entity
	TrackAction(ctx context.Context, entityType string, entityID uuid.UUID, action models.AuditAction, userID uuid.UUID, changes models.JSONB, metadata models.JSONB) error
}

// Differ defines the interface for generating diffs between objects
type Differ interface {
	// GenerateDiff generates a diff between before and after objects
	GenerateDiff(before, after interface{}) (models.JSONB, error)

	// GenerateFeatureFlagDiff generates a specialized diff for feature flags
	GenerateFeatureFlagDiff(before, after *models.FeatureFlag) (models.JSONB, error)

	// GenerateEnvironmentDiff generates a diff for environment configurations
	GenerateEnvironmentDiff(before, after models.EnvironmentConfig) (models.JSONB, error)
}

// MetadataExtractor defines the interface for extracting metadata from context
type MetadataExtractor interface {
	// ExtractMetadata extracts audit metadata from the context
	ExtractMetadata(ctx context.Context) models.JSONB

	// ExtractUserInfo extracts user information from context
	ExtractUserInfo(ctx context.Context) (userID uuid.UUID, exists bool)

	// ExtractRequestInfo extracts request information from context
	ExtractRequestInfo(ctx context.Context) models.JSONB
}