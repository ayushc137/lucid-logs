// Package templates provides template data access using SurrealDB SDK.
package templates

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Repository defines the template data access interface.
type Repository interface {
	// FindByID retrieves a template by ID for a specific user.
	FindByID(ctx context.Context, id, userID string) (*TaskTemplate, error)

	// FindPaginated retrieves templates for a user with pagination.
	FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*TaskTemplate, int64, error)

	// FindQuickLog retrieves quick-log templates for a user.
	FindQuickLog(ctx context.Context, userID string) ([]*TaskTemplate, error)

	// Create creates a new template.
	Create(ctx context.Context, req *CreateRequest, userID string) (*TaskTemplate, error)

	// Update updates an existing template.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*TaskTemplate, error)

	// IncrementUseCount increments the use count and sets last_used_at.
	IncrementUseCount(ctx context.Context, id string) error

	// Delete soft-deletes a template.
	Delete(ctx context.Context, id, userID string) error

	// LinkGoal links a template to a goal via template_goals relation.
	LinkGoal(ctx context.Context, templateID string, req *LinkGoalRequest, userID string) error

	// UnlinkGoal removes a template-goal link.
	UnlinkGoal(ctx context.Context, templateID, goalID, userID string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new template Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "templates").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

type templateDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	CreatedBy string          `json:"created_by"`

	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`

	DefaultDuration int `json:"default_duration,omitempty"`

	IsQuickLog    bool `json:"is_quick_log"`
	QuickLogOrder int  `json:"quick_log_order,omitempty"`

	QuantityEnabled bool    `json:"quantity_enabled"`
	QuantityDefault float64 `json:"quantity_default,omitempty"`
	QuantityStep    float64 `json:"quantity_step,omitempty"`

	ExpectedQuadrant string `json:"expected_quadrant,omitempty"`
	DefaultEmotionID string `json:"default_emotion_id,omitempty"`

	UseCount   int                   `json:"use_count"`
	LastUsedAt *database.SurrealTime `json:"last_used_at,omitempty"`

	CreatedAt database.SurrealTime  `json:"created_at"`
	UpdatedAt database.SurrealTime  `json:"updated_at"`
	DeletedAt *database.SurrealTime `json:"deleted_at,omitempty"`

	// Populated via subquery
	Category *categoryDB `json:"category,omitempty"`
}

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

func (t *templateDB) toTemplate() *TaskTemplate {
	template := &TaskTemplate{
		ID:        database.ToStringID(t.ID),
		CreatedBy: t.CreatedBy,

		Title:       t.Title,
		Description: t.Description,
		Icon:        t.Icon,

		DefaultDuration: t.DefaultDuration,

		IsQuickLog:    t.IsQuickLog,
		QuickLogOrder: t.QuickLogOrder,

		QuantityEnabled: t.QuantityEnabled,
		QuantityDefault: t.QuantityDefault,
		QuantityStep:    t.QuantityStep,

		ExpectedQuadrant: t.ExpectedQuadrant,
		DefaultEmotionID: t.DefaultEmotionID,

		UseCount:  t.UseCount,
		CreatedAt: t.CreatedAt.Time,
		UpdatedAt: t.UpdatedAt.Time,
	}

	if t.Category != nil {
		template.Category = t.Category.toCategory()
	}

	if t.LastUsedAt != nil && !t.LastUsedAt.IsZero() {
		lt := t.LastUsedAt.Time
		template.LastUsedAt = &lt
	}

	if t.DeletedAt != nil && !t.DeletedAt.IsZero() {
		dt := t.DeletedAt.Time
		template.DeletedAt = &dt
	}

	return template
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByID(ctx context.Context, id, userID string) (*TaskTemplate, error) {
	templateID := database.MustRecordID(Table, id)

	tmpl, err := database.QueryFirst[templateDB](ctx, r.db, `
		SELECT *,
			(SELECT out as category FROM in_category WHERE in = $parent.id)[0].category as category
		FROM type::thing($id)
	`, map[string]any{
		"id": templateID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("template_id", id).Msg("query failed for template fetch")
		return nil, err
	}

	if tmpl == nil {
		return nil, errors.ErrNotFound
	}

	if tmpl.CreatedBy != userID {
		return nil, errors.ErrNotFound
	}

	if tmpl.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return tmpl.toTemplate(), nil
}

func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*TaskTemplate, int64, error) {
	// Count
	countQuery := `
		RETURN (SELECT count() FROM templates 
			WHERE created_by = $user 
			  AND deleted_at IS NONE 
			GROUP ALL)[0].count OR 0
	`
	total, err := database.QueryScalar[float64](ctx, r.db, countQuery, map[string]any{"user": userID})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("count query failed")
		return nil, 0, err
	}

	// List with category via in_category edge
	dataQuery := `
		SELECT *,
			(SELECT out as category FROM in_category WHERE in = $parent.id)[0].category as category
		FROM templates 
		WHERE created_by = $user 
		  AND deleted_at IS NONE
		ORDER BY is_quick_log DESC, quick_log_order ASC, created_at DESC
		LIMIT $limit START $offset
	`
	templatesDB, err := database.QueryAll[templateDB](ctx, r.db, dataQuery, map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("list query failed")
		return nil, 0, err
	}

	templates := make([]*TaskTemplate, len(templatesDB))
	for i := range templatesDB {
		templates[i] = templatesDB[i].toTemplate()
	}

	return templates, int64(total), nil
}

func (r *repository) FindQuickLog(ctx context.Context, userID string) ([]*TaskTemplate, error) {
	templatesDB, err := database.QueryAll[templateDB](ctx, r.db, `
		SELECT *,
			(SELECT out as category FROM in_category WHERE in = $parent.id)[0].category as category
		FROM templates 
		WHERE created_by = $user 
		  AND deleted_at IS NONE
		  AND is_quick_log = true
		ORDER BY quick_log_order ASC, created_at DESC
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("quick log query failed")
		return nil, err
	}

	templates := make([]*TaskTemplate, len(templatesDB))
	for i := range templatesDB {
		templates[i] = templatesDB[i].toTemplate()
	}

	return templates, nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*TaskTemplate, error) {
	templateID := generateRecordID()
	now := time.Now().UTC()

	createData := map[string]any{
		"created_by":         userID,
		"title":              req.Title,
		"description":        req.Description,
		"icon":               req.Icon,
		"default_duration":   req.DefaultDuration,
		"is_quick_log":       req.IsQuickLog,
		"quick_log_order":    req.QuickLogOrder,
		"quantity_enabled":   req.QuantityEnabled,
		"quantity_default":   req.QuantityDefault,
		"quantity_step":      req.QuantityStep,
		"expected_quadrant":  req.ExpectedQuadrant,
		"default_emotion_id": req.DefaultEmotionID,
		"use_count":          0,
		"created_at":         now,
		"updated_at":         now,
	}

	_, err := database.QueryAll[templateDB](ctx, r.db, `
		CREATE type::thing($id) CONTENT $data
	`, map[string]any{
		"id":   templateID,
		"data": createData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("create template failed")
		return nil, err
	}

	templateIDStr := database.ToStringID(templateID)
	r.logger.Info().Str("template_id", templateIDStr).Msg("template created")

	// Link category via in_category relation
	if req.CategoryID != "" {
		if err := r.linkCategory(ctx, templateIDStr, req.CategoryID, userID); err != nil {
			r.logger.Warn().Err(err).Msg("failed to link category")
		}
	}

	// Link goals via template_goals relation
	for _, goalLink := range req.GoalLinks {
		if err := r.LinkGoal(ctx, templateIDStr, &LinkGoalRequest{
			GoalID:             goalLink.GoalID,
			AutoLinkTasks:      goalLink.AutoLinkTasks,
			QuantityMultiplier: goalLink.QuantityMultiplier,
		}, userID); err != nil {
			r.logger.Warn().Err(err).Str("goal_id", goalLink.GoalID).Msg("failed to link goal")
		}
	}

	return r.FindByID(ctx, templateIDStr, userID)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*TaskTemplate, error) {
	// Verify ownership
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	templateID := database.MustRecordID(Table, id)
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
	if req.DefaultDuration != nil {
		updateData["default_duration"] = *req.DefaultDuration
	}
	if req.IsQuickLog != nil {
		updateData["is_quick_log"] = *req.IsQuickLog
	}
	if req.QuickLogOrder != nil {
		updateData["quick_log_order"] = *req.QuickLogOrder
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
	if req.ExpectedQuadrant != nil {
		updateData["expected_quadrant"] = *req.ExpectedQuadrant
	}
	if req.DefaultEmotionID != nil {
		updateData["default_emotion_id"] = *req.DefaultEmotionID
	}

	_, err = database.QueryAll[templateDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE $data
	`, map[string]any{
		"id":   templateID,
		"data": updateData,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("template_id", id).Msg("update template failed")
		return nil, err
	}

	r.logger.Info().Str("template_id", id).Msg("template updated")

	return r.FindByID(ctx, id, userID)
}

func (r *repository) IncrementUseCount(ctx context.Context, id string) error {
	templateID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err := database.QueryAll[any](ctx, r.db, `
		UPDATE type::thing($id) SET 
			use_count += 1,
			last_used_at = $now
	`, map[string]any{
		"id":  templateID,
		"now": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("template_id", id).Msg("increment use count failed")
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

	templateID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err = database.QueryAll[templateDB](ctx, r.db, `
		UPDATE type::thing($id) MERGE {
			deleted_at: $now,
			updated_at: $now
		}
	`, map[string]any{
		"id":  templateID,
		"now": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("template_id", id).Msg("soft delete template failed")
		return err
	}

	r.logger.Info().Str("template_id", id).Msg("template deleted")
	return nil
}

// =============================================================================
// GOAL LINKING (via template_goals relation)
// =============================================================================

func (r *repository) LinkGoal(ctx context.Context, templateID string, req *LinkGoalRequest, userID string) error {
	tID := database.MustRecordID(Table, templateID)
	gID := database.MustRecordID("goals", req.GoalID)
	now := time.Now().UTC()

	multiplier := req.QuantityMultiplier
	if multiplier == 0 {
		multiplier = 1.0
	}

	_, err := database.QueryAll[any](ctx, r.db, `
		RELATE $template -> template_goals -> $goal SET {
			auto_link_tasks: $auto_link,
			quantity_multiplier: $multiplier,
			created_by: $user,
			created_at: $now
		}
	`, map[string]any{
		"template":   tID,
		"goal":       gID,
		"auto_link":  req.AutoLinkTasks,
		"multiplier": multiplier,
		"user":       userID,
		"now":        now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("template", templateID).Str("goal", req.GoalID).Msg("link goal failed")
		return err
	}

	return nil
}

func (r *repository) UnlinkGoal(ctx context.Context, templateID, goalID, userID string) error {
	tID := database.MustRecordID(Table, templateID)
	gID := database.MustRecordID("goals", goalID)

	_, err := database.QueryAll[any](ctx, r.db, `
		DELETE template_goals WHERE in = $template AND out = $goal
	`, map[string]any{
		"template": tID,
		"goal":     gID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("template", templateID).Str("goal", goalID).Msg("unlink goal failed")
		return err
	}

	return nil
}

// =============================================================================
// CATEGORY LINKING (via in_category relation)
// =============================================================================

func (r *repository) linkCategory(ctx context.Context, templateID, categoryID, userID string) error {
	tID := database.MustRecordID(Table, templateID)
	now := time.Now().UTC()

	// Remove existing category link
	_, _ = database.QueryAll[any](ctx, r.db, `
		DELETE in_category WHERE in = $template_id
	`, map[string]any{
		"template_id": tID,
	})

	if categoryID == "" {
		return nil
	}

	cID := database.MustRecordID("categories", categoryID)

	_, err := database.QueryAll[any](ctx, r.db, `
		RELATE $template -> in_category -> $category SET {
			created_by: $user,
			created_at: $now
		}
	`, map[string]any{
		"template": tID,
		"category": cID,
		"user":     userID,
		"now":      now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("template", templateID).Str("category", categoryID).Msg("link category failed")
		return err
	}

	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func generateRecordID() models.RecordID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}
