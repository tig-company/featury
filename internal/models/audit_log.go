package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry in the system
type AuditLog struct {
	ID         uuid.UUID   `json:"id" db:"id"`
	EntityType string      `json:"entity_type" db:"entity_type"` // feature_flag, user, api_key
	EntityID   uuid.UUID   `json:"entity_id" db:"entity_id"`
	Action     AuditAction `json:"action" db:"action"`
	UserID     uuid.UUID   `json:"user_id" db:"user_id"`
	Changes    JSONB       `json:"changes,omitempty" db:"changes"`    // Before/after diff
	Metadata   JSONB       `json:"metadata,omitempty" db:"metadata"`  // Request info, IP, etc.
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
}

// CreateAuditLogRequest represents a request to create a new audit log entry
type CreateAuditLogRequest struct {
	EntityType string      `json:"entity_type" binding:"required"`
	EntityID   uuid.UUID   `json:"entity_id" binding:"required"`
	Action     AuditAction `json:"action" binding:"required"`
	UserID     uuid.UUID   `json:"user_id" binding:"required"`
	Changes    JSONB       `json:"changes,omitempty"`
	Metadata   JSONB       `json:"metadata,omitempty"`
}

// Validate validates the create audit log request
func (car *CreateAuditLogRequest) Validate() error {
	if car.EntityType == "" {
		return NewValidationError("entity_type", "entity type is required")
	}
	
	if !car.Action.Valid() {
		return NewValidationError("action", "invalid audit action")
	}
	
	// Validate entity type
	switch car.EntityType {
	case "feature_flag", "user", "api_key":
		// Valid entity types
	default:
		return NewValidationError("entity_type", "invalid entity type")
	}
	
	return nil
}

// AuditLogFilter represents filters for audit log queries
type AuditLogFilter struct {
	EntityType string      `json:"entity_type,omitempty" form:"entity_type"`
	EntityID   *uuid.UUID  `json:"entity_id,omitempty" form:"entity_id"`
	Action     AuditAction `json:"action,omitempty" form:"action"`
	UserID     *uuid.UUID  `json:"user_id,omitempty" form:"user_id"`
	FromDate   *time.Time  `json:"from_date,omitempty" form:"from_date"`
	ToDate     *time.Time  `json:"to_date,omitempty" form:"to_date"`
}

// AuditLogResponse represents an audit log entry in API responses
type AuditLogResponse struct {
	ID         uuid.UUID   `json:"id"`
	EntityType string      `json:"entity_type"`
	EntityID   uuid.UUID   `json:"entity_id"`
	Action     AuditAction `json:"action"`
	UserID     uuid.UUID   `json:"user_id"`
	Changes    JSONB       `json:"changes,omitempty"`
	Metadata   JSONB       `json:"metadata,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

// ToResponse converts an AuditLog to AuditLogResponse
func (al *AuditLog) ToResponse() *AuditLogResponse {
	return &AuditLogResponse{
		ID:         al.ID,
		EntityType: al.EntityType,
		EntityID:   al.EntityID,
		Action:     al.Action,
		UserID:     al.UserID,
		Changes:    al.Changes,
		Metadata:   al.Metadata,
		CreatedAt:  al.CreatedAt,
	}
}

// ChangesDiff represents the changes made to an entity
type ChangesDiff struct {
	Before map[string]interface{} `json:"before,omitempty"`
	After  map[string]interface{} `json:"after,omitempty"`
	Fields []string               `json:"fields"`
}

// RequestMetadata represents metadata about the request that triggered the audit log
type RequestMetadata struct {
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	Source    string `json:"source,omitempty"` // api, cli, web
}

// NewChangesDiff creates a new changes diff
func NewChangesDiff(before, after map[string]interface{}) *ChangesDiff {
	diff := &ChangesDiff{
		Before: before,
		After:  after,
		Fields: make([]string, 0),
	}
	
	// Track which fields changed
	allFields := make(map[string]bool)
	for field := range before {
		allFields[field] = true
	}
	for field := range after {
		allFields[field] = true
	}
	
	for field := range allFields {
		beforeVal, beforeExists := before[field]
		afterVal, afterExists := after[field]
		
		// Field was added, removed, or changed
		if !beforeExists || !afterExists || beforeVal != afterVal {
			diff.Fields = append(diff.Fields, field)
		}
	}
	
	return diff
}