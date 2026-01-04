// Package templates provides task template management functionality.
//
// This package implements:
//   - CRUD operations for task templates
//   - Quick-log template management
//   - Template instantiation (creating tasks from templates)
//   - Template-goal linking via template_goals relation
//
// Templates are reusable blueprints for creating tasks quickly, especially
// useful for habits where users log the same activity repeatedly.
//
// Database Architecture:
//   - template_goals: RELATE table linking templates to goals
//   - in_category: RELATE table for category assignment
//   - created_from: RELATE table tracking task origin (on tasks table)
package templates

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/features/goals"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// TaskTemplate represents a reusable task blueprint.
//
// Templates are linked to goals via the template_goals relation.
// When a task is created from a template with auto_link_tasks=true,
// the task automatically gets linked to the goal via task_goals.
//
// @Description Reusable task blueprint
type TaskTemplate struct {
	ID        string `json:"id,omitempty"`
	CreatedBy string `json:"-"`

	// Core fields
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"` // Emoji

	// Defaults for tasks created from this template
	DefaultDuration int `json:"default_duration,omitempty"` // seconds

	// Quick log settings
	IsQuickLog    bool `json:"is_quick_log"`
	QuickLogOrder int  `json:"quick_log_order,omitempty"`

	// Quantity settings
	QuantityEnabled bool    `json:"quantity_enabled"`
	QuantityDefault float64 `json:"quantity_default,omitempty"`
	QuantityStep    float64 `json:"quantity_step,omitempty"`
	// Note: quantity unit is inherited from linked goal's target.unit_id

	// Emotion defaults
	ExpectedQuadrant string `json:"expected_quadrant,omitempty"` // green, yellow, red, blue
	DefaultEmotionID string `json:"default_emotion_id,omitempty"`

	// Usage stats (auto-updated)
	UseCount   int        `json:"use_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// Populated via graph queries (not stored on template)
	Goals    []*goals.Goal        `json:"goals,omitempty"`    // From template_goals edge
	Category *categories.Category `json:"category,omitempty"` // From in_category edge (or inherited)
}

// TemplateGoalLink represents a link between a template and a goal.
//
// @Description Link metadata for template-goal relationship
type TemplateGoalLink struct {
	GoalID             string  `json:"goal_id"`
	AutoLinkTasks      bool    `json:"auto_link_tasks"`     // Auto-link tasks to goal
	QuantityMultiplier float64 `json:"quantity_multiplier"` // Multiply quantity when linking
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

	DefaultDuration int `json:"default_duration,omitempty" example:"1800"` // seconds (30 min)

	IsQuickLog    bool `json:"is_quick_log,omitempty" example:"true"`
	QuickLogOrder int  `json:"quick_log_order,omitempty" example:"1"`

	QuantityEnabled bool    `json:"quantity_enabled,omitempty" example:"true"`
	QuantityDefault float64 `json:"quantity_default,omitempty" validate:"min=0" example:"5.0"`
	QuantityStep    float64 `json:"quantity_step,omitempty" validate:"min=0" example:"0.5"`

	ExpectedQuadrant string `json:"expected_quadrant,omitempty" validate:"omitempty,oneof=green yellow red blue" example:"yellow"`
	DefaultEmotionID string `json:"default_emotion_id,omitempty" example:"emotions:E16"`

	// Category assignment (creates in_category edge)
	CategoryID string `json:"category_id,omitempty" example:"categories:health123"`

	// Goal linking (creates template_goals edge)
	GoalLinks []GoalLinkInput `json:"goal_links,omitempty"`
}

// GoalLinkInput is the input for linking a template to a goal.
type GoalLinkInput struct {
	GoalID             string  `json:"goal_id" validate:"required" example:"goals:hydration123"`
	AutoLinkTasks      bool    `json:"auto_link_tasks,omitempty" example:"true"`
	QuantityMultiplier float64 `json:"quantity_multiplier,omitempty" validate:"min=0" example:"1.0"`
}

// UpdateRequest is the request payload for updating a template.
//
// @Description Request payload for updating a template
type UpdateRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=500" example:"Evening Run"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000" example:"Post-work stress relief run"`
	Icon        *string `json:"icon,omitempty" validate:"omitempty,max=50" example:"🌙"`

	DefaultDuration *int `json:"default_duration,omitempty" example:"2700"`

	IsQuickLog    *bool `json:"is_quick_log,omitempty" example:"true"`
	QuickLogOrder *int  `json:"quick_log_order,omitempty" example:"2"`

	QuantityEnabled *bool    `json:"quantity_enabled,omitempty" example:"true"`
	QuantityDefault *float64 `json:"quantity_default,omitempty" validate:"omitempty,min=0" example:"7.0"`
	QuantityStep    *float64 `json:"quantity_step,omitempty" validate:"omitempty,min=0" example:"1.0"`

	ExpectedQuadrant *string `json:"expected_quadrant,omitempty" validate:"omitempty,oneof=green yellow red blue" example:"green"`
	DefaultEmotionID *string `json:"default_emotion_id,omitempty" example:"emotions:E25"`
}

// LinkGoalRequest is the request for linking a template to a goal.
//
// @Description Request for linking template to goal
type LinkGoalRequest struct {
	GoalID             string  `json:"goal_id" validate:"required" example:"goals:hydration123"`
	AutoLinkTasks      bool    `json:"auto_link_tasks,omitempty" example:"true"`
	QuantityMultiplier float64 `json:"quantity_multiplier,omitempty" validate:"min=0" example:"1.0"`
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
//
// @Description Paginated list of templates
type TemplatePageResponse struct {
	Items   []*TaskTemplate `json:"items"`
	Total   int64           `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
	HasMore bool            `json:"has_more"`
}

// QuickLogTemplate represents a template configured for quick logging.
//
// @Description Quick log template with goal info
type QuickLogTemplate struct {
	Template *TaskTemplate `json:"template"`
	GoalID   string        `json:"goal_id,omitempty"`
	GoalIcon string        `json:"goal_icon,omitempty"`
	UnitID   string        `json:"unit_id,omitempty"` // From linked goal's target
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for templates.
	Table = "templates"

	// TemplateGoalsTable is the SurrealDB relation table name.
	TemplateGoalsTable = "template_goals"
)
