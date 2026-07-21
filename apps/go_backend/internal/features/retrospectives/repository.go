package retrospectives

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	models "github.com/lucid-logs/go-backend/internal/shared/recordid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the retrospective data access interface.
type Repository interface {
	// FindByID retrieves a retrospective by ID.
	FindByID(ctx context.Context, id, userID string) (*Retrospective, error)

	// FindByDateRange finds a retrospective for a specific date range.
	FindByDateRange(ctx context.Context, userID string, retroType string, start, end time.Time) (*Retrospective, error)

	// FindPaginated retrieves retrospectives with pagination.
	FindPaginated(ctx context.Context, userID string, limit, offset int) ([]*Retrospective, int, error)

	// Create creates a new retrospective.
	Create(ctx context.Context, retro *Retrospective) (*Retrospective, error)

	// Update updates a retrospective.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Retrospective, error)

	// UpdateAutoSummary updates just the auto_summary JSON for a retrospective.
	UpdateAutoSummary(ctx context.Context, id, userID string, summary RetroAutoSummary) (*Retrospective, error)

	// Delete soft-deletes a retrospective.
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new retrospectives Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "retrospectives").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

type retroDB struct {
	ID        models.RecordID `json:"id,omitempty"`
	CreatedBy string          `json:"created_by"`

	RetroType string            `json:"retro_type"`
	StartDate database.FlexTime `json:"start_date"`
	EndDate   database.FlexTime `json:"end_date"`

	AutoSummary any `json:"auto_summary"` // Store as JSON object
	UserContent any `json:"user_content"`

	Status string `json:"status"`

	GeneratedAt database.FlexTime  `json:"generated_at"`
	CreatedAt   database.FlexTime  `json:"created_at"`
	UpdatedAt   database.FlexTime  `json:"updated_at"`
	DeletedAt   *database.FlexTime `json:"deleted_at,omitempty"`
}

func (r *retroDB) toRetrospective() *Retrospective {
	retro := &Retrospective{
		ID:          database.ToStringID(r.ID),
		CreatedBy:   r.CreatedBy,
		RetroType:   r.RetroType,
		StartDate:   r.StartDate.Time,
		EndDate:     r.EndDate.Time,
		Status:      r.Status,
		GeneratedAt: r.GeneratedAt.Time,
		CreatedAt:   r.CreatedAt.Time,
		UpdatedAt:   r.UpdatedAt.Time,
	}

	if r.DeletedAt != nil && !r.DeletedAt.IsZero() {
		t := r.DeletedAt.Time
		retro.DeletedAt = &t
	}

	// Parse auto_summary and user_content from any
	// They're stored as JSON TEXT columns in SQLite
	if r.AutoSummary != nil {
		if raw, ok := r.AutoSummary.(string); ok && raw != "" {
			var summary RetroAutoSummary
			if err := json.Unmarshal([]byte(raw), &summary); err == nil {
				retro.AutoSummary = summary
			}
		} else if summary, ok := r.AutoSummary.(map[string]interface{}); ok {
			retro.AutoSummary = parseAutoSummary(summary)
		}
	}
	if r.UserContent != nil {
		if raw, ok := r.UserContent.(string); ok && raw != "" {
			var content UserReflection
			if err := json.Unmarshal([]byte(raw), &content); err == nil {
				retro.UserContent = content
			}
		} else if content, ok := r.UserContent.(map[string]interface{}); ok {
			retro.UserContent = parseUserContent(content)
		}
	}

	return retro
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByID(ctx context.Context, id, userID string) (*Retrospective, error) {
	retroID := database.MustRecordID(Table, id)

	result, err := database.QueryFirst[retroDB](ctx, r.db, `
		SELECT * FROM retrospectives WHERE id = $id
	`, map[string]any{
		"id": database.ToStringID(retroID),
	})
	if err != nil {
		r.logger.Error().Err(err).Str("retro_id", id).Msg("query failed")
		return nil, err
	}

	if result == nil || result.CreatedBy != userID || result.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return result.toRetrospective(), nil
}

func (r *repository) FindByDateRange(ctx context.Context, userID, retroType string, start, end time.Time) (*Retrospective, error) {
	result, err := database.QueryFirst[retroDB](ctx, r.db, `
		SELECT * FROM retrospectives
		WHERE created_by = $user
		  AND retro_type = $type
		  AND start_date = $start
		  AND end_date = $end
		  AND deleted_at IS NULL
		LIMIT 1
	`, map[string]any{
		"user":  userID,
		"type":  retroType,
		"start": start.Format(time.RFC3339Nano),
		"end":   end.Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil // Not found, but not an error
	}

	return result.toRetrospective(), nil
}

func (r *repository) FindPaginated(ctx context.Context, userID string, limit, offset int) ([]*Retrospective, int, error) {
	// Get total count
	total, err := database.QueryScalar[int64](ctx, r.db, `
		SELECT COUNT(*) FROM retrospectives
		WHERE created_by = $user AND deleted_at IS NULL
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	results, err := database.QueryAll[retroDB](ctx, r.db, `
		SELECT * FROM retrospectives
		WHERE created_by = $user AND deleted_at IS NULL
		ORDER BY start_date DESC
		LIMIT $limit OFFSET $offset
	`, map[string]any{
		"user":   userID,
		"limit":  limit,
		"offset": offset,
	})
	if err != nil {
		return nil, 0, err
	}

	retros := make([]*Retrospective, len(results))
	for i, res := range results {
		retros[i] = res.toRetrospective()
	}

	return retros, int(total), nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

func (r *repository) Create(ctx context.Context, retro *Retrospective) (*Retrospective, error) {
	retroID := generateRecordID()
	now := time.Now().UTC()

	autoSummaryJSON, err := json.Marshal(retro.AutoSummary)
	if err != nil {
		r.logger.Error().Err(err).Msg("marshal auto_summary failed")
		return nil, err
	}
	userContentJSON, err := json.Marshal(retro.UserContent)
	if err != nil {
		r.logger.Error().Err(err).Msg("marshal user_content failed")
		return nil, err
	}

	createData := map[string]any{
		"id":           database.ToStringID(retroID),
		"created_by":   retro.CreatedBy,
		"retro_type":   retro.RetroType,
		"start_date":   retro.StartDate,
		"end_date":     retro.EndDate,
		"auto_summary": string(autoSummaryJSON),
		"user_content": string(userContentJSON),
		"status":       retro.Status,
		"generated_at": retro.GeneratedAt,
		"created_at":   now,
		"updated_at":   now,
	}

	_, err = database.Create[retroDB](ctx, r.db, Table, createData)
	if err != nil {
		r.logger.Error().Err(err).Msg("create retrospective failed")
		return nil, err
	}

	r.logger.Info().
		Str("retro_id", database.ToStringID(retroID)).
		Str("type", retro.RetroType).
		Msg("retrospective created")

	return r.FindByID(ctx, database.ToStringID(retroID), retro.CreatedBy)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Retrospective, error) {
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	retroID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now,
	}

	if req.UserContent != nil {
		userContentJSON, err := json.Marshal(req.UserContent)
		if err != nil {
			r.logger.Error().Err(err).Msg("marshal user_content failed")
			return nil, err
		}
		updateData["user_content"] = string(userContentJSON)
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
	}

	_, err = database.Merge[retroDB](ctx, r.db, database.ToStringID(retroID), updateData)
	if err != nil {
		r.logger.Error().Err(err).Str("retro_id", id).Msg("update failed")
		return nil, err
	}

	return r.FindByID(ctx, id, userID)
}

func (r *repository) UpdateAutoSummary(ctx context.Context, id, userID string, summary RetroAutoSummary) (*Retrospective, error) {
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	retroID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		r.logger.Error().Err(err).Msg("marshal auto_summary failed")
		return nil, err
	}

	_, err = database.Merge[retroDB](ctx, r.db, database.ToStringID(retroID), map[string]any{
		"auto_summary": string(summaryJSON),
		"updated_at":   now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("retro_id", id).Msg("update auto_summary failed")
		return nil, err
	}

	return r.FindByID(ctx, id, userID)
}

func (r *repository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.FindByID(ctx, id, userID)
	if err != nil {
		return err
	}

	retroID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	_, err = database.Merge[retroDB](ctx, r.db, database.ToStringID(retroID), map[string]any{
		"deleted_at": now,
		"updated_at": now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("retro_id", id).Msg("delete failed")
		return err
	}

	r.logger.Info().Str("retro_id", id).Msg("retrospective deleted")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func generateRecordID() models.RecordID {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}

// parseAutoSummary converts raw map to RetroAutoSummary.
func parseAutoSummary(data map[string]interface{}) RetroAutoSummary {
	summary := RetroAutoSummary{}
	// In production, use proper JSON unmarshaling
	// For now, we store/retrieve as JSON and let Gin handle serialization
	return summary
}

// parseUserContent converts raw map to UserReflection.
func parseUserContent(data map[string]interface{}) UserReflection {
	content := UserReflection{}
	if v, ok := data["what_went_well"].(string); ok {
		content.WhatWentWell = v
	}
	if v, ok := data["what_didnt_go_well"].(string); ok {
		content.WhatDidntGoWell = v
	}
	if v, ok := data["what_learned"].(string); ok {
		content.WhatLearned = v
	}
	if v, ok := data["proud_of"].(string); ok {
		content.ProudOf = v
	}
	if v, ok := data["change_tomorrow"].(string); ok {
		content.ChangeTomorrow = v
	}
	if v, ok := data["additional_notes"].(string); ok {
		content.AdditionalNotes = v
	}
	if arr, ok := data["gratitude"].([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				content.Gratitude = append(content.Gratitude, s)
			}
		}
	}
	return content
}
