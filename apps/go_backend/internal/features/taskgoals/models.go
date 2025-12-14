// Package taskgoals provides task-goal relationship management.
//
// This package implements:
//   - Task-goal linking with impact tracking
//   - SurrealDB relation table operations (task_goals)
//   - Many-to-many relationship: one task can link to multiple goals
//
// Database Architecture:
//
// Uses SurrealDB RELATE for creating edges:
//
//	RELATE tasks:abc -> goals:xyz SET impact_type = "positive"
//
// The task_goals table stores:
//   - impact_type: "positive", "negative", "neutral"
//   - impact_magnitude: 1-5 (how much impact)
//   - quantity_value/unit: for measurable goals
//   - notes: optional context
//   - source: "manual" or "auto" (activity_key match)
package taskgoals

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/features/tasks"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// TaskGoal represents a link between a task and a goal with impact metadata.
//
// This is the edge data stored in the task_goals relation table.
// Each link tracks how the task impacts the goal (positive/negative) and
// optionally stores quantity for measurable goal progress.
type TaskGoal struct {
	ID string `json:"id,omitempty"` // Edge ID: "task_goals:xyz"

	// Related entities
	TaskID string `json:"task_id"` // "tasks:abc"
	GoalID string `json:"goal_id"` // "goals:xyz"

	// Impact tracking
	ImpactType      string `json:"impact_type"`      // "positive", "negative", "neutral"
	ImpactMagnitude int    `json:"impact_magnitude"` // 1-5

	// Quantity (for measurable goals)
	QuantityValue *float64 `json:"quantity_value,omitempty"` // e.g., 5.0
	QuantityUnit  *string  `json:"quantity_unit,omitempty"`  // e.g., "km"

	// Context
	Notes  string `json:"notes,omitempty"`
	Source string `json:"source"` // "manual" or "auto"

	// Metadata
	CreatedAt time.Time `json:"created_at"`
}

// TaskGoalWithGoal includes the full goal details for task→goals queries.
type TaskGoalWithGoal struct {
	TaskGoal
	Goal *goals.Goal `json:"goal,omitempty"`
}

// TaskGoalWithTask includes the full task details for goal→tasks queries.
type TaskGoalWithTask struct {
	TaskGoal
	Task *tasks.Task `json:"task,omitempty"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// LinkRequest is the request payload for linking a task to a goal.
//
// @Description Request payload for creating a task-goal link
type LinkRequest struct {
	GoalID          string   `json:"goal_id" validate:"required" example:"goals:abc123"`
	ImpactType      string   `json:"impact_type" validate:"required,oneof=positive negative neutral" example:"positive"`
	ImpactMagnitude int      `json:"impact_magnitude" validate:"min=1,max=5" example:"3"`
	QuantityValue   *float64 `json:"quantity_value,omitempty" example:"5.0"`
	QuantityUnit    *string  `json:"quantity_unit,omitempty" example:"km"`
	Notes           string   `json:"notes,omitempty" validate:"max=1000"`
}

// BatchLinkRequest allows linking a task to multiple goals at once.
//
// @Description Request payload for linking a task to multiple goals
type BatchLinkRequest struct {
	Links []LinkRequest `json:"links" validate:"required,min=1,dive"`
}

// UpdateLinkRequest is the request payload for updating a task-goal link.
//
// All fields are optional. Only provided fields will be updated.
//
// @Description Request payload for updating a task-goal link
type UpdateLinkRequest struct {
	ImpactType      *string  `json:"impact_type,omitempty" validate:"omitempty,oneof=positive negative neutral" example:"positive"`
	ImpactMagnitude *int     `json:"impact_magnitude,omitempty" validate:"omitempty,min=1,max=5" example:"4"`
	QuantityValue   *float64 `json:"quantity_value,omitempty" example:"8.0"`
	QuantityUnit    *string  `json:"quantity_unit,omitempty" example:"km"`
	Notes           *string  `json:"notes,omitempty" validate:"omitempty,max=1000" example:"Great progress on this workout"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// GoalsForTaskResponse is the response for GET /tasks/:taskId/goals.
type GoalsForTaskResponse struct {
	TaskID string              `json:"task_id"`
	Links  []*TaskGoalWithGoal `json:"links"`
	Count  int                 `json:"count"`
}

// TasksForGoalResponse is the response for GET /goals/:goalId/tasks.
type TasksForGoalResponse struct {
	GoalID string              `json:"goal_id"`
	Links  []*TaskGoalWithTask `json:"links"`
	Count  int                 `json:"count"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB relation table name.
	Table = "task_goals"

	// Impact types
	ImpactPositive = "positive"
	ImpactNegative = "negative"
	ImpactNeutral  = "neutral"

	// Source types
	SourceManual = "manual"
	SourceAuto   = "auto"
)

// ValidImpactTypes for validation.
var ValidImpactTypes = []string{ImpactPositive, ImpactNegative, ImpactNeutral}
