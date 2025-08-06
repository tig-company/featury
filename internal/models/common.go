package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleEditor UserRole = "editor"
	RoleViewer UserRole = "viewer"
)

// Valid returns true if the user role is valid
func (ur UserRole) Valid() bool {
	switch ur {
	case RoleAdmin, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

// AuditAction represents the type of action performed
type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionView   AuditAction = "view"
	AuditActionEnable AuditAction = "enable"
	AuditActionDisable AuditAction = "disable"
)

// Valid returns true if the audit action is valid
func (aa AuditAction) Valid() bool {
	switch aa {
	case AuditActionCreate, AuditActionUpdate, AuditActionDelete, AuditActionView, AuditActionEnable, AuditActionDisable:
		return true
	default:
		return false
	}
}

// Permission represents a specific permission for API keys
type Permission string

const (
	PermissionReadFeatureFlags   Permission = "read:feature_flags"
	PermissionWriteFeatureFlags  Permission = "write:feature_flags"
	PermissionDeleteFeatureFlags Permission = "delete:feature_flags"
	PermissionReadUsers          Permission = "read:users"
	PermissionWriteUsers         Permission = "write:users"
	PermissionReadAPIKeys        Permission = "read:api_keys"
	PermissionWriteAPIKeys       Permission = "write:api_keys"
	PermissionReadAuditLogs      Permission = "read:audit_logs"
)

// Valid returns true if the permission is valid
func (p Permission) Valid() bool {
	switch p {
	case PermissionReadFeatureFlags, PermissionWriteFeatureFlags, PermissionDeleteFeatureFlags,
		PermissionReadUsers, PermissionWriteUsers,
		PermissionReadAPIKeys, PermissionWriteAPIKeys,
		PermissionReadAuditLogs:
		return true
	default:
		return false
	}
}

// JSONB represents a PostgreSQL JSONB field
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// BaseModel contains common fields for all entities
type BaseModel struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted returns true if the entity is soft deleted
func (bm BaseModel) IsDeleted() bool {
	return bm.DeletedAt != nil
}

// PaginationParams represents pagination parameters for queries
type PaginationParams struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
	Offset   int `json:"-"`
}

// Normalize ensures pagination parameters have valid values
func (p *PaginationParams) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	p.Offset = (p.Page - 1) * p.PageSize
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int64       `json:"total_count"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, params PaginationParams, totalCount int64) *PaginatedResponse {
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginatedResponse{
		Data:       data,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", ve.Field, ve.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}