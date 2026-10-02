package project

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"strings"
)

// Service defines Service values.
type Service struct {
	repo Repository
}

// NewService initializes or executes NewService behavior.
func NewService(projectRepository Repository) *Service {
	return &Service{repo: projectRepository}
}

// List executes List behavior.
func (s *Service) List(ctx context.Context) ([]domain.Project, error) {
	return s.repo.List(ctx)
}

// Details executes Details behavior.
func (s *Service) Details(ctx context.Context, req domain.ProjectIDRequest) (domain.Project, error) {
	if req.ID <= 0 {
		return domain.Project{}, app.ErrProjectInvalidInput
	}
	return s.repo.Details(ctx, req)
}

// Create executes Create behavior.
func (s *Service) Create(ctx context.Context, req domain.ProjectCreateRequest) (domain.ProjectIDRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return domain.ProjectIDRequest{}, app.ErrProjectInvalidInput
	}
	return s.repo.Create(ctx, req)
}

// UpdateProject executes UpdateProject behavior.
func (s *Service) UpdateProject(ctx context.Context, req domain.ProjectUpdateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.ID <= 0 || req.Name == "" {
		return app.ErrProjectInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// Delete executes Delete behavior.
func (s *Service) Delete(ctx context.Context, req domain.ProjectIDRequest) error {
	if req.ID <= 0 {
		return app.ErrProjectInvalidInput
	}
	err := s.repo.Delete(ctx, req)
	if app.IsError(err, app.ErrProjectHasChanges) {
		return s.repo.Deactivate(ctx, req)
	}
	return err
}

// Config returns only the selected configuration, without a default fallback.
func (s *Service) Config(ctx context.Context, req domain.ProjectIDRequest) (domain.Config, error) {
	if req.ID <= 0 {
		return domain.Config{}, app.ErrProjectInvalidInput
	}
	return s.repo.Config(ctx, req)
}
