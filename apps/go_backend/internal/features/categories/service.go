package categories

import (
	"context"

	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the category business logic interface.
type Service interface {
	List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Category], error)
	Get(ctx context.Context, id, userID string) (*Category, error)
	Create(ctx context.Context, req *CreateRequest, userID string) (*Category, error)
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Category, error)
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new category Service.
func NewService(repo Repository) Service {
	return &service{
		repo:   repo,
		logger: log.With().Str("service", "categories").Logger(),
	}
}

func (s *service) List(ctx context.Context, userID string, params pagination.Params) (*pagination.Response[*Category], error) {
	cats, total, err := s.repo.FindPaginated(ctx, userID, params)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list categories")
		return nil, err
	}

	resp := pagination.NewResponse(cats, total, params)
	return &resp, nil
}

func (s *service) Get(ctx context.Context, id, userID string) (*Category, error) {
	cat, err := s.repo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		return nil, err
	}
	return cat, nil
}

func (s *service) Create(ctx context.Context, req *CreateRequest, userID string) (*Category, error) {
	cat, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrCategoryNameExists) {
			return nil, errors.ErrCategoryNameExists
		}
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to create category")
		return nil, err
	}

	s.logger.Info().Str("category_id", cat.ID).Msg("category created")
	return cat, nil
}

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*Category, error) {
	cat, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, errors.ErrNotFound
		}
		if errors.Is(err, errors.ErrCategoryNameExists) {
			return nil, errors.ErrCategoryNameExists
		}
		s.logger.Error().Err(err).Str("category_id", id).Msg("failed to update category")
		return nil, err
	}

	s.logger.Info().Str("category_id", id).Msg("category updated")
	return cat, nil
}

func (s *service) Delete(ctx context.Context, id, userID string) error {
	err := s.repo.Delete(ctx, id, userID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return errors.ErrNotFound
		}
		s.logger.Error().Err(err).Str("category_id", id).Msg("failed to delete category")
		return err
	}

	s.logger.Info().Str("category_id", id).Msg("category deleted")
	return nil
}
