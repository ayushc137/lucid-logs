// Package retrospectives provides retrospective generation and management.
//
// This package implements:
//   - Auto-generation of daily/weekly/monthly retrospectives
//   - Mood, habit, task, goal analysis for a period
//   - User-editable content and reflections
//   - Scheduling integration for automatic generation
//
// Retrospectives aggregate user data into actionable insights
// and reflection prompts for self-improvement.
package retrospectives

import (
	"time"
)

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// Retrospective represents a reflection on a time period.
//
// @Description A retrospective contains auto-generated summaries and user reflections
type Retrospective struct {
	ID        string `json:"id,omitempty"`
	CreatedBy string `json:"-"`

	// Type & Scope
	RetroType string    `json:"retro_type"` // daily, weekly, monthly, quarterly, yearly, custom
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`

	// Auto-Generated Content
	AutoSummary RetroAutoSummary `json:"auto_summary"`

	// User-Editable Content
	UserContent UserReflection `json:"user_content"`

	// Status
	Status string `json:"status"` // draft, completed

	// Metadata
	GeneratedAt time.Time  `json:"generated_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// =============================================================================
// AUTO-GENERATED SUMMARY
// =============================================================================

// RetroAutoSummary contains computed analytics for the period.
type RetroAutoSummary struct {
	Mood       MoodSummary       `json:"mood"`
	Habits     HabitsSummary     `json:"habits"`
	Tasks      TasksSummary      `json:"tasks"`
	Goals      GoalsSummary      `json:"goals"`
	Categories CategoriesSummary `json:"categories"`
	Insights   []string          `json:"insights,omitempty"`         // AI-generated insights
	AINarrative string           `json:"ai_narrative,omitempty"`     // AI-generated narrative paragraph
}

// MoodSummary contains emotion analysis for the period.
type MoodSummary struct {
	AverageValence       float64      `json:"average_valence"`
	AverageArousal       float64      `json:"average_arousal"`
	DominantQuadrant     string       `json:"dominant_quadrant"`
	QuadrantDistribution QuadrantDist `json:"quadrant_distribution"`
	NotableSpikes        []MoodEvent  `json:"notable_spikes,omitempty"`
	NotableDips          []MoodEvent  `json:"notable_dips,omitempty"`
}

// QuadrantDist shows percentage in each mood quadrant.
type QuadrantDist struct {
	Yellow float64 `json:"yellow"`
	Green  float64 `json:"green"`
	Red    float64 `json:"red"`
	Blue   float64 `json:"blue"`
}

// MoodEvent represents a notable emotion spike or dip.
type MoodEvent struct {
	Date    time.Time `json:"date"`
	Emotion string    `json:"emotion"`
	Context string    `json:"context,omitempty"`
}

// HabitsSummary contains habit tracking analysis.
type HabitsSummary struct {
	Met          []HabitStatus  `json:"met"`
	PartiallyMet []HabitStatus  `json:"partially_met"`
	Missed       []HabitStatus  `json:"missed"`
	Streaks      StreaksSummary `json:"streaks"`
}

// HabitStatus shows status for a single habit.
type HabitStatus struct {
	HabitID     string  `json:"habit_id"`
	Name        string  `json:"name"`
	SuccessRate float64 `json:"success_rate,omitempty"`
}

// StreaksSummary tracks streak changes.
type StreaksSummary struct {
	Continued []StreakUpdate `json:"continued,omitempty"`
	Broken    []StreakUpdate `json:"broken,omitempty"`
	Started   []StreakUpdate `json:"started,omitempty"`
}

// StreakUpdate represents a change in streak.
type StreakUpdate struct {
	HabitID string `json:"habit_id"`
	Name    string `json:"name"`
	Streak  int    `json:"streak,omitempty"`
	Was     int    `json:"was,omitempty"`
	Now     int    `json:"now,omitempty"`
}

// TasksSummary contains task completion analysis.
type TasksSummary struct {
	Completed          int             `json:"completed"`
	Postponed          int             `json:"postponed"`
	Canceled           int             `json:"canceled"`
	NotStarted         int             `json:"not_started"`
	TotalDurationHours float64         `json:"total_duration_hours"`
	ByCategory         []CategoryCount `json:"by_category,omitempty"`
}

// CategoryCount shows task count per category.
type CategoryCount struct {
	Category      string  `json:"category"`
	Count         int     `json:"count"`
	DurationHours float64 `json:"duration_hours"`
}

// GoalsSummary contains goal progress analysis.
type GoalsSummary struct {
	NetImpact             []GoalImpact    `json:"net_impact,omitempty"`
	SignificantlyAdvanced []GoalHighlight `json:"significantly_advanced,omitempty"`
	NegativelyImpacted    []GoalHighlight `json:"negatively_impacted,omitempty"`
}

// GoalImpact shows positive/negative impact on a goal.
type GoalImpact struct {
	GoalID   string `json:"goal_id"`
	Name     string `json:"name"`
	Positive int    `json:"positive"`
	Negative int    `json:"negative"`
}

// GoalHighlight identifies goals with significant changes.
type GoalHighlight struct {
	GoalID        string  `json:"goal_id"`
	Name          string  `json:"name"`
	ProgressDelta float64 `json:"progress_delta,omitempty"`
	Reason        string  `json:"reason,omitempty"`
}

// CategoriesSummary contains time distribution analysis.
type CategoriesSummary struct {
	TimeDistribution []CategoryTime  `json:"time_distribution"`
	Neglected        []NeglectedArea `json:"neglected,omitempty"`
}

// CategoryTime shows time spent in a category.
type CategoryTime struct {
	Category   string  `json:"category"`
	Hours      float64 `json:"hours"`
	Percentage float64 `json:"percentage"`
}

// NeglectedArea identifies categories with no recent activity.
type NeglectedArea struct {
	Category          string `json:"category"`
	DaysSinceLastTask int    `json:"days_since_last_task"`
}

// =============================================================================
// USER CONTENT
// =============================================================================

// UserReflection contains user-editable reflection content.
type UserReflection struct {
	WhatWentWell    string   `json:"what_went_well,omitempty" example:"Completed all my morning routines consistently"`
	WhatDidntGoWell string   `json:"what_didnt_go_well,omitempty" example:"Procrastinated on the design review task"`
	WhatLearned     string   `json:"what_learned,omitempty" example:"Breaking tasks into smaller chunks helps me focus"`
	Gratitude       []string `json:"gratitude,omitempty"`
	ProudOf         string   `json:"proud_of,omitempty" example:"Maintained my exercise streak for 30 days"`
	ChangeTomorrow  string   `json:"change_tomorrow,omitempty" example:"Start with the hardest task first thing in the morning"`
	AdditionalNotes string   `json:"additional_notes,omitempty" example:"Need to schedule a dentist appointment"`
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// GenerateRequest is for generating a new retrospective.
//
// @Description Request to generate a retrospective
type GenerateRequest struct {
	RetroType string     `json:"retro_type" validate:"required,oneof=daily weekly monthly quarterly yearly custom" example:"daily"`
	Date      *time.Time `json:"date,omitempty" example:"2025-12-14T00:00:00Z"`       // For daily retros (defaults to today)
	StartDate *time.Time `json:"start_date,omitempty" example:"2025-12-01T00:00:00Z"` // For custom range
	EndDate   *time.Time `json:"end_date,omitempty" example:"2025-12-14T23:59:59Z"`   // For custom range
}

// UpdateRequest is for updating user content on a retrospective.
//
// @Description Request to update retrospective user content
type UpdateRequest struct {
	UserContent *UserReflection `json:"user_content,omitempty"`
	Status      *string         `json:"status,omitempty" validate:"omitempty,oneof=draft completed" example:"completed"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// ListResponse for paginated retrospective list.
type ListResponse struct {
	Retrospectives []*Retrospective `json:"retrospectives"`
	Total          int              `json:"total"`
	Limit          int              `json:"limit"`
	Offset         int              `json:"offset"`
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Retro types
	RetroTypeDaily     = "daily"
	RetroTypeWeekly    = "weekly"
	RetroTypeMonthly   = "monthly"
	RetroTypeQuarterly = "quarterly"
	RetroTypeYearly    = "yearly"
	RetroTypeCustom    = "custom"

	// Statuses
	StatusDraft     = "draft"
	StatusCompleted = "completed"

	// Table name
	Table = "retrospectives"
)
