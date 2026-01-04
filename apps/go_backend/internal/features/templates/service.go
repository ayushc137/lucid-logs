package templates

import (
	"context"

	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/lucid-logs/go-backend/internal/shared/pagination"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the template business logic interface.
type Service interface {
	// List retrieves paginated templates for a user.
	List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*TaskTemplate], error)

	// Get retrieves a single template by ID.
	Get(ctx context.Context, id, userID string) (*TaskTemplate, error)

	// GetQuickLog retrieves quick-log templates for a user.
	GetQuickLog(ctx context.Context, userID string) ([]*TaskTemplate, error)

	// Create creates a new template.
	Create(ctx context.Context, req *CreateRequest, userID string) (*TaskTemplate, error)

	// Update updates an existing template.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*TaskTemplate, error)

	// Delete soft-deletes a template.
	Delete(ctx context.Context, id, userID string) error

	// IncrementUseCount updates the template usage stats.
	IncrementUseCount(ctx context.Context, id string) error
}

// =============================================================================
// GOAL TEMPLATE CREATOR
// =============================================================================

// GoalTemplateCreator implements goals.TemplateCreator interface.
// This allows goals service to auto-create templates without circular imports.
type GoalTemplateCreator struct {
	repo Repository
}

// NewGoalTemplateCreator creates a new GoalTemplateCreator.
func NewGoalTemplateCreator(repo Repository) *GoalTemplateCreator {
	return &GoalTemplateCreator{repo: repo}
}

// CreateForGoal creates a template linked to a goal.
func (c *GoalTemplateCreator) CreateForGoal(ctx context.Context, goal *goals.Goal, userID string) (string, error) {
	// Infer icon from goal title or use default
	icon := goal.Icon
	if icon == "" {
		icon = "⚡"
	}

	// Build template from goal
	req := &CreateRequest{
		Title:       goal.Title,
		Description: goal.Description,
		Icon:        icon,
		IsQuickLog:  true,
	}

	// Set quantity from goal target
	if goal.Target != nil {
		req.QuantityEnabled = true
		req.QuantityDefault = goal.Target.Value
		req.QuantityStep = 1

		// Link to the goal via template_goals
		req.GoalLinks = []GoalLinkInput{{
			GoalID:        goal.ID,
			AutoLinkTasks: true,
		}}
	}

	template, err := c.repo.Create(ctx, req, userID)
	if err != nil {
		return "", err
	}

	log.Info().
		Str("template_id", template.ID).
		Str("goal_id", goal.ID).
		Msg("template auto-created for goal")

	return template.ID, nil
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new template Service.
func NewService(repo Repository) Service {
	return &service{
		repo:   repo,
		logger: log.With().Str("service", "templates").Logger(),
	}
}

// =============================================================================
// LIST
// =============================================================================

func (s *service) List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*TaskTemplate], error) {
	templates, total, err := s.repo.FindPaginated(ctx, userID, params)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list templates")
		return nil, err
	}

	s.logger.Debug().
		Str("user_id", userID).
		Int("count", len(templates)).
		Int64("total", total).
		Msg("templates listed")

	resp := pagination.NewResponse(templates, total, params)
	return &resp, nil
}

// =============================================================================
// GET
// =============================================================================

func (s *service) Get(ctx context.Context, id, userID string) (*TaskTemplate, error) {
	template, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to get template")
		return nil, err
	}

	return template, nil
}

func (s *service) GetQuickLog(ctx context.Context, userID string) ([]*TaskTemplate, error) {
	templates, err := s.repo.FindQuickLog(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get quick log templates")
		return nil, err
	}

	s.logger.Debug().
		Str("user_id", userID).
		Int("count", len(templates)).
		Msg("quick log templates fetched")

	return templates, nil
}

// =============================================================================
// CREATE
// =============================================================================

func (s *service) Create(ctx context.Context, req *CreateRequest, userID string) (*TaskTemplate, error) {
	template, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to create template")
		return nil, err
	}

	s.logger.Info().
		Str("template_id", template.ID).
		Str("user_id", userID).
		Msg("template created")

	return template, nil
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*TaskTemplate, error) {
	template, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		if errors.Is(err, errors.ErrBadRequest) {
			return nil, err
		}
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to update template")
		return nil, err
	}

	s.logger.Info().
		Str("template_id", id).
		Str("user_id", userID).
		Msg("template updated")

	return template, nil
}

// =============================================================================
// DELETE
// =============================================================================

func (s *service) Delete(ctx context.Context, id, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return errors.ErrNotFound
		}
		if errors.Is(err, errors.ErrBadRequest) {
			return err
		}
		s.logger.Error().Err(err).Str("template_id", id).Msg("failed to delete template")
		return err
	}

	s.logger.Info().
		Str("template_id", id).
		Str("user_id", userID).
		Msg("template deleted")

	return nil
}

// =============================================================================
// USE COUNT
// =============================================================================

func (s *service) IncrementUseCount(ctx context.Context, id string) error {
	return s.repo.IncrementUseCount(ctx, id)
}
