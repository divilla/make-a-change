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
	return s.repo.Get(ctx, req.ID)
}

// CreateProject executes CreateProject behavior.
func (s *Service) CreateProject(ctx context.Context, req domain.ProjectCreateRequest) (domain.Project, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.Project{}, apperror.ErrProjectInvalidInput
	}
	return s.repo.Create(ctx, name)
}

// UpdateProject executes UpdateProject behavior.
func (s *Service) UpdateProject(ctx context.Context, req domain.ProjectUpdateRequest) (domain.Project, error) {
	name := strings.TrimSpace(req.Name)
	if req.ID <= 0 || name == "" {
		return domain.Project{}, apperror.ErrProjectInvalidInput
	}
	return s.repo.Update(ctx, req.ID, name)
}

// DeleteProject executes DeleteProject behavior.
func (s *Service) DeleteProject(ctx context.Context, req domain.ProjectIDRequest) error {
	if req.ID <= 0 {
		return apperror.ErrProjectInvalidInput
	}
	return s.repo.Delete(ctx, req.ID)
}
