package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	Role      UserRole  `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Email string   `json:"email" binding:"required,email"`
	Name  string   `json:"name" binding:"required"`
	Role  UserRole `json:"role" binding:"required"`
}

// Validate validates the create user request
func (cur *CreateUserRequest) Validate() error {
	if !cur.Role.Valid() {
		return NewValidationError("role", "invalid role")
	}
	return nil
}

// UpdateUserRequest represents a request to update an existing user
type UpdateUserRequest struct {
	Name *string   `json:"name,omitempty"`
	Role *UserRole `json:"role,omitempty"`
}

// Validate validates the update user request
func (uur *UpdateUserRequest) Validate() error {
	if uur.Role != nil && !uur.Role.Valid() {
		return NewValidationError("role", "invalid role")
	}
	return nil
}

// HasChanges returns true if the update request has any changes
func (uur *UpdateUserRequest) HasChanges() bool {
	return uur.Name != nil || uur.Role != nil
}

// UserFilter represents filters for user queries
type UserFilter struct {
	Email string   `json:"email,omitempty" form:"email"`
	Role  UserRole `json:"role,omitempty" form:"role"`
	Name  string   `json:"name,omitempty" form:"name"`
}

// UserResponse represents a user in API responses (excludes sensitive info)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts a User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}