package epic

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"
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
		Details(ctx context.Context, req domain.EpicIDRequest) (domain.Epic, error)
		Create(ctx context.Context, req domain.EpicCreateRequest) (domain.EpicIDRequest, error)
		Update(ctx context.Context, req domain.EpicUpdateRequest) error
		UpdateActive(ctx context.Context, req domain.EpicUpdateActiveRequest) error
		Delete(ctx context.Context, req domain.EpicIDRequest) error
		Deactivate(ctx context.Context, req domain.EpicIDRequest) error
	}
)

// NewService initializes or executes NewService behavior.
func NewService(epicRepository Repository) *Service {
	return &Service{repo: epicRepository}
}

// List executes List behavior.
func (s *Service) List(ctx context.Context, req domain.EpicListRequest) ([]domain.Epic, error) {
	if req.ProjectID <= 0 {
		return nil, app.ErrEpicInvalidInput
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

// Details executes Details behavior.
func (s *Service) Details(ctx context.Context, req domain.EpicIDRequest) (domain.Epic, error) {
	if req.ID <= 0 {
		return domain.Epic{}, app.ErrEpicInvalidInput
	}
	item, err := s.repo.Details(ctx, req)
	if err != nil {
		return domain.Epic{}, err
	}
	return withCompletion(item), nil
}

// Create executes Create behavior.
func (s *Service) Create(ctx context.Context, req domain.EpicCreateRequest) (domain.EpicIDRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.ProjectID <= 0 || req.Name == "" {
		return domain.EpicIDRequest{}, app.ErrEpicInvalidInput
	}
	return s.repo.Create(ctx, req)
}

// UpdateEpic executes UpdateEpic behavior.
func (s *Service) UpdateEpic(ctx context.Context, req domain.EpicUpdateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.ID <= 0 || req.Name == "" {
		return app.ErrEpicInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// UpdateActive requires an explicit boolean, including false.
func (s *Service) UpdateActive(ctx context.Context, req domain.EpicUpdateActiveRequest) error {
	if req.ID <= 0 || req.Active == nil {
		return app.ErrEpicInvalidInput
	}
	return s.repo.UpdateActive(ctx, req)
}

// Delete executes Delete behavior.
func (s *Service) Delete(ctx context.Context, req domain.EpicIDRequest) error {
	if req.ID <= 0 {
		return app.ErrEpicInvalidInput
	}
	err := s.repo.Delete(ctx, req)
	if app.IsError(err, app.ErrEpicHasChanges) {
		return s.repo.Deactivate(ctx, req)
	}
	return err
}

func withCompletion(item domain.Epic) domain.Epic {
	item.Completed = 0
	if item.TotalTC != 0 {
		item.Completed = 100 * item.DoneTC / item.TotalTC
	}
	return item
}
