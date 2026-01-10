// Package activitylogs provides unified activity logging for all entities.
//
// This package implements:
//   - Recording events for goals, tasks, and other entities
//   - Querying activity history by entity or user
//   - Supporting rich change data for frontend display
//
// Database Architecture:
//   - activity_logs: Main table for all activity events
package activitylogs

import (
	"time"
)

// =============================================================================
// DOMAIN MODELS
// =============================================================================

// ActivityLog represents a single activity event in the system.
//
// @Description Activity log entry for any entity
type ActivityLog struct {
	ID string `json:"id,omitempty"` // activity_logs:xxx

	// Entity information
	EntityType string `json:"entity_type"` // "goal", "task", "category", etc.
	EntityID   string `json:"entity_id"`   // The actual entity ID

	// Event information
	Event   string         `json:"event"`             // created, updated, deleted, completed, etc.
	Changes map[string]any `json:"changes,omitempty"` // What changed

	// Display helpers (populated for frontend)
	EntityTitle string `json:"entity_title,omitempty"` // Title of the entity for display
	EntityIcon  string `json:"entity_icon,omitempty"`  // Icon of the entity for display

	// Metadata
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// =============================================================================
// EVENT TYPES
// =============================================================================

const (
	// Common events
	EventCreated = "created"
	EventUpdated = "updated"
	EventDeleted = "deleted"

	// Goal-specific events
	EventCompleted     = "completed"
	EventArchived      = "archived"
	EventReactivated   = "reactivated"
	EventStreakUpdated = "streak_updated"
	EventTargetMet     = "target_met"
	EventChildAdded    = "child_added"
	EventChildRemoved  = "child_removed"

	// Task-specific events
	EventStarted  = "started"
	EventPaused   = "paused"
	EventResumed  = "resumed"
	EventFinished = "finished"
	EventLinked   = "linked"   // linked to goal
	EventUnlinked = "unlinked" // unlinked from goal
)

// =============================================================================
// ENTITY TYPES
// =============================================================================

const (
	EntityTypeGoal     = "goal"
	EntityTypeTask     = "task"
	EntityTypeCategory = "category"
	EntityTypeTemplate = "template"
)

// =============================================================================
// REQUEST TYPES
// =============================================================================

// LogEventRequest is used to create an activity log entry.
type LogEventRequest struct {
	EntityType  string
	EntityID    string
	EntityTitle string
	EntityIcon  string
	Event       string
	Changes     map[string]any
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// ActivityLogsResponse is the response for listing activity logs.
//
// @Description Activity log list response
type ActivityLogsResponse struct {
	Logs  []*ActivityLog `json:"logs"`
	Total int64          `json:"total"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name.
	Table = "activity_logs"
)
