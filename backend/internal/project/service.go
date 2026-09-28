package project

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
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

// ListProjects executes ListProjects behavior.
func (s *Service) ListProjects(ctx context.Context) ([]domain.Project, error) {
	return s.repo.List(ctx)
}

// GetProject executes GetProject behavior.
func (s *Service) GetProject(ctx context.Context, req domain.ProjectIDRequest) (domain.Project, error) {
	if req.ID <= 0 {
		return domain.Project{}, apperror.ErrProjectInvalidInput
	}
	return s.repo.Get(ctx, req)
}

// CreateProject executes CreateProject behavior.
func (s *Service) CreateProject(ctx context.Context, req domain.ProjectCreateRequest) (domain.ProjectIDRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return domain.ProjectIDRequest{}, apperror.ErrProjectInvalidInput
	}
	return s.repo.Create(ctx, req)
}

// UpdateProject executes UpdateProject behavior.
func (s *Service) UpdateProject(ctx context.Context, req domain.ProjectUpdateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.ID <= 0 || req.Name == "" {
		return apperror.ErrProjectInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// DeleteProject executes DeleteProject behavior.
func (s *Service) DeleteProject(ctx context.Context, req domain.ProjectIDRequest) error {
	if req.ID <= 0 {
		return apperror.ErrProjectInvalidInput
	}
	return s.repo.Delete(ctx, req)
}

// Config returns only the selected configuration, without a default fallback.
func (s *Service) Config(ctx context.Context, req domain.ProjectIDRequest) (domain.Config, error) {
	if req.ID <= 0 {
		return domain.Config{}, apperror.ErrProjectInvalidInput
	}
	return s.repo.Config(ctx, req)
}
