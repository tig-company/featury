package auth

import (
	"fmt"
	"strings"

	"github.com/tig-company/featury/internal/models"
)

// PermissionLevel represents the hierarchical level of a permission
type PermissionLevel int

const (
	PermissionLevelRead PermissionLevel = iota
	PermissionLevelWrite
	PermissionLevelDelete
	PermissionLevelAdmin
)

// ResourceType represents the type of resource being accessed
type ResourceType string

const (
	ResourceTypeFeatureFlags ResourceType = "feature_flags"
	ResourceTypeUsers        ResourceType = "users"
	ResourceTypeAPIKeys      ResourceType = "api_keys"
	ResourceTypeAuditLogs    ResourceType = "audit_logs"
)

// PermissionChecker provides methods to check permissions
type PermissionChecker struct{}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker() *PermissionChecker {
	return &PermissionChecker{}
}

// HasPermission checks if the given permissions include a specific permission
func (pc *PermissionChecker) HasPermission(userPermissions []models.Permission, required models.Permission) bool {
	for _, permission := range userPermissions {
		if permission == required {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the given permissions include any of the required permissions
func (pc *PermissionChecker) HasAnyPermission(userPermissions []models.Permission, required ...models.Permission) bool {
	for _, requiredPerm := range required {
		if pc.HasPermission(userPermissions, requiredPerm) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the given permissions include all required permissions
func (pc *PermissionChecker) HasAllPermissions(userPermissions []models.Permission, required ...models.Permission) bool {
	for _, requiredPerm := range required {
		if !pc.HasPermission(userPermissions, requiredPerm) {
			return false
		}
	}
	return true
}

// CheckResourceAccess checks if the permissions allow access to a resource with specific action
func (pc *PermissionChecker) CheckResourceAccess(userPermissions []models.Permission, resource ResourceType, action string) bool {
	// Convert action to lowercase for consistency
	action = strings.ToLower(action)

	// Determine required permission based on resource and action
	var requiredPermission models.Permission

	switch resource {
	case ResourceTypeFeatureFlags:
		switch action {
		case "read", "get", "list":
			requiredPermission = models.PermissionReadFeatureFlags
		case "write", "create", "update", "post", "put", "patch":
			requiredPermission = models.PermissionWriteFeatureFlags
		case "delete":
			requiredPermission = models.PermissionDeleteFeatureFlags
		default:
			return false
		}
	case ResourceTypeUsers:
		switch action {
		case "read", "get", "list":
			requiredPermission = models.PermissionReadUsers
		case "write", "create", "update", "post", "put", "patch":
			requiredPermission = models.PermissionWriteUsers
		default:
			return false
		}
	case ResourceTypeAPIKeys:
		switch action {
		case "read", "get", "list":
			requiredPermission = models.PermissionReadAPIKeys
		case "write", "create", "update", "post", "put", "patch", "delete":
			requiredPermission = models.PermissionWriteAPIKeys
		default:
			return false
		}
	case ResourceTypeAuditLogs:
		switch action {
		case "read", "get", "list":
			requiredPermission = models.PermissionReadAuditLogs
		default:
			return false
		}
	default:
		return false
	}

	return pc.HasPermission(userPermissions, requiredPermission)
}

// GetPermissionLevel returns the hierarchical level of a permission
func (pc *PermissionChecker) GetPermissionLevel(permission models.Permission) PermissionLevel {
	permStr := string(permission)
	
	if strings.HasPrefix(permStr, "read:") {
		return PermissionLevelRead
	}
	if strings.HasPrefix(permStr, "write:") {
		return PermissionLevelWrite
	}
	if strings.HasPrefix(permStr, "delete:") {
		return PermissionLevelDelete
	}
	
	return PermissionLevelRead // Default to lowest level
}

// IsHigherOrEqualLevel checks if permission1 is higher or equal level than permission2
func (pc *PermissionChecker) IsHigherOrEqualLevel(permission1, permission2 models.Permission) bool {
	level1 := pc.GetPermissionLevel(permission1)
	level2 := pc.GetPermissionLevel(permission2)
	return level1 >= level2
}

// GetResourceFromPermission extracts the resource type from a permission string
func (pc *PermissionChecker) GetResourceFromPermission(permission models.Permission) ResourceType {
	permStr := string(permission)
	parts := strings.Split(permStr, ":")
	if len(parts) != 2 {
		return ""
	}
	
	switch parts[1] {
	case "feature_flags":
		return ResourceTypeFeatureFlags
	case "users":
		return ResourceTypeUsers
	case "api_keys":
		return ResourceTypeAPIKeys
	case "audit_logs":
		return ResourceTypeAuditLogs
	default:
		return ResourceType(parts[1])
	}
}

// ValidatePermissions validates a slice of permissions
func (pc *PermissionChecker) ValidatePermissions(permissions []models.Permission) error {
	if len(permissions) == 0 {
		return fmt.Errorf("at least one permission is required")
	}

	for _, permission := range permissions {
		if !permission.Valid() {
			return fmt.Errorf("invalid permission: %s", permission)
		}
	}

	return nil
}

// GetPermissionsByResource groups permissions by resource type
func (pc *PermissionChecker) GetPermissionsByResource(permissions []models.Permission) map[ResourceType][]models.Permission {
	result := make(map[ResourceType][]models.Permission)
	
	for _, permission := range permissions {
		resource := pc.GetResourceFromPermission(permission)
		if resource != "" {
			result[resource] = append(result[resource], permission)
		}
	}
	
	return result
}

// MinimumPermissionsForRole returns minimum permissions required for each user role
func (pc *PermissionChecker) MinimumPermissionsForRole(role models.UserRole) []models.Permission {
	switch role {
	case models.RoleViewer:
		return []models.Permission{
			models.PermissionReadFeatureFlags,
		}
	case models.RoleEditor:
		return []models.Permission{
			models.PermissionReadFeatureFlags,
			models.PermissionWriteFeatureFlags,
		}
	case models.RoleAdmin:
		return []models.Permission{
			models.PermissionReadFeatureFlags,
			models.PermissionWriteFeatureFlags,
			models.PermissionDeleteFeatureFlags,
			models.PermissionReadUsers,
			models.PermissionWriteUsers,
			models.PermissionReadAPIKeys,
			models.PermissionWriteAPIKeys,
			models.PermissionReadAuditLogs,
		}
	default:
		return []models.Permission{}
	}
}

// CanAccessResource is a convenience method that combines resource and action checking
func (pc *PermissionChecker) CanAccessResource(permissions []models.Permission, resource string, action string) bool {
	resourceType := ResourceType(resource)
	return pc.CheckResourceAccess(permissions, resourceType, action)
}