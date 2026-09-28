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
		List(ctx context.Context, req domain.EpicListRequest) ([]domain.Epic, error)
		Get(ctx context.Context, req domain.EpicIDRequest) (domain.Epic, error)
		Create(ctx context.Context, req domain.EpicCreateRequest) (domain.EpicIDRequest, error)
		Update(ctx context.Context, req domain.EpicUpdateRequest) error
		Delete(ctx context.Context, req domain.EpicIDRequest) error
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
	items, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i] = withCompletion(items[i])
	}
	return items, nil
}

// GetEpic executes GetEpic behavior.
func (s *Service) GetEpic(ctx context.Context, req domain.EpicIDRequest) (domain.Epic, error) {
	if req.ID <= 0 {
		return domain.Epic{}, apperror.ErrEpicInvalidInput
	}
	item, err := s.repo.Get(ctx, req)
	if err != nil {
		return domain.Epic{}, err
	}
	return withCompletion(item), nil
}

// CreateEpic executes CreateEpic behavior.
func (s *Service) CreateEpic(ctx context.Context, req domain.EpicCreateRequest) (domain.EpicIDRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.ProjectID <= 0 || req.Name == "" {
		return domain.EpicIDRequest{}, apperror.ErrEpicInvalidInput
	}
	return s.repo.Create(ctx, req)
}

// UpdateEpic executes UpdateEpic behavior.
func (s *Service) UpdateEpic(ctx context.Context, req domain.EpicUpdateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.ID <= 0 || req.Name == "" {
		return apperror.ErrEpicInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// DeleteEpic executes DeleteEpic behavior.
func (s *Service) DeleteEpic(ctx context.Context, req domain.EpicIDRequest) error {
	if req.ID <= 0 {
		return apperror.ErrEpicInvalidInput
	}
	return s.repo.Delete(ctx, req)
}

func withCompletion(item domain.Epic) domain.Epic {
	item.Completed = 0
	if item.TotalTC != 0 {
		item.Completed = 100 * item.DoneTC / item.TotalTC
	}
	return item
}
