// Package activitylogs provides activity log repository for database operations.
package activitylogs

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
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the activity logs data access interface.
type Repository interface {
	// LogEvent records an activity event.
	LogEvent(ctx context.Context, req *LogEventRequest, userID string) (*ActivityLog, error)

	// FindByEntity retrieves logs for a specific entity.
	FindByEntity(ctx context.Context, entityType, entityID, userID string, params pagination.Params) ([]*ActivityLog, int64, error)

	// FindByUser retrieves all activity logs for a user.
	FindByUser(ctx context.Context, userID string, params pagination.Params, entityType string) ([]*ActivityLog, int64, error)
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new activity logs Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "activitylogs").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

type activityLogDB struct {
	ID          models.RecordID   `json:"id,omitempty"`
	EntityType  string            `json:"entity_type"`
	EntityID    string            `json:"entity_id"`
	Event       string            `json:"event_type"`
	Changes     map[string]any    `json:"changes,omitempty"`
	EntityTitle string            `json:"entity_title,omitempty"`
	EntityIcon  string            `json:"entity_icon,omitempty"`
	CreatedBy   string            `json:"created_by"`
	CreatedAt   database.FlexTime `json:"created_at"`
}

func (l *activityLogDB) toActivityLog() *ActivityLog {
	return &ActivityLog{
		ID:          database.ToStringID(l.ID),
		EntityType:  l.EntityType,
		EntityID:    l.EntityID,
		Event:       l.Event,
		Changes:     l.Changes,
		EntityTitle: l.EntityTitle,
		EntityIcon:  l.EntityIcon,
		CreatedBy:   l.CreatedBy,
		CreatedAt:   l.CreatedAt.Time,
	}
}

// =============================================================================
// LOG EVENT
// =============================================================================

func (r *repository) LogEvent(ctx context.Context, req *LogEventRequest, userID string) (*ActivityLog, error) {
	now := time.Now().UTC()

	changes := req.Changes
	if changes == nil {
		changes = map[string]any{}
	}

	result, err := database.Create[activityLogDB](ctx, r.db, "activity_logs", map[string]any{
		"id":           database.ToStringID(generateRecordID()),
		"entity_type":  req.EntityType,
		"entity_id":    req.EntityID,
		"event_type":   req.Event,
		"changes":      mustJSON(changes),
		"entity_title": req.EntityTitle,
		"entity_icon":  req.EntityIcon,
		"created_by":   userID,
		"created_at":   now,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("entity_type", req.EntityType).
			Str("entity_id", req.EntityID).
			Str("event", req.Event).
			Msg("failed to log activity event")
		return nil, errors.ErrDatabase.Wrap(err)
	}

	r.logger.Debug().
		Str("entity_type", req.EntityType).
		Str("entity_id", req.EntityID).
		Str("event", req.Event).
		Msg("activity event logged")

	return result.toActivityLog(), nil
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByEntity(ctx context.Context, entityType, entityID, userID string, params pagination.Params) ([]*ActivityLog, int64, error) {
	// Count total
	total, err := database.QueryScalar[int64](ctx, r.db, `
		SELECT COUNT(*) FROM activity_logs
		WHERE entity_type = $entity_type
		  AND entity_id = $entity_id
		  AND created_by = $user
	`, map[string]any{
		"entity_type": entityType,
		"entity_id":   entityID,
		"user":        userID,
	})
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	// Fetch logs
	logsDB, err := database.QueryAll[activityLogDB](ctx, r.db, `
		SELECT * FROM activity_logs
		WHERE entity_type = $entity_type
		  AND entity_id = $entity_id
		  AND created_by = $user
		ORDER BY created_at DESC
		LIMIT $limit OFFSET $offset
	`, map[string]any{
		"entity_type": entityType,
		"entity_id":   entityID,
		"user":        userID,
		"limit":       params.Limit,
		"offset":      params.Offset,
	})
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	logs := make([]*ActivityLog, len(logsDB))
	for i, l := range logsDB {
		logs[i] = l.toActivityLog()
	}

	return logs, total, nil
}

func (r *repository) FindByUser(ctx context.Context, userID string, params pagination.Params, entityType string) ([]*ActivityLog, int64, error) {
	// Build query based on entity type filter
	whereClause := "created_by = $user"
	queryVars := map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	}

	if entityType != "" {
		whereClause += " AND entity_type = $entity_type"
		queryVars["entity_type"] = entityType
	}

	// Count total
	total, err := database.QueryScalar[int64](ctx, r.db, `
		SELECT COUNT(*) FROM activity_logs WHERE `+whereClause, queryVars)
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	// Fetch logs
	logsDB, err := database.QueryAll[activityLogDB](ctx, r.db, `
		SELECT * FROM activity_logs
		WHERE `+whereClause+`
		ORDER BY created_at DESC
		LIMIT $limit OFFSET $offset
	`, queryVars)
	if err != nil {
		return nil, 0, errors.ErrDatabase.Wrap(err)
	}

	logs := make([]*ActivityLog, len(logsDB))
	for i, l := range logsDB {
		logs[i] = l.toActivityLog()
	}

	return logs, total, nil
}

// =============================================================================
// ACTIVITY LOGGER INTERFACE IMPLEMENTATION
// =============================================================================

// ActivityLogger provides a simple interface for logging activities from other packages.
type ActivityLogger struct {
	repo Repository
}

// NewActivityLogger creates a new ActivityLogger.
func NewActivityLogger(repo Repository) *ActivityLogger {
	return &ActivityLogger{repo: repo}
}

// Log logs an activity event.
func (l *ActivityLogger) Log(ctx context.Context, entityType, entityID, entityTitle, entityIcon, event, userID string, changes map[string]any) error {
	_, err := l.repo.LogEvent(ctx, &LogEventRequest{
		EntityType:  entityType,
		EntityID:    entityID,
		EntityTitle: entityTitle,
		EntityIcon:  entityIcon,
		Event:       event,
		Changes:     changes,
	}, userID)
	return err
}

// mustJSON serializes a map to a JSON string for SQLite storage.
func mustJSON(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// generateRecordID creates a new table:value record identifier.
func generateRecordID() models.RecordID {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return database.NewRecordID(Table, hex.EncodeToString(bytes))
}
