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

	// FindByActivityKey retrieves a template by its activity key.
	FindByActivityKey(ctx context.Context, activityKey, userID string) (*TaskTemplate, error)

	// Create creates a new template.
	Create(ctx context.Context, req *CreateRequest, userID string) (*TaskTemplate, error)

	// Update updates an existing template.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*TaskTemplate, error)

	// IncrementUseCount increments the use count and sets last_used_at.
	IncrementUseCount(ctx context.Context, id string) error

	// Delete soft-deletes a template.
	Delete(ctx context.Context, id, userID string) error
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
	Color       string `json:"color,omitempty"`

	DefaultDuration int         `json:"default_duration,omitempty"`
	DefaultPriority int         `json:"default_priority,omitempty"`
	DefaultCategory *categoryDB `json:"default_category,omitempty"`

	IsQuickLog    bool `json:"is_quick_log"`
	QuickLogOrder int  `json:"quick_log_order,omitempty"`

	QuantityEnabled bool    `json:"quantity_enabled"`
	QuantityDefault float64 `json:"quantity_default,omitempty"`
	QuantityUnit    string  `json:"quantity_unit,omitempty"`
	QuantityStep    float64 `json:"quantity_step,omitempty"`

	ExpectedQuadrant string `json:"expected_quadrant,omitempty"`
	DefaultEmotionID string `json:"default_emotion_id,omitempty"`

	ActivityKey string `json:"activity_key,omitempty"`
	GoalID      string `json:"goal_id,omitempty"`

	ShowFields map[string]bool `json:"show_fields,omitempty"`

	IsDefault    bool   `json:"is_default"`
	SourceTaskID string `json:"source_task_id,omitempty"`

	UseCount   int                   `json:"use_count"`
	LastUsedAt *database.SurrealTime `json:"last_used_at,omitempty"`

	CreatedAt database.SurrealTime  `json:"created_at"`
	UpdatedAt database.SurrealTime  `json:"updated_at"`
	DeletedAt *database.SurrealTime `json:"deleted_at,omitempty"`
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
		Color:       t.Color,

		DefaultDuration: t.DefaultDuration,
		DefaultPriority: t.DefaultPriority,

		IsQuickLog:    t.IsQuickLog,
		QuickLogOrder: t.QuickLogOrder,

		QuantityEnabled: t.QuantityEnabled,
		QuantityDefault: t.QuantityDefault,
		QuantityUnit:    t.QuantityUnit,
		QuantityStep:    t.QuantityStep,

		ExpectedQuadrant: t.ExpectedQuadrant,
		DefaultEmotionID: t.DefaultEmotionID,

		ActivityKey: t.ActivityKey,
		GoalID:      t.GoalID,

		IsDefault:    t.IsDefault,
		SourceTaskID: t.SourceTaskID,

		UseCount:  t.UseCount,
		CreatedAt: t.CreatedAt.Time,
		UpdatedAt: t.UpdatedAt.Time,
	}

	if t.DefaultCategory != nil {
		template.DefaultCategory = t.DefaultCategory.toCategory()
	}

	if t.LastUsedAt != nil && !t.LastUsedAt.IsZero() {
		lt := t.LastUsedAt.Time
		template.LastUsedAt = &lt
	}

	if t.DeletedAt != nil && !t.DeletedAt.IsZero() {
		dt := t.DeletedAt.Time
		template.DeletedAt = &dt
	}

	if t.ShowFields != nil {
		template.ShowFields = &ShowFields{
			Journal:            t.ShowFields["journal"],
			Duration:           t.ShowFields["duration"],
			Quantity:           t.ShowFields["quantity"],
			Emotion:            t.ShowFields["emotion"],
			PositivesNegatives: t.ShowFields["positives_negatives"],
			Notes:              t.ShowFields["notes"],
		}
	}

	return template
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByID(ctx context.Context, id, userID string) (*TaskTemplate, error) {
	templateID := database.MustRecordID(Table, id)

	tmpl, err := database.QueryFirst[templateDB](ctx, r.db, `
		SELECT * FROM type::thing($id) FETCH default_category
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

	// Allow access to default templates OR user's own templates
	if tmpl.CreatedBy != userID && !tmpl.IsDefault {
		return nil, errors.ErrNotFound
	}

	if tmpl.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return tmpl.toTemplate(), nil
}

func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*TaskTemplate, int64, error) {
	// Count (user's templates + default templates)
	countQuery := `
		RETURN (SELECT count() FROM templates 
			WHERE (created_by = $user OR is_default = true) 
			  AND deleted_at IS NONE 
			GROUP ALL)[0].count OR 0
	`
	total, err := database.QueryScalar[float64](ctx, r.db, countQuery, map[string]any{"user": userID})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("count query failed")
		return nil, 0, err
	}

	// List
	dataQuery := `
		SELECT * FROM templates 
		WHERE (created_by = $user OR is_default = true) 
		  AND deleted_at IS NONE
		ORDER BY is_quick_log DESC, quick_log_order ASC, created_at DESC
		LIMIT $limit START $offset
		FETCH default_category
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
		SELECT * FROM templates 
		WHERE (created_by = $user OR is_default = true) 
		  AND deleted_at IS NONE
		  AND is_quick_log = true
		ORDER BY quick_log_order ASC, created_at DESC
		FETCH default_category
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

func (r *repository) FindByActivityKey(ctx context.Context, activityKey, userID string) (*TaskTemplate, error) {
	tmpl, err := database.QueryFirst[templateDB](ctx, r.db, `
		SELECT * FROM templates 
		WHERE (created_by = $user OR is_default = true)
		  AND activity_key = $key 
		  AND deleted_at IS NONE
		FETCH default_category
	`, map[string]any{
		"user": userID,
		"key":  activityKey,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("activity_key", activityKey).Msg("query failed for activity key lookup")
		return nil, err
	}

	if tmpl == nil {
		return nil, errors.ErrNotFound
	}

	return tmpl.toTemplate(), nil
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
		"color":              req.Color,
		"default_duration":   req.DefaultDuration,
		"default_priority":   req.DefaultPriority,
		"is_quick_log":       req.IsQuickLog,
		"quick_log_order":    req.QuickLogOrder,
		"quantity_enabled":   req.QuantityEnabled,
		"quantity_default":   req.QuantityDefault,
		"quantity_unit":      req.QuantityUnit,
		"quantity_step":      req.QuantityStep,
		"expected_quadrant":  req.ExpectedQuadrant,
		"default_emotion_id": req.DefaultEmotionID,
		"activity_key":       req.ActivityKey,
		"goal_id":            req.GoalID,
		"is_default":         false,
		"use_count":          0,
		"created_at":         now,
		"updated_at":         now,
	}

	if req.DefaultCategoryID != "" {
		createData["default_category"] = database.MustRecordID("categories", req.DefaultCategoryID)
	}

	if req.ShowFields != nil {
		createData["show_fields"] = map[string]bool{
			"journal":             boolValue(req.ShowFields.Journal),
			"duration":            boolValue(req.ShowFields.Duration),
			"quantity":            boolValue(req.ShowFields.Quantity),
			"emotion":             boolValue(req.ShowFields.Emotion),
			"positives_negatives": boolValue(req.ShowFields.PositivesNegatives),
			"notes":               boolValue(req.ShowFields.Notes),
		}
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

	r.logger.Info().Str("template_id", database.ToStringID(templateID)).Msg("template created")

	return r.FindByID(ctx, database.ToStringID(templateID), userID)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*TaskTemplate, error) {
	// Verify ownership
	existing, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Don't allow updating default templates
	if existing.IsDefault {
		return nil, errors.ErrBadRequest.WithMessage("Cannot modify default templates")
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
	if req.Color != nil {
		updateData["color"] = *req.Color
	}
	if req.DefaultDuration != nil {
		updateData["default_duration"] = *req.DefaultDuration
	}
	if req.DefaultPriority != nil {
		updateData["default_priority"] = *req.DefaultPriority
	}
	if req.DefaultCategoryID != nil {
		if *req.DefaultCategoryID == "" {
			updateData["default_category"] = nil
		} else {
			updateData["default_category"] = database.MustRecordID("categories", *req.DefaultCategoryID)
		}
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
	if req.QuantityUnit != nil {
		updateData["quantity_unit"] = *req.QuantityUnit
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

	if req.ShowFields != nil {
		showFields := make(map[string]bool)
		if existing.ShowFields != nil {
			showFields["journal"] = existing.ShowFields.Journal
			showFields["duration"] = existing.ShowFields.Duration
			showFields["quantity"] = existing.ShowFields.Quantity
			showFields["emotion"] = existing.ShowFields.Emotion
			showFields["positives_negatives"] = existing.ShowFields.PositivesNegatives
			showFields["notes"] = existing.ShowFields.Notes
		}
		if req.ShowFields.Journal != nil {
			showFields["journal"] = *req.ShowFields.Journal
		}
		if req.ShowFields.Duration != nil {
			showFields["duration"] = *req.ShowFields.Duration
		}
		if req.ShowFields.Quantity != nil {
			showFields["quantity"] = *req.ShowFields.Quantity
		}
		if req.ShowFields.Emotion != nil {
			showFields["emotion"] = *req.ShowFields.Emotion
		}
		if req.ShowFields.PositivesNegatives != nil {
			showFields["positives_negatives"] = *req.ShowFields.PositivesNegatives
		}
		if req.ShowFields.Notes != nil {
			showFields["notes"] = *req.ShowFields.Notes
		}
		updateData["show_fields"] = showFields
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
	existing, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	// Don't allow deleting default templates
	if existing.IsDefault {
		return errors.ErrBadRequest.WithMessage("Cannot delete default templates")
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
// HELPERS
// =============================================================================

func generateRecordID() models.RecordID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}

func boolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
