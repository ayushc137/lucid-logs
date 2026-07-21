// Package goals provides goal management functionality.
//
// This package implements:
//   - CRUD operations for goals (simple, measurable, habits, grouped)
//   - Graph-inferred goal nature (no goal_type enum)
//   - Target with operators (gte, lte, eq) for achievement/avoidance goals
//   - Streak tracking for recurring goals
//   - Goal history via goal_logs relation
//
// Goal Nature (Graph-Inferred):
//   - Has children via goal_children → Grouped goal
//   - Has recurrence → Habit
//   - Has target → Measurable goal
//   - Has target.operator="lte" or "eq" → Avoidance/limit goal
//   - None of above → Simple goal (target.value=1 implied)
//
// Status:
//   - active: Currently working on
//   - completed: Goal achieved
//   - archived: Hidden from active views
//
// Database Architecture:
//   - Uses the in_category relation table for categories
//   - Child goals via goal_children relation
//   - History tracking via goal_logs relation
package goals

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Goal represents a goal, habit, or project in the system.
//
// The goal's nature is inferred from its structure:
//   - Has children (via goal_children) → Grouped goal
//   - Has recurrence → Habit
//   - Has target → Measurable
//   - target.operator="lte"/"eq" → Avoidance
//
// @Description Goal entity with graph-inferred nature
type Goal struct {
	ID        string `json:"id,omitempty"`
	CreatedBy string `json:"-"`

	// Core fields
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"` // Emoji

	// Target (optional - defines measurable objectives)
	// If nil, goal is a simple "complete it" goal (target.value=1 implied)
	Target *Target `json:"target,omitempty"`

	// Recurrence (optional - if present, this is a habit)
	Recurrence *Recurrence `json:"recurrence,omitempty"`

	// Status: only 3 states
	Status string `json:"status"` // "active", "completed", "archived"

	// Computed statistics (populated on read from related data)
	Stats *GoalStats `json:"stats,omitempty"`

	// Organization
	Priority int `json:"priority"` // 1-3

	// Timeline
	StartDate   *time.Time `json:"start_date,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// Populated via graph queries (not stored on goal record)
	Category      *categories.Category `json:"category,omitempty"`        // From in_category edge
	Children      []*Goal              `json:"children,omitempty"`        // From goal_children (forward)
	Parent        *Goal                `json:"parent,omitempty"`          // From goal_children (reverse)
	LinkedTaskIDs []string             `json:"linked_task_ids,omitempty"` // From task_goals edge (for highlighting)
}

// Target defines what success looks like for a measurable goal.
//
// Operators:
//   - "gte": At least (≥) - achievement goals (e.g., "Run 100km")
//   - "lte": At most (≤) - limit goals (e.g., "Max 2 coffees/day")
//   - "eq": Exactly (=) - strict goals (e.g., "Zero cigarettes")
//
// @Description Target for measurable goals with comparison operator
type Target struct {
	Value              float64 `json:"value"`                // Target amount (e.g., 100, 3, 0)
	Operator           string  `json:"operator"`             // "gte", "lte", "eq"
	UnitID             string  `json:"unit_id"`              // Reference to units table (e.g., "units:km")
	TrackCompletedOnly bool    `json:"track_completed_only"` // true = only count completed tasks
}

// GoalStats contains computed statistics for a goal.
// These are calculated on read from related data, not stored on the goal.
//
// @Description Computed goal statistics
type GoalStats struct {
	// Progress toward target
	CurrentValue    float64 `json:"current_value"`    // Sum from task_goals
	ProgressPercent float64 `json:"progress_percent"` // 0-100 (or >100 if exceeded)

	// Streak tracking (for habits with recurrence)
	CurrentStreak     int        `json:"current_streak"`
	LongestStreak     int        `json:"longest_streak"`
	LastCompletedDate *time.Time `json:"last_completed_date,omitempty"`
	TodayStatus       string     `json:"today_status,omitempty"` // "pending", "met", "exceeded"

	// For grouped goals (with children)
	ChildrenTotal     int `json:"children_total,omitempty"`
	ChildrenCompleted int `json:"children_completed,omitempty"`

	// Overall metrics
	TotalContributions int `json:"total_contributions"` // Count of task_goals links
}

// GoalTaskLink represents a linked task with impact metadata.
// This is populated from the task_goals relation.
//
// @Description Task linked to goal with impact data
type GoalTaskLink struct {
	TaskID        string   `json:"task_id"`
	TaskTitle     string   `json:"task_title"`
	ImpactType    string   `json:"impact_type"` // "positive", "negative", "neutral"
	QuantityValue *float64 `json:"quantity_value,omitempty"`
	UnitID        *string  `json:"unit_id,omitempty"`

	// Additional task details for rich display
	TaskJournal   string     `json:"task_journal,omitempty"`
	TaskStartDate *time.Time `json:"task_start_date,omitempty"`
	TaskEndDate   *time.Time `json:"task_end_date,omitempty"`
	TaskCompleted bool       `json:"task_completed"`
	TaskEmotionID *string    `json:"task_emotion_id,omitempty"`
	TaskCategory  *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"task_category,omitempty"`

	// Link metadata
	LinkedAt *time.Time `json:"linked_at,omitempty"`
	Notes    string     `json:"notes,omitempty"`
}

// Recurrence defines how often a recurring goal/habit should be completed.
//
// @Description Recurrence settings for habits
type Recurrence struct {
	Frequency  int      `json:"frequency"`             // Times per period (e.g., 5)
	Period     string   `json:"period"`                // "day", "week", "month"
	ActiveDays []string `json:"active_days,omitempty"` // ["mon", "tue", ...] or nil for all
	BeforeTime string   `json:"before_time,omitempty"` // "22:00" (complete before)
	AfterTime  string   `json:"after_time,omitempty"`  // "06:00" (complete after)
	GraceDays  int      `json:"grace_days,omitempty"`  // Can miss X days without breaking streak
}

// =============================================================================
// GOAL NATURE HELPERS
// =============================================================================

// IsHabit returns true if this goal has a recurrence defined.
func (g *Goal) IsHabit() bool {
	return g.Recurrence != nil
}

// IsMeasurable returns true if this goal has a target defined.
func (g *Goal) IsMeasurable() bool {
	return g.Target != nil
}

// IsGrouped returns true if this goal has children.
func (g *Goal) IsGrouped() bool {
	return len(g.Children) > 0
}

// IsAvoidance returns true if this goal has a limit/avoidance target.
func (g *Goal) IsAvoidance() bool {
	return g.Target != nil && (g.Target.Operator == OperatorLTE || g.Target.Operator == OperatorEQ)
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new goal.
//
// The goal's nature is inferred from the provided fields:
//   - Provide recurrence → creates a habit
//   - Provide target → creates a measurable goal
//   - Neither → creates a simple goal (auto target.value=1)
//
// @Description Request payload for creating a goal
type CreateRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=500" example:"Drink 3L water daily"`
	Description string `json:"description,omitempty" validate:"max=2000" example:"Stay hydrated throughout the day"`
	Icon        string `json:"icon,omitempty" validate:"max=50" example:"💧"`

	// Target (optional - if nil, implies simple goal with target=1)
	Target *TargetInput `json:"target,omitempty"`

	// Recurrence (optional - if set, creates a habit)
	Recurrence *RecurrenceInput `json:"recurrence,omitempty"`

	// Timeline
	StartDate *string `json:"start_date,omitempty" validate:"omitempty,datetime_flexible" example:"2025-01-01T00:00:00Z"`
	Deadline  *string `json:"deadline,omitempty" validate:"omitempty,datetime_flexible" example:"2025-12-31T23:59:59Z"`

	// Organization
	Priority   int    `json:"priority,omitempty" validate:"min=0,max=3" example:"2"`
	CategoryID string `json:"category_id,omitempty" example:"categories:health123"`

	// Parent goal (optional - adds this as child of parent via goal_children)
	ParentGoalID string `json:"parent_goal_id,omitempty" example:"goals:launch_saas"`
}

// TargetInput is the input format for target settings.
//
// @Description Target configuration for measurable goals
type TargetInput struct {
	Value              float64 `json:"value" validate:"gte=0" example:"3"`
	Operator           string  `json:"operator,omitempty" validate:"omitempty,oneof=gte lte eq" example:"gte"`
	UnitID             string  `json:"unit_id" validate:"required" example:"units:l"`
	TrackCompletedOnly bool    `json:"track_completed_only,omitempty" example:"true"`
}

// RecurrenceInput is the input format for recurrence settings.
//
// @Description Recurrence configuration for habits
type RecurrenceInput struct {
	Frequency  int      `json:"frequency" validate:"required,min=1,max=365" example:"1"`
	Period     string   `json:"period" validate:"required,oneof=day week month" example:"day"`
	ActiveDays []string `json:"active_days,omitempty"`
	BeforeTime string   `json:"before_time,omitempty" example:"22:00"`
	AfterTime  string   `json:"after_time,omitempty" example:"06:00"`
	GraceDays  int      `json:"grace_days,omitempty" validate:"min=0,max=7" example:"1"`
}

// UpdateRequest is the request payload for updating a goal.
//
// All fields are optional. Only provided fields will be updated.
//
// @Description Request payload for updating a goal
type UpdateRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	Icon        *string `json:"icon,omitempty" validate:"omitempty,max=50"`

	Target     *TargetInput     `json:"target,omitempty"`
	Recurrence *RecurrenceInput `json:"recurrence,omitempty"`

	StartDate *string `json:"start_date,omitempty" validate:"omitempty,datetime_flexible"`
	Deadline  *string `json:"deadline,omitempty" validate:"omitempty,datetime_flexible"`

	Status   *string `json:"status,omitempty" validate:"omitempty,oneof=active completed archived"`
	Priority *int    `json:"priority,omitempty" validate:"omitempty,min=0,max=3"`

	CategoryID *string `json:"category_id,omitempty"`
}

// AddChildRequest is the request for adding a child goal to a grouped goal.
//
// @Description Request for adding a child to a grouped goal
type AddChildRequest struct {
	ChildGoalID string `json:"child_goal_id" validate:"required" example:"goals:design_ui"`
	Order       int    `json:"order,omitempty" example:"1"`
	Required    *bool  `json:"required,omitempty" example:"true"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// GoalPageResponse documents the paginated response returned by the List endpoint.
//
// @Description Paginated list of goals
type GoalPageResponse struct {
	Items   []*Goal `json:"items"`
	Total   int64   `json:"total"`
	Limit   int     `json:"limit"`
	Offset  int     `json:"offset"`
	HasMore bool    `json:"has_more"`
}

// TodayGoal represents a recurring goal with today's status.
//
// @Description Today's status for a habit/recurring goal
type TodayGoal struct {
	Goal       *Goal    `json:"goal"`
	TodayMet   bool     `json:"today_met"`
	TodayValue *float64 `json:"today_value,omitempty"`
	Streak     int      `json:"streak"`
}

// TodayGoalsResponse is the response for GET /goals/today.
//
// @Description Today's habits and their status
type TodayGoalsResponse struct {
	Date  string       `json:"date"` // YYYY-MM-DD
	Goals []*TodayGoal `json:"goals"`
}

// ChildGoalLink represents a child goal in a grouped goal.
//
// @Description Child goal in a group with ordering
type ChildGoalLink struct {
	GoalID   string `json:"goal_id"`
	Order    int    `json:"order"`
	Required bool   `json:"required"`
}

// GoalTasksResponse is the response for GET /goals/{id}/tasks.
//
// @Description Tasks linked to a goal
type GoalTasksResponse struct {
	GoalID string         `json:"goal_id"`
	Tasks  []GoalTaskLink `json:"tasks"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the database table name for goals.
	Table = "goals"

	// Status values (only 3 now)
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusArchived  = "archived"

	// Recurrence periods
	PeriodDay   = "day"
	PeriodWeek  = "week"
	PeriodMonth = "month"

	// Target operators
	OperatorGTE = "gte" // Greater than or equal (≥) - achievement
	OperatorLTE = "lte" // Less than or equal (≤) - limit/avoidance
	OperatorEQ  = "eq"  // Exactly equal (=) - strict target

	// Default operator
	DefaultOperator = OperatorGTE

	// Today status values
	TodayStatusPending  = "pending"
	TodayStatusMet      = "met"
	TodayStatusExceeded = "exceeded"
)

// ValidStatuses for validation.
var ValidStatuses = []string{StatusActive, StatusCompleted, StatusArchived}

// ValidOperators for validation.
var ValidOperators = []string{OperatorGTE, OperatorLTE, OperatorEQ}
