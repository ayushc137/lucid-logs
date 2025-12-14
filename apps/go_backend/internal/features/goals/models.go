// Package goals provides goal/habit management functionality.
//
// This package implements:
//   - CRUD operations for goals (one-time and recurring/habits)
//   - Activity key auto-generation for template/task linking
//   - Streak tracking for recurring goals
//   - Integration with templates for quick logging
//
// Goal Types:
//   - discrete: One-time goal without measurable target
//   - measurable: Goal with quantifiable target (e.g., "Run 100km")
//   - avoidance: Goal to NOT do something (e.g., "No junk food")
//   - epic: Parent goal with child milestones
//
// Database Architecture:
//   - Schemaless SurrealDB table with record links
//   - activity_key enables automatic task-goal matching
//   - linked_template holds auto-created template ID
package goals

import (
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Goal represents a goal or habit in the system.
//
// Goals and habits are unified: a habit is simply a goal with recurrence set.
// This matches how users naturally think about goals.
type Goal struct {
	ID          string `json:"id,omitempty"`
	CreatedBy   string `json:"-"`
	ActivityKey string `json:"activity_key"` // Auto-generated unique key for matching

	// Core fields
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Why         string `json:"why,omitempty"` // "Why does this matter?" - for retros
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`

	// Goal type
	GoalType string `json:"goal_type"` // "discrete", "measurable", "epic", "avoidance"

	// Recurrence (null = one-time, populated = recurring/habit)
	Recurrence *Recurrence `json:"recurrence,omitempty"`

	// Target (for measurable goals)
	Target *Target `json:"target,omitempty"`

	// Timeline
	StartDate *time.Time `json:"start_date,omitempty"`
	Deadline  *time.Time `json:"deadline,omitempty"`

	// Status & progress
	Status         string     `json:"status"` // "active", "completed", "paused", "abandoned"
	CompletionDate *time.Time `json:"completion_date,omitempty"`

	// Streak tracking (computed, for recurring goals)
	CurrentStreak     int        `json:"current_streak"`
	LongestStreak     int        `json:"longest_streak"`
	LastCompletedDate *time.Time `json:"last_completed_date,omitempty"`
	GraceDaysUsed     int        `json:"grace_days_used"`

	// Completion settings (for epic goals)
	CompletionMode string `json:"completion_mode,omitempty"` // "all" (AND) or "any" (OR)

	// Organization
	Priority   int                  `json:"priority"`    // 1 (low), 2 (medium), 3 (high)
	ValueScore int                  `json:"value_score"` // 1-5, how meaningful
	Category   *categories.Category `json:"category,omitempty"`
	ParentGoal *string              `json:"parent_goal,omitempty"` // For milestones under epics
	LifeDomain string               `json:"life_domain,omitempty"` // health, work, learning, etc.

	// Linked template (auto-created)
	LinkedTemplate *string `json:"linked_template,omitempty"`

	// Privacy
	IsPrivate bool `json:"is_private"`

	// Metadata
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// Linked tasks (populated via FETCH)
	LinkedTasks []GoalTaskLink `json:"linked_tasks,omitempty"`

	// Child goals (populated for epic goals)
	ChildGoals []*Goal `json:"child_goals,omitempty"`
}

// GoalTaskLink represents a linked task with impact metadata.
// This is populated via SurrealDB FETCH from the task_goals relation.
type GoalTaskLink struct {
	TaskID          string   `json:"task_id"`
	TaskTitle       string   `json:"task_title"`
	ImpactType      string   `json:"impact_type"`      // "positive", "negative", "neutral"
	ImpactMagnitude int      `json:"impact_magnitude"` // 1-5
	QuantityValue   *float64 `json:"quantity_value,omitempty"`
	QuantityUnit    *string  `json:"quantity_unit,omitempty"`
}

// Recurrence defines how often a recurring goal/habit should be completed.
type Recurrence struct {
	Frequency  int      `json:"frequency"`             // Times per period (e.g., 5)
	Period     string   `json:"period"`                // "day", "week", "month"
	ActiveDays []string `json:"active_days,omitempty"` // ["mon", "tue", ...] or nil for all

	// Time constraints
	BeforeTime string `json:"before_time,omitempty"` // "22:00" (complete before 10pm)
	AfterTime  string `json:"after_time,omitempty"`  // "06:00" (complete after 6am)

	// Streak settings
	GraceDays int `json:"grace_days,omitempty"` // Can miss X days without breaking streak
}

// Target defines the measurable target for a goal.
type Target struct {
	Value        float64 `json:"value"`                // e.g., 1000, 3
	Unit         string  `json:"unit"`                 // "km", "L", "pages", "minutes"
	CurrentValue float64 `json:"current_value"`        // Auto-computed from linked tasks
	PerPeriod    bool    `json:"per_period,omitempty"` // true = "3L per day", false = "1000km total"
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a new goal.
//
// @Description Request payload for creating a goal
type CreateRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=500" example:"Drink 3L water daily"`
	Description string `json:"description,omitempty" validate:"max=2000" example:"Stay hydrated throughout the day for better health and focus"`
	Why         string `json:"why,omitempty" validate:"max=1000" example:"Improve energy levels and skin health"`
	Icon        string `json:"icon,omitempty" validate:"max=50" example:"💧"`
	Color       string `json:"color,omitempty" validate:"max=20" example:"#3B82F6"`

	GoalType string `json:"goal_type" validate:"required,oneof=discrete measurable epic avoidance" example:"measurable"`

	Recurrence *RecurrenceInput `json:"recurrence,omitempty"`
	Target     *TargetInput     `json:"target,omitempty"`

	StartDate *string `json:"start_date,omitempty" validate:"omitempty,datetime_flexible" example:"2025-01-01T00:00:00Z"`
	Deadline  *string `json:"deadline,omitempty" validate:"omitempty,datetime_flexible" example:"2025-12-31T23:59:59Z"`

	Priority       int    `json:"priority,omitempty" validate:"min=0,max=3" example:"2"`
	ValueScore     int    `json:"value_score,omitempty" validate:"min=0,max=5" example:"4"`
	CategoryID     string `json:"category_id,omitempty" example:"categories:health123"`
	ParentGoal     string `json:"parent_goal,omitempty" example:"goals:epic456"`
	LifeDomain     string `json:"life_domain,omitempty" validate:"max=50" example:"health"`
	CompletionMode string `json:"completion_mode,omitempty" validate:"omitempty,oneof=all any" example:"all"`

	IsPrivate bool `json:"is_private,omitempty" example:"false"`
}

// RecurrenceInput is the input format for recurrence settings.
type RecurrenceInput struct {
	Frequency  int      `json:"frequency" validate:"required,min=1,max=365" example:"1"`
	Period     string   `json:"period" validate:"required,oneof=day week month" example:"day"`
	ActiveDays []string `json:"active_days,omitempty"`
	BeforeTime string   `json:"before_time,omitempty" example:"22:00"`
	AfterTime  string   `json:"after_time,omitempty" example:"06:00"`
	GraceDays  int      `json:"grace_days,omitempty" validate:"min=0,max=7" example:"1"`
}

// TargetInput is the input format for target settings.
type TargetInput struct {
	Value     float64 `json:"value" validate:"required,gt=0" example:"3"`
	Unit      string  `json:"unit" validate:"required,min=1,max=50" example:"liters"`
	PerPeriod bool    `json:"per_period,omitempty" example:"true"`
}

// UpdateRequest is the request payload for updating a goal.
//
// All fields are optional. Only provided fields will be updated.
//
// @Description Request payload for updating a goal
type UpdateRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	Why         *string `json:"why,omitempty" validate:"omitempty,max=1000"`
	Icon        *string `json:"icon,omitempty" validate:"omitempty,max=50"`
	Color       *string `json:"color,omitempty" validate:"omitempty,max=20"`

	GoalType *string `json:"goal_type,omitempty" validate:"omitempty,oneof=discrete measurable epic avoidance"`

	Recurrence *RecurrenceInput `json:"recurrence,omitempty"`
	Target     *TargetInput     `json:"target,omitempty"`

	StartDate *string `json:"start_date,omitempty" validate:"omitempty,datetime_flexible"`
	Deadline  *string `json:"deadline,omitempty" validate:"omitempty,datetime_flexible"`

	Status *string `json:"status,omitempty" validate:"omitempty,oneof=active completed paused abandoned"`

	Priority       *int    `json:"priority,omitempty" validate:"omitempty,min=0,max=3"`
	ValueScore     *int    `json:"value_score,omitempty" validate:"omitempty,min=0,max=5"`
	CategoryID     *string `json:"category_id,omitempty"`
	LifeDomain     *string `json:"life_domain,omitempty" validate:"omitempty,max=50"`
	CompletionMode *string `json:"completion_mode,omitempty" validate:"omitempty,oneof=all any"`

	IsPrivate *bool `json:"is_private,omitempty"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// GoalPageResponse documents the paginated response returned by the List endpoint.
type GoalPageResponse struct {
	Items   []*Goal `json:"items"`
	Total   int64   `json:"total"`
	Limit   int     `json:"limit"`
	Offset  int     `json:"offset"`
	HasMore bool    `json:"has_more"`
}

// TodayGoal represents a recurring goal with today's status.
type TodayGoal struct {
	Goal       *Goal    `json:"goal"`
	TodayMet   bool     `json:"today_met"`
	TodayValue *float64 `json:"today_value,omitempty"`
	Streak     int      `json:"streak"`
}

// TodayGoalsResponse is the response for GET /goals/today.
type TodayGoalsResponse struct {
	Date  string       `json:"date"` // YYYY-MM-DD
	Goals []*TodayGoal `json:"goals"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for goals.
	Table = "goals"

	// Goal types
	GoalTypeDiscrete   = "discrete"
	GoalTypeMeasurable = "measurable"
	GoalTypeEpic       = "epic"
	GoalTypeAvoidance  = "avoidance"

	// Goal statuses
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusPaused    = "paused"
	StatusAbandoned = "abandoned"

	// Recurrence periods
	PeriodDay   = "day"
	PeriodWeek  = "week"
	PeriodMonth = "month"

	// Completion modes (for epic goals)
	CompletionModeAll = "all" // AND logic: complete when ALL children are done
	CompletionModeAny = "any" // OR logic: complete when ANY child is done
)
