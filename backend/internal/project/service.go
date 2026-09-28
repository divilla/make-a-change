package project

import (
	"context"
	"errors"
	"strings"

	"mch_api/internal/domain"
)

var (
	// ErrInvalidInput is a package-level value.
	ErrInvalidInput = errors.New("invalid project input")
	// ErrNotFound is returned when a project cannot be found.
	ErrNotFound = errors.New("project not found")
	// ErrProjectHasChanges is returned when deleting a project that still has changes.
	ErrProjectHasChanges = errors.New("project has changes")
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
		return domain.Project{}, ErrInvalidInput
	}
	return s.repo.Get(ctx, req.ID)
}

// CreateProject executes CreateProject behavior.
func (s *Service) CreateProject(ctx context.Context, req domain.ProjectCreateRequest) (domain.Project, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.Project{}, ErrInvalidInput
	}
	return s.repo.Create(ctx, name)
}

// UpdateProject executes UpdateProject behavior.
func (s *Service) UpdateProject(ctx context.Context, req domain.ProjectUpdateRequest) (domain.Project, error) {
	name := strings.TrimSpace(req.Name)
	if req.ID <= 0 || name == "" {
		return domain.Project{}, ErrInvalidInput
	}
	return s.repo.Update(ctx, req.ID, name)
}

// DeleteProject executes DeleteProject behavior.
func (s *Service) DeleteProject(ctx context.Context, req domain.ProjectIDRequest) error {
	if req.ID <= 0 {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, req.ID)
}
