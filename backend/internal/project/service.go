package project

import (
	"context"
	"errors"
	"strings"

	"aipm/internal/dto"
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
func (s *Service) ListProjects(ctx context.Context) ([]dto.Project, error) {
	return s.repo.List(ctx)
}

// GetProject executes GetProject behavior.
func (s *Service) GetProject(ctx context.Context, req dto.ProjectIDRequest) (dto.Project, error) {
	if req.ID <= 0 {
		return dto.Project{}, ErrInvalidInput
	}
	return s.repo.Get(ctx, req.ID)
}

// CreateProject executes CreateProject behavior.
func (s *Service) CreateProject(ctx context.Context, req dto.ProjectCreateRequest) (dto.Project, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return dto.Project{}, ErrInvalidInput
	}
	return s.repo.Create(ctx, name)
}

// UpdateProject executes UpdateProject behavior.
func (s *Service) UpdateProject(ctx context.Context, req dto.ProjectUpdateRequest) (dto.Project, error) {
	name := strings.TrimSpace(req.Name)
	if req.ID <= 0 || name == "" {
		return dto.Project{}, ErrInvalidInput
	}
	return s.repo.Update(ctx, req.ID, name)
}

// DeleteProject executes DeleteProject behavior.
func (s *Service) DeleteProject(ctx context.Context, req dto.ProjectIDRequest) error {
	if req.ID <= 0 {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, req.ID)
}
