// Package goals provides goal management functionality using SurrealDB SDK.
//
// This package implements:
//   - CRUD operations for goals using typed SDK methods
//   - Activity key generation and lookup
//   - Category linking (record links)
//   - Soft delete support
//   - Pagination with filtering
//
// SDK Methods Used:
//   - database.QueryFirst[T]() - Single record queries
//   - database.QueryAll[T]() - Multi-record queries
//   - database.QueryScalar[T]() - Scalar value queries
package goals

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
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

	// FindByActivityKey retrieves a goal by its activity key.
	FindByActivityKey(ctx context.Context, activityKey, userID string) (*Goal, error)

	// FindPaginated retrieves goals for a user with pagination and filters.
	FindPaginated(ctx context.Context, userID string, params pagination.Params, filters GoalFilters) ([]*Goal, int64, error)

	// FindRecurringForDate retrieves all recurring goals for a user that should be tracked on a given date.
	FindRecurringForDate(ctx context.Context, userID string, date time.Time) ([]*Goal, error)

	// Create creates a new goal.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Goal, error)

	// Update updates an existing goal.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Goal, error)

	// UpdateStreak updates streak-related fields.
	UpdateStreak(ctx context.Context, id string, currentStreak, longestStreak int, lastCompleted *time.Time, userID string) error

	// UpdateLinkedTemplate sets the linked template ID.
	UpdateLinkedTemplate(ctx context.Context, id string, templateID string, userID string) error

	// Delete soft-deletes a goal.
	Delete(ctx context.Context, id, userID string) error

	// FindChildGoals retrieves all child goals of a parent goal.
	FindChildGoals(ctx context.Context, parentGoalID, userID string) ([]*Goal, error)

	// UpdateStatus updates a goal's status (for auto-completion).
	UpdateStatus(ctx context.Context, id, status, userID string) error
}

// GoalFilters contains optional filters for listing goals.
type GoalFilters struct {
	Status      string // Filter by status
	GoalType    string // Filter by goal type
	LifeDomain  string // Filter by life domain
	IsRecurring *bool  // Filter recurring (true) vs one-time (false)
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
	ID          models.RecordID `json:"id,omitempty"`
	CreatedBy   string          `json:"created_by"`
	ActivityKey string          `json:"activity_key"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Why         string `json:"why,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`

	GoalType string `json:"goal_type"`

	Recurrence map[string]any `json:"recurrence,omitempty"`
	Target     map[string]any `json:"target,omitempty"`

	StartDate *database.SurrealTime `json:"start_date,omitempty"`
	Deadline  *database.SurrealTime `json:"deadline,omitempty"`

	Status         string                `json:"status"`
	CompletionDate *database.SurrealTime `json:"completion_date,omitempty"`

	CurrentStreak     int                   `json:"current_streak"`
	LongestStreak     int                   `json:"longest_streak"`
	LastCompletedDate *database.SurrealTime `json:"last_completed_date,omitempty"`
	GraceDaysUsed     int                   `json:"grace_days_used"`

	Priority       int         `json:"priority"`
	ValueScore     int         `json:"value_score"`
	Category       *categoryDB `json:"category,omitempty"`
	ParentGoal     *string     `json:"parent_goal,omitempty"`
	LifeDomain     string      `json:"life_domain,omitempty"`
	LinkedTemplate *string     `json:"linked_template,omitempty"`
	CompletionMode string      `json:"completion_mode,omitempty"`

	IsPrivate bool `json:"is_private"`

	// Linked tasks (populated via subquery)
	LinkedTasks []goalTaskDB `json:"linked_tasks,omitempty"`

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
	QuantityUnit    *string  `json:"quantity_unit,omitempty"`
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
		ID:          database.ToStringID(g.ID),
		CreatedBy:   g.CreatedBy,
		ActivityKey: g.ActivityKey,

		Title:       g.Title,
		Description: g.Description,
		Why:         g.Why,
		Icon:        g.Icon,
		Color:       g.Color,

		GoalType: g.GoalType,
		Status:   g.Status,

		CurrentStreak: g.CurrentStreak,
		LongestStreak: g.LongestStreak,
		GraceDaysUsed: g.GraceDaysUsed,

		Priority:       g.Priority,
		ValueScore:     g.ValueScore,
		ParentGoal:     g.ParentGoal,
		LifeDomain:     g.LifeDomain,
		LinkedTemplate: g.LinkedTemplate,
		CompletionMode: g.CompletionMode,

		IsPrivate: g.IsPrivate,

		CreatedAt: g.CreatedAt.Time,
		UpdatedAt: g.UpdatedAt.Time,
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
	if g.CompletionDate != nil && !g.CompletionDate.IsZero() {
		t := g.CompletionDate.Time
		goal.CompletionDate = &t
	}
	if g.LastCompletedDate != nil && !g.LastCompletedDate.IsZero() {
		t := g.LastCompletedDate.Time
		goal.LastCompletedDate = &t
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
			goal.LinkedTasks[i] = GoalTaskLink{
				TaskID:          lt.TaskID,
				TaskTitle:       lt.TaskTitle,
				ImpactType:      lt.ImpactType,
				ImpactMagnitude: lt.ImpactMagnitude,
				QuantityValue:   lt.QuantityValue,
				QuantityUnit:    lt.QuantityUnit,
			}
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
	t := &Target{}
	if v, ok := m["value"].(float64); ok {
		t.Value = v
	}
	if v, ok := m["unit"].(string); ok {
		t.Unit = v
	}
	if v, ok := m["current_value"].(float64); ok {
		t.CurrentValue = v
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
				quantity_unit
			 FROM task_goals WHERE out = $parent.id) as linked_tasks
		FROM type::thing($id) FETCH category
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

func (r *repository) FindByActivityKey(ctx context.Context, activityKey, userID string) (*Goal, error) {
	goal, err := database.QueryFirst[goalDB](ctx, r.db, `
		SELECT * FROM goals 
		WHERE created_by = $user 
		  AND activity_key = $key 
		  AND deleted_at IS NONE
		FETCH category
	`, map[string]any{
		"user": userID,
		"key":  activityKey,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_key", activityKey).Msg("query failed for activity key lookup")
		return nil, err
	}

	if goal == nil {
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
	if filters.GoalType != "" {
		conditions = append(conditions, "goal_type = $goal_type")
		queryVars["goal_type"] = filters.GoalType
	}
	if filters.LifeDomain != "" {
		conditions = append(conditions, "life_domain = $life_domain")
		queryVars["life_domain"] = filters.LifeDomain
	}
	if filters.IsRecurring != nil {
		if *filters.IsRecurring {
			conditions = append(conditions, "recurrence IS NOT NONE")
		} else {
			conditions = append(conditions, "recurrence IS NONE")
		}
	}
	// Search filter - search in title and description
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
		// Map allowed sort fields
		switch sortField {
		case "title":
			orderClause = "ORDER BY title " + sortDir
		case "streak":
			orderClause = "ORDER BY current_streak " + sortDir
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
	dataQuery := "SELECT *, (SELECT type::string(in) as task_id, in.title as task_title, impact_type, impact_magnitude, quantity_value, quantity_unit FROM task_goals WHERE out = $parent.id) as linked_tasks FROM goals WHERE " + whereClause + " " + orderClause + " LIMIT $limit START $offset FETCH category"
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
		FETCH category
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
	// Generate activity key from title
	activityKey := generateActivityKey(req.Title)

	// Check for duplicate activity key
	existing, _ := r.FindByActivityKey(ctx, activityKey, userID)
	if existing != nil {
		// Make unique by appending random suffix
		activityKey = activityKey + "_" + generateShortID()
	}

	goalID := generateRecordID()
	now := time.Now().UTC()

	createData := map[string]any{
		"created_by":      userID,
		"activity_key":    activityKey,
		"title":           req.Title,
		"description":     req.Description,
		"why":             req.Why,
		"icon":            req.Icon,
		"color":           req.Color,
		"goal_type":       req.GoalType,
		"status":          StatusActive,
		"current_streak":  0,
		"longest_streak":  0,
		"grace_days_used": 0,
		"priority":        req.Priority,
		"value_score":     req.ValueScore,
		"life_domain":     req.LifeDomain,
		"is_private":      req.IsPrivate,
		"created_at":      now,
		"updated_at":      now,
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

	// Handle target
	if req.Target != nil {
		createData["target"] = map[string]any{
			"value":         req.Target.Value,
			"unit":          req.Target.Unit,
			"current_value": 0,
			"per_period":    req.Target.PerPeriod,
		}
	}

	// Handle category
	if req.CategoryID != "" {
		createData["category"] = database.MustRecordID("categories", req.CategoryID)
	}

	// Handle parent goal
	if req.ParentGoal != "" {
		createData["parent_goal"] = req.ParentGoal
	}

	// Handle completion mode (default to "all" for epic goals)
	if req.CompletionMode != "" {
		createData["completion_mode"] = req.CompletionMode
	} else if req.GoalType == GoalTypeEpic {
		createData["completion_mode"] = CompletionModeAll
	}

	// Create the goal
	_, err := database.QueryAll[goalDB](ctx, r.db, `
		CREATE type::thing($id) CONTENT $data
	`, map[string]any{
		"id":   goalID,
		"data": createData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("create goal failed")
		return nil, err
	}

	r.logger.Info().Str("goal_id", database.ToStringID(goalID)).Str("activity_key", activityKey).Msg("goal created")

	return r.FindByID(ctx, database.ToStringID(goalID), userID)
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
	if req.Why != nil {
		updateData["why"] = *req.Why
	}
	if req.Icon != nil {
		updateData["icon"] = *req.Icon
	}
	if req.Color != nil {
		updateData["color"] = *req.Color
	}
	if req.GoalType != nil {
		updateData["goal_type"] = *req.GoalType
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
		if *req.Status == StatusCompleted {
			updateData["completion_date"] = now
		}
	}
	if req.Priority != nil {
		updateData["priority"] = *req.Priority
	}
	if req.ValueScore != nil {
		updateData["value_score"] = *req.ValueScore
	}
	if req.LifeDomain != nil {
		updateData["life_domain"] = *req.LifeDomain
	}
	if req.IsPrivate != nil {
		updateData["is_private"] = *req.IsPrivate
	}
	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			updateData["category"] = nil
		} else {
			updateData["category"] = database.MustRecordID("categories", *req.CategoryID)
		}
	}
	if req.CompletionMode != nil {
		updateData["completion_mode"] = *req.CompletionMode
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
		updateData["target"] = map[string]any{
			"value":      req.Target.Value,
			"unit":       req.Target.Unit,
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

	r.logger.Info().Str("goal_id", id).Msg("goal updated")

	return r.FindByID(ctx, id, userID)
}

func (r *repository) UpdateStreak(ctx context.Context, id string, currentStreak, longestStreak int, lastCompleted *time.Time, userID string) error {
	goalID := database.MustRecordID(Table, id)
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
		UPDATE type::thing($id) MERGE $data WHERE created_by = $user
	`, map[string]any{
		"id":   goalID,
		"data": updateData,
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", id).Msg("update streak failed")
		return err
	}

	return nil
}

func (r *repository) UpdateLinkedTemplate(ctx context.Context, id string, templateID string, userID string) error {
	goalID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err := database.QueryAll[goalDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			linked_template: $template_id,
			updated_at: $now
		} WHERE created_by = $user
	`, map[string]any{
		"id":          goalID,
		"template_id": templateID,
		"now":         now,
		"user":        userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", id).Msg("update linked template failed")
		return err
	}

	return nil
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
// CHILD GOALS OPERATIONS
// =============================================================================

func (r *repository) FindChildGoals(ctx context.Context, parentGoalID, userID string) ([]*Goal, error) {
	goalsDB, err := database.QueryAll[goalDB](ctx, r.db, `
		SELECT * FROM goals 
		WHERE created_by = $user 
		  AND parent_goal = $parent_id
		  AND deleted_at IS NONE
		FETCH category
	`, map[string]any{
		"user":      userID,
		"parent_id": parentGoalID,
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

func (r *repository) UpdateStatus(ctx context.Context, id, status, userID string) error {
	goalID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"status":     status,
		"updated_at": now,
	}

	// Set completion_date if completing
	if status == StatusCompleted {
		updateData["completion_date"] = now
	}

	_, err := database.QueryAll[goalDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE $data WHERE created_by = $user
	`, map[string]any{
		"id":   goalID,
		"data": updateData,
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("goal_id", id).Msg("update status failed")
		return err
	}

	r.logger.Info().Str("goal_id", id).Str("status", status).Msg("goal status updated")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// generateActivityKey creates a URL-safe key from the goal title.
// Example: "Drink 3L water daily" → "drink_3l_water_daily"
func generateActivityKey(title string) string {
	key := strings.ToLower(title)
	key = strings.ReplaceAll(key, " ", "_")
	key = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(key, "")
	// Remove consecutive underscores
	key = regexp.MustCompile(`_+`).ReplaceAllString(key, "_")
	// Trim underscores from ends
	key = strings.Trim(key, "_")
	// Limit length
	if len(key) > 100 {
		key = key[:100]
	}
	return key
}

func generateRecordID() models.RecordID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}

func generateShortID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
