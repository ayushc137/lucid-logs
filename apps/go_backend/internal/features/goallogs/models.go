// Package goallogs provides goal history tracking via the goal_logs relation.
//
// This package implements:
//   - Recording all significant goal events (created, completed, streak updates)
//   - Point-in-time snapshots of goal state
//   - Querying goal history for analytics and debugging
//
// Database Architecture:
//   - goal_logs: RELATE table connecting goals to goal_snapshots
//   - goal_snapshots: Table storing point-in-time goal state
package goallogs

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/goals"
)

// =============================================================================
// DOMAIN MODELS
// =============================================================================

// GoalLog represents a single event in a goal's history.
//
// @Description Goal history event record
type GoalLog struct {
	ID        string `json:"id,omitempty"` // goal_logs:xxx
	GoalID    string `json:"goal_id"`      // goals:xxx
	CreatedBy string `json:"-"`

	// Event type
	Event string `json:"event"` // created, updated, completed, archived, etc.

	// What changed (optional, depends on event type)
	Changes map[string]any `json:"changes,omitempty"`

	// What triggered this event (optional)
	TriggeredByTaskID string `json:"triggered_by_task_id,omitempty"` // tasks:xxx

	// Snapshot reference
	SnapshotID string `json:"snapshot_id,omitempty"` // goal_snapshots:xxx

	// Metadata
	CreatedAt time.Time `json:"created_at"`
}

// GoalSnapshot represents a point-in-time snapshot of goal state.
//
// @Description Point-in-time goal state snapshot
type GoalSnapshot struct {
	ID     string `json:"id,omitempty"` // goal_snapshots:xxx
	GoalID string `json:"goal_id"`      // goals:xxx

	// Snapshot of goal state at this point
	Status string           `json:"status"`
	Stats  *goals.GoalStats `json:"stats,omitempty"`
	Target *goals.Target    `json:"target,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
}

// =============================================================================
// EVENT TYPES
// =============================================================================

const (
	// Lifecycle events
	EventCreated     = "created"     // Goal was created
	EventUpdated     = "updated"     // Goal properties changed
	EventCompleted   = "completed"   // Status changed to completed
	EventArchived    = "archived"    // Status changed to archived
	EventReactivated = "reactivated" // Status changed back to active

	// Progress events
	EventStreakUpdated  = "streak_updated"  // Streak value changed
	EventTargetMet      = "target_met"      // Current value reached target
	EventTargetExceeded = "target_exceeded" // Avoidance goal exceeded limit
	EventPeriodReset    = "period_reset"    // Recurring goal period ended, value reset

	// Structure events
	EventChildAdded   = "child_added"   // Child goal added to group
	EventChildRemoved = "child_removed" // Child goal removed from group
)

// ValidEvents for validation.
var ValidEvents = []string{
	EventCreated, EventUpdated, EventCompleted, EventArchived, EventReactivated,
	EventStreakUpdated, EventTargetMet, EventTargetExceeded, EventPeriodReset,
	EventChildAdded, EventChildRemoved,
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// LogEventRequest is used internally to create a goal log entry.
type LogEventRequest struct {
	GoalID            string
	Event             string
	Changes           map[string]any
	TriggeredByTaskID string
	Stats             *goals.GoalStats // Optional: creates a snapshot if provided
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// GoalLogsResponse is the response for GET /goals/:id/logs.
//
// @Description Goal history log entries
type GoalLogsResponse struct {
	GoalID string     `json:"goal_id"`
	Logs   []*GoalLog `json:"logs"`
	Total  int        `json:"total"`
}

// GoalLogsSummary provides aggregated history data.
//
// @Description Summary of goal history
type GoalLogsSummary struct {
	DaysMet    int     `json:"days_met"`
	DaysMissed int     `json:"days_missed"`
	AvgDaily   float64 `json:"avg_daily,omitempty"`
	BestDay    *struct {
		Date  string  `json:"date"`
		Value float64 `json:"value"`
	} `json:"best_day,omitempty"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// LogsTable is the SurrealDB relation table name.
	LogsTable = "goal_logs"

	// SnapshotsTable is the SurrealDB table for snapshots.
	SnapshotsTable = "goal_snapshots"
)
