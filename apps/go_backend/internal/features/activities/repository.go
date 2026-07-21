package activities

import (
	"context"
	"fmt"
	"strings"
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

	// Populated via subquery
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

	result, err := database.QueryFirst[activityDB](ctx, r.db, `
		SELECT *,
			(SELECT
				out.id AS goal_id,
				out.title AS goal_title,
				out.icon AS goal_icon,
				auto_link_tasks,
				quantity_multiplier,
				default_quantity,
				default_impact,
				out.target.unit_id AS target_unit_id
			FROM ->activity_goals
			WHERE out.deleted_at IS NONE
			) AS goals,
			(SELECT out AS category FROM ->in_category LIMIT 1)[0].category AS category
		FROM $id
	`, map[string]any{
		"id": activityID,
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

	if result.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return result.toActivity(), nil
}

// FindPaginated retrieves paginated activities for a user.
func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Activity], error) {
	// Count
	countQuery := `
		RETURN (SELECT count() FROM activities 
			WHERE created_by = $user 
			  AND deleted_at IS NONE 
			GROUP ALL)[0].count OR 0
	`
	total, err := database.QueryScalar[float64](ctx, r.db, countQuery, map[string]any{"user": userID})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("count query failed")
		return nil, err
	}

	// List with goals and category
	dataQuery := `
		SELECT *,
			(SELECT
				out.id AS goal_id,
				out.title AS goal_title,
				out.icon AS goal_icon,
				auto_link_tasks,
				quantity_multiplier,
				default_quantity,
				default_impact,
				out.target.unit_id AS target_unit_id
			FROM ->activity_goals
			WHERE out.deleted_at IS NONE
			) AS goals,
			(SELECT out AS category FROM ->in_category LIMIT 1)[0].category AS category
		FROM activities
		WHERE created_by = $user
		  AND deleted_at IS NONE
		ORDER BY pinned DESC, sort_order ASC, last_used_at DESC
		LIMIT $limit START $offset
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
		Total:   int64(total),
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: int64(params.Offset+len(items)) < int64(total),
	}, nil
}

// FindPinned retrieves pinned activities for quick bar.
func (r *repository) FindPinned(ctx context.Context, userID string) ([]*Activity, error) {
	query := `
		SELECT *,
			(SELECT
				out.id AS goal_id,
				out.title AS goal_title,
				out.icon AS goal_icon,
				auto_link_tasks,
				quantity_multiplier,
				default_quantity,
				default_impact,
				out.target.unit_id AS target_unit_id
			FROM ->activity_goals
			WHERE out.deleted_at IS NONE AND out.status = "active"
			) AS goals,
			(SELECT out AS category FROM ->in_category LIMIT 1)[0].category AS category
		FROM activities
		WHERE created_by = $user
		  AND deleted_at IS NONE
		  AND pinned = true
		ORDER BY sort_order ASC
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

	createQuery := `
		CREATE activities CONTENT {
			title: $title,
			icon: $icon,
			description: $description,
			default_duration: $default_duration,
			default_emotion_id: $default_emotion_id,
			default_priority: $default_priority,
			default_completed: $default_completed,
			quantity_enabled: $quantity_enabled,
			quantity_default: $quantity_default,
			quantity_step: $quantity_step,
			quantity_unit_id: $quantity_unit_id,
			default_impact: $default_impact,
			category_id: $category_id,
			pinned: $pinned,
			sort_order: $sort_order,
			use_count: 0,
			created_by: $user_id,
			created_at: time::now(),
			updated_at: time::now()
		};
	`

	result, err := database.QueryFirst[activityDB](ctx, r.db, createQuery, map[string]any{
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
		"user_id":            userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Msg("create query failed")
		return nil, err
	}

	activityID := database.ToStringID(result.ID)

	// Link to category if provided
	if req.CategoryID != "" {
		if err := r.SetCategory(ctx, activityID, req.CategoryID); err != nil {
			r.logger.Warn().Err(err).Str("activity_id", activityID).Msg("failed to link category")
		}
	}

	// Link to goals if provided
	for _, link := range req.GoalLinks {
		if err := r.LinkGoal(ctx, activityID, &link); err != nil {
			r.logger.Warn().Err(err).Str("activity_id", activityID).Str("goal_id", link.GoalID).Msg("failed to link goal")
		}
	}

	// Fetch with relations populated
	return r.FindByID(ctx, activityID, userID)
}

// Update updates an existing activity.
func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Activity, error) {
	// Build dynamic update
	updates := []string{"updated_at = time::now()"}
	params := map[string]interface{}{
		"table":   Table,
		"id":      database.ExtractID(id),
		"user_id": database.ExtractID(userID),
	}

	if req.Title != nil {
		updates = append(updates, "title = $title")
		params["title"] = *req.Title
	}
	if req.Icon != nil {
		updates = append(updates, "icon = $icon")
		params["icon"] = *req.Icon
	}
	if req.Description != nil {
		updates = append(updates, "description = $description")
		params["description"] = *req.Description
	}
	if req.DefaultDuration != nil {
		updates = append(updates, "default_duration = $default_duration")
		params["default_duration"] = *req.DefaultDuration
	}
	if req.DefaultEmotionID != nil {
		updates = append(updates, "default_emotion_id = $default_emotion_id")
		params["default_emotion_id"] = *req.DefaultEmotionID
	}
	if req.DefaultPriority != nil {
		updates = append(updates, "default_priority = $default_priority")
		params["default_priority"] = *req.DefaultPriority
	}
	if req.DefaultCompleted != nil {
		updates = append(updates, "default_completed = $default_completed")
		params["default_completed"] = *req.DefaultCompleted
	}
	if req.QuantityEnabled != nil {
		updates = append(updates, "quantity_enabled = $quantity_enabled")
		params["quantity_enabled"] = *req.QuantityEnabled
	}
	if req.QuantityDefault != nil {
		updates = append(updates, "quantity_default = $quantity_default")
		params["quantity_default"] = *req.QuantityDefault
	}
	if req.QuantityStep != nil {
		updates = append(updates, "quantity_step = $quantity_step")
		params["quantity_step"] = *req.QuantityStep
	}
	if req.QuantityUnitID != nil {
		updates = append(updates, "quantity_unit_id = $quantity_unit_id")
		params["quantity_unit_id"] = *req.QuantityUnitID
	}
	if req.DefaultImpact != nil {
		updates = append(updates, "default_impact = $default_impact")
		params["default_impact"] = *req.DefaultImpact
	}
	if req.Pinned != nil {
		updates = append(updates, "pinned = $pinned")
		params["pinned"] = *req.Pinned
	}
	if req.SortOrder != nil {
		updates = append(updates, "sort_order = $sort_order")
		params["sort_order"] = *req.SortOrder
	}

	activityID := database.MustRecordID(Table, id)
	params["id"] = activityID

	query := fmt.Sprintf(`
		UPDATE $id SET %s
		WHERE created_by = $user_id
		  AND deleted_at IS NONE
	`, strings.Join(updates, ", "))

	_, err := database.QueryFirst[activityDB](ctx, r.db, query, params)
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", id).Msg("update query failed")
		return nil, err
	}

	return r.FindByID(ctx, id, userID)
}

// Delete soft-deletes an activity.
func (r *repository) Delete(ctx context.Context, id, userID string) error {
	activityID := database.MustRecordID(Table, id)

	_, err := database.QueryFirst[activityDB](ctx, r.db, `
		UPDATE $id SET 
			deleted_at = time::now(),
			updated_at = time::now()
		WHERE created_by = $user_id
		  AND deleted_at IS NONE
	`, map[string]any{
		"id":      activityID,
		"user_id": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", id).Msg("delete query failed")
		return err
	}

	return nil
}

// IncrementUseCount updates usage statistics.
func (r *repository) IncrementUseCount(ctx context.Context, id string) error {
	activityID := database.MustRecordID(Table, id)

	_, err := database.QueryFirst[activityDB](ctx, r.db, `
		UPDATE $id SET 
			use_count = use_count + 1,
			last_used_at = time::now(),
			updated_at = time::now()
	`, map[string]any{
		"id": activityID,
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
		INSERT INTO activity_goals (activity_id, goal_id, auto_link_tasks, quantity_multiplier, default_quantity, default_impact, created_at)
		VALUES ($activity_id, $goal_id, $auto_link, $multiplier, $default_quantity, $impact, $now)
		ON CONFLICT(activity_id, goal_id) DO UPDATE SET
			auto_link_tasks = excluded.auto_link_tasks,
			quantity_multiplier = excluded.quantity_multiplier,
			default_quantity = excluded.default_quantity,
			default_impact = excluded.default_impact
	`, map[string]any{
		"activity_id":      actID.String(),
		"goal_id":          goalID.String(),
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
		DELETE activity_goals 
		WHERE in = $activity_id
		  AND out = $goal_id
	`, map[string]any{
		"activity_id": actID,
		"goal_id":     gID,
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
			out.id AS goal_id,
			out.title AS goal_title,
			out.icon AS goal_icon,
			(SELECT out.color FROM out->in_category LIMIT 1)[0] AS goal_color,
			auto_link_tasks,
			quantity_multiplier,
			default_quantity,
			default_impact,
			out.target.unit_id AS target_unit_id,
			(SELECT symbol FROM units WHERE id = out.target.unit_id)[0].symbol AS target_unit_symbol
		FROM activity_goals
		WHERE in = $activity_id
		  AND out.deleted_at IS NONE
	`, map[string]any{
		"activity_id": actID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", activityID).Msg("find linked goals failed")
		return nil, err
	}

	return results, nil
}

// SetCategory sets the activity's category.
func (r *repository) SetCategory(ctx context.Context, activityID, categoryID string) error {
	// First remove existing category link
	if err := r.ClearCategory(ctx, activityID); err != nil {
		r.logger.Warn().Err(err).Msg("failed to clear existing category")
	}

	actID := database.MustRecordID(Table, activityID)
	catID := database.MustRecordID("categories", categoryID)

	_, err := database.QueryAll[any](ctx, r.db, `
		RELATE $activity_id -> in_category -> $category_id
		SET created_at = time::now()
	`, map[string]any{
		"activity_id": actID,
		"category_id": catID,
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
		DELETE in_category WHERE in = $activity_id
	`, map[string]any{
		"activity_id": actID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_id", activityID).Msg("clear category failed")
		return err
	}

	return nil
}
