// Package activities provides activity management for quick task logging.
//
// Activities are reusable task blueprints that support three modes:
//   - Instant Log: One-tap task creation with defaults (completed immediately)
//   - Schedule: Pre-fill task form with defaults for manual editing
//   - Timer: Flowmodoro-style time tracking with break management
//
// Database Architecture:
//   - activity_goals: RELATE table linking activities to multiple goals
//   - in_category: RELATE table for category assignment
//   - created_from_activity: RELATE table tracking task origin
//   - timer_sessions: Table for active Flowmodoro sessions
package activities

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/features/units"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Activity represents a reusable task blueprint.
//
// Activities can be linked to multiple goals. When an activity is used,
// tasks are automatically linked to those goals based on the activity_goals config.
//
// @Description Reusable task blueprint for quick logging
type Activity struct {
	ID        string `json:"id,omitempty"`
	CreatedBy string `json:"-"`

	// Identity
	Title       string `json:"title"`
	Icon        string `json:"icon,omitempty"`        // Emoji
	Description string `json:"description,omitempty"` // Template for task journal

	// Task Defaults (pre-fill when creating task)
	DefaultDuration  int    `json:"default_duration,omitempty"`   // seconds
	DefaultEmotionID string `json:"default_emotion_id,omitempty"` // emotions:xxx
	DefaultPriority  int    `json:"default_priority"`             // 1-5
	DefaultCompleted bool   `json:"default_completed"`            // true = instant log creates completed task

	// Quantity Settings
	QuantityEnabled bool    `json:"quantity_enabled"`
	QuantityDefault float64 `json:"quantity_default,omitempty"` // Deprecated: use goal-link default_quantity
	QuantityStep    float64 `json:"quantity_step,omitempty"`
	QuantityUnitID  string  `json:"quantity_unit_id,omitempty"` // Deprecated: unit comes from goal's target

	// Goal Link Defaults
	DefaultImpact string `json:"default_impact"` // positive, negative, neutral

	// Display & Organization
	Pinned    bool `json:"pinned"`     // Show in quick access bar
	SortOrder int  `json:"sort_order"` // Order in quick bar

	// Usage Statistics
	UseCount   int        `json:"use_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Active Timer Session (if any)
	ActiveSession *TimerSession `json:"active_session,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// Populated via graph queries (not stored on activity)
	Goals    []ActivityGoalLink   `json:"goals,omitempty"`    // From activity_goals edge
	Category *categories.Category `json:"category,omitempty"` // From in_category edge
}

// ActivityGoalLink represents a link between an activity and a goal.
//
// @Description Link configuration for activity-goal relationship
type ActivityGoalLink struct {
	GoalID             string   `json:"goal_id"`
	AutoLinkTasks      bool     `json:"auto_link_tasks"`            // Auto-link tasks to this goal
	QuantityMultiplier float64  `json:"quantity_multiplier"`        // Multiply quantity when linking
	DefaultQuantity    *float64 `json:"default_quantity,omitempty"` // Default quantity for this goal (unit from goal's target)
	DefaultImpact      string   `json:"default_impact"`             // Override impact type

	// Populated goal details (for display)
	Goal *goals.Goal `json:"goal,omitempty"`
}

// TimerSession tracks an active Flowmodoro session.
//
// @Description Active Flowmodoro timer session
type TimerSession struct {
	TaskID    string       `json:"task_id"`
	StartedAt time.Time    `json:"started_at"`
	Breaks    []TimerBreak `json:"breaks,omitempty"`
}

// TimerBreak records a break during Flowmodoro.
//
// @Description Break period during timer session
type TimerBreak struct {
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Duration  int        `json:"duration,omitempty"` // seconds
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new activity.
//
// @Description Request payload for creating an activity
type CreateRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=500" example:"Morning Run"`
	Icon        string `json:"icon,omitempty" validate:"max=50" example:"🏃"`
	Description string `json:"description,omitempty" validate:"max=2000" example:"Quick morning jog"`

	// Task Defaults
	DefaultDuration  int    `json:"default_duration,omitempty" example:"1800"`
	DefaultEmotionID string `json:"default_emotion_id,omitempty" example:"emotions:E16"`
	DefaultPriority  int    `json:"default_priority,omitempty" validate:"min=1,max=5" example:"3"`
	DefaultCompleted bool   `json:"default_completed,omitempty" example:"true"`

	// Quantity Settings
	QuantityEnabled bool    `json:"quantity_enabled,omitempty" example:"true"`
	QuantityDefault float64 `json:"quantity_default,omitempty" validate:"min=0" example:"1.0"`
	QuantityStep    float64 `json:"quantity_step,omitempty" validate:"min=0" example:"0.5"`
	QuantityUnitID  string  `json:"quantity_unit_id,omitempty" example:"units:km"`

	// Goal Link Defaults
	DefaultImpact string `json:"default_impact,omitempty" validate:"omitempty,oneof=positive negative neutral" example:"positive"`

	// Display
	Pinned    bool `json:"pinned,omitempty" example:"true"`
	SortOrder int  `json:"sort_order,omitempty" example:"1"`

	// Category assignment
	CategoryID string `json:"category_id,omitempty" example:"categories:health"`

	// Goal linking
	GoalLinks []GoalLinkInput `json:"goal_links,omitempty"`
}

// GoalLinkInput is the input for linking an activity to a goal.
//
// @Description Input for activity-goal link configuration
type GoalLinkInput struct {
	GoalID             string   `json:"goal_id" validate:"required" example:"goals:hydration"`
	AutoLinkTasks      bool     `json:"auto_link_tasks,omitempty" example:"true"`
	QuantityMultiplier float64  `json:"quantity_multiplier,omitempty" validate:"min=0" example:"1.0"`
	DefaultQuantity    *float64 `json:"default_quantity,omitempty" validate:"omitempty,min=0" example:"1.0"`
	DefaultImpact      string   `json:"default_impact,omitempty" validate:"omitempty,oneof=positive negative neutral" example:"positive"`
}

// UpdateRequest is the request payload for updating an activity.
//
// @Description Request payload for updating an activity
type UpdateRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Icon        *string `json:"icon,omitempty" validate:"omitempty,max=50"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`

	DefaultDuration  *int    `json:"default_duration,omitempty"`
	DefaultEmotionID *string `json:"default_emotion_id,omitempty"`
	DefaultPriority  *int    `json:"default_priority,omitempty" validate:"omitempty,min=1,max=5"`
	DefaultCompleted *bool   `json:"default_completed,omitempty"`

	QuantityEnabled *bool    `json:"quantity_enabled,omitempty"`
	QuantityDefault *float64 `json:"quantity_default,omitempty" validate:"omitempty,min=0"`
	QuantityStep    *float64 `json:"quantity_step,omitempty" validate:"omitempty,min=0"`
	QuantityUnitID  *string  `json:"quantity_unit_id,omitempty"`

	DefaultImpact *string `json:"default_impact,omitempty" validate:"omitempty,oneof=positive negative neutral"`

	Pinned    *bool `json:"pinned,omitempty"`
	SortOrder *int  `json:"sort_order,omitempty"`
}

// InstantLogRequest creates a completed task immediately.
//
// @Description Request for instant task logging
type InstantLogRequest struct {
	Quantity  *float64 `json:"quantity,omitempty" example:"2.5"`
	Notes     string   `json:"notes,omitempty" example:"Felt great today"`
	Timestamp *string  `json:"timestamp,omitempty" example:"2025-01-25T10:00:00Z"`
}

// ScheduleRequest returns pre-filled task data.
//
// @Description Request for scheduling a task from activity
type ScheduleRequest struct {
	StartDate string `json:"start_date,omitempty" example:"2025-01-25T14:00:00Z"`
}

// TimerStartRequest begins a Flowmodoro session.
//
// @Description Request to start timer session
type TimerStartRequest struct {
	Notes string `json:"notes,omitempty" example:"Focus session"`
}

// TimerStopRequest ends the session.
//
// @Description Request to stop timer session
type TimerStopRequest struct {
	Notes string `json:"notes,omitempty" example:"Completed deep work"`
}

// TimerBreakRequest records a break.
//
// @Description Request to record a break
type TimerBreakRequest struct {
	Action string `json:"action" validate:"required,oneof=start end" example:"start"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// ActivityPageResponse is the paginated response for listing activities.
//
// @Description Paginated list of activities
type ActivityPageResponse struct {
	Items   []*Activity `json:"items"`
	Total   int64       `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"has_more"`
}

// InstantLogResponse returns the created task and goal updates.
//
// @Description Response for instant logging
type InstantLogResponse struct {
	TaskID       string              `json:"task_id"`
	TaskTitle    string              `json:"task_title"`
	GoalsUpdated []GoalUpdateSummary `json:"goals_updated,omitempty"`
}

// GoalUpdateSummary shows how a goal was impacted.
//
// @Description Goal progress update summary
type GoalUpdateSummary struct {
	GoalID      string  `json:"goal_id"`
	GoalTitle   string  `json:"goal_title"`
	GoalIcon    string  `json:"goal_icon,omitempty"`
	ValueAdded  float64 `json:"value_added"`
	NewTotal    float64 `json:"new_total"`
	TargetValue float64 `json:"target_value,omitempty"`
	IsCompleted bool    `json:"is_completed"`
}

// ScheduleResponse returns pre-filled task data.
//
// @Description Response with pre-filled task defaults
type ScheduleResponse struct {
	Activity     *Activity         `json:"activity"`
	TaskDefaults TaskDefaults      `json:"task_defaults"`
	GoalLinks    []GoalLinkDefault `json:"goal_links"`
}

// TaskDefaults contains pre-filled values for task form.
//
// @Description Pre-filled task form values
type TaskDefaults struct {
	Title        string  `json:"title"`
	Journal      string  `json:"journal,omitempty"`
	Duration     int     `json:"duration,omitempty"`
	Priority     int     `json:"priority"`
	CategoryID   string  `json:"category_id,omitempty"`
	EmotionID    string  `json:"emotion_id,omitempty"`
	Quantity     float64 `json:"quantity,omitempty"`
	QuantityUnit string  `json:"quantity_unit,omitempty"`
}

// GoalLinkDefault is a pre-configured goal link for task creation.
//
// @Description Pre-configured goal link
type GoalLinkDefault struct {
	GoalID          string   `json:"goal_id"`
	GoalTitle       string   `json:"goal_title"`
	GoalIcon        string   `json:"goal_icon,omitempty"`
	ImpactType      string   `json:"impact_type"`
	DefaultQuantity *float64 `json:"default_quantity,omitempty"` // Per-goal default quantity
	QuantityUnit    string   `json:"quantity_unit,omitempty"`    // From goal's target.unit_id
	QuantityStep    float64  `json:"quantity_step,omitempty"`    // From activity (shared)
}

// TimerStartResponse returns the in-progress task.
//
// @Description Response for starting timer
type TimerStartResponse struct {
	TaskID    string    `json:"task_id"`
	SessionID string    `json:"session_id"`
	StartedAt time.Time `json:"started_at"`
}

// TimerStopResponse returns the completed task.
//
// @Description Response for stopping timer
type TimerStopResponse struct {
	TaskID        string              `json:"task_id"`
	TaskTitle     string              `json:"task_title"`
	TotalDuration int                 `json:"total_duration"` // seconds
	WorkDuration  int                 `json:"work_duration"`  // seconds (excluding breaks)
	BreakDuration int                 `json:"break_duration"` // seconds
	Breaks        []TimerBreak        `json:"breaks"`
	GoalsUpdated  []GoalUpdateSummary `json:"goals_updated,omitempty"`
}

// ActivityGoalLinkDetail contains full goal details for activity display.
//
// @Description Goal linked to activity with full details
type ActivityGoalLinkDetail struct {
	GoalID             string   `json:"goal_id"`
	GoalTitle          string   `json:"goal_title"`
	GoalIcon           string   `json:"goal_icon,omitempty"`
	GoalColor          string   `json:"goal_color,omitempty"`
	AutoLinkTasks      bool     `json:"auto_link_tasks"`
	QuantityMultiplier float64  `json:"quantity_multiplier"`
	DefaultQuantity    *float64 `json:"default_quantity,omitempty"`
	DefaultImpact      string   `json:"default_impact"`
	TargetUnitID       string   `json:"target_unit_id,omitempty"`
	TargetUnitSymbol   string   `json:"target_unit_symbol,omitempty"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for activities.
	Table = "activities"

	// ActivityGoalsTable is the SurrealDB relation table name.
	ActivityGoalsTable = "activity_goals"

	// TimerSessionsTable is the SurrealDB table for timer sessions.
	TimerSessionsTable = "timer_sessions"

	// CreatedFromActivityTable tracks task origin.
	CreatedFromActivityTable = "created_from_activity"

	// Impact types
	ImpactPositive = "positive"
	ImpactNegative = "negative"
	ImpactNeutral  = "neutral"

	// Timer session statuses
	SessionActive    = "active"
	SessionPaused    = "paused"
	SessionCompleted = "completed"
	SessionAbandoned = "abandoned"

	// Task creation modes
	ModeInstant  = "instant"
	ModeSchedule = "schedule"
	ModeTimer    = "timer"
)

// ValidImpactTypes for validation.
var ValidImpactTypes = []string{ImpactPositive, ImpactNegative, ImpactNeutral}

// ValidModes for validation.
var ValidModes = []string{ModeInstant, ModeSchedule, ModeTimer}

// Ensure units import is used (for future unit lookups)
var _ = units.Table
