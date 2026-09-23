package epic

import (
	"context"
	"errors"
	"strings"

	"aipm/internal/dto"
)

var (
	// ErrInvalidInput is a package-level value.
	ErrInvalidInput = errors.New("invalid epic input")
	// ErrNotFound is returned when an epic cannot be found.
	ErrNotFound = errors.New("epic not found")
	// ErrEpicHasChanges is returned when deleting an epic that still has changes.
	ErrEpicHasChanges = errors.New("epic has changes")
)

type (
	// Service defines Service values.
	Service struct {
		repo Repository
	}

	// Repository defines Repository values.
	Repository interface {
		List(ctx context.Context, projectID int) ([]dto.Epic, error)
		Get(ctx context.Context, id int) (dto.Epic, error)
		Create(ctx context.Context, req dto.EpicCreateRequest) (dto.Epic, error)
		Update(ctx context.Context, req dto.EpicUpdateRequest) (dto.Epic, error)
		Delete(ctx context.Context, id int) error
	}
)

// NewService initializes or executes NewService behavior.
func NewService(epicRepository Repository) *Service {
	return &Service{repo: epicRepository}
}

// ListEpics executes ListEpics behavior.
func (s *Service) ListEpics(ctx context.Context, req dto.EpicListRequest) ([]dto.Epic, error) {
	if req.ProjectID <= 0 {
		return nil, ErrInvalidInput
	}
	return s.repo.List(ctx, req.ProjectID)
}

// GetEpic executes GetEpic behavior.
func (s *Service) GetEpic(ctx context.Context, req dto.EpicIDRequest) (dto.Epic, error) {
	if req.ID <= 0 {
		return dto.Epic{}, ErrInvalidInput
	}
	return s.repo.Get(ctx, req.ID)
}

// CreateEpic executes CreateEpic behavior.
func (s *Service) CreateEpic(ctx context.Context, req dto.EpicCreateRequest) (dto.Epic, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.ProjectID <= 0 || req.Name == "" {
		return dto.Epic{}, ErrInvalidInput
	}
	return s.repo.Create(ctx, req)
}

// UpdateEpic executes UpdateEpic behavior.
func (s *Service) UpdateEpic(ctx context.Context, req dto.EpicUpdateRequest) (dto.Epic, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.ID <= 0 || req.Name == "" {
		return dto.Epic{}, ErrInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// DeleteEpic executes DeleteEpic behavior.
func (s *Service) DeleteEpic(ctx context.Context, req dto.EpicIDRequest) error {
	if req.ID <= 0 {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, req.ID)
}
