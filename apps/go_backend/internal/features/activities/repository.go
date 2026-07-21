package activities

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"
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

// Repository defines the data access interface for activities.
type Repository interface {
	// FindByID retrieves a single activity by ID with goals populated.
	FindByID(ctx context.Context, id, userID string) (*Activity, error)

	// FindPaginated retrieves paginated activities for a user.
	FindPaginated(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Activity], error)

	// FindPinned retrieves pinned activities for quick bar.
	FindPinned(ctx context.Context, userID string) ([]*Activity, error)

	// Create creates a new activity.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Activity, error)

	// Update updates an existing activity.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Activity, error)

	// Delete soft-deletes an activity.
	Delete(ctx context.Context, id, userID string) error

	// IncrementUseCount updates usage statistics.
	IncrementUseCount(ctx context.Context, id string) error

	// LinkGoal links an activity to a goal.
	LinkGoal(ctx context.Context, activityID string, link *GoalLinkInput) error

	// UnlinkGoal removes a goal link.
	UnlinkGoal(ctx context.Context, activityID, goalID string) error

	// FindLinkedGoals retrieves goals linked to an activity.
	FindLinkedGoals(ctx context.Context, activityID, userID string) ([]ActivityGoalLinkDetail, error)

	// SetCategory sets the activity's category.
	SetCategory(ctx context.Context, activityID, categoryID string) error

	// ClearCategory removes the activity's category.
	ClearCategory(ctx context.Context, activityID string) error
}

// =============================================================================
// DATABASE MODELS
// =============================================================================

// activityDB is the database representation of Activity.
//
// The activities table stores the category as a foreign key (category_id) and
// goal links in the activity_goals join table. Both are hydrated via SQL
// subqueries (json_object / json_group_array) into the nested structs below.
type activityDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	CreatedBy string          `json:"created_by"`

	Title       string `json:"title"`
	Icon        string `json:"icon,omitempty"`
	Description string `json:"description,omitempty"`

	DefaultDuration  int    `json:"default_duration,omitempty"`
	DefaultEmotionID string `json:"default_emotion_id,omitempty"`
	DefaultPriority  int    `json:"default_priority"`
	DefaultCompleted bool   `json:"default_completed"`

	QuantityEnabled bool    `json:"quantity_enabled"`
	QuantityDefault float64 `json:"quantity_default,omitempty"`
	QuantityStep    float64 `json:"quantity_step,omitempty"`
	QuantityUnitID  string  `json:"quantity_unit_id,omitempty"`

	DefaultImpact string `json:"default_impact"`

	CategoryID string `json:"category_id,omitempty"`
	Pinned     bool   `json:"pinned"`
	SortOrder  int    `json:"sort_order"`

	UseCount   int                `json:"use_count"`
	LastUsedAt *database.FlexTime `json:"last_used_at,omitempty"`

	ActiveSession *TimerSession `json:"active_session,omitempty"`

	CreatedAt database.FlexTime  `json:"created_at"`
	UpdatedAt database.FlexTime  `json:"updated_at"`
	DeletedAt *database.FlexTime `json:"deleted_at,omitempty"`

	// Populated via subqueries
	Category *categoryDB  `json:"category,omitempty"`
	Goals    []goalLinkDB `json:"goals,omitempty"`
}

type categoryDB struct {
	ID        models.RecordID    `json:"id,omitempty"`
	Name      string             `json:"name"`
	Color     string             `json:"color"`
	CreatedAt database.FlexTime  `json:"created_at"`
	UpdatedAt database.FlexTime  `json:"updated_at"`
	DeletedAt *database.FlexTime `json:"deleted_at,omitempty"`
	CreatedBy string             `json:"created_by"`
}

type goalLinkDB struct {
	GoalID             string   `json:"goal_id"`
	GoalTitle          string   `json:"goal_title"`
	GoalIcon           string   `json:"goal_icon,omitempty"`
	AutoLinkTasks      bool     `json:"auto_link_tasks"`
	QuantityMultiplier float64  `json:"quantity_multiplier"`
	DefaultQuantity    *float64 `json:"default_quantity,omitempty"`
	DefaultImpact      string   `json:"default_impact"`
	// Goal's target info for unit
	TargetUnitID string `json:"target_unit_id,omitempty"`
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

func (a *activityDB) toActivity() *Activity {
	activity := &Activity{
		ID:        database.ToStringID(a.ID),
		CreatedBy: a.CreatedBy,

		Title:       a.Title,
		Icon:        a.Icon,
		Description: a.Description,

		DefaultDuration:  a.DefaultDuration,
		DefaultEmotionID: a.DefaultEmotionID,
		DefaultPriority:  a.DefaultPriority,
		DefaultCompleted: a.DefaultCompleted,

		QuantityEnabled: a.QuantityEnabled,
		QuantityDefault: a.QuantityDefault,
		QuantityStep:    a.QuantityStep,
		QuantityUnitID:  a.QuantityUnitID,

		DefaultImpact: a.DefaultImpact,

		Pinned:    a.Pinned,
		SortOrder: a.SortOrder,

		UseCount:      a.UseCount,
		ActiveSession: a.ActiveSession,

		CreatedAt: a.CreatedAt.Time,
		UpdatedAt: a.UpdatedAt.Time,
	}

	if a.Category != nil {
		activity.Category = a.Category.toCategory()
	}

	// Convert goal links
	for _, gl := range a.Goals {
		activity.Goals = append(activity.Goals, ActivityGoalLink{
			GoalID:             gl.GoalID,
			AutoLinkTasks:      gl.AutoLinkTasks,
			QuantityMultiplier: gl.QuantityMultiplier,
			DefaultQuantity:    gl.DefaultQuantity,
			DefaultImpact:      gl.DefaultImpact,
		})
	}

	if a.LastUsedAt != nil && !a.LastUsedAt.IsZero() {
		lt := a.LastUsedAt.Time
		activity.LastUsedAt = &lt
	}

	if a.DeletedAt != nil && !a.DeletedAt.IsZero() {
		dt := a.DeletedAt.Time
		activity.DeletedAt = &dt
	}

	return activity
}

// =============================================================================
// SQL FRAGMENTS
// =============================================================================

// baseColumns enumerates the scalar activity columns with COALESCE so that
// empty-string defaults decode cleanly into the struct.
const baseColumns = `
	a.id AS id,
	a.created_by AS created_by,
	a.title AS title,
	COALESCE(a.icon, '') AS icon,
	COALESCE(a.description, '') AS description,
	a.default_duration AS default_duration,
	COALESCE(a.default_emotion_id, '') AS default_emotion_id,
	a.default_priority AS default_priority,
	a.default_completed AS default_completed,
	a.quantity_enabled AS quantity_enabled,
	COALESCE(a.quantity_default, 0) AS quantity_default,
	COALESCE(a.quantity_step, 0) AS quantity_step,
	COALESCE(a.quantity_unit_id, '') AS quantity_unit_id,
	a.default_impact AS default_impact,
	COALESCE(a.category_id, '') AS category_id,
	a.pinned AS pinned,
	a.sort_order AS sort_order,
	a.use_count AS use_count,
	a.last_used_at AS last_used_at,
	a.created_at AS created_at,
	a.updated_at AS updated_at,
	a.deleted_at AS deleted_at
`

// categorySubquery hydrates the activity's category from the category_id FK.
const categorySubquery = `
	(
		SELECT json_object(
			'id', c.id,
			'name', c.name,
			'color', c.color,
			'created_at', c.created_at,
			'updated_at', c.updated_at,
			'deleted_at', c.deleted_at,
			'created_by', c.created_by
		)
		FROM categories c
		WHERE c.id = a.category_id
		LIMIT 1
	) AS category
`

// goalsSubquery hydrates the linked goals from the activity_goals join table.
// The goal's target unit comes from the goals.target JSON payload.
func goalsSubquery(extraCondition string) string {
	return fmt.Sprintf(`
	(
		SELECT json_group_array(json_object(
			'goal_id', g.id,
			'goal_title', g.title,
			'goal_icon', COALESCE(g.icon, ''),
			'auto_link_tasks', CASE WHEN ag.auto_link_tasks != 0 THEN json('true') ELSE json('false') END,
			'quantity_multiplier', ag.quantity_multiplier,
			'default_quantity', ag.default_quantity,
			'default_impact', ag.default_impact,
			'target_unit_id', COALESCE(json_extract(g.target, '$.unit_id'), '')
		))
		FROM activity_goals ag
		JOIN goals g ON g.id = ag.goal_id
		WHERE ag.activity_id = a.id
		  AND g.deleted_at IS NULL
		  %s
	) AS goals
	`, extraCondition)
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new activities Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "activities").Logger(),
	}
}

// FindByID retrieves a single activity by ID with goals populated.
func (r *repository) FindByID(ctx context.Context, id, userID string) (*Activity, error) {
	activityID := database.MustRecordID(Table, id)

	query := `SELECT ` + baseColumns + `,
		` + categorySubquery + `,
		` + goalsSubquery("") + `
		FROM activities a
		WHERE a.id = $id
	`

	result, err := database.QueryFirst[activityDB](ctx, r.db, query, map[string]any{
		"id": database.ToStringID(activityID),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", id).Msg("query failed")
		return nil, err
	}

	if result == nil {
		return nil, errors.ErrNotFound
	}

	if result.CreatedBy != userID {
		return nil, errors.ErrNotFound
	}

	if result.DeletedAt != nil && !result.DeletedAt.IsZero() {
		return nil, errors.ErrNotFound
	}

	return result.toActivity(), nil
}

// FindPaginated retrieves paginated activities for a user.
func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Activity], error) {
	// Count
	total, err := database.QueryScalar[int64](ctx, r.db, `
		SELECT COUNT(*) FROM activities
		WHERE created_by = $user
		  AND deleted_at IS NULL
	`, map[string]any{"user": userID})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("count query failed")
		return nil, err
	}

	// List with goals and category
	dataQuery := `SELECT ` + baseColumns + `,
		` + categorySubquery + `,
		` + goalsSubquery("") + `
		FROM activities a
		WHERE a.created_by = $user
		  AND a.deleted_at IS NULL
		ORDER BY a.pinned DESC, a.sort_order ASC, a.last_used_at DESC
		LIMIT $limit OFFSET $offset
	`
	activitiesDB, err := database.QueryAll[activityDB](ctx, r.db, dataQuery, map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("list query failed")
		return nil, err
	}

	items := make([]*Activity, 0, len(activitiesDB))
	for _, a := range activitiesDB {
		items = append(items, a.toActivity())
	}

	return &pagination.Response[*Activity]{
		Items:   items,
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: int64(params.Offset+len(items)) < total,
	}, nil
}

// FindPinned retrieves pinned activities for quick bar.
func (r *repository) FindPinned(ctx context.Context, userID string) ([]*Activity, error) {
	query := `SELECT ` + baseColumns + `,
		` + categorySubquery + `,
		` + goalsSubquery(`AND g.status = 'active'`) + `
		FROM activities a
		WHERE a.created_by = $user
		  AND a.deleted_at IS NULL
		  AND a.pinned = 1
		ORDER BY a.sort_order ASC
	`

	activitiesDB, err := database.QueryAll[activityDB](ctx, r.db, query, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("pinned query failed")
		return nil, err
	}

	items := make([]*Activity, 0, len(activitiesDB))
	for _, a := range activitiesDB {
		items = append(items, a.toActivity())
	}

	return items, nil
}

// Create creates a new activity.
func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Activity, error) {
	// Set defaults
	defaultImpact := req.DefaultImpact
	if defaultImpact == "" {
		defaultImpact = ImpactPositive
	}
	defaultPriority := req.DefaultPriority
	if defaultPriority == 0 {
		defaultPriority = 3
	}

	activityID := generateRecordID()
	now := time.Now().UTC()

	result, err := database.Create[activityDB](ctx, r.db, Table, map[string]any{
		"id":                 database.ToStringID(activityID),
		"title":              req.Title,
		"icon":               req.Icon,
		"description":        req.Description,
		"default_duration":   req.DefaultDuration,
		"default_emotion_id": req.DefaultEmotionID,
		"default_priority":   defaultPriority,
		"default_completed":  req.DefaultCompleted,
		"quantity_enabled":   req.QuantityEnabled,
		"quantity_default":   req.QuantityDefault,
		"quantity_step":      req.QuantityStep,
		"quantity_unit_id":   req.QuantityUnitID,
		"default_impact":     defaultImpact,
		"category_id":        req.CategoryID,
		"pinned":             req.Pinned,
		"sort_order":         req.SortOrder,
		"use_count":          0,
		"created_by":         userID,
		"created_at":         now.Format(time.RFC3339Nano),
		"updated_at":         now.Format(time.RFC3339Nano),
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("create query failed")
		return nil, err
	}

	newID := database.ToStringID(result.ID)

	// Link to category if provided
	if req.CategoryID != "" {
		if err := r.SetCategory(ctx, newID, req.CategoryID); err != nil {
			r.logger.Warn().Err(err).Str("activity_id", newID).Msg("failed to link category")
		}
	}

	// Link to goals if provided
	for _, link := range req.GoalLinks {
		if err := r.LinkGoal(ctx, newID, &link); err != nil {
			r.logger.Warn().Err(err).Str("activity_id", newID).Str("goal_id", link.GoalID).Msg("failed to link goal")
		}
	}

	// Fetch with relations populated
	return r.FindByID(ctx, newID, userID)
}

// Update updates an existing activity.
func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Activity, error) {
	// Verify ownership
	if _, err := r.FindByID(ctx, id, userID); err != nil {
		return nil, err
	}

	activityID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now.Format(time.RFC3339Nano),
	}

	if req.Title != nil {
		updateData["title"] = *req.Title
	}
	if req.Icon != nil {
		updateData["icon"] = *req.Icon
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
	}
	if req.DefaultDuration != nil {
		updateData["default_duration"] = *req.DefaultDuration
	}
	if req.DefaultEmotionID != nil {
		updateData["default_emotion_id"] = *req.DefaultEmotionID
	}
	if req.DefaultPriority != nil {
		updateData["default_priority"] = *req.DefaultPriority
	}
	if req.DefaultCompleted != nil {
		updateData["default_completed"] = *req.DefaultCompleted
	}
	if req.QuantityEnabled != nil {
		updateData["quantity_enabled"] = *req.QuantityEnabled
	}
	if req.QuantityDefault != nil {
		updateData["quantity_default"] = *req.QuantityDefault
	}
	if req.QuantityStep != nil {
		updateData["quantity_step"] = *req.QuantityStep
	}
	if req.QuantityUnitID != nil {
		updateData["quantity_unit_id"] = *req.QuantityUnitID
	}
	if req.DefaultImpact != nil {
		updateData["default_impact"] = *req.DefaultImpact
	}
	if req.Pinned != nil {
		updateData["pinned"] = *req.Pinned
	}
	if req.SortOrder != nil {
		updateData["sort_order"] = *req.SortOrder
	}

	if _, err := database.Merge[activityDB](ctx, r.db, database.ToStringID(activityID), updateData); err != nil {
		r.logger.Error().Err(err).Str("activity_id", id).Msg("update query failed")
		return nil, err
	}

	return r.FindByID(ctx, id, userID)
}

// Delete soft-deletes an activity.
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	// Verify ownership
	if _, err := r.FindByID(ctx, id, userID); err != nil {
		return err
	}

	activityID := database.MustRecordID(Table, id)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := database.Merge[activityDB](ctx, r.db, database.ToStringID(activityID), map[string]any{
		"deleted_at": now,
		"updated_at": now,
	}); err != nil {
		r.logger.Error().Err(err).Str("activity_id", id).Msg("delete query failed")
		return err
	}

	return nil
}

// IncrementUseCount updates usage statistics.
func (r *repository) IncrementUseCount(ctx context.Context, id string) error {
	activityID := database.MustRecordID(Table, id)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := database.QueryAll[any](ctx, r.db, `
		UPDATE activities SET
			use_count = use_count + 1,
			last_used_at = $now,
			updated_at = $now
		WHERE id = $id
	`, map[string]any{
		"id":  database.ToStringID(activityID),
		"now": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", id).Msg("increment use count failed")
		return err
	}

	return nil
}

// LinkGoal links an activity to a goal.
func (r *repository) LinkGoal(ctx context.Context, activityID string, link *GoalLinkInput) error {
	// Set defaults
	multiplier := link.QuantityMultiplier
	if multiplier == 0 {
		multiplier = 1.0
	}
	impact := link.DefaultImpact
	if impact == "" {
		impact = ImpactPositive
	}

	actID := database.MustRecordID(Table, activityID)
	goalID := database.MustRecordID("goals", link.GoalID)

	_, err := database.QueryAll[any](ctx, r.db, `
		INSERT INTO activity_goals (id, activity_id, goal_id, auto_link_tasks, quantity_multiplier, default_quantity, default_impact, created_at)
		VALUES ($id, $activity_id, $goal_id, $auto_link, $multiplier, $default_quantity, $impact, $now)
		ON CONFLICT(activity_id, goal_id) DO UPDATE SET
			auto_link_tasks = excluded.auto_link_tasks,
			quantity_multiplier = excluded.quantity_multiplier,
			default_quantity = excluded.default_quantity,
			default_impact = excluded.default_impact
	`, map[string]any{
		"id":               database.ToStringID(generateRecordIDFor("activity_goals")),
		"activity_id":      database.ToStringID(actID),
		"goal_id":          database.ToStringID(goalID),
		"auto_link":        link.AutoLinkTasks,
		"multiplier":       multiplier,
		"default_quantity": link.DefaultQuantity,
		"impact":           impact,
		"now":              time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", activityID).Str("goal_id", link.GoalID).Msg("link goal failed")
		return err
	}

	return nil
}

// UnlinkGoal removes a goal link.
func (r *repository) UnlinkGoal(ctx context.Context, activityID, goalID string) error {
	actID := database.MustRecordID(Table, activityID)
	gID := database.MustRecordID("goals", goalID)

	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE FROM activity_goals
		WHERE activity_id = $activity_id
		  AND goal_id = $goal_id
	`, map[string]any{
		"activity_id": database.ToStringID(actID),
		"goal_id":     database.ToStringID(gID),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", activityID).Str("goal_id", goalID).Msg("unlink goal failed")
		return err
	}

	return nil
}

// FindLinkedGoals retrieves goals linked to an activity.
func (r *repository) FindLinkedGoals(ctx context.Context, activityID, userID string) ([]ActivityGoalLinkDetail, error) {
	actID := database.MustRecordID(Table, activityID)

	results, err := database.QueryAll[ActivityGoalLinkDetail](ctx, r.db, `
		SELECT
			g.id AS goal_id,
			g.title AS goal_title,
			COALESCE(g.icon, '') AS goal_icon,
			COALESCE(gc.color, '') AS goal_color,
			ag.auto_link_tasks AS auto_link_tasks,
			ag.quantity_multiplier AS quantity_multiplier,
			ag.default_quantity AS default_quantity,
			ag.default_impact AS default_impact,
			COALESCE(json_extract(g.target, '$.unit_id'), '') AS target_unit_id,
			COALESCE(u.symbol, '') AS target_unit_symbol
		FROM activity_goals ag
		JOIN goals g ON g.id = ag.goal_id
		LEFT JOIN categories gc ON gc.id = g.category_id
		LEFT JOIN units u ON u.id = json_extract(g.target, '$.unit_id')
		WHERE ag.activity_id = $activity_id
		  AND g.deleted_at IS NULL
	`, map[string]any{
		"activity_id": database.ToStringID(actID),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", activityID).Msg("find linked goals failed")
		return nil, err
	}

	return results, nil
}

// SetCategory sets the activity's category.
func (r *repository) SetCategory(ctx context.Context, activityID, categoryID string) error {
	actID := database.MustRecordID(Table, activityID)
	catID := database.MustRecordID("categories", categoryID)

	_, err := database.QueryAll[any](ctx, r.db, `
		UPDATE activities SET
			category_id = $category_id,
			updated_at = $now
		WHERE id = $activity_id
	`, map[string]any{
		"activity_id": database.ToStringID(actID),
		"category_id": database.ToStringID(catID),
		"now":         time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", activityID).Str("category_id", categoryID).Msg("set category failed")
		return err
	}

	return nil
}

// ClearCategory removes the activity's category.
func (r *repository) ClearCategory(ctx context.Context, activityID string) error {
	actID := database.MustRecordID(Table, activityID)

	_, err := database.QueryAll[any](ctx, r.db, `
		UPDATE activities SET
			category_id = NULL,
			updated_at = $now
		WHERE id = $activity_id
	`, map[string]any{
		"activity_id": database.ToStringID(actID),
		"now":         time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", activityID).Msg("clear category failed")
		return err
	}

	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// generateRecordID creates a new activities record identifier.
func generateRecordID() models.RecordID {
	return generateRecordIDFor(Table)
}

// generateRecordIDFor creates a new table:value record identifier.
func generateRecordIDFor(table string) models.RecordID {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return database.NewRecordID(table, hex.EncodeToString(bytes))
}
