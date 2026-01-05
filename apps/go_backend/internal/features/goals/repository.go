// Package goals provides goal management functionality using SurrealDB SDK.
//
// This package implements:
//   - CRUD operations for goals using typed SDK methods
//   - Graph-inferred goal nature (no goal_type enum)
//   - Category linking via in_category relation
//   - Child goals via goal_children relation
//   - Soft delete support
//   - Pagination with filtering
//
// Database Architecture:
//   - in_category: RELATE table for category assignment
//   - goal_children: RELATE table for parent-child relationships
//   - goal_logs: RELATE table for history tracking
package goals

import (
	"context"
	"strings"
	"time"

	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go/pkg/models"
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
}

// GoalFilters contains optional filters for listing goals.
type GoalFilters struct {
	Status      string // Filter by status (active, completed, archived)
	IsRecurring *bool  // Filter recurring (true) vs one-time (false)
	HasTarget   *bool  // Filter measurable goals
	HasChildren *bool  // Filter grouped goals
	Search      string // Search in title and description
	SortBy      string // Sort field with optional -desc suffix
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
type goalDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	CreatedBy string          `json:"created_by"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`

	Recurrence map[string]any `json:"recurrence,omitempty"`
	Target     map[string]any `json:"target,omitempty"`

	StartDate   *database.SurrealTime `json:"start_date,omitempty"`
	Deadline    *database.SurrealTime `json:"deadline,omitempty"`
	CompletedAt *database.SurrealTime `json:"completed_at,omitempty"`

	Status   string `json:"status"`
	Priority int    `json:"priority"`

	// Denormalized streak fields (stored for fast reads)
	CurrentStreak     int                   `json:"current_streak"`
	LongestStreak     int                   `json:"longest_streak"`
	LastCompletedDate *database.SurrealTime `json:"last_completed_date,omitempty"`

	// Linked tasks (populated via subquery)
	LinkedTasks []goalTaskDB `json:"linked_tasks,omitempty"`

	// Category (populated via in_category edge)
	Category *categoryDB `json:"category,omitempty"`

	// Children (populated via goal_children edge)
	Children []goalChildDB `json:"children,omitempty"`

	CreatedAt database.SurrealTime  `json:"created_at"`
	UpdatedAt database.SurrealTime  `json:"updated_at"`
	DeletedAt *database.SurrealTime `json:"deleted_at,omitempty"`
}

// goalTaskDB represents a linked task from the task_goals relation.
type goalTaskDB struct {
	TaskID          string   `json:"task_id"`
	TaskTitle       string   `json:"task_title"`
	ImpactType      string   `json:"impact_type"`
	ImpactMagnitude int      `json:"impact_magnitude"`
	QuantityValue   *float64 `json:"quantity_value,omitempty"`
	UnitID          *string  `json:"unit_id,omitempty"`
}

// goalChildDB represents a child goal link.
type goalChildDB struct {
	GoalID   string `json:"goal_id"`
	Order    int    `json:"order"`
	Required bool   `json:"required"`
}

// categoryDB is the database representation of a category when fetched.
type categoryDB struct {
	ID        models.RecordID       `json:"id,omitempty"`
	Name      string                `json:"name"`
	Color     string                `json:"color"`
	CreatedAt database.SurrealTime  `json:"created_at"`
	UpdatedAt database.SurrealTime  `json:"updated_at"`
	DeletedAt *database.SurrealTime `json:"deleted_at,omitempty"`
	CreatedBy string                `json:"created_by"`
}

func (c *categoryDB) toCategory() *categories.Category {
	if c == nil {
		return nil
	}
	var deletedAt *time.Time
	if c.DeletedAt != nil && !c.DeletedAt.IsZero() {
		dt := c.DeletedAt.Time
		deletedAt = &dt
	}
	return &categories.Category{
		ID:        database.ToStringID(c.ID),
		Name:      c.Name,
		Color:     c.Color,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
		DeletedAt: deletedAt,
		CreatedBy: c.CreatedBy,
	}
}

// toGoal converts the database model to the domain model.
func (g *goalDB) toGoal() *Goal {
	goal := &Goal{
		ID:        database.ToStringID(g.ID),
		CreatedBy: g.CreatedBy,

		Title:       g.Title,
		Description: g.Description,
		Icon:        g.Icon,

		Status:   g.Status,
		Priority: g.Priority,

		// Denormalized streak fields (stored for fast reads)
		CurrentStreak: g.CurrentStreak,
		LongestStreak: g.LongestStreak,

		CreatedAt: g.CreatedAt.Time,
		UpdatedAt: g.UpdatedAt.Time,
	}

	// Convert last completed date
	if g.LastCompletedDate != nil && !g.LastCompletedDate.IsZero() {
		t := g.LastCompletedDate.Time
		goal.LastCompletedDate = &t
	}

	// Convert category
	if g.Category != nil {
		goal.Category = g.Category.toCategory()
	}

	// Convert dates
	if g.StartDate != nil && !g.StartDate.IsZero() {
		t := g.StartDate.Time
		goal.StartDate = &t
	}
	if g.Deadline != nil && !g.Deadline.IsZero() {
		t := g.Deadline.Time
		goal.Deadline = &t
	}
	if g.CompletedAt != nil && !g.CompletedAt.IsZero() {
		t := g.CompletedAt.Time
		goal.CompletedAt = &t
	}
	if g.DeletedAt != nil && !g.DeletedAt.IsZero() {
		t := g.DeletedAt.Time
		goal.DeletedAt = &t
	}

	// Convert recurrence
	if g.Recurrence != nil {
		goal.Recurrence = mapToRecurrence(g.Recurrence)
	}

	// Convert target
	if g.Target != nil {
		goal.Target = mapToTarget(g.Target)
	}

	// Convert linked tasks
	if len(g.LinkedTasks) > 0 {
		goal.LinkedTasks = make([]GoalTaskLink, len(g.LinkedTasks))
		for i, lt := range g.LinkedTasks {
			goal.LinkedTasks[i] = GoalTaskLink(lt)
		}
	}

	return goal
}

func mapToRecurrence(m map[string]any) *Recurrence {
	if m == nil {
		return nil
	}
	r := &Recurrence{}
	if v, ok := m["frequency"].(float64); ok {
		r.Frequency = int(v)
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
	if v, ok := m["grace_days"].(float64); ok {
		r.GraceDays = int(v)
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
	if v, ok := m["value"].(float64); ok {
		t.Value = v
	}
	if v, ok := m["unit_id"].(string); ok {
		t.UnitID = v
	}
	if v, ok := m["operator"].(string); ok {
		t.Operator = v
	}
	if v, ok := m["per_period"].(bool); ok {
		t.PerPeriod = v
	}
	return t
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByID(ctx context.Context, id, userID string) (*Goal, error) {
	goalID := database.MustRecordID(Table, id)

	goal, err := database.QueryFirst[goalDB](ctx, r.db, `
		SELECT *,
			(SELECT 
				type::string(in) as task_id,
				in.title as task_title,
				impact_type,
				impact_magnitude,
				quantity_value,
				unit_id
			 FROM task_goals WHERE out = $parent.id) as linked_tasks,
			(SELECT out as category FROM in_category WHERE in = $parent.id)[0].category as category
		FROM type::thing($id)
	`, map[string]any{
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
	conditions := []string{"created_by = $user", "deleted_at IS NONE"}
	queryVars := map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	}

	if filters.Status != "" {
		conditions = append(conditions, "status = $status")
		queryVars["status"] = filters.Status
	}
	if filters.IsRecurring != nil {
		if *filters.IsRecurring {
			conditions = append(conditions, "recurrence IS NOT NONE")
		} else {
			conditions = append(conditions, "recurrence IS NONE")
		}
	}
	if filters.HasTarget != nil {
		if *filters.HasTarget {
			conditions = append(conditions, "target IS NOT NONE")
		} else {
			conditions = append(conditions, "target IS NONE")
		}
	}
	if filters.Search != "" {
		conditions = append(conditions, "(string::lowercase(title) CONTAINS string::lowercase($search) OR string::lowercase(description) CONTAINS string::lowercase($search))")
		queryVars["search"] = filters.Search
	}

	whereClause := strings.Join(conditions, " AND ")

	// Determine sort order
	orderClause := "ORDER BY created_at DESC" // default
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
			orderClause = "ORDER BY title " + sortDir
		case "priority":
			orderClause = "ORDER BY priority " + sortDir
		case "updated_at":
			orderClause = "ORDER BY updated_at " + sortDir
		case "created_at":
			orderClause = "ORDER BY created_at " + sortDir
		default:
			orderClause = "ORDER BY created_at DESC"
		}
	}

	// Count query
	countQuery := "RETURN (SELECT count() FROM goals WHERE " + whereClause + " GROUP ALL)[0].count OR 0"
	total, err := database.QueryScalar[float64](ctx, r.db, countQuery, queryVars)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("count query failed")
		return nil, 0, err
	}

	// Main query with linked tasks subquery
	dataQuery := `SELECT *, 
		(SELECT type::string(in) as task_id, in.title as task_title, impact_type, impact_magnitude, quantity_value, unit_id FROM task_goals WHERE out = $parent.id) as linked_tasks,
		(SELECT out as category FROM in_category WHERE in = $parent.id)[0].category as category
		FROM goals WHERE ` + whereClause + " " + orderClause + " LIMIT $limit START $offset"
	goalsDB, err := database.QueryAll[goalDB](ctx, r.db, dataQuery, queryVars)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("list query failed")
		return nil, 0, err
	}

	goals := make([]*Goal, len(goalsDB))
	for i := range goalsDB {
		goals[i] = goalsDB[i].toGoal()
	}

	return goals, int64(total), nil
}

func (r *repository) FindRecurringForDate(ctx context.Context, userID string, date time.Time) ([]*Goal, error) {
	goalsDB, err := database.QueryAll[goalDB](ctx, r.db, `
		SELECT * FROM goals 
		WHERE created_by = $user 
		  AND deleted_at IS NONE
		  AND status = "active"
		  AND recurrence IS NOT NONE
	`, map[string]any{
		"user": userID,
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

	createData := map[string]any{
		"created_by":  userID,
		"title":       req.Title,
		"description": req.Description,
		"icon":        req.Icon,
		"status":      StatusActive,
		"priority":    req.Priority,
		"created_at":  now,
		"updated_at":  now,
	}

	// Handle recurrence
	if req.Recurrence != nil {
		createData["recurrence"] = map[string]any{
			"frequency":   req.Recurrence.Frequency,
			"period":      req.Recurrence.Period,
			"active_days": req.Recurrence.ActiveDays,
			"before_time": req.Recurrence.BeforeTime,
			"after_time":  req.Recurrence.AfterTime,
			"grace_days":  req.Recurrence.GraceDays,
		}
	}

	// Handle target with operator
	if req.Target != nil {
		operator := req.Target.Operator
		if operator == "" {
			operator = DefaultOperator
		}
		createData["target"] = map[string]any{
			"value":      req.Target.Value,
			"unit_id":    req.Target.UnitID,
			"operator":   operator,
			"per_period": req.Target.PerPeriod,
		}
	}

	// Handle start date
	if req.StartDate != nil {
		startDate, err := time.Parse(time.RFC3339, *req.StartDate)
		if err == nil {
			createData["start_date"] = startDate
		}
	}

	// Handle deadline
	if req.Deadline != nil {
		deadline, err := time.Parse(time.RFC3339, *req.Deadline)
		if err == nil {
			createData["deadline"] = deadline
		}
	}

	// Create the goal
	result, err := database.QueryFirst[goalDB](ctx, r.db, `
		CREATE goals CONTENT $data
	`, map[string]any{
		"data": createData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("create goal failed")
		return nil, err
	}

	goalID := database.ToStringID(result.ID)

	// Link category via in_category relation
	if req.CategoryID != "" {
		if err := r.UpdateCategory(ctx, goalID, req.CategoryID, userID); err != nil {
			r.logger.Warn().Err(err).Msg("failed to link category")
		}
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

	goalID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now,
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
			updateData["completed_at"] = now
		}
	}
	if req.Priority != nil {
		updateData["priority"] = *req.Priority
	}

	if req.Recurrence != nil {
		updateData["recurrence"] = map[string]any{
			"frequency":   req.Recurrence.Frequency,
			"period":      req.Recurrence.Period,
			"active_days": req.Recurrence.ActiveDays,
			"before_time": req.Recurrence.BeforeTime,
			"after_time":  req.Recurrence.AfterTime,
			"grace_days":  req.Recurrence.GraceDays,
		}
	}

	if req.Target != nil {
		operator := req.Target.Operator
		if operator == "" {
			operator = DefaultOperator
		}
		updateData["target"] = map[string]any{
			"value":      req.Target.Value,
			"unit_id":    req.Target.UnitID,
			"operator":   operator,
			"per_period": req.Target.PerPeriod,
		}
	}

	_, err = database.QueryAll[goalDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE $data
	`, map[string]any{
		"id":   goalID,
		"data": updateData,
	})
	if err != nil {
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

	goalID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err = database.QueryAll[goalDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			deleted_at: $now,
			updated_at: $now
		}
	`, map[string]any{
		"id":  goalID,
		"now": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", id).Msg("soft delete goal failed")
		return err
	}

	r.logger.Info().Str("goal_id", id).Msg("goal deleted")
	return nil
}

// =============================================================================
// CHILD GOALS OPERATIONS (via goal_children relation)
// =============================================================================

func (r *repository) FindChildren(ctx context.Context, parentGoalID, userID string) ([]*Goal, error) {
	parentID := database.MustRecordID(Table, parentGoalID)

	goalsDB, err := database.QueryAll[goalDB](ctx, r.db, `
		SELECT out.* FROM goal_children 
		WHERE in = $parent_id AND out.created_by = $user AND out.deleted_at IS NONE
		ORDER BY order ASC
	`, map[string]any{
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

func (r *repository) AddChild(ctx context.Context, parentID, childID, userID string, order int, required bool) error {
	pID := database.MustRecordID(Table, parentID)
	cID := database.MustRecordID(Table, childID)
	now := time.Now().UTC()

	_, err := database.QueryAll[any](ctx, r.db, `
		RELATE $parent -> goal_children -> $child SET {
			order: $order,
			required: $required,
			created_by: $user,
			created_at: $now
		}
	`, map[string]any{
		"parent":   pID,
		"child":    cID,
		"order":    order,
		"required": required,
		"user":     userID,
		"now":      now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("parent", parentID).Str("child", childID).Msg("add child goal failed")
		return err
	}

	return nil
}

func (r *repository) RemoveChild(ctx context.Context, parentID, childID, userID string) error {
	pID := database.MustRecordID(Table, parentID)
	cID := database.MustRecordID(Table, childID)

	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE goal_children WHERE in = $parent AND out = $child
	`, map[string]any{
		"parent": pID,
		"child":  cID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("parent", parentID).Str("child", childID).Msg("remove child goal failed")
		return err
	}

	return nil
}

// =============================================================================
// CATEGORY OPERATIONS (via in_category relation)
// =============================================================================

func (r *repository) UpdateCategory(ctx context.Context, goalID, categoryID, userID string) error {
	gID := database.MustRecordID(Table, goalID)
	now := time.Now().UTC()

	// First, remove existing category link
	_, _ = database.QueryAll[any](ctx, r.db, `
		DELETE in_category WHERE in = $goal_id
	`, map[string]any{
		"goal_id": gID,
	})

	// If empty category ID, just remove
	if categoryID == "" {
		return nil
	}

	cID := database.MustRecordID("categories", categoryID)

	// Create new category link
	_, err := database.QueryAll[any](ctx, r.db, `
		RELATE $goal -> in_category -> $category SET {
			created_by: $user,
			created_at: $now
		}
	`, map[string]any{
		"goal":     gID,
		"category": cID,
		"user":     userID,
		"now":      now,
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
	gID := database.MustRecordID(Table, goalID)

	// Get goal first (includes denormalized streak fields)
	goal, err := r.FindByID(ctx, goalID, userID)
	if err != nil {
		return nil, err
	}

	stats := &GoalStats{
		// Use denormalized streak values from goal record (materialized view pattern)
		CurrentStreak:     goal.CurrentStreak,
		LongestStreak:     goal.LongestStreak,
		LastCompletedDate: goal.LastCompletedDate,
	}

	// Sum quantity values from task_goals
	sumResult, err := database.QueryFirst[struct {
		Total float64 `json:"total"`
		Count int     `json:"count"`
	}](ctx, r.db, `
		SELECT 
			math::sum(quantity_value) as total,
			count() as count
		FROM task_goals 
		WHERE out = $goal_id
		GROUP ALL
	`, map[string]any{
		"goal_id": gID,
	})
	if err == nil && sumResult != nil {
		stats.CurrentValue = sumResult.Total
		stats.TotalContributions = sumResult.Count
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
			if stats.CurrentValue == goal.Target.Value {
				stats.TodayStatus = TodayStatusMet
			} else if stats.CurrentValue > goal.Target.Value {
				stats.TodayStatus = TodayStatusExceeded
			} else {
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
			count() as total,
			count(out.status = "completed") as completed
		FROM goal_children 
		WHERE in = $goal_id
		GROUP ALL
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
	gID := database.MustRecordID(Table, goalID)
	now := time.Now().UTC()

	updateData := map[string]any{
		"current_streak": currentStreak,
		"longest_streak": longestStreak,
		"updated_at":     now,
	}

	if lastCompleted != nil {
		updateData["last_completed_date"] = *lastCompleted
	}

	_, err := database.QueryAll[goalDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE $data
	`, map[string]any{
		"id":   gID,
		"data": updateData,
	})
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
