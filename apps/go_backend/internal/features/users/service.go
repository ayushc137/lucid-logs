package users

import (
	"context"
	"strings"

	"github.com/lucid-logs/go-backend/internal/shared/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Service defines business logic for users.
type Service interface {
	Get(ctx context.Context, requesterID, targetID string) (*User, error)
	Update(ctx context.Context, requesterID, targetID string, req *UpdateRequest) (*User, error)
	Delete(ctx context.Context, requesterID, targetID string) error
}

type service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a user service.
func NewService(repo Repository) Service {
	return &service{
		repo:   repo,
		logger: log.With().Str("service", "users").Logger(),
	}
}

func (s *service) Get(ctx context.Context, requesterID, targetID string) (*User, error) {
	requester, target, err := s.fetchAndAuthorize(ctx, requesterID, targetID)
	if err != nil {
		return nil, err
	}

	if requester.ID != target.ID && !requester.IsAdmin {
		return nil, errors.ErrForbidden.WithMessage("Only admins can view other users")
	}

	return target, nil
}

func (s *service) Update(ctx context.Context, requesterID, targetID string, req *UpdateRequest) (*User, error) {
	if req.Email == nil && req.Password == nil {
		return nil, errors.ErrBadRequest.WithMessage("Provide email and/or password to update")
	}

	requester, target, err := s.fetchAndAuthorize(ctx, requesterID, targetID)
	if err != nil {
		return nil, err
	}

	if requester.ID != target.ID && !requester.IsAdmin {
		return nil, errors.ErrForbidden.WithMessage("Only admins can update other users")
	}

	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email == "" {
			return nil, errors.ErrValidationFailed.WithMessage("Email cannot be empty")
		}
		existing, err := s.repo.FindByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != target.ID {
			return nil, errors.ErrConflict.WithMessage("Email already in use")
		}

		target, err = s.repo.UpdateEmail(ctx, target.ID, email)
		if err != nil {
			return nil, err
		}
	}

	if req.Password != nil {
		if err := s.repo.UpdatePassword(ctx, target.ID, *req.Password); err != nil {
			return nil, err
		}
		// Refresh user to capture updated_at change
		target, err = s.repo.FindByID(ctx, target.ID)
		if err != nil {
			return nil, err
		}
	}

	return target, nil
}

func (s *service) Delete(ctx context.Context, requesterID, targetID string) error {
	requester, target, err := s.fetchAndAuthorize(ctx, requesterID, targetID)
	if err != nil {
		return err
	}

	isSelf := requester.ID == target.ID
	if !isSelf && !requester.IsAdmin {
		return errors.ErrForbidden.WithMessage("Only admins can delete other users")
	}

	if target.IsAdmin {
		admins, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return errors.ErrBadRequest.WithMessage("At least one admin user must remain")
		}
	}

	if err := s.repo.Delete(ctx, target.ID); err != nil {
		return err
	}

	s.logger.Info().
		Str("requester_id", requester.ID).
		Str("target_id", target.ID).
		Msg("user deleted")

	return nil
}

func (s *service) fetchAndAuthorize(ctx context.Context, requesterID, targetID string) (*User, *User, error) {
	requester, err := s.repo.FindByID(ctx, requesterID)
	if err != nil {
		return nil, nil, err
	}

	target, err := s.repo.FindByID(ctx, targetID)
	if err != nil {
		return nil, nil, err
	}

	return requester, target, nil
}
