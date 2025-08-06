package models

import (
	"time"
)

type Feature struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	Rules       map[string]interface{} `json:"rules" db:"rules"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

type CreateFeatureRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Rules       map[string]interface{} `json:"rules"`
}

type UpdateFeatureRequest struct {
	Description *string                `json:"description"`
	Enabled     *bool                  `json:"enabled"`
	Rules       map[string]interface{} `json:"rules"`
}