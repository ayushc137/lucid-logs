// Package goals provides goal management functionality on libSQL/SQLite.
//
// This package implements:
//   - CRUD operations for goals using SQL queries
//   - Graph-inferred goal nature (no goal_type enum)
//   - Category linking via the goals.category_id foreign key
//   - Child goals via the goal_children join table
//   - Soft delete support
//   - Pagination with filtering
//
// Database Architecture:
//   - goals.category_id: FK to categories(id) (replaces in_category edge)
//   - goal_children: join table for parent-child relationships
//   - task_goals: join table linking tasks to goals with impact metadata
//   - goal_daily_stats: pre-aggregated daily progress per goal
package goals

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the goal data access interface.
type Repository interface {
	// FindByID retrieves a goal by ID for a specific user.
	FindByID(ctx context.Context, id, userID string) (*Goal, error)

	// FindPaginated retrieves goals for a user with pagination and filters.
	FindPaginated(ctx context.Context, userID string, params pagination.Params, filters GoalFilters) ([]*Goal, int64, error)

	// FindRecurringForDate retrieves all recurring goals for a user that should be tracked on a given date.
	FindRecurringForDate(ctx context.Context, userID string, date time.Time) ([]*Goal, error)

	// Create creates a new goal.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Goal, error)

	// Update updates an existing goal.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Goal, error)

	// Delete soft-deletes a goal.
	Delete(ctx context.Context, id, userID string) error

	// FindChildren retrieves all child goals of a parent goal.
	FindChildren(ctx context.Context, parentGoalID, userID string) ([]*Goal, error)

	// FindTasksForGoal retrieves all tasks linked to a goal.
	FindTasksForGoal(ctx context.Context, goalID, userID string) ([]GoalTaskLink, error)

	// AddChild links a child goal to a parent.
	AddChild(ctx context.Context, parentID, childID, userID string, order int, required bool) error

	// RemoveChild removes a child goal from a parent.
	RemoveChild(ctx context.Context, parentID, childID, userID string) error

	// UpdateCategory sets or updates the category for a goal.
	UpdateCategory(ctx context.Context, goalID, categoryID, userID string) error

	// ComputeStats calculates the current stats for a goal.
	ComputeStats(ctx context.Context, goalID, userID string) (*GoalStats, error)

	// UpdateStreaks updates the denormalized streak fields on a goal.
	// Called when a task is completed or a habit entry is logged.
	UpdateStreaks(ctx context.Context, goalID string, currentStreak, longestStreak int, lastCompleted *time.Time) error

	// ==========================================================================
	// ANALYTICS METHODS (Pre-Aggregated Tables)
	// ==========================================================================

	// GetDailyProgress retrieves daily progress stats for a goal within a date range.
	// Reads from pre-computed goal_daily_stats table for O(1) lookups.
	GetDailyProgress(ctx context.Context, goalID, userID string, startDate, endDate time.Time) ([]DailyStats, error)

	// BackfillDailyStats populates goal_daily_stats from existing task_goals data.
	BackfillDailyStats(ctx context.Context, userID string) error

	// RecalculateDailyStats recalculates stats for a specific goal and date.
	RecalculateDailyStats(ctx context.Context, goalID string, date time.Time) error
}

// GoalFilters contains optional filters for listing goals.
type GoalFilters struct {
	Status       string     // Filter by status (active, completed, archived)
	IsRecurring  *bool      // Filter recurring (true) vs one-time (false)
	HasTarget    *bool      // Filter measurable goals
	HasChildren  *bool      // Filter grouped goals
	Search       string     // Search in title and description
	SortBy       string     // Sort field with optional -desc suffix
	DateFrom     *time.Time // Filter stats from this date
	DateTo       *time.Time // Filter stats to this date
	ProgressDate *time.Time // Calculate progress as of this specific date (for timeline views)
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new goal Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "goals").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

// goalDB is the internal database representation of a goal.
//
// The goals table stores recurrence/target as JSON-encoded TEXT columns. We
// decode them into map[string]any to preserve compatibility with the existing
// domain conversion helpers (mapToRecurrence / mapToTarget).
//
// Computed columns (filtered_stats_total, filtered_stats_count,
// children_stats_total, children_stats_completed, linked_task_ids,
// category_*) are populated via JOINs/subqueries in the SELECT queries and
// are decoded into the nested structs below.
type goalDB struct {
	ID        string `json:"id"`
	CreatedBy string `json:"created_by"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`

	Recurrence map[string]any `json:"recurrence,omitempty"`
	Target     map[string]any `json:"target,omitempty"`

	StartDate   *time.Time `json:"start_date,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Status   string `json:"status"`
	Priority int    `json:"priority"`

	// Denormalized streak fields (stored for fast reads)
	CurrentStreak     int        `json:"current_streak"`
	LongestStreak     int        `json:"longest_streak"`
	LastCompletedDate *time.Time `json:"last_completed_date,omitempty"`

	// Category (populated via LEFT JOIN on goals.category_id)
	Category *categoryDB `json:"category,omitempty"`

	// Children (populated via goal_children join — currently unused by reads
	// but kept for API compatibility with the previous struct shape).
	Children []goalChildDB `json:"children,omitempty"`

	// Linked task IDs (populated via GROUP_CONCAT over task_goals)
	LinkedTaskIDs []string `json:"linked_task_ids,omitempty"`

	// Computed stats (populated via subqueries in optimized fetches)
	FilteredStats *struct {
		Total any `json:"total"`
		Count any `json:"count"`
	} `json:"filtered_stats,omitempty"`

	ChildrenStats *struct {
		Total     any `json:"total"`
		Completed any `json:"completed"`
	} `json:"children_stats,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// goalChildDB represents a child goal link.
type goalChildDB struct {
	GoalID   string `json:"goal_id"`
	Order    int    `json:"order"`
	Required bool   `json:"required"`
}

// categoryDB is the database representation of a category when fetched.
type categoryDB struct {
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Color     string     `json:"color"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy string     `json:"created_by"`
}

func (c *categoryDB) toCategory() *categories.Category {
	if c == nil {
		return nil
	}
	return &categories.Category{
		ID:        c.ID,
		Name:      c.Name,
		Color:     c.Color,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		DeletedAt: c.DeletedAt,
		CreatedBy: c.CreatedBy,
	}
}

// toGoal converts the database model to the domain model.
func (g *goalDB) toGoal() *Goal {
	goal := &Goal{
		ID:        g.ID,
		CreatedBy: g.CreatedBy,

		Title:       g.Title,
		Description: g.Description,
		Icon:        g.Icon,

		Status:   g.Status,
		Priority: g.Priority,

		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}

	// Convert category
	if g.Category != nil {
		goal.Category = g.Category.toCategory()
	}

	// Convert dates (pointer-copied directly since goalDB now uses *time.Time)
	goal.StartDate = g.StartDate
	goal.Deadline = g.Deadline
	goal.CompletedAt = g.CompletedAt
	goal.DeletedAt = g.DeletedAt

	// Convert recurrence
	if g.Recurrence != nil {
		goal.Recurrence = mapToRecurrence(g.Recurrence)
	}

	// Convert target
	if g.Target != nil {
		goal.Target = mapToTarget(g.Target)
	}

	// Convert computed stats using helper functions for any -> numeric conversion
	var computedTaskCount, computedChildrenTotal, computedChildrenCompleted int
	var computedCurrentValue float64

	if g.FilteredStats != nil {
		computedTaskCount = anyToInt(g.FilteredStats.Count)
		computedCurrentValue = anyToFloat64(g.FilteredStats.Total)
	}
	if g.ChildrenStats != nil {
		computedChildrenTotal = anyToInt(g.ChildrenStats.Total)
		computedChildrenCompleted = anyToInt(g.ChildrenStats.Completed)
	}

	// Map computed stats from DB to Goal.Stats
	// Always initialize stats with stored streak data
	stats := &GoalStats{
		CurrentValue:       computedCurrentValue,
		TotalContributions: computedTaskCount,
		CurrentStreak:      g.CurrentStreak,
		LongestStreak:      g.LongestStreak,
		ChildrenTotal:      computedChildrenTotal,
		ChildrenCompleted:  computedChildrenCompleted,
	}

	stats.LastCompletedDate = g.LastCompletedDate

	// Calculate progress percent and status logic
	if goal.Target != nil && goal.Target.Value > 0 {
		stats.ProgressPercent = (stats.CurrentValue / goal.Target.Value) * 100

		switch goal.Target.Operator {
		case OperatorGTE:
			if stats.CurrentValue >= goal.Target.Value {
				stats.TodayStatus = TodayStatusMet
			} else {
				stats.TodayStatus = TodayStatusPending
			}
		case OperatorLTE:
			if stats.CurrentValue <= goal.Target.Value {
				stats.TodayStatus = TodayStatusMet
			} else {
				stats.TodayStatus = TodayStatusExceeded
			}
		case OperatorEQ:
			switch {
			case stats.CurrentValue == goal.Target.Value:
				stats.TodayStatus = TodayStatusMet
			case stats.CurrentValue > goal.Target.Value:
				stats.TodayStatus = TodayStatusExceeded
			default:
				stats.TodayStatus = TodayStatusPending
			}
		}
	}

	goal.Stats = stats

	// Copy linked task IDs for highlighting
	goal.LinkedTaskIDs = g.LinkedTaskIDs

	return goal
}

// anyToFloat64 converts an any type (from JSON decoding) to float64
func anyToFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case int:
		return float64(n)
	case uint64:
		return float64(n)
	case uint32:
		return float64(n)
	case uint:
		return float64(n)
	default:
		return 0
	}
}

// anyToInt converts an any type (from JSON decoding) to int
func anyToInt(v any) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int64:
		return int(n) //nolint:gosec // safe cast
	case int32:
		return int(n)
	case int:
		return n
	case uint64:
		return int(n) //nolint:gosec // safe cast
	case uint32:
		return int(n) //nolint:gosec // safe cast
	case uint:
		return int(n) //nolint:gosec // safe cast
	default:
		return 0
	}
}

func mapToRecurrence(m map[string]any) *Recurrence {
	if m == nil {
		return nil
	}
	r := &Recurrence{}
	// Handle frequency - JSON decoding returns float64 for numbers
	switch v := m["frequency"].(type) {
	case float64:
		r.Frequency = int(v)
	case float32:
		r.Frequency = int(v)
	case int64:
		r.Frequency = int(v) //nolint:gosec // configuration value
	case int32:
		r.Frequency = int(v)
	case int:
		r.Frequency = v
	case uint64:
		r.Frequency = int(v) //nolint:gosec // configuration value
	case uint32:
		r.Frequency = int(v) //nolint:gosec // configuration value
	case uint:
		r.Frequency = int(v) //nolint:gosec // configuration value
	}
	if v, ok := m["period"].(string); ok {
		r.Period = v
	}
	if v, ok := m["active_days"].([]any); ok {
		for _, d := range v {
			if s, ok := d.(string); ok {
				r.ActiveDays = append(r.ActiveDays, s)
			}
		}
	}
	if v, ok := m["before_time"].(string); ok {
		r.BeforeTime = v
	}
	if v, ok := m["after_time"].(string); ok {
		r.AfterTime = v
	}
	// Handle grace_days - same as frequency
	switch v := m["grace_days"].(type) {
	case float64:
		r.GraceDays = int(v)
	case float32:
		r.GraceDays = int(v)
	case int64:
		r.GraceDays = int(v) //nolint:gosec // configuration value
	case int32:
		r.GraceDays = int(v)
	case int:
		r.GraceDays = v
	case uint64:
		r.GraceDays = int(v) //nolint:gosec // configuration value
	case uint32:
		r.GraceDays = int(v) //nolint:gosec // configuration value
	case uint:
		r.GraceDays = int(v) //nolint:gosec // configuration value
	}
	return r
}

func mapToTarget(m map[string]any) *Target {
	if m == nil {
		return nil
	}
	t := &Target{
		Operator: DefaultOperator, // Default to GTE
	}
	// Handle value - JSON decoding returns float64 for numbers
	switch v := m["value"].(type) {
	case float64:
		t.Value = v
	case float32:
		t.Value = float64(v)
	case int64:
		t.Value = float64(v)
	case int32:
		t.Value = float64(v)
	case int:
		t.Value = float64(v)
	case uint64:
		t.Value = float64(v)
	case uint32:
		t.Value = float64(v)
	case uint:
		t.Value = float64(v)
	}
	if v, ok := m["unit_id"].(string); ok {
		t.UnitID = v
	}
	if v, ok := m["operator"].(string); ok {
		t.Operator = v
	}
	if v, ok := m["track_completed_only"].(bool); ok {
		t.TrackCompletedOnly = v
	}
	return t
}

// =============================================================================
// SQL HELPERS
// =============================================================================

// goalSelectColumns is the canonical column list for selecting goals, with
// JSON columns decoded and priority coerced to integer for the goalDB struct.
// Aliases match the json tags on goalDB.
const goalSelectColumns = `g.id, g.created_by, g.title, g.description, g.icon,
	g.recurrence, g.target, g.start_date, g.deadline, g.completed_at,
	g.status, COALESCE(CAST(g.priority AS INTEGER), 0) AS priority,
	g.current_streak, g.longest_streak, g.last_completed_date,
	g.created_at, g.updated_at, g.deleted_at`

// resolveGoalID normalizes a caller-supplied id (with or without "goals:"
// prefix) into the canonical "goals:<id>" form used as the primary key.
func resolveGoalID(id string) string {
	return database.RecordID(Table, id)
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByID(ctx context.Context, id, userID string) (*Goal, error) {
	goalID := resolveGoalID(id)

	// Single query that fetches the goal along with its category (LEFT JOIN),
	// aggregated task contributions (filtered_stats), and child stats.
	query := `
		SELECT
			` + goalSelectColumns + `,
			JSON_OBJECT(
				'id', c.id, 'name', c.name, 'color', c.color,
				'created_at', c.created_at, 'updated_at', c.updated_at,
				'deleted_at', c.deleted_at, 'created_by', c.created_by
			) AS category,
			JSON_OBJECT(
				'total', COALESCE((
					SELECT SUM(tg.quantity_value)
					FROM task_goals tg
					JOIN tasks t ON t.id = tg.task_id
					WHERE tg.goal_id = g.id
					  AND (json_extract(g.target, '$.track_completed_only') IS NULL
					       OR json_extract(g.target, '$.track_completed_only') != 1
					       OR t.completed = 1)
					  AND (
						g.recurrence IS NULL
						OR (
						  json_extract(g.recurrence, '$.period') = 'day'
						  AND date(COALESCE(t.completed_at, t.start_date)) = date('now')
						)
						OR (
						  json_extract(g.recurrence, '$.period') = 'week'
						  AND strftime('%W', COALESCE(t.completed_at, t.start_date)) = strftime('%W', 'now')
						  AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', 'now')
						)
						OR (
						  json_extract(g.recurrence, '$.period') = 'month'
						  AND strftime('%m', COALESCE(t.completed_at, t.start_date)) = strftime('%m', 'now')
						  AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', 'now')
						)
					  )
					), 0),
				'count', COALESCE((
					SELECT COUNT(*)
					FROM task_goals tg
					JOIN tasks t ON t.id = tg.task_id
					WHERE tg.goal_id = g.id
					  AND (json_extract(g.target, '$.track_completed_only') IS NULL
					       OR json_extract(g.target, '$.track_completed_only') != 1
					       OR t.completed = 1)
					  AND (
						g.recurrence IS NULL
						OR (
						  json_extract(g.recurrence, '$.period') = 'day'
						  AND date(COALESCE(t.completed_at, t.start_date)) = date('now')
						)
						OR (
						  json_extract(g.recurrence, '$.period') = 'week'
						  AND strftime('%W', COALESCE(t.completed_at, t.start_date)) = strftime('%W', 'now')
						  AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', 'now')
						)
						OR (
						  json_extract(g.recurrence, '$.period') = 'month'
						  AND strftime('%m', COALESCE(t.completed_at, t.start_date)) = strftime('%m', 'now')
						  AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', 'now')
						)
					  )
					), 0)
			) AS filtered_stats,
			JSON_OBJECT(
				'total', (SELECT COUNT(*) FROM goal_children gc WHERE gc.parent_goal_id = g.id),
				'completed', (SELECT COUNT(*) FROM goal_children gc
							  JOIN goals cg ON cg.id = gc.child_goal_id
							  WHERE gc.parent_goal_id = g.id AND cg.status = 'completed')
			) AS children_stats
		FROM goals g
		LEFT JOIN categories c ON c.id = g.category_id
		WHERE g.id = $id
	`

	goal, err := database.QueryFirst[goalDB](ctx, r.db, query, map[string]any{
		"id": goalID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", id).Msg("query failed for goal fetch")
		return nil, err
	}

	if goal == nil {
		return nil, errors.ErrNotFound
	}

	if goal.CreatedBy != userID || goal.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return goal.toGoal(), nil
}

func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params, filters GoalFilters) ([]*Goal, int64, error) {
	// Build WHERE clause dynamically
	conditions := []string{"g.created_by = $user", "g.deleted_at IS NULL"}
	queryVars := map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	}

	if filters.Status != "" {
		conditions = append(conditions, "g.status = $status")
		queryVars["status"] = filters.Status
	}
	if filters.IsRecurring != nil {
		if *filters.IsRecurring {
			conditions = append(conditions, "g.recurrence IS NOT NULL")
		} else {
			conditions = append(conditions, "g.recurrence IS NULL")
		}
	}
	if filters.HasTarget != nil {
		if *filters.HasTarget {
			conditions = append(conditions, "g.target IS NOT NULL")
		} else {
			conditions = append(conditions, "g.target IS NULL")
		}
	}
	if filters.Search != "" {
		conditions = append(conditions, "(LOWER(g.title) LIKE '%' || LOWER($search) || '%' OR LOWER(g.description) LIKE '%' || LOWER($search) || '%')")
		queryVars["search"] = filters.Search
	}

	whereClause := strings.Join(conditions, " AND ")

	// Determine sort order
	orderClause := "ORDER BY g.created_at DESC" // default
	if filters.SortBy != "" {
		sortField := filters.SortBy
		sortDir := "ASC"
		if strings.HasSuffix(sortField, "-desc") {
			sortField = strings.TrimSuffix(sortField, "-desc")
			sortDir = "DESC"
		} else if strings.HasSuffix(sortField, "-asc") {
			sortField = strings.TrimSuffix(sortField, "-asc")
			sortDir = "ASC"
		}
		switch sortField {
		case "title":
			orderClause = "ORDER BY g.title " + sortDir
		case "priority":
			orderClause = "ORDER BY g.priority " + sortDir
		case "updated_at":
			orderClause = "ORDER BY g.updated_at " + sortDir
		case "created_at":
			orderClause = "ORDER BY g.created_at " + sortDir
		default:
			orderClause = "ORDER BY g.created_at DESC"
		}
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM goals g WHERE " + whereClause
	total, err := database.QueryScalar[int64](ctx, r.db, countQuery, queryVars)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("count query failed")
		return nil, 0, err
	}

	// Build the time reference for progress filtering
	// Default is now(), but for timeline views we use progress_date
	timeRef := "'now'"
	if filters.ProgressDate != nil {
		queryVars["progress_date"] = filters.ProgressDate.UTC().Format(time.RFC3339Nano)
		timeRef = "$progress_date"
	}

	// Main query with computed stats and linked task IDs.
	// For recurring goals: prefer pre-computed goal_daily_stats; fallback to
	// computing from task_goals with a period filter.
	// For non-recurring: sum all contributions.
	// linked_task_ids is populated via a JSON aggregation over task_goals so
	// the UI can highlight tasks linked to this goal.
	dataQuery := `
		SELECT
			` + goalSelectColumns + `,
			JSON_OBJECT(
				'id', c.id, 'name', c.name, 'color', c.color,
				'created_at', c.created_at, 'updated_at', c.updated_at,
				'deleted_at', c.deleted_at, 'created_by', c.created_by
			) AS category,
			COALESCE((
				SELECT JSON_GROUP_ARRAY(tg.task_id)
				FROM task_goals tg
				WHERE tg.goal_id = g.id
			), '[]') AS linked_task_ids,
			CASE
				WHEN g.recurrence IS NOT NULL THEN
					COALESCE(
						(SELECT JSON_OBJECT(
							'total', daily_value,
							'count', contribution_count
						 )
						 FROM goal_daily_stats
						 WHERE goal_id = g.id AND date = date(` + timeRef + `)),
						(SELECT JSON_OBJECT(
							'total', COALESCE(SUM(tg.quantity_value), 0),
							'count', COUNT(*)
						 )
						 FROM task_goals tg
						 JOIN tasks t ON t.id = tg.task_id
						 WHERE tg.goal_id = g.id
						   AND (json_extract(g.target, '$.track_completed_only') IS NULL
						        OR json_extract(g.target, '$.track_completed_only') != 1
						        OR t.completed = 1)
						   AND (
						     (json_extract(g.recurrence, '$.period') = 'day'
						      AND date(COALESCE(t.completed_at, t.start_date)) = date(` + timeRef + `))
						     OR (json_extract(g.recurrence, '$.period') = 'week'
						      AND strftime('%W', COALESCE(t.completed_at, t.start_date)) = strftime('%W', ` + timeRef + `)
						      AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', ` + timeRef + `))
						     OR (json_extract(g.recurrence, '$.period') = 'month'
						      AND strftime('%m', COALESCE(t.completed_at, t.start_date)) = strftime('%m', ` + timeRef + `)
						      AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', ` + timeRef + `))
						   )
						),
						JSON_OBJECT('total', 0, 'count', 0)
					)
				ELSE
					COALESCE(
						(SELECT JSON_OBJECT(
							'total', COALESCE(SUM(tg.quantity_value), 0),
							'count', COUNT(*)
						 )
						 FROM task_goals tg
						 JOIN tasks t ON t.id = tg.task_id
						 WHERE tg.goal_id = g.id
						   AND (json_extract(g.target, '$.track_completed_only') IS NULL
						        OR json_extract(g.target, '$.track_completed_only') != 1
						        OR t.completed = 1)
						),
						JSON_OBJECT('total', 0, 'count', 0)
					)
			END AS filtered_stats,
			JSON_OBJECT(
				'total', (SELECT COUNT(*) FROM goal_children gc WHERE gc.parent_goal_id = g.id),
				'completed', (SELECT COUNT(*) FROM goal_children gc
							  JOIN goals cg ON cg.id = gc.child_goal_id
							  WHERE gc.parent_goal_id = g.id AND cg.status = 'completed')
			) AS children_stats
		FROM goals g
		LEFT JOIN categories c ON c.id = g.category_id
		WHERE ` + whereClause + `
		` + orderClause + `
		LIMIT $limit OFFSET $offset`

	goalsDB, err := database.QueryAll[goalDB](ctx, r.db, dataQuery, queryVars)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("list query failed")
		return nil, 0, err
	}

	goals := make([]*Goal, len(goalsDB))
	for i := range goalsDB {
		goals[i] = goalsDB[i].toGoal()
	}

	return goals, total, nil
}

func (r *repository) FindRecurringForDate(ctx context.Context, userID string, date time.Time) ([]*Goal, error) {
	dateStr := date.UTC().Format(time.RFC3339Nano)
	query := `
		SELECT
			` + goalSelectColumns + `,
			JSON_OBJECT(
				'id', c.id, 'name', c.name, 'color', c.color,
				'created_at', c.created_at, 'updated_at', c.updated_at,
				'deleted_at', c.deleted_at, 'created_by', c.created_by
			) AS category,
			COALESCE(
				(SELECT JSON_OBJECT(
					'total', daily_value,
					'count', contribution_count
				 )
				 FROM goal_daily_stats
				 WHERE goal_id = g.id AND date = date($date)),
				JSON_OBJECT('total', 0, 'count', 0)
			) AS filtered_stats
		FROM goals g
		LEFT JOIN categories c ON c.id = g.category_id
		WHERE g.created_by = $user
		  AND g.deleted_at IS NULL
		  AND g.status = 'active'
		  AND g.recurrence IS NOT NULL
	`
	goalsDB, err := database.QueryAll[goalDB](ctx, r.db, query, map[string]any{
		"user": userID,
		"date": dateStr,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("recurring goals query failed")
		return nil, err
	}

	goals := make([]*Goal, len(goalsDB))
	for i := range goalsDB {
		goals[i] = goalsDB[i].toGoal()
	}

	return goals, nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Goal, error) {
	now := time.Now().UTC()
	goalID := generateGoalRecordID()

	createData := map[string]any{
		"id":          goalID,
		"created_by":  userID,
		"title":       req.Title,
		"description": req.Description,
		"icon":        req.Icon,
		"status":      StatusActive,
		"priority":    req.Priority,
		"created_at":  now.Format(time.RFC3339Nano),
		"updated_at":  now.Format(time.RFC3339Nano),
	}

	// Handle recurrence
	if req.Recurrence != nil {
		recurrenceMap := map[string]any{
			"frequency":   req.Recurrence.Frequency,
			"period":      req.Recurrence.Period,
			"active_days": req.Recurrence.ActiveDays,
			"before_time": req.Recurrence.BeforeTime,
			"after_time":  req.Recurrence.AfterTime,
			"grace_days":  req.Recurrence.GraceDays,
		}
		encoded, err := json.Marshal(recurrenceMap)
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to encode recurrence")
			return nil, errors.ErrInternal.Wrap(err)
		}
		createData["recurrence"] = string(encoded)
	}

	// Handle target with operator
	if req.Target != nil {
		operator := req.Target.Operator
		if operator == "" {
			operator = DefaultOperator
		}
		targetMap := map[string]any{
			"value":                req.Target.Value,
			"unit_id":              req.Target.UnitID,
			"operator":             operator,
			"track_completed_only": req.Target.TrackCompletedOnly,
		}
		encoded, err := json.Marshal(targetMap)
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to encode target")
			return nil, errors.ErrInternal.Wrap(err)
		}
		createData["target"] = string(encoded)
	}

	// Handle start date
	if req.StartDate != nil {
		startDate, err := time.Parse(time.RFC3339, *req.StartDate)
		if err == nil {
			createData["start_date"] = startDate.UTC().Format(time.RFC3339Nano)
		}
	}

	// Handle deadline
	if req.Deadline != nil {
		deadline, err := time.Parse(time.RFC3339, *req.Deadline)
		if err == nil {
			createData["deadline"] = deadline.UTC().Format(time.RFC3339Nano)
		}
	}

	// Link category (direct column)
	if req.CategoryID != "" {
		createData["category_id"] = database.RecordID("categories", req.CategoryID)
	}

	// Create the goal via the typed Create helper
	_, err := database.Create[goalDB](ctx, r.db, Table, createData)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("create goal failed")
		return nil, err
	}

	// Link to parent goal if specified
	if req.ParentGoalID != "" {
		if err := r.AddChild(ctx, req.ParentGoalID, goalID, userID, 0, true); err != nil {
			r.logger.Warn().Err(err).Msg("failed to link to parent goal")
		}
	}

	r.logger.Info().Str("goal_id", goalID).Msg("goal created")

	return r.FindByID(ctx, goalID, userID)
}

// =============================================================================
// UPDATE OPERATIONS
// =============================================================================

func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Goal, error) {
	// Verify ownership
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	goalID := resolveGoalID(id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now.Format(time.RFC3339Nano),
	}

	if req.Title != nil {
		updateData["title"] = *req.Title
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}
	if req.Icon != nil {
		updateData["icon"] = *req.Icon
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
		if *req.Status == StatusCompleted {
			updateData["completed_at"] = now.Format(time.RFC3339Nano)
		}
	}
	if req.Priority != nil {
		updateData["priority"] = *req.Priority
	}

	if req.Recurrence != nil {
		recurrenceMap := map[string]any{
			"frequency":   req.Recurrence.Frequency,
			"period":      req.Recurrence.Period,
			"active_days": req.Recurrence.ActiveDays,
			"before_time": req.Recurrence.BeforeTime,
			"after_time":  req.Recurrence.AfterTime,
			"grace_days":  req.Recurrence.GraceDays,
		}
		encoded, err := json.Marshal(recurrenceMap)
		if err != nil {
			return nil, errors.ErrInternal.Wrap(err)
		}
		updateData["recurrence"] = string(encoded)
	}

	if req.Target != nil {
		operator := req.Target.Operator
		if operator == "" {
			operator = DefaultOperator
		}
		targetMap := map[string]any{
			"value":                req.Target.Value,
			"unit_id":              req.Target.UnitID,
			"operator":             operator,
			"track_completed_only": req.Target.TrackCompletedOnly,
		}
		encoded, err := json.Marshal(targetMap)
		if err != nil {
			return nil, errors.ErrInternal.Wrap(err)
		}
		updateData["target"] = string(encoded)
	}

	if _, err := database.Merge[goalDB](ctx, r.db, goalID, updateData); err != nil {
		r.logger.Error().Err(err).Str("goal_id", id).Msg("update goal failed")
		return nil, err
	}

	// Update category if provided
	if req.CategoryID != nil {
		if err := r.UpdateCategory(ctx, id, *req.CategoryID, userID); err != nil {
			r.logger.Warn().Err(err).Msg("failed to update category")
		}
	}

	r.logger.Info().Str("goal_id", id).Msg("goal updated")

	return r.FindByID(ctx, id, userID)
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

func (r *repository) Delete(ctx context.Context, id, userID string) error {
	// Verify ownership
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	goalID := resolveGoalID(id)
	now := time.Now().UTC()

	_, err = database.Merge[goalDB](ctx, r.db, goalID, map[string]any{
		"deleted_at": now.Format(time.RFC3339Nano),
		"updated_at": now.Format(time.RFC3339Nano),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", id).Msg("soft delete goal failed")
		return err
	}

	r.logger.Info().Str("goal_id", id).Msg("goal deleted")
	return nil
}

// =============================================================================
// CHILD GOALS OPERATIONS (via goal_children join table)
// =============================================================================

func (r *repository) FindChildren(ctx context.Context, parentGoalID, userID string) ([]*Goal, error) {
	parentID := resolveGoalID(parentGoalID)

	query := `
		SELECT
			` + goalSelectColumns + `,
			JSON_OBJECT(
				'id', c.id, 'name', c.name, 'color', c.color,
				'created_at', c.created_at, 'updated_at', c.updated_at,
				'deleted_at', c.deleted_at, 'created_by', c.created_by
			) AS category
		FROM goal_children gc
		JOIN goals g ON g.id = gc.child_goal_id
		LEFT JOIN categories c ON c.id = g.category_id
		WHERE gc.parent_goal_id = $parent_id
		  AND g.created_by = $user
		  AND g.deleted_at IS NULL
		ORDER BY gc.sort_order ASC
	`
	goalsDB, err := database.QueryAll[goalDB](ctx, r.db, query, map[string]any{
		"parent_id": parentID,
		"user":      userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("parent_goal_id", parentGoalID).Msg("find child goals failed")
		return nil, err
	}

	goals := make([]*Goal, len(goalsDB))
	for i := range goalsDB {
		goals[i] = goalsDB[i].toGoal()
	}

	return goals, nil
}

func (r *repository) FindTasksForGoal(ctx context.Context, goalID, userID string) ([]GoalTaskLink, error) {
	gID := resolveGoalID(goalID)

	type taskCategoryDB struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	type taskLinkDB struct {
		TaskID        string          `json:"task_id"`
		TaskTitle     string          `json:"task_title"`
		ImpactType    string          `json:"impact_type"`
		QuantityValue *float64        `json:"quantity_value,omitempty"`
		UnitID        *string         `json:"unit_id,omitempty"`
		TaskJournal   string          `json:"task_journal,omitempty"`
		TaskStartDate *time.Time      `json:"task_start_date,omitempty"`
		TaskEndDate   *time.Time      `json:"task_end_date,omitempty"`
		TaskCompleted bool            `json:"task_completed"`
		TaskEmotionID *string         `json:"task_emotion_id,omitempty"`
		TaskCategory  *taskCategoryDB `json:"task_category,omitempty"`
		LinkedAt      *time.Time      `json:"linked_at,omitempty"`
		Notes         string          `json:"notes,omitempty"`
	}

	query := `
		SELECT
			tg.task_id AS task_id,
			t.title AS task_title,
			tg.impact_type,
			tg.quantity_value,
			tg.unit_id,
			t.journal AS task_journal,
			t.start_date AS task_start_date,
			t.end_date AS task_end_date,
			CASE WHEN t.completed = 1 THEN 1 ELSE 0 END AS task_completed,
			t.emotion_id AS task_emotion_id,
			CASE WHEN t.category_id IS NOT NULL THEN
				JSON_OBJECT(
					'id', tc.id, 'name', tc.name, 'color', tc.color
				)
			ELSE NULL END AS task_category,
			tg.created_at AS linked_at,
			tg.notes
		FROM task_goals tg
		JOIN tasks t ON t.id = tg.task_id
		LEFT JOIN categories tc ON tc.id = t.category_id
		WHERE tg.goal_id = $goal_id
		  AND t.deleted_at IS NULL
	`
	tasksDB, err := database.QueryAll[taskLinkDB](ctx, r.db, query, map[string]any{
		"goal_id": gID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("find tasks for goal failed")
		return nil, err
	}

	tasks := make([]GoalTaskLink, len(tasksDB))
	for i, t := range tasksDB {
		link := GoalTaskLink{
			TaskID:        t.TaskID,
			TaskTitle:     t.TaskTitle,
			ImpactType:    t.ImpactType,
			QuantityValue: t.QuantityValue,
			UnitID:        t.UnitID,
			TaskJournal:   t.TaskJournal,
			TaskCompleted: t.TaskCompleted,
			TaskEmotionID: t.TaskEmotionID,
			Notes:         t.Notes,
		}

		link.TaskStartDate = t.TaskStartDate
		link.TaskEndDate = t.TaskEndDate
		link.LinkedAt = t.LinkedAt

		if t.TaskCategory != nil {
			link.TaskCategory = &struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			}{
				ID:    t.TaskCategory.ID,
				Name:  t.TaskCategory.Name,
				Color: t.TaskCategory.Color,
			}
		}

		tasks[i] = link
	}

	return tasks, nil
}

func (r *repository) AddChild(ctx context.Context, parentID, childID, userID string, order int, required bool) error {
	pID := resolveGoalID(parentID)
	cID := resolveGoalID(childID)
	now := time.Now().UTC()

	r.logger.Debug().
		Str("parent_id", parentID).
		Str("child_id", childID).
		Str("pID", pID).
		Str("cID", cID).
		Msg("adding child goal")

	requiredInt := 0
	if required {
		requiredInt = 1
	}

	_, err := r.db.SQL().ExecContext(ctx, `
		INSERT INTO goal_children (parent_goal_id, child_goal_id, sort_order, required, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(parent_goal_id, child_goal_id) DO UPDATE SET
			sort_order = excluded.sort_order,
			required = excluded.required
	`, pID, cID, order, requiredInt, now.Format(time.RFC3339Nano))
	if err != nil {
		r.logger.Error().Err(err).Str("parent", parentID).Str("child", childID).Msg("add child goal failed")
		return err
	}

	return nil
}

func (r *repository) RemoveChild(ctx context.Context, parentID, childID, userID string) error {
	pID := resolveGoalID(parentID)
	cID := resolveGoalID(childID)

	_, err := r.db.SQL().ExecContext(ctx,
		`DELETE FROM goal_children WHERE parent_goal_id = ? AND child_goal_id = ?`,
		pID, cID,
	)
	if err != nil {
		r.logger.Error().Err(err).Str("parent", parentID).Str("child", childID).Msg("remove child goal failed")
		return err
	}

	return nil
}

// =============================================================================
// CATEGORY OPERATIONS (direct column on goals)
// =============================================================================

func (r *repository) UpdateCategory(ctx context.Context, goalID, categoryID, userID string) error {
	gID := resolveGoalID(goalID)

	var catVal any
	if categoryID != "" {
		catVal = database.RecordID("categories", categoryID)
	}

	_, err := database.Merge[goalDB](ctx, r.db, gID, map[string]any{
		"category_id": catVal,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal", goalID).Str("category", categoryID).Msg("update category failed")
		return err
	}

	return nil
}

// =============================================================================
// STATS COMPUTATION
// =============================================================================

func (r *repository) ComputeStats(ctx context.Context, goalID, userID string) (*GoalStats, error) {
	gID := resolveGoalID(goalID)

	// Get goal first (includes denormalized streak fields)
	goal, err := r.FindByID(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	stats := &GoalStats{}
	if goal.Stats != nil {
		stats.CurrentStreak = goal.Stats.CurrentStreak
		stats.LongestStreak = goal.Stats.LongestStreak
		stats.LastCompletedDate = goal.Stats.LastCompletedDate
	}

	// For recurring goals: try pre-computed goal_daily_stats, fallback to computing
	// For non-recurring goals: aggregate from task_goals join table
	if goal.Recurrence != nil {
		// Recurring goal: try pre-computed daily stats first (O(1))
		dailyStats, err := database.QueryFirst[struct {
			DailyValue        float64 `json:"daily_value"`
			ContributionCount int     `json:"contribution_count"`
		}](ctx, r.db, `
			SELECT daily_value, contribution_count
			FROM goal_daily_stats
			WHERE goal_id = $goal_id AND date = date('now')
		`, map[string]any{
			"goal_id": gID,
		})
		if err == nil && dailyStats != nil && (dailyStats.DailyValue > 0 || dailyStats.ContributionCount > 0) {
			stats.CurrentValue = dailyStats.DailyValue
			stats.TotalContributions = dailyStats.ContributionCount
		} else {
			// Fallback: compute from task_goals with period filter
			var periodCondition string
			switch goal.Recurrence.Period {
			case PeriodDay:
				periodCondition = "date(COALESCE(t.completed_at, t.start_date)) = date('now')"
			case PeriodWeek:
				periodCondition = "strftime('%W', COALESCE(t.completed_at, t.start_date)) = strftime('%W', 'now') AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', 'now')"
			case PeriodMonth:
				periodCondition = "strftime('%m', COALESCE(t.completed_at, t.start_date)) = strftime('%m', 'now') AND strftime('%Y', COALESCE(t.completed_at, t.start_date)) = strftime('%Y', 'now')"
			}

			conditions := []string{periodCondition}
			if goal.Target != nil && goal.Target.TrackCompletedOnly {
				conditions = append(conditions, "t.completed = 1")
			}

			computed, err := database.QueryFirst[struct {
				Total float64 `json:"total"`
				Count int     `json:"count"`
			}](ctx, r.db, `
				SELECT
					COALESCE(SUM(tg.quantity_value), 0) AS total,
					COUNT(*) AS count
				FROM task_goals tg
				JOIN tasks t ON t.id = tg.task_id
				WHERE tg.goal_id = $goal_id
				  AND `+strings.Join(conditions, " AND ")+`
			`, map[string]any{
				"goal_id": gID,
			})
			if err == nil && computed != nil {
				stats.CurrentValue = computed.Total
				stats.TotalContributions = computed.Count
			}
		}
	} else {
		// Non-recurring goal: aggregate all contributions
		conditions := []string{"tg.goal_id = $goal_id"}
		if goal.Target != nil && goal.Target.TrackCompletedOnly {
			conditions = append(conditions, "t.completed = 1")
		}

		sumResult, err := database.QueryFirst[struct {
			Total float64 `json:"total"`
			Count int     `json:"count"`
		}](ctx, r.db, `
			SELECT
				COALESCE(SUM(tg.quantity_value), 0) AS total,
				COUNT(*) AS count
			FROM task_goals tg
			JOIN tasks t ON t.id = tg.task_id
			WHERE `+strings.Join(conditions, " AND ")+`
		`, map[string]any{
			"goal_id": gID,
		})
		if err == nil && sumResult != nil {
			stats.CurrentValue = sumResult.Total
			stats.TotalContributions = sumResult.Count
		}
	}

	// Calculate progress percentage and today status
	if goal.Target != nil && goal.Target.Value > 0 {
		stats.ProgressPercent = (stats.CurrentValue / goal.Target.Value) * 100

		// Determine today status based on operator
		switch goal.Target.Operator {
		case OperatorGTE:
			if stats.CurrentValue >= goal.Target.Value {
				stats.TodayStatus = TodayStatusMet
			} else {
				stats.TodayStatus = TodayStatusPending
			}
		case OperatorLTE:
			if stats.CurrentValue <= goal.Target.Value {
				stats.TodayStatus = TodayStatusMet
			} else {
				stats.TodayStatus = TodayStatusExceeded
			}
		case OperatorEQ:
			switch {
			case stats.CurrentValue == goal.Target.Value:
				stats.TodayStatus = TodayStatusMet
			case stats.CurrentValue > goal.Target.Value:
				stats.TodayStatus = TodayStatusExceeded
			default:
				stats.TodayStatus = TodayStatusPending
			}
		}
	}

	// Count children if this is a grouped goal
	childrenResult, err := database.QueryFirst[struct {
		Total     int `json:"total"`
		Completed int `json:"completed"`
	}](ctx, r.db, `
		SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN cg.status = 'completed' THEN 1 ELSE 0 END) AS completed
		FROM goal_children gc
		LEFT JOIN goals cg ON cg.id = gc.child_goal_id
		WHERE gc.parent_goal_id = $goal_id
	`, map[string]any{
		"goal_id": gID,
	})
	if err == nil && childrenResult != nil {
		stats.ChildrenTotal = childrenResult.Total
		stats.ChildrenCompleted = childrenResult.Completed
	}

	return stats, nil
}

// =============================================================================
// STREAK OPERATIONS
// =============================================================================

// UpdateStreaks updates the denormalized streak fields directly on the goal.
// This is called when tasks are completed or habit entries are logged.
func (r *repository) UpdateStreaks(ctx context.Context, goalID string, currentStreak, longestStreak int, lastCompleted *time.Time) error {
	gID := resolveGoalID(goalID)
	now := time.Now().UTC()

	updateData := map[string]any{
		"current_streak": currentStreak,
		"longest_streak": longestStreak,
		"updated_at":     now.Format(time.RFC3339Nano),
	}

	if lastCompleted != nil {
		updateData["last_completed_date"] = lastCompleted.UTC().Format(time.RFC3339Nano)
	}

	_, err := database.Merge[goalDB](ctx, r.db, gID, updateData)
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", goalID).Msg("update streaks failed")
		return err
	}

	r.logger.Debug().
		Str("goal_id", goalID).
		Int("current_streak", currentStreak).
		Int("longest_streak", longestStreak).
		Msg("goal streaks updated")

	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// generateGoalRecordID generates a unique goal ID as a "goals:<hex>" string.
func generateGoalRecordID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes) //nolint:errcheck // crypto/rand.Read never fails in practice
	return database.RecordID(Table, hex.EncodeToString(bytes))
}
