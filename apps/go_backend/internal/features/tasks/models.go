// Package tasks provides task management functionality.
//
// This package implements:
//   - CRUD operations for tasks
//   - Category linking (record links)
//   - Soft delete support
//   - Pagination
//
// Database Architecture:
//
// Tasks use SurrealDB's schemaless tables with record links:
//   - task.category = categories:abc123 (record link)
//   - FETCH hydrates the linked category automatically
//   - Permissions use $auth.id for ownership
package tasks

import (
	"time"

	"github.com/daily-journal/go-backend/internal/features/categories"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Task represents a task/journal entry in the system.
//
// Fields marked with json:"-" are hidden from API responses.
// System-managed fields (created_at, updated_at, deleted_at) are read-only.
type Task struct {
	ID        string               `json:"id,omitempty"`
	Title     string               `json:"title"`
	Journal   string               `json:"journal"`
	StartDate time.Time            `json:"start_date"`
	EndDate   time.Time            `json:"end_date"`
	Completed bool                 `json:"completed"`
	Priority  int                  `json:"priority"`
	Source    string               `json:"source"`
	Note      string               `json:"note"`
	Positives []string             `json:"positives"`
	Negatives []string             `json:"negatives"`
	Category  *categories.Category `json:"category,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	DeletedAt *time.Time           `json:"deleted_at,omitempty"`
	CreatedBy string               `json:"-"` // Hidden: ownership field
	UpdatedBy string               `json:"-"` // Hidden: audit field
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new task.
//
// Required fields: title, start_date, end_date
// Optional fields: journal, priority, source, note, positives, negatives, category_id
//
// @Description Request payload for creating a task
type CreateRequest struct {
	Title      string   `json:"title" validate:"required,min=1,max=500" example:"Plan tomorrow"`
	Journal    string   `json:"journal" validate:"max=10000" example:"Capture high-level goals"`
	StartDate  string   `json:"start_date" validate:"required,datetime_flexible" example:"2025-11-24T09:00:00Z"`
	EndDate    string   `json:"end_date" validate:"required,datetime_flexible" example:"2025-11-25T17:00:00Z"`
	Priority   int      `json:"priority" validate:"min=-100,max=100" example:"1"`
	Source     string   `json:"source,omitempty" example:"manual"`
	Note       string   `json:"note,omitempty" validate:"max=5000" example:"Focus on top priorities"`
	Positives  []string `json:"positives,omitempty" example:"Great progress,In flow"`
	Negatives  []string `json:"negatives,omitempty" example:"Some distractions"`
	CategoryID string   `json:"category_id,omitempty" example:"categories:work123"`
}

// UpdateRequest is the request payload for updating a task.
//
// All fields are optional. Only provided fields will be updated.
// Use null or empty string for category_id to remove the category link.
//
// @Description Request payload for updating a task
type UpdateRequest struct {
	Title      *string  `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Journal    *string  `json:"journal,omitempty" validate:"omitempty,max=10000"`
	StartDate  *string  `json:"start_date,omitempty" validate:"omitempty,datetime_flexible"`
	EndDate    *string  `json:"end_date,omitempty" validate:"omitempty,datetime_flexible"`
	Completed  *bool    `json:"completed,omitempty"`
	Priority   *int     `json:"priority,omitempty" validate:"omitempty,min=-100,max=100"`
	Note       *string  `json:"note,omitempty" validate:"omitempty,max=5000"`
	Positives  []string `json:"positives,omitempty"`
	Negatives  []string `json:"negatives,omitempty"`
	CategoryID *string  `json:"category_id,omitempty"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for tasks.
	Table = "tasks"

	// SourceManual is the default source for manually created tasks.
	SourceManual = "manual"
)
