// Package taskgoals provides task-goal relationship management.
//
// This package implements:
//   - Task-goal linking with impact tracking
//   - SurrealDB relation table operations (task_goals)
//   - Many-to-many relationship: one task can link to multiple goals
//   - Milestone tracking for significant task completions
//
// Database Architecture:
//
// Uses SurrealDB RELATE for creating edges:
//
//	RELATE tasks:abc -> task_goals -> goals:xyz SET {
//	    impact_type: "positive",
//	    quantity_value: 5.0,
//	    unit_id: units:km,
//	    is_milestone: true
//	}
//
// The task_goals table stores:
//   - impact_type: "positive", "negative", "neutral"
//   - quantity_value/unit_id: for measurable goals
//   - is_milestone: marks significant progress points
//   - notes: optional context
//   - source: "manual" or "auto" (from template)
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
//
// @Description Task-goal relationship edge with impact data
type TaskGoal struct {
	ID string `json:"id,omitempty"` // Edge ID: "task_goals:xyz"

	// Related entities (populated as record IDs)
	TaskID string `json:"task_id"` // "tasks:abc"
	GoalID string `json:"goal_id"` // "goals:xyz"

	// Impact tracking
	ImpactType string `json:"impact_type"` // "positive", "negative", "neutral"

	// Quantity (for measurable goals)
	QuantityValue *float64 `json:"quantity_value,omitempty"` // e.g., 5.0
	UnitID        *string  `json:"unit_id,omitempty"`        // e.g., "units:km"

	// Milestone tracking (replaces goal_actions)
	IsMilestone    bool   `json:"is_milestone,omitempty"`
	MilestoneLabel string `json:"milestone_label,omitempty"` // e.g., "Module 3 Complete"
	MilestoneOrder int    `json:"milestone_order,omitempty"` // Display order

	// Context
	Notes  string `json:"notes,omitempty"`
	Source string `json:"source"` // "manual" or "auto"

	// Metadata
	CreatedAt time.Time `json:"created_at"`
}

// TaskGoalWithGoal includes the full goal details for task→goals queries.
//
// @Description Task-goal link with full goal details
type TaskGoalWithGoal struct {
	TaskGoal
	Goal *goals.Goal `json:"goal,omitempty"`
}

// TaskGoalWithTask includes the full task details for goal→tasks queries.
//
// @Description Task-goal link with full task details
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
	GoalID        string   `json:"goal_id" validate:"required" example:"goals:abc123"`
	ImpactType    string   `json:"impact_type" validate:"required,oneof=positive negative neutral" example:"positive"`
	QuantityValue *float64 `json:"quantity_value,omitempty" example:"5.0"`
	UnitID        *string  `json:"unit_id,omitempty" example:"units:km"`

	// Milestone fields (for significant completions)
	IsMilestone    bool   `json:"is_milestone,omitempty" example:"false"`
	MilestoneLabel string `json:"milestone_label,omitempty" validate:"max=200" example:"Design Phase Complete"`
	MilestoneOrder int    `json:"milestone_order,omitempty" example:"1"`

	Notes string `json:"notes,omitempty" validate:"max=1000"`
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
	ImpactType     *string  `json:"impact_type,omitempty" validate:"omitempty,oneof=positive negative neutral" example:"positive"`
	QuantityValue  *float64 `json:"quantity_value,omitempty" example:"8.0"`
	UnitID         *string  `json:"unit_id,omitempty" example:"units:km"`
	IsMilestone    *bool    `json:"is_milestone,omitempty" example:"true"`
	MilestoneLabel *string  `json:"milestone_label,omitempty" validate:"omitempty,max=200" example:"Updated milestone"`
	MilestoneOrder *int     `json:"milestone_order,omitempty" example:"2"`
	Notes          *string  `json:"notes,omitempty" validate:"omitempty,max=1000" example:"Great progress on this workout"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// GoalsForTaskResponse is the response for GET /tasks/:taskId/goals.
//
// @Description List of goals linked to a task
type GoalsForTaskResponse struct {
	TaskID string              `json:"task_id"`
	Links  []*TaskGoalWithGoal `json:"links"`
	Count  int                 `json:"count"`
}

// TasksForGoalResponse is the response for GET /goals/:goalId/tasks.
//
// @Description List of tasks linked to a goal
type TasksForGoalResponse struct {
	GoalID string              `json:"goal_id"`
	Links  []*TaskGoalWithTask `json:"links"`
	Count  int                 `json:"count"`
}

// MilestonesResponse is the response for GET /goals/:goalId/milestones.
//
// @Description List of milestones for a goal
type MilestonesResponse struct {
	GoalID     string              `json:"goal_id"`
	Milestones []*TaskGoalWithTask `json:"milestones"`
	Completed  int                 `json:"completed"`
	Total      int                 `json:"total"`
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
