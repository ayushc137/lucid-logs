// Package units provides unit repository for database operations.
package units

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	models "github.com/lucid-logs/go-backend/internal/shared/recordid"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the unit data access interface.
type Repository interface {
	// FindAll retrieves all units (system + user's custom units).
	FindAll(ctx context.Context, userID string) ([]*Unit, error)

	// FindByID retrieves a unit by ID.
	FindByID(ctx context.Context, id string) (*Unit, error)

	// FindSystemUnits retrieves only system-provided units.
	FindSystemUnits(ctx context.Context) ([]*Unit, error)

	// Create creates a new custom unit.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Unit, error)

	// Update updates a custom unit (cannot update system units).
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Unit, error)

	// Delete deletes a custom unit (cannot delete system units).
	Delete(ctx context.Context, id, userID string) error

	// SeedSystemUnits ensures all system units exist in the database.
	SeedSystemUnits(ctx context.Context) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new unit Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: zerolog.New(zerolog.NewConsoleWriter()).With().Str("pkg", "units").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

type unitDB struct {
	ID        models.RecordID      `json:"id,omitempty"`
	Name      string               `json:"name"`
	Symbol    string               `json:"symbol"`
	Type      string               `json:"type"`
	IsSystem  bool                 `json:"is_system"`
	CreatedBy string               `json:"created_by"`
	CreatedAt database.SurrealTime `json:"created_at"`
	UpdatedAt database.SurrealTime `json:"updated_at"`
}

func (u *unitDB) toUnit() *Unit {
	return &Unit{
		ID:        database.ToStringID(u.ID),
		Name:      u.Name,
		Symbol:    u.Symbol,
		Type:      u.Type,
		IsSystem:  u.IsSystem,
		CreatedBy: u.CreatedBy,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindAll(ctx context.Context, userID string) ([]*Unit, error) {
	unitsDB, err := database.QueryAll[unitDB](ctx, r.db, `
		SELECT * FROM units
		WHERE is_system = true OR created_by = $user
		ORDER BY is_system DESC, type ASC, name ASC
	`, map[string]any{
		"user": userID,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	units := make([]*Unit, len(unitsDB))
	for i, u := range unitsDB {
		units[i] = u.toUnit()
	}
	return units, nil
}

func (r *repository) FindByID(ctx context.Context, id string) (*Unit, error) {
	unitID := database.MustRecordID(Table, id)

	unit, err := database.QueryFirst[unitDB](ctx, r.db, `
		SELECT * FROM $id
	`, map[string]any{
		"id": unitID,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}
	if unit == nil {
		return nil, errors.ErrNotFound
	}

	return unit.toUnit(), nil
}

func (r *repository) FindSystemUnits(ctx context.Context) ([]*Unit, error) {
	unitsDB, err := database.QueryAll[unitDB](ctx, r.db, `
		SELECT * FROM units
		WHERE is_system = true
		ORDER BY type ASC, name ASC
	`, nil)
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	units := make([]*Unit, len(unitsDB))
	for i, u := range unitsDB {
		units[i] = u.toUnit()
	}
	return units, nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*Unit, error) {
	now := time.Now().UTC()
	unitType := req.Type
	if unitType == "" {
		unitType = TypeCustom
	}

	result, err := database.QueryFirst[unitDB](ctx, r.db, `
		CREATE units CONTENT {
			name: $name,
			symbol: $symbol,
			type: $type,
			is_system: false,
			created_by: $user,
			created_at: $now,
			updated_at: $now
		}
	`, map[string]any{
		"name":   req.Name,
		"symbol": req.Symbol,
		"type":   unitType,
		"user":   userID,
		"now":    now,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result.toUnit(), nil
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Unit, error) {
	// Verify unit exists and is not system unit
	existing, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.IsSystem {
		return nil, errors.ErrForbidden.WithMessage("Cannot modify system units")
	}
	if existing.CreatedBy != userID {
		return nil, errors.ErrForbidden
	}

	unitID := database.MustRecordID(Table, id)
	now := time.Now().UTC()

	updateData := map[string]any{
		"updated_at": now,
	}
	if req.Name != nil {
		updateData["name"] = *req.Name
	}
	if req.Symbol != nil {
		updateData["symbol"] = *req.Symbol
	}

	result, err := database.QueryFirst[unitDB](ctx, r.db, `
		UPDATE $id MERGE $data RETURN AFTER
	`, map[string]any{
		"id":   unitID,
		"data": updateData,
	})
	if err != nil {
		return nil, errors.ErrDatabase.Wrap(err)
	}

	return result.toUnit(), nil
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

func (r *repository) Delete(ctx context.Context, id, userID string) error {
	// Verify unit exists and is not system unit
	existing, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return errors.ErrForbidden.WithMessage("Cannot delete system units")
	}
	if existing.CreatedBy != userID {
		return errors.ErrForbidden
	}

	unitID := database.MustRecordID(Table, id)

	_, err = database.QueryAll[any](ctx, r.db, `
		DELETE $id
	`, map[string]any{
		"id": unitID,
	})
	if err != nil {
		return errors.ErrDatabase.Wrap(err)
	}

	return nil
}

// =============================================================================
// SEED OPERATION
// =============================================================================

func (r *repository) SeedSystemUnits(ctx context.Context) error {
	now := time.Now().UTC()

	for _, unit := range SystemUnits {
		// Use UPSERT to create or update system units
		_, err := database.QueryAll[any](ctx, r.db, `
			UPSERT $id CONTENT {
				name: $name,
				symbol: $symbol,
				type: $type,
				is_system: true,
				created_by: "",
				created_at: $now,
				updated_at: $now
			}
		`, map[string]any{
			"id":     database.MustRecordID(Table, unit.ID),
			"name":   unit.Name,
			"symbol": unit.Symbol,
			"type":   unit.Type,
			"now":    now,
		})
		if err != nil {
			r.logger.Warn().Err(err).Str("unit", unit.ID).Msg("Failed to seed unit")
		}
	}

	r.logger.Info().Int("count", len(SystemUnits)).Msg("System units seeded")
	return nil
}
