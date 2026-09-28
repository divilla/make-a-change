package epic

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"strings"
)

type (
	// Service defines Service values.
	Service struct {
		repo Repository
	}

	// Repository defines Repository values.
	Repository interface {
		List(ctx context.Context, projectID int) ([]domain.Epic, error)
		Get(ctx context.Context, id int) (domain.Epic, error)
		Create(ctx context.Context, req domain.EpicCreateRequest) (domain.Epic, error)
		Update(ctx context.Context, req domain.EpicUpdateRequest) (domain.Epic, error)
		Delete(ctx context.Context, id int) error
	}
)

// NewService initializes or executes NewService behavior.
func NewService(epicRepository Repository) *Service {
	return &Service{repo: epicRepository}
}

// ListEpics executes ListEpics behavior.
func (s *Service) ListEpics(ctx context.Context, req domain.EpicListRequest) ([]domain.Epic, error) {
	if req.ProjectID <= 0 {
		return nil, apperror.ErrEpicInvalidInput
	}
	return s.repo.List(ctx, req.ProjectID)
}

// GetEpic executes GetEpic behavior.
func (s *Service) GetEpic(ctx context.Context, req domain.EpicIDRequest) (domain.Epic, error) {
	if req.ID <= 0 {
		return domain.Epic{}, apperror.ErrEpicInvalidInput
	}
	return s.repo.Get(ctx, req.ID)
}

// CreateEpic executes CreateEpic behavior.
func (s *Service) CreateEpic(ctx context.Context, req domain.EpicCreateRequest) (domain.Epic, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.ProjectID <= 0 || req.Name == "" {
		return domain.Epic{}, apperror.ErrEpicInvalidInput
	}
	return s.repo.Create(ctx, req)
}

// UpdateEpic executes UpdateEpic behavior.
func (s *Service) UpdateEpic(ctx context.Context, req domain.EpicUpdateRequest) (domain.Epic, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.ID <= 0 || req.Name == "" {
		return domain.Epic{}, apperror.ErrEpicInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// DeleteEpic executes DeleteEpic behavior.
func (s *Service) DeleteEpic(ctx context.Context, req domain.EpicIDRequest) error {
	if req.ID <= 0 {
		return apperror.ErrEpicInvalidInput
	}
	return s.repo.Delete(ctx, req.ID)
}
