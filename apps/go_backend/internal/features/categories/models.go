// Package categories provides category management functionality.
//
// Categories are used to organize tasks. Each user can have multiple
// categories, and each task can be linked to one category.
//
// Features:
//   - Unique category names per user
//   - Color customization
//   - Soft delete support
package categories

import "time"

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Category represents a task category.
type Category struct {
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Color     string     `json:"color"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy string     `json:"-"` // Hidden from API
	UpdatedBy string     `json:"-"` // Hidden from API
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a category.
//
// @Description Request payload for creating a category
type CreateRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=100" example:"Work"`
	Color string `json:"color" validate:"required,min=1,max=50" example:"#3B82F6"`
}

// UpdateRequest is the request payload for updating a category.
//
// @Description Request payload for updating a category
type UpdateRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1,max=100" example:"Personal"`
	Color *string `json:"color,omitempty" validate:"omitempty,min=1,max=50" example:"#10B981"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for categories.
	Table = "categories"
)

// CategoryPageResponse documents the paginated response for categories.
type CategoryPageResponse struct {
	Items   []*Category `json:"items"`
	Total   int64       `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"has_more"`
}
