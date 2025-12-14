// Package goalactions provides goal action (subtask) management functionality.
//
// This package implements:
//   - CRUD operations for goal actions/subtasks
//   - Marking actions complete (triggers goal auto-completion check)
//   - Reordering actions within a goal
//
// GoalActions serve two purposes:
//   - For discrete goals: subtasks to complete the goal
//   - For recurring goals: templates for quick logging activities
//
// When all actions of a discrete goal are completed, the goal
// auto-completes based on its completion_mode setting.
package goalactions

import (
	"time"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// GoalAction represents a subtask or action item within a goal.
//
// For discrete goals: these are the steps to complete the goal.
// For recurring goals: these define activities that count toward the goal.
type GoalAction struct {
	ID        string `json:"id,omitempty"` // "goal_actions:abc123"
	GoalID    string `json:"goal_id"`      // Parent goal ID
	CreatedBy string `json:"-"`            // User ownership

	// Action definition
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order"` // Display order

	// Quantity (for measurable contributions)
	QuantityValue *float64 `json:"quantity_value,omitempty"` // e.g., 5.0
	QuantityUnit  *string  `json:"quantity_unit,omitempty"`  // e.g., "km"

	// Completion status (for discrete goals only)
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new goal action.
//
// @Description Request payload for creating a goal action
type CreateRequest struct {
	Title         string   `json:"title" validate:"required,min=1,max=500" example:"Complete Python Basics course"`
	Description   string   `json:"description,omitempty" validate:"max=2000" example:"Finish all lessons in Module 1 including exercises"`
	Order         *int     `json:"order,omitempty" example:"1"`
	QuantityValue *float64 `json:"quantity_value,omitempty" example:"5.0"`
	QuantityUnit  *string  `json:"quantity_unit,omitempty" validate:"omitempty,max=50" example:"km"`
}

// UpdateRequest is the request payload for updating a goal action.
//
// All fields are optional. Only provided fields will be updated.
// Setting Completed to true/false handles marking complete/incomplete.
//
// @Description Request payload for updating a goal action
type UpdateRequest struct {
	Title         *string  `json:"title,omitempty" validate:"omitempty,min=1,max=500" example:"Complete Advanced Python"`
	Description   *string  `json:"description,omitempty" validate:"omitempty,max=2000" example:"Updated description"`
	Order         *int     `json:"order,omitempty" example:"2"`
	QuantityValue *float64 `json:"quantity_value,omitempty" example:"10.0"`
	QuantityUnit  *string  `json:"quantity_unit,omitempty" validate:"omitempty,max=50" example:"pages"`
	Completed     *bool    `json:"completed,omitempty" example:"true"` // Set completion status
}

// ReorderRequest is for reordering actions within a goal.
//
// @Description Request payload for reordering goal actions
type ReorderRequest struct {
	ActionIDs []string `json:"action_ids" validate:"required,min=1"` // Ordered list of action IDs
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// ActionListResponse is the response for GET /goals/:id/actions.
type ActionListResponse struct {
	GoalID  string        `json:"goal_id"`
	Actions []*GoalAction `json:"actions"`
	Count   int           `json:"count"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for goal actions.
	Table = "goal_actions"
)
