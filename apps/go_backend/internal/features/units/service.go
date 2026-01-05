// Package units provides unit management functionality.
package units

import (
	"context"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the unit business logic interface.
type Service interface {
	// List retrieves all units (system + user's custom units).
	List(ctx context.Context, userID string, systemOnly bool) (*UnitListResponse, error)

	// Get retrieves a unit by ID.
	Get(ctx context.Context, id string) (*Unit, error)

	// Create creates a new custom unit.
	Create(ctx context.Context, req *CreateRequest, userID string) (*Unit, error)

	// Update updates a custom unit.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Unit, error)

	// Delete deletes a custom unit.
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo Repository
}

// NewService creates a new unit service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// List retrieves all units or only system units.
func (s *service) List(ctx context.Context, userID string, systemOnly bool) (*UnitListResponse, error) {
	var units []*Unit
	var err error

	if systemOnly {
		units, err = s.repo.FindSystemUnits(ctx)
	} else {
		units, err = s.repo.FindAll(ctx, userID)
	}

	if err != nil {
		return nil, err
	}

	return &UnitListResponse{
		Items:      units,
		SystemOnly: systemOnly,
	}, nil
}

// Get retrieves a unit by ID.
func (s *service) Get(ctx context.Context, id string) (*Unit, error) {
	return s.repo.FindByID(ctx, id)
}

// Create creates a new custom unit.
func (s *service) Create(ctx context.Context, req *CreateRequest, userID string) (*Unit, error) {
	return s.repo.Create(ctx, req, userID)
}

// Update updates a custom unit.
func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Unit, error) {
	return s.repo.Update(ctx, id, req, userID)
}

// Delete deletes a custom unit.
func (s *service) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}
