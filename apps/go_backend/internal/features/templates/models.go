// Package templates provides task template management functionality.
//
// This package implements:
//   - CRUD operations for task templates
//   - Quick-log template management
//   - Template instantiation (creating tasks from templates)
//   - Activity key inheritance for goal linking
//
// Templates are reusable blueprints for creating tasks quickly, especially
// useful for habits where users log the same activity repeatedly.
package templates

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// TaskTemplate represents a reusable task blueprint.
//
// Templates can be:
// - Auto-created from goals (for quick logging)
// - User-created for any recurring task
// - System-provided defaults (is_default = true)
type TaskTemplate struct {
	ID        string `json:"id,omitempty"`
	CreatedBy string `json:"-"`

	// Core
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`

	// Defaults for tasks created from this template
	DefaultDuration int                  `json:"default_duration,omitempty"` // seconds
	DefaultPriority int                  `json:"default_priority,omitempty"`
	DefaultCategory *categories.Category `json:"default_category,omitempty"`

	// Quick log settings
	IsQuickLog    bool `json:"is_quick_log"`
	QuickLogOrder int  `json:"quick_log_order,omitempty"`

	// Quantity settings
	QuantityEnabled bool    `json:"quantity_enabled"`
	QuantityDefault float64 `json:"quantity_default,omitempty"`
	QuantityUnit    string  `json:"quantity_unit,omitempty"`
	QuantityStep    float64 `json:"quantity_step,omitempty"`

	// Emotion defaults
	ExpectedQuadrant string `json:"expected_quadrant,omitempty"` // green, yellow, red, blue
	DefaultEmotionID string `json:"default_emotion_id,omitempty"`

	// Goal/Activity linking
	ActivityKey string `json:"activity_key,omitempty"` // For auto-linking to goals
	GoalID      string `json:"goal_id,omitempty"`      // Source goal if auto-created

	// Fields to show in quick-log UI
	ShowFields *ShowFields `json:"show_fields,omitempty"`

	// Source
	IsDefault    bool   `json:"is_default"` // System-provided template
	SourceTaskID string `json:"source_task_id,omitempty"`

	// Usage stats
	UseCount   int        `json:"use_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ShowFields controls which fields are displayed in the quick-log UI.
type ShowFields struct {
	Journal            bool `json:"journal"`
	Duration           bool `json:"duration"`
	Quantity           bool `json:"quantity"`
	Emotion            bool `json:"emotion"`
	PositivesNegatives bool `json:"positives_negatives"`
	Notes              bool `json:"notes"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new template.
//
// @Description Request payload for creating a template
type CreateRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=500" example:"Morning Run"`
	Description string `json:"description,omitempty" validate:"max=2000" example:"Quick morning jog before work"`
	Icon        string `json:"icon,omitempty" validate:"max=50" example:"🏃"`
	Color       string `json:"color,omitempty" validate:"max=20" example:"#10B981"`

	DefaultDuration   int    `json:"default_duration,omitempty" example:"1800"` // seconds (30 min)
	DefaultPriority   int    `json:"default_priority,omitempty" validate:"min=0,max=3" example:"2"`
	DefaultCategoryID string `json:"default_category_id,omitempty" example:"categories:fitness123"`

	IsQuickLog    bool `json:"is_quick_log,omitempty" example:"true"`
	QuickLogOrder int  `json:"quick_log_order,omitempty" example:"1"`

	QuantityEnabled bool    `json:"quantity_enabled,omitempty" example:"true"`
	QuantityDefault float64 `json:"quantity_default,omitempty" validate:"min=0" example:"5.0"`
	QuantityUnit    string  `json:"quantity_unit,omitempty" validate:"max=50" example:"km"`
	QuantityStep    float64 `json:"quantity_step,omitempty" validate:"min=0" example:"0.5"`

	ExpectedQuadrant string `json:"expected_quadrant,omitempty" validate:"omitempty,oneof=green yellow red blue" example:"yellow"`
	DefaultEmotionID string `json:"default_emotion_id,omitempty" example:"emotions:E16"`

	ActivityKey string `json:"activity_key,omitempty" example:"running"`
	GoalID      string `json:"goal_id,omitempty" example:"goals:run100km"`

	ShowFields *ShowFieldsInput `json:"show_fields,omitempty"`
}

// ShowFieldsInput is the input format for show_fields.
type ShowFieldsInput struct {
	Journal            *bool `json:"journal,omitempty" example:"false"`
	Duration           *bool `json:"duration,omitempty" example:"true"`
	Quantity           *bool `json:"quantity,omitempty" example:"true"`
	Emotion            *bool `json:"emotion,omitempty" example:"true"`
	PositivesNegatives *bool `json:"positives_negatives,omitempty" example:"false"`
	Notes              *bool `json:"notes,omitempty" example:"true"`
}

// UpdateRequest is the request payload for updating a template.
//
// @Description Request payload for updating a template
type UpdateRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=500" example:"Evening Run"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000" example:"Post-work stress relief run"`
	Icon        *string `json:"icon,omitempty" validate:"omitempty,max=50" example:"🌙"`
	Color       *string `json:"color,omitempty" validate:"omitempty,max=20" example:"#6366F1"`

	DefaultDuration   *int    `json:"default_duration,omitempty" example:"2700"`
	DefaultPriority   *int    `json:"default_priority,omitempty" validate:"omitempty,min=0,max=3" example:"2"`
	DefaultCategoryID *string `json:"default_category_id,omitempty" example:"categories:fitness123"`

	IsQuickLog    *bool `json:"is_quick_log,omitempty" example:"true"`
	QuickLogOrder *int  `json:"quick_log_order,omitempty" example:"2"`

	QuantityEnabled *bool    `json:"quantity_enabled,omitempty" example:"true"`
	QuantityDefault *float64 `json:"quantity_default,omitempty" validate:"omitempty,min=0" example:"7.0"`
	QuantityUnit    *string  `json:"quantity_unit,omitempty" validate:"omitempty,max=50" example:"km"`
	QuantityStep    *float64 `json:"quantity_step,omitempty" validate:"omitempty,min=0" example:"1.0"`

	ExpectedQuadrant *string `json:"expected_quadrant,omitempty" validate:"omitempty,oneof=green yellow red blue" example:"green"`
	DefaultEmotionID *string `json:"default_emotion_id,omitempty" example:"emotions:E25"`

	ShowFields *ShowFieldsInput `json:"show_fields,omitempty"`
}

// InstantiateRequest is the request for creating a task from a template.
//
// @Description Request payload for creating a task from a template
type InstantiateRequest struct {
	StartDate string   `json:"start_date" validate:"required" example:"2025-12-14T07:00:00Z"`
	EndDate   string   `json:"end_date,omitempty" example:"2025-12-14T07:30:00Z"`
	Quantity  *float64 `json:"quantity,omitempty" example:"5.5"`
	Notes     string   `json:"notes,omitempty" example:"Felt great today, pushed a bit harder"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// TemplatePageResponse documents the paginated response returned by the List endpoint.
type TemplatePageResponse struct {
	Items   []*TaskTemplate `json:"items"`
	Total   int64           `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
	HasMore bool            `json:"has_more"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for templates.
	Table = "templates"
)
